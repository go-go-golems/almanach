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
