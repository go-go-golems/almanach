package app

import (
	"image"
	"image/color"
	"testing"
)

// solidGray builds a w x h image filled with a single gray level.
func solidGray(w, h int, level uint8) image.Image {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetGray(x, y, color.Gray{Y: level})
		}
	}
	return img
}

func countBlack(bm *Bitmap, w, h int) int {
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if bm.Data[y*bm.BytesPerRow+x/8]&(byte(0x80)>>(x%8)) != 0 {
				n++
			}
		}
	}
	return n
}

// A dark solid image thresholds to all-black; a light one to all-white.
func TestThresholdRegionSolid(t *testing.T) {
	w, h := 16, 8
	dark := imageToBitmapRegions(solidGray(w, h, 80), 128, nil)
	if got := countBlack(dark, w, h); got != w*h {
		t.Errorf("dark: black=%d, want %d", got, w*h)
	}
	light := imageToBitmapRegions(solidGray(w, h, 200), 128, nil)
	if got := countBlack(light, w, h); got != 0 {
		t.Errorf("light: black=%d, want 0", got)
	}
}

// Atkinson on a mid-gray produces a dithered mix (~half), not all-or-nothing —
// the qualitative difference from a hard threshold.
func TestAtkinsonMidGrayDithers(t *testing.T) {
	w, h := 32, 32
	regions := []rasterRegion{{YStart: 0, YEnd: h, Mode: "atkinson"}}
	bm := imageToBitmapRegions(solidGray(w, h, 128), 128, regions)
	black := countBlack(bm, w, h)
	total := w * h
	// A hard threshold would give 0 or all; dithering should land mid-range.
	if black < total*3/10 || black > total*7/10 {
		t.Errorf("atkinson mid-gray black=%d/%d, want ~half (dithered)", black, total)
	}
}

// Regions are independent: a threshold top band stays solid while a dithered
// bottom band mixes, and rows are attributed to the right band.
func TestMixedRegions(t *testing.T) {
	w, h := 24, 24
	img := solidGray(w, h, 128)
	regions := []rasterRegion{
		{YStart: 0, YEnd: 12, Mode: "", Threshold: 200}, // threshold: 128<200 -> all black
		{YStart: 12, YEnd: 24, Mode: "atkinson"},        // dithered ~half
	}
	bm := imageToBitmapRegions(img, 128, regions)

	topBlack := 0
	for y := 0; y < 12; y++ {
		for x := 0; x < w; x++ {
			if bm.Data[y*bm.BytesPerRow+x/8]&(byte(0x80)>>(x%8)) != 0 {
				topBlack++
			}
		}
	}
	if topBlack != 12*w {
		t.Errorf("top threshold band black=%d, want %d (all black)", topBlack, 12*w)
	}

	botBlack := 0
	for y := 12; y < 24; y++ {
		for x := 0; x < w; x++ {
			if bm.Data[y*bm.BytesPerRow+x/8]&(byte(0x80)>>(x%8)) != 0 {
				botBlack++
			}
		}
	}
	bandTotal := 12 * w
	if botBlack < bandTotal*3/10 || botBlack > bandTotal*7/10 {
		t.Errorf("bottom dither band black=%d/%d, want ~half", botBlack, bandTotal)
	}
}

// Floyd-Steinberg on a mid-gray also dithers to ~half coverage — and must not
// fall through to a hard threshold (which would give all-or-nothing).
func TestFloydSteinbergMidGrayDithers(t *testing.T) {
	w, h := 32, 32
	regions := []rasterRegion{{YStart: 0, YEnd: h, Mode: "floydSteinberg"}}
	bm := imageToBitmapRegions(solidGray(w, h, 128), 128, regions)
	black := countBlack(bm, w, h)
	total := w * h
	if black < total*3/10 || black > total*7/10 {
		t.Errorf("floydSteinberg mid-gray black=%d/%d, want ~half (dithered)", black, total)
	}
}

