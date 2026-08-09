package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	// defaultHeatGapDensity is the density used for the non-overridden bands of
	// a mixed-heat page whose layout sets no page-level printerDensity. 38 is
	// the paper-verified text density (ALMANACH-PIXELFONT); the firmware has no
	// way to read the current density back, so gaps need an explicit value once
	// any block override has mutated the printer state.
	defaultHeatGapDensity = 38

	printerFeedLinePixels         = 24
	maxPrinterBitmapBodyBytes     = 1024 * 1024 // 1MB hard limit; anything above maxSafePrinterBitmapBodyBytes gets segmented
	maxSafePrinterBitmapBodyBytes = 36 * 1024   // ESP32 httpd cannot reliably receive > ~38 KiB
	// At 9600 baud, a 90 KiB bitmap can take ~77 s to UART-transfer.
	// Allow generous headroom so large prints outlive the transfer.
	printerHTTPTimeout = 120 * time.Second
)

// sendBitmapWithOptions applies explicit page-level printer state before sending
// the bitmap. Setting failures are fatal: printing with stale heat while claiming
// success would violate the layout contract and waste paper.
func sendBitmapWithOptions(printerURL string, bitmap *Bitmap, feedLines int, opts RenderOptions) (map[string]any, error) {
	if opts.PrinterSpeed > 0 {
		if err := setPrinterSpeed(printerURL, opts.PrinterSpeed); err != nil {
			return nil, fmt.Errorf("set printer speed %d: %w", opts.PrinterSpeed, err)
		}
	}
	if opts.PrinterDensitySet {
		if err := setPrinterDensity(printerURL, opts.PrinterDensity); err != nil {
			return nil, fmt.Errorf("set printer density %d: %w", opts.PrinterDensity, err)
		}
	}
	return sendBitmapToPrinter(printerURL, bitmap, feedLines)
}

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

// heatGapDensity resolves the density used for the non-overridden bands of a
// mixed-heat page: the page-level density when set, else the paper-verified
// text default. Never 0 — once a block override mutates the printer's density
// state there is no way to read the previous value back, so gap bands must
// always set an explicit density. An explicit zero is the firmware's minimum
// density and must not be confused with an unset page-level option.
func heatGapDensity(pageDensity int, pageDensitySet bool) int {
	if pageDensitySet {
		return pageDensity
	}
	return defaultHeatGapDensity
}

// densityBand is a contiguous run of bitmap rows [YStart, YEnd) to print at
// Density. Every generated band has an explicit density, including zero.
type densityBand struct {
	YStart  int
	YEnd    int
	Density int
}

// densityBands turns per-region heat requests into a top-to-bottom cover of all
// rows: each heat region becomes a band at its density, gaps get defaultDensity.
// Assumes regions do not overlap (block boxes are vertically disjoint).
func densityBands(height int, regions []HeatRegion, defaultDensity int) []densityBand {
	sorted := append([]HeatRegion(nil), regions...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].YStart < sorted[j].YStart })
	var bands []densityBand
	y := 0
	for _, r := range sorted {
		ys := clampInt(r.YStart, 0, height)
		ye := clampInt(r.YEnd, 0, height)
		if ye <= ys || ye <= y {
			continue
		}
		if ys > y {
			bands = append(bands, densityBand{y, ys, defaultDensity})
		}
		bands = append(bands, densityBand{maxInt(ys, y), ye, r.Density})
		y = ye
	}
	if y < height {
		bands = append(bands, densityBand{y, height, defaultDensity})
	}
	return bands
}

// sliceBitmapRows returns the sub-bitmap covering rows [y0, y1).
func sliceBitmapRows(bm *Bitmap, y0, y1 int) *Bitmap {
	y0 = clampInt(y0, 0, bm.Height)
	y1 = clampInt(y1, 0, bm.Height)
	if y1 <= y0 {
		return &Bitmap{Width: bm.Width, Height: 0, BytesPerRow: bm.BytesPerRow}
	}
	return &Bitmap{
		Width:       bm.Width,
		Height:      y1 - y0,
		BytesPerRow: bm.BytesPerRow,
		Data:        bm.Data[y0*bm.BytesPerRow : y1*bm.BytesPerRow],
	}
}

// sendBitmapWithHeat prints a bitmap in density bands (Phase 6, per-segment
// heat): it sets the printer density before each band, so a single page can burn
// text hot and photos cool. Feed rows are baked into the final band only.
func sendBitmapWithHeat(printerURL string, bitmap *Bitmap, feedLines int, bands []densityBand) (map[string]any, error) {
	if feedLines > 20 {
		feedLines = 20
	}
	type unit struct {
		bm      *Bitmap
		density int
	}
	var units []unit
	for _, band := range bands {
		s := sliceBitmapRows(bitmap, band.YStart, band.YEnd)
		if s.Height == 0 {
			continue
		}
		units = append(units, unit{bm: s, density: band.Density})
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("no printable rows")
	}
	units[len(units)-1].bm = bitmapWithTrailingBlankRows(units[len(units)-1].bm, feedLines)

	var lastResp map[string]any
	total := 0
	for ui, u := range units {
		if err := setPrinterDensity(printerURL, u.density); err != nil {
			log.Printf("warning: set density %d for heat band failed: %v", u.density, err)
		}
		segs := splitBitmap(u.bm, maxSafePrinterBitmapBodyBytes)
		for si, seg := range segs {
			segFeed := 0
			if ui == len(units)-1 && si == len(segs)-1 {
				segFeed = feedLines
			}
			resp, err := sendSingleBitmap(printerURL, seg, segFeed)
			if err != nil {
				return nil, fmt.Errorf("heat band %d segment %d: %w", ui+1, si+1, err)
			}
			lastResp = resp
			total++
		}
	}
	if lastResp == nil {
		lastResp = map[string]any{}
	}
	lastResp["segments"] = total
	return lastResp, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// setPrinterDensity POSTs a head density/heat value to the printer's
// /api/printer/density endpoint, derived from the bitmap print URL. Density is a
// per-page render option (DSL v2, Phase 5): text prints best hotter (~38),
// photos cooler (~20). Returns an error the caller may choose to warn on rather
// than abort.
func setPrinterDensity(printerBitmapURL string, density int) error {
	return postPrinterSetting(printerBitmapURL, "/api/printer/density", "density", density)
}

// setPrinterSpeed POSTs a feed speed to the printer's /api/printer/speed
// endpoint. The firmware accepts only a discrete set of speeds (validated in
// validateRenderOptions); slower speeds give the head more dwell time per row,
// which reduces heat-related artifacts on dense pages.
func setPrinterSpeed(printerBitmapURL string, speed int) error {
	return postPrinterSetting(printerBitmapURL, "/api/printer/speed", "speed", speed)
}

// postPrinterSetting POSTs {key: value} to a printer settings endpoint, deriving
// the printer base URL from the bitmap print URL.
func postPrinterSetting(printerBitmapURL, path, key string, value int) error {
	base := strings.TrimSuffix(printerBitmapURL, "/api/print/bitmap")
	if base == printerBitmapURL {
		// URL didn't match the expected shape; best-effort strip of the path.
		if i := strings.Index(printerBitmapURL, "/api/"); i >= 0 {
			base = printerBitmapURL[:i]
		}
	}
	url := base + path
	payload, err := json.Marshal(map[string]int{key: value})
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create %s request: %w", key, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("set %s request failed: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set %s returned %d: %s", key, resp.StatusCode, body)
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
