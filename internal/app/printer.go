package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	printerFeedLinePixels         = 24
	maxPrinterBitmapBodyBytes     = 1024 * 1024 // 1MB hard limit; anything above maxSafePrinterBitmapBodyBytes gets segmented
	maxSafePrinterBitmapBodyBytes = 36 * 1024   // ESP32 httpd cannot reliably receive > ~38 KiB
	// At 9600 baud, a 90 KiB bitmap can take ~77 s to UART-transfer.
	// Allow generous headroom so large prints outlive the transfer.
	printerHTTPTimeout = 120 * time.Second
)

// sendBitmapToPrinter sends a 1-bit bitmap to the ESP32's /api/print/bitmap endpoint.
// If the bitmap exceeds the ESP32's safe receive limit (~38 KiB), it is split into
// segments that are sent as sequential print commands. Only the final segment carries
// the paper feed (baked as trailing blank rows + X-Feed header).
//
// Note: ESC d n (X-Feed) alone is not visually reliable on the K118 mechanism —
// the paper may not actually advance. Baking blank raster rows into the bitmap
// ensures the printer physically prints the feed gap. We set X-Feed as a fallback
// so the firmware can issue ESC d n too, but the baked rows are the primary feed.
func sendBitmapToPrinter(printerURL string, bitmap *Bitmap, feedLines int) (map[string]any, error) {
	if bitmap == nil {
		return nil, fmt.Errorf("bitmap is nil")
	}
	if len(bitmap.Data) > maxPrinterBitmapBodyBytes {
		return nil, fmt.Errorf("bitmap too large: got %d bytes, max %d bytes; use a shorter layout or future segmented print endpoint", len(bitmap.Data), maxPrinterBitmapBodyBytes)
	}

	if feedLines > 20 {
		feedLines = 20
	}

	// Bake feed rows into the bitmap for reliable paper advance.
	// ESC d n (X-Feed) alone is not reliable on this printer mechanism.
	bitmapWithFeed := bitmapWithTrailingBlankRows(bitmap, feedLines)

	// If the bitmap (with feed) fits in a single safe request, send it directly.
	if len(bitmapWithFeed.Data) <= maxSafePrinterBitmapBodyBytes {
		return sendSingleBitmap(printerURL, bitmapWithFeed, feedLines)
	}

	// Split into segments that each fit under the safe limit.
	// Feed rows are baked into the last segment.
	segments := splitBitmap(bitmapWithFeed, maxSafePrinterBitmapBodyBytes)
	var lastResp map[string]any
	for i, seg := range segments {
		segFeed := 0
		if i == len(segments)-1 {
			segFeed = feedLines
		}
		resp, err := sendSingleBitmap(printerURL, seg, segFeed)
		if err != nil {
			return nil, fmt.Errorf("segment %d/%d failed: %w", i+1, len(segments), err)
		}
		lastResp = resp
	}
	lastResp["segments"] = len(segments)
	return lastResp, nil
}

// printerBaseFromBitmapURL derives the device base URL (scheme://host) from a
// full bitmap endpoint URL like http://host/api/print/bitmap.
func printerBaseFromBitmapURL(bitmapURL string) string {
	if i := strings.Index(bitmapURL, "/api/"); i >= 0 {
		return bitmapURL[:i]
	}
	return strings.TrimRight(bitmapURL, "/")
}

// setPrinterHeat sets the K118 density (0..39, the "heat"/darkness knob) and/or
// speed (lower = more heat dwell = darker) before a print. Either may be nil to
// leave the firmware default. baseOrBitmapURL may be the device base URL or the
// full bitmap endpoint URL. Errors are returned so callers can surface them.
func setPrinterHeat(baseOrBitmapURL string, density, speed *int) error {
	base := printerBaseFromBitmapURL(baseOrBitmapURL)
	if density != nil {
		if err := postPrinterJSON(base+"/api/printer/density", fmt.Sprintf(`{"density":%d}`, *density)); err != nil {
			return fmt.Errorf("set density: %w", err)
		}
	}
	if speed != nil {
		if err := postPrinterJSON(base+"/api/printer/speed", fmt.Sprintf(`{"speed":%d}`, *speed)); err != nil {
			return fmt.Errorf("set speed: %w", err)
		}
	}
	return nil
}

