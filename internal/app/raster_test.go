package app

import (
	"image"
	"image/color"
	"testing"
)

// syntheticGradient builds a horizontal 0..255 grayscale gradient image.
func syntheticGradient(w, h int) image.Image {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetGray(x, y, color.Gray{Y: uint8(x * 255 / (w - 1))})
		}
	}
	return img
}

// TestThresholdByteIdentical proves the new Rasterize threshold path produces
// exactly the same packed bytes as the legacy imageToBitmap, so switching the
// pluggable rasterizer in is fully reversible.
func TestThresholdByteIdentical(t *testing.T) {
	img := syntheticGradient(37, 8) // non-multiple-of-8 width exercises padding
	for _, thr := range []uint8{0, 64, 128, 160, 255} {
		legacy, err := imageToBitmap(img, thr)
		if err != nil {
			t.Fatalf("imageToBitmap: %v", err)
		}
		got, err := Rasterize(img, RasterOptions{Mode: RasterThreshold, Threshold: thr})
		if err != nil {
			t.Fatalf("Rasterize: %v", err)
		}
		if legacy.Width != got.Width || legacy.Height != got.Height || legacy.BytesPerRow != got.BytesPerRow {
			t.Fatalf("dims differ at thr=%d: legacy %dx%d/%d got %dx%d/%d", thr,
				legacy.Width, legacy.Height, legacy.BytesPerRow, got.Width, got.Height, got.BytesPerRow)
		}
		if string(legacy.Data) != string(got.Data) {
			t.Fatalf("bytes differ at threshold=%d", thr)
		}
	}
}

// TestRasterModesPackShape checks each dither mode returns a correctly-shaped,
// padded bitmap and that a mid-gray gradient yields a plausible black fraction.
func TestRasterModesPackShape(t *testing.T) {
	img := syntheticGradient(64, 16)
	for _, mode := range []RasterMode{RasterAtkinson, RasterFloyd, RasterBayer8} {
		bm, err := Rasterize(img, RasterOptions{Mode: mode, Threshold: 128, Gamma: 1, Contrast: 1})
		if err != nil {
			t.Fatalf("mode %s: %v", mode, err)
		}
		if bm.Width != 64 || bm.Height != 16 || bm.BytesPerRow != 8 {
			t.Fatalf("mode %s wrong shape: %dx%d/%d", mode, bm.Width, bm.Height, bm.BytesPerRow)
		}
		black := 0
		for _, b := range bm.Data {
			for i := 0; i < 8; i++ {
				if b&(1<<i) != 0 {
					black++
				}
			}
		}
		frac := float64(black) / float64(64*16)
		if frac < 0.2 || frac > 0.8 {
			t.Fatalf("mode %s black fraction %.2f out of range for a 0..255 ramp", mode, frac)
		}
	}
}

// TestGammaLightensMidtones verifies gamma<1 reduces black coverage on a mid
// image (lightens) relative to gamma=1, confirming the v'=v^gamma convention.
func TestGammaLightensMidtones(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetGray(x, y, color.Gray{Y: 110}) // slightly-dark mid tone
		}
	}
	count := func(g float64) int {
		bm, _ := Rasterize(img, RasterOptions{Mode: RasterAtkinson, Threshold: 128, Gamma: g, Contrast: 1})
		n := 0
		for _, b := range bm.Data {
			for i := 0; i < 8; i++ {
				if b&(1<<i) != 0 {
					n++
				}
			}
		}
		return n
	}
	if count(0.6) >= count(1.0) {
		t.Fatalf("gamma 0.6 should lighten (fewer black dots) than gamma 1.0: got %d vs %d", count(0.6), count(1.0))
	}
}

func TestUnknownModeErrors(t *testing.T) {
	img := syntheticGradient(8, 8)
	if _, err := Rasterize(img, RasterOptions{Mode: "nope"}); err == nil {
		t.Fatal("expected error for unknown mode")
	}
}