// Bayer ordered dithering on a mid-gray produces the regular ~50% crosshatch;
// before ALMANACH-WORKSLIP PR review it silently fell back to a hard threshold.
func TestBayerMidGrayDithers(t *testing.T) {
	w, h := 32, 32
	regions := []rasterRegion{{YStart: 0, YEnd: h, Mode: "bayer"}}
	bm := imageToBitmapRegions(solidGray(w, h, 128), 128, regions)
	black := countBlack(bm, w, h)
	total := w * h
	if black < total*4/10 || black > total*6/10 {
		t.Errorf("bayer mid-gray black=%d/%d, want ~half (ordered dither)", black, total)
	}
	// Ordered dithering is position-deterministic: the same 4x4 tile repeats.
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			a := bm.Data[y*bm.BytesPerRow+x/8]&(byte(0x80)>>(x%8)) != 0
			b := bm.Data[(y+4)*bm.BytesPerRow+(x+4)/8]&(byte(0x80)>>((x+4)%8)) != 0
			if a != b {
				t.Fatalf("bayer pattern not tiled at (%d,%d)", x, y)
			}
		}
	}
}

// Page-level rasterMode/gamma produce full-page regions covering the rows that
// per-block overrides don't claim (and the whole page when there are none).
func TestPageRasterRegions(t *testing.T) {
	// Plain threshold page: no regions.
	if got := pageRasterRegions("", 0, nil); got != nil {
		t.Errorf("threshold page produced regions: %v", got)
	}
	if got := pageRasterRegions("threshold", 1, nil); got != nil {
		t.Errorf("explicit threshold page produced regions: %v", got)
	}

	// Full-page dither with no block overrides: one region covering everything.
	full := pageRasterRegions("atkinson", 0.8, nil)
	if len(full) != 1 || full[0].YStart != 0 || full[0].Mode != "atkinson" || full[0].Gamma != 0.8 {
		t.Fatalf("full-page regions = %+v", full)
	}

	// Block override in the middle: page regions cover the complement.
	blocks := []rasterRegion{{YStart: 100, YEnd: 200, Mode: ""}}
	gaps := pageRasterRegions("bayer", 0, blocks)
	if len(gaps) != 2 {
		t.Fatalf("expected 2 gap regions, got %+v", gaps)
	}
	if gaps[0].YStart != 0 || gaps[0].YEnd != 100 || gaps[1].YStart != 200 {
		t.Errorf("gap regions = %+v", gaps)
	}

	// Gamma-only page (threshold + tone curve) still needs regions.
	g := pageRasterRegions("", 0.8, nil)
	if len(g) != 1 || g[0].Mode != "" || g[0].Gamma != 0.8 {
		t.Errorf("gamma-only page regions = %+v", g)
	}
}

// End to end: a page-level atkinson region dithers rows outside a block's
// threshold band.
func TestPageLevelDitherAroundBlock(t *testing.T) {
	w, h := 24, 30
	blocks := []rasterRegion{{YStart: 10, YEnd: 20, Mode: "", Threshold: 200}} // block: all black at gray 128
	regions := append(blocks, pageRasterRegions("atkinson", 0, blocks)...)
	bm := imageToBitmapRegions(solidGray(w, h, 128), 128, regions)

	blockBlack := 0
	for y := 10; y < 20; y++ {
		for x := 0; x < w; x++ {
			if bm.Data[y*bm.BytesPerRow+x/8]&(byte(0x80)>>(x%8)) != 0 {
				blockBlack++
			}
		}
	}
	if blockBlack != 10*w {
		t.Errorf("block band black=%d, want %d (threshold 200 on gray 128)", blockBlack, 10*w)
	}
	outside := countBlack(bm, w, h) - blockBlack
	outsideTotal := (h - 10) * w
	if outside < outsideTotal*3/10 || outside > outsideTotal*7/10 {
		t.Errorf("page dither outside block black=%d/%d, want ~half", outside, outsideTotal)
	}
}

// Gamma < 1 lifts a dark gray above a threshold, reducing black coverage.
func TestGammaLiftsShadows(t *testing.T) {
	w, h := 16, 16
	img := solidGray(w, h, 100) // below threshold 128 -> would be all black
	noGamma := imageToBitmapRegions(img, 128, []rasterRegion{{YStart: 0, YEnd: h, Mode: "", Threshold: 128}})
	withGamma := imageToBitmapRegions(img, 128, []rasterRegion{{YStart: 0, YEnd: h, Mode: "", Gamma: 0.5, Threshold: 128}})
	// gamma 0.5 maps 100 -> 255*(100/255)^0.5 ~= 159 > 128 -> white
	if countBlack(noGamma, w, h) != w*h {
		t.Errorf("no gamma: expected all black")
	}
	if countBlack(withGamma, w, h) != 0 {
		t.Errorf("gamma 0.5: expected all white (shadows lifted), got %d", countBlack(withGamma, w, h))
	}
}
