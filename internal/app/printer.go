package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const printerFeedLinePixels = 24

// sendBitmapToPrinter sends a 1-bit bitmap to the ESP32's /api/print/bitmap endpoint.
func sendBitmapToPrinter(printerURL string, bitmap *Bitmap, feedLines int) (map[string]any, error) {
	bitmapToSend := bitmapWithTrailingBlankRows(bitmap, feedLines)
	body := bytes.NewReader(bitmapToSend.Data)

	req, err := http.NewRequest("POST", printerURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Width", fmt.Sprintf("%d", bitmapToSend.Width))
	req.Header.Set("X-Height", fmt.Sprintf("%d", bitmapToSend.Height))
	// Feed is baked into the bitmap as trailing blank raster rows. The firmware
	// still supports X-Feed, but several printer runs showed ESC d n after a
	// bitmap was not visually reliable on this mechanism.
	req.Header.Set("X-Feed", "0")

	client := &http.Client{Timeout: 30 * time.Second}
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
