package app

import (
	"encoding/json"
	"image"
	"math"
	"sort"

	layoutv1 "github.com/go-go-golems/almanach/gen/almanach/layout/v1"
)

// blockMetric is one block's bounding box as measured in the browser, in CSS px
// relative to the screenshotted element's top. At deviceScaleFactor 1 (and after
// supersample downscaling) these map 1:1 to final bitmap rows.
type blockMetric struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"`
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
}

// collectBlockMetricsJS returns JS that measures every .block-wrap[data-block-id]
// inside the screenshot selector, relative to that selector's top.
func collectBlockMetricsJS(selector string) string {
	sel, _ := json.Marshal(selector)
	return `(() => {
	const root = document.querySelector(` + string(sel) + `);
	if (!root) return [];
	const rootTop = root.getBoundingClientRect().top;
	return Array.from(root.querySelectorAll('.block-wrap[data-block-id]')).map((el) => {
		const r = el.getBoundingClientRect();
		return {
			id: el.getAttribute('data-block-id'),
			type: el.getAttribute('data-block-type'),
			top: r.top - rootTop,
			bottom: r.bottom - rootTop,
		};
	});
})()`
}

// blockRasterRegions turns per-block render overrides + measured bounding boxes
// into raster regions. Only blocks whose override actually changes rasterization
// (a dither mode, a tone-curve gamma, or a custom threshold) produce a region;
// heat-only overrides (printerDensity) do not.
func blockRasterRegions(metrics []blockMetric, perBlock map[string]*layoutv1.RenderOptions) []rasterRegion {
	if len(metrics) == 0 || len(perBlock) == 0 {
		return nil
	}
	var regions []rasterRegion
	for _, m := range metrics {
		ro := perBlock[m.ID]
		if ro == nil {
			continue
		}
		mode := rasterModeString(ro.GetRasterMode())
		hasGamma := ro.Gamma != nil
		hasThreshold := ro.Threshold != nil
		if mode == "" && !hasGamma && !hasThreshold {
			continue // e.g. printerDensity-only override — no raster change
		}
		reg := rasterRegion{
			YStart: int(math.Round(m.Top)),
			YEnd:   int(math.Round(m.Bottom)),
			Mode:   mode,
		}
		if hasGamma {
			reg.Gamma = ro.GetGamma()
		}
		if hasThreshold {
			reg.Threshold = clampToUint8(int(ro.GetThreshold()))
			reg.ThresholdSet = true
		}
		if reg.YEnd > reg.YStart {
			regions = append(regions, reg)
		}
	}
	return regions
}

// rasterize.go implements block-aware rasterization (DSL v2, Phase 6): different
// vertical regions of one page can be converted to 1-bit with different
// techniques. Text wants a hard threshold (crisp strokes survive), photographs
// want Atkinson dithering plus a gamma tone curve (ALMANACH-RASTER-LAB finding).
// Regions come from per-block render options mapped onto per-block bounding
// boxes; rows not covered by any region use the page default threshold.

// rasterRegion is a horizontal band [YStart, YEnd) of the screenshot with its
// own rasterization technique. Y values are image pixel rows.
type rasterRegion struct {
	YStart       int
	YEnd         int
	Mode         string  // "" (threshold) | "atkinson" | "floydSteinberg" | "bayer"; unknown falls back to threshold
	Gamma        float64 // <=0 means 1.0 (no tone curve)
	Threshold    uint8
	ThresholdSet bool // distinguishes an explicit zero cutoff from page-default inheritance
}

// pageRasterRegions turns page-level render options (Layout.render rasterMode /
// gamma) into raster regions covering every row NOT already claimed by a
// per-block region, so a full-page photo layout can request e.g. Atkinson
// without per-block overrides. Returns nil when the page options request plain
// default-threshold rasterization. YEnd of the trailing region is MaxInt32 —
// imageToBitmapRegions clamps to the image height.
func pageRasterRegions(mode string, gamma float64, blockRegions []rasterRegion) []rasterRegion {
	if mode == "threshold" {
		mode = ""
	}
	if mode == "" && (gamma <= 0 || gamma == 1) {
		return nil
	}

	sorted := append([]rasterRegion(nil), blockRegions...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].YStart < sorted[j].YStart })

	var out []rasterRegion
	y := 0
	for _, r := range sorted {
		if r.YStart > y {
			out = append(out, rasterRegion{YStart: y, YEnd: r.YStart, Mode: mode, Gamma: gamma})
		}
		y = maxInt(y, r.YEnd)
	}
	out = append(out, rasterRegion{YStart: y, YEnd: math.MaxInt32, Mode: mode, Gamma: gamma})
	return out
}