func postPrinterJSON(url, body string) error {
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("printer returned %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// sendSingleBitmap sends one bitmap chunk to the printer.
func sendSingleBitmap(printerURL string, bitmap *Bitmap, feedLines int) (map[string]any, error) {
	body := bytes.NewReader(bitmap.Data)

	req, err := http.NewRequest("POST", printerURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Width", fmt.Sprintf("%d", bitmap.Width))
	req.Header.Set("X-Height", fmt.Sprintf("%d", bitmap.Height))
	// Use X-Feed as a supplementary feed signal. The primary feed is the baked
	// blank rows in the bitmap, but the firmware also issues ESC d n as backup.
	req.Header.Set("X-Feed", fmt.Sprintf("%d", feedLines))
	req.Header.Set("Connection", "close")

	// Use a fresh connection per request. The ESP32 httpd has short
	// recv/send timeouts and the UART bitmap transfer can take seconds,
	// so reusing a stale connection causes unexpected EOF.
	transport := &http.Transport{
		DisableKeepAlives: true,
		IdleConnTimeout:   printerHTTPTimeout,
	}
	client := &http.Client{
		Timeout:   printerHTTPTimeout,
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("printer request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("printer returned %d: %s", resp.StatusCode, respBody)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return map[string]any{"raw": string(respBody)}, nil
	}
	return result, nil
}

// splitBitmap divides a bitmap into horizontal segments that each fit under maxBytes.
// Each segment has the same width and BytesPerRow; only the height and data are sliced.
func splitBitmap(bitmap *Bitmap, maxBytes int) []*Bitmap {
	if len(bitmap.Data) <= maxBytes || bitmap.BytesPerRow == 0 {
		return []*Bitmap{bitmap}
	}

	// Calculate max rows per segment.
	maxRows := maxBytes / bitmap.BytesPerRow
	if maxRows <= 0 {
		maxRows = 1
	}

	var segments []*Bitmap
	remaining := bitmap.Height
	offset := 0
	for remaining > 0 {
		rows := remaining
		if rows > maxRows {
			rows = maxRows
		}
		start := offset * bitmap.BytesPerRow
		end := start + rows*bitmap.BytesPerRow
		if end > len(bitmap.Data) {
			end = len(bitmap.Data)
		}
		segments = append(segments, &Bitmap{
			Width:       bitmap.Width,
			Height:      rows,
			BytesPerRow: bitmap.BytesPerRow,
			Data:        bitmap.Data[start:end],
		})
		offset += rows
		remaining -= rows
	}
	return segments
}

// bitmapWithTrailingBlankRows appends blank raster rows to a bitmap for feed.
// Kept for compatibility with tests and the dry-run path, but the actual
// sendBitmapToPrinter now uses X-Feed instead of baked rows.
func bitmapWithTrailingBlankRows(bitmap *Bitmap, feedLines int) *Bitmap {
	if bitmap == nil || feedLines <= 0 || bitmap.BytesPerRow <= 0 {
		return bitmap
	}
	if feedLines > 20 {
		feedLines = 20
	}

	blankRows := feedLines * printerFeedLinePixels
	if blankRows <= 0 {
		return bitmap
	}

	newData := make([]byte, len(bitmap.Data)+bitmap.BytesPerRow*blankRows)
	copy(newData, bitmap.Data)
	return &Bitmap{
		Width:       bitmap.Width,
		Height:      bitmap.Height + blankRows,
		BytesPerRow: bitmap.BytesPerRow,
		Data:        newData,
	}
}
