package app

import (
	"image"
	"image/color"
	"testing"
)

// TestDownscaleBoxGrayAverages checks the box average produces the mean luminance
// of each scale x scale block and the correct output dimensions.
func TestDownscaleBoxGrayAverages(t *testing.T) {
	// 4x2 image, scale 2 -> 2x1. Left block all black (0), right block all white.
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			c := color.RGBA{0, 0, 0, 255}
			if x >= 2 {
				c = color.RGBA{255, 255, 255, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	out := downscaleBoxGray(img, 2)
	if out.Bounds().Dx() != 2 || out.Bounds().Dy() != 1 {
		t.Fatalf("expected 2x1, got %dx%d", out.Bounds().Dx(), out.Bounds().Dy())
	}
	g := out.(*image.Gray)
	if g.Pix[0] != 0 {
		t.Fatalf("left block should be black (0), got %d", g.Pix[0])
	}
	if g.Pix[1] < 254 {
		t.Fatalf("right block should be white (~255), got %d", g.Pix[1])
	}

	// A half-black / half-white block should average to mid gray.
	img2 := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img2.SetRGBA(0, 0, color.RGBA{0, 0, 0, 255})
	img2.SetRGBA(1, 0, color.RGBA{0, 0, 0, 255})
	img2.SetRGBA(0, 1, color.RGBA{255, 255, 255, 255})
	img2.SetRGBA(1, 1, color.RGBA{255, 255, 255, 255})
	out2 := downscaleBoxGray(img2, 2).(*image.Gray)
	if out2.Pix[0] < 120 || out2.Pix[0] > 135 {
		t.Fatalf("half/half block should be ~127 gray, got %d", out2.Pix[0])
	}
}

// TestDownscaleScale1Passthrough verifies scale<=1 returns the input unchanged.
func TestDownscaleScale1Passthrough(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	if got := downscaleBoxGray(img, 1); got != image.Image(img) {
		t.Fatal("scale 1 should return the same image")
	}
}

// TestSupersampledThresholdEquivalated confirms a supersampled all-white image is
// all white and an all-black image is all black after 1-bit conversion.
func TestSupersampledSolids(t *testing.T) {
	for _, tc := range []struct {
		name  string
		gray  uint8
		black bool
	}{{"white", 255, false}, {"black", 0, true}} {
		img := image.NewGray(image.Rect(0, 0, 24, 6))
		for i := range img.Pix {
			img.Pix[i] = tc.gray
		}
		down := downscaleBoxGray(img, 3)
		bm, err := imageToBitmap(down, 128)
		if err != nil {
			t.Fatal(err)
		}
		anySet := false
		for _, b := range bm.Data {
			if b != 0 {
				anySet = true
			}
		}
		if anySet != tc.black {
			t.Fatalf("%s: expected black=%v, got bits set=%v", tc.name, tc.black, anySet)
		}
	}
}