// grayAt returns the BT.601 luminance (0..255) of a pixel.
func grayAt(img image.Image, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256.0
}

// applyGamma maps a 0..255 value through a gamma tone curve. gamma < 1 lifts
// shadows (brightens midtones), which keeps photos from turning into black mud
// on a hot thermal head.
func applyGamma(v, gamma float64) float64 {
	if gamma <= 0 || gamma == 1 {
		return v
	}
	return 255 * math.Pow(v/255, gamma)
}

// imageToBitmapRegions converts an image to a 1-bit packed bitmap, applying each
// region's technique to its rows and the page default threshold to the rest.
func imageToBitmapRegions(img image.Image, defaultThreshold uint8, regions []rasterRegion) *Bitmap {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	wPadded := ((w + 7) / 8) * 8
	bytesPerRow := wPadded / 8
	data := make([]byte, bytesPerRow*h)

	// black[y*w+x] == true -> burn a dot.
	black := make([]bool, w*h)
	covered := make([]bool, h)

	for _, reg := range regions {
		ys := clampInt(reg.YStart, 0, h)
		ye := clampInt(reg.YEnd, 0, h)
		if ye <= ys {
			continue
		}
		switch reg.Mode {
		case "atkinson":
			atkinsonBand(img, black, bounds, w, ys, ye, reg.Gamma)
		case "floydSteinberg":
			floydSteinbergBand(img, black, bounds, w, ys, ye, reg.Gamma)
		case "bayer":
			bayerBand(img, black, bounds, w, ys, ye, reg.Gamma)
		default:
			// "" (threshold, possibly with gamma/custom threshold) and any
			// unknown mode: hard threshold.
			thr := reg.Threshold
			if !reg.ThresholdSet && thr == 0 {
				thr = defaultThreshold
			}
			thresholdBand(img, black, bounds, w, ys, ye, reg.Gamma, thr)
		}
		for y := ys; y < ye; y++ {
			covered[y] = true
		}
	}

	// Rows not covered by any region: page default threshold.
	for y := 0; y < h; y++ {
		if covered[y] {
			continue
		}
		for x := 0; x < w; x++ {
			if uint8(grayAt(img, bounds.Min.X+x, bounds.Min.Y+y)) < defaultThreshold {
				black[y*w+x] = true
			}
		}
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if black[y*w+x] {
				data[y*bytesPerRow+x/8] |= byte(0x80) >> (x % 8)
			}
		}
	}

	return &Bitmap{
		Width:       wPadded,
		Height:      h,
		BytesPerRow: bytesPerRow,
		Data:        data,
	}
}

// thresholdBand hard-thresholds a row band (with optional gamma).
func thresholdBand(img image.Image, black []bool, bounds image.Rectangle, w, ys, ye int, gamma float64, threshold uint8) {
	for y := ys; y < ye; y++ {
		for x := 0; x < w; x++ {
			v := applyGamma(grayAt(img, bounds.Min.X+x, bounds.Min.Y+y), gamma)
			if uint8(v) < threshold {
				black[y*w+x] = true
			}
		}
	}
}

// atkinsonBand runs Atkinson error-diffusion dithering over a row band. Error is
// confined to the band so an image's dithering never bleeds into the text above
// or below it. Atkinson diffuses only 6/8 of the error (the rest is dropped),
// which reads cleaner than Floyd-Steinberg on a 1-bit thermal head.
func atkinsonBand(img image.Image, black []bool, bounds image.Rectangle, w, ys, ye int, gamma float64) {
	bh := ye - ys
	buf := make([]float64, w*bh)
	for j := 0; j < bh; j++ {
		for x := 0; x < w; x++ {
			buf[j*w+x] = applyGamma(grayAt(img, bounds.Min.X+x, bounds.Min.Y+ys+j), gamma)
		}
	}
	diffuse := func(j, x int, e float64) {
		if x < 0 || x >= w || j < 0 || j >= bh {
			return
		}
		buf[j*w+x] += e
	}
	for j := 0; j < bh; j++ {
		for x := 0; x < w; x++ {
			old := buf[j*w+x]
			var newv float64
			if old >= 128 {
				newv = 255
			} else {
				newv = 0
				black[(ys+j)*w+x] = true
			}
			e := (old - newv) / 8
			diffuse(j, x+1, e)
			diffuse(j, x+2, e)
			diffuse(j+1, x-1, e)
			diffuse(j+1, x, e)
			diffuse(j+1, x+1, e)
			diffuse(j+2, x, e)
		}
	}
}

// floydSteinbergBand runs classic Floyd-Steinberg error diffusion over a row
// band (7/16 right, 3/16 below-left, 5/16 below, 1/16 below-right). Error is
// confined to the band, matching atkinsonBand's containment guarantee. Compared
// to Atkinson it diffuses the full error, which preserves more tonal accuracy
// but reads slightly noisier on a thermal head.
func floydSteinbergBand(img image.Image, black []bool, bounds image.Rectangle, w, ys, ye int, gamma float64) {
	bh := ye - ys
	buf := make([]float64, w*bh)
	for j := 0; j < bh; j++ {
		for x := 0; x < w; x++ {
			buf[j*w+x] = applyGamma(grayAt(img, bounds.Min.X+x, bounds.Min.Y+ys+j), gamma)
		}
	}
	diffuse := func(j, x int, e float64) {
		if x < 0 || x >= w || j < 0 || j >= bh {
			return
		}
		buf[j*w+x] += e
	}
	for j := 0; j < bh; j++ {
		for x := 0; x < w; x++ {
			old := buf[j*w+x]
			var newv float64
			if old >= 128 {
				newv = 255
			} else {
				newv = 0
				black[(ys+j)*w+x] = true
			}
			e := old - newv
			diffuse(j, x+1, e*7/16)
			diffuse(j+1, x-1, e*3/16)
			diffuse(j+1, x, e*5/16)
			diffuse(j+1, x+1, e*1/16)
		}
	}
}

// bayer4 is the classic 4x4 ordered-dither matrix (values 0..15).
var bayer4 = [4][4]float64{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// bayerBand runs 4x4 ordered (Bayer) dithering over a row band: each pixel is
// compared against a position-dependent threshold, producing a regular
// crosshatch pattern. No error diffusion means no state, so the pattern is
// stable across renders — useful when dither noise between prints matters more
// than tonal accuracy.
func bayerBand(img image.Image, black []bool, bounds image.Rectangle, w, ys, ye int, gamma float64) {
	for y := ys; y < ye; y++ {
		for x := 0; x < w; x++ {
			v := applyGamma(grayAt(img, bounds.Min.X+x, bounds.Min.Y+y), gamma)
			t := (bayer4[y%4][x%4] + 0.5) / 16 * 255
			if v < t {
				black[y*w+x] = true
			}
		}
	}
}

// blockHeatRegions maps per-block printerDensity overrides + bounding boxes to
// printer heat regions (Phase 6, per-segment heat). Only blocks that set a
// printerDensity produce a region.
func blockHeatRegions(metrics []blockMetric, perBlock map[string]*layoutv1.RenderOptions) []HeatRegion {
	if len(metrics) == 0 || len(perBlock) == 0 {
		return nil
	}
	var regions []HeatRegion
	for _, m := range metrics {
		ro := perBlock[m.ID]
		if ro == nil || ro.PrinterDensity == nil {
			continue
		}
		ys := int(math.Round(m.Top))
		ye := int(math.Round(m.Bottom))
		if ye > ys {
			regions = append(regions, HeatRegion{YStart: ys, YEnd: ye, Density: int(ro.GetPrinterDensity())})
		}
	}
	return regions
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
