package app

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"math"
)

// RasterMode selects how a grayscale image is reduced to 1-bit.
type RasterMode string

const (
	// RasterThreshold is the legacy fixed-threshold conversion. It is kept
	// byte-identical to imageToBitmap so switching the default is reversible.
	RasterThreshold RasterMode = "threshold"
	RasterAtkinson  RasterMode = "atkinson"
	RasterFloyd     RasterMode = "floyd-steinberg"
	RasterBayer8    RasterMode = "bayer8"
)

// RasterOptions controls host-side 1-bit conversion. The tone curve
// (Gamma/Brightness/Contrast) is applied before dithering; it is skipped for
// RasterThreshold so that mode stays byte-identical with the legacy pipeline.
//
// Paper-verified default for photos (ALMANACH-RASTER-LAB): Atkinson, Gamma 0.8.
type RasterOptions struct {
	Mode       RasterMode
	Threshold  uint8   // quantization threshold, default 128
	Gamma      float64 // exponent on normalized gray; <1 lightens midtones
	Brightness float64 // added to normalized gray (fraction of full scale)
	Contrast   float64 // scale around 0.5; 1.0 = unchanged
}

// DefaultRasterOptions returns the legacy-compatible defaults.
func DefaultRasterOptions() RasterOptions {
	return RasterOptions{Mode: RasterThreshold, Threshold: 128, Gamma: 1, Brightness: 0, Contrast: 1}
}

// diffusionTap is one error-diffusion destination: (dx, dy) with weight/divisor.
type diffusionTap struct {
	dx, dy int
	w      float64
}

type diffusionKernel struct {
	div  float64
	taps []diffusionTap
}

var (
	atkinsonKernel = diffusionKernel{div: 8, taps: []diffusionTap{
		{1, 0, 1}, {2, 0, 1}, {-1, 1, 1}, {0, 1, 1}, {1, 1, 1}, {0, 2, 1},
	}}
	floydKernel = diffusionKernel{div: 16, taps: []diffusionTap{
		{1, 0, 7}, {-1, 1, 3}, {0, 1, 5}, {1, 1, 1},
	}}
	// bayer8 threshold matrix, values 0..63.
	bayer8 = [8][8]float64{
		{0, 48, 12, 60, 3, 51, 15, 63}, {32, 16, 44, 28, 35, 19, 47, 31},
		{8, 56, 4, 52, 11, 59, 7, 55}, {40, 24, 36, 20, 43, 27, 39, 23},
		{2, 50, 14, 62, 1, 49, 13, 61}, {34, 18, 46, 30, 33, 17, 45, 29},
		{10, 58, 6, 54, 9, 57, 5, 53}, {42, 26, 38, 22, 41, 25, 37, 21},
	}
)

// PngToBitmapRaster decodes a PNG and rasterizes it with the given options.
func PngToBitmapRaster(pngData []byte, opts RasterOptions) (*Bitmap, error) {
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("decode PNG: %w", err)
	}
	return Rasterize(img, opts)
}

// Rasterize converts an image to a 1-bit packed Bitmap using opts.Mode.
func Rasterize(img image.Image, opts RasterOptions) (*Bitmap, error) {
	// Threshold mode delegates to the legacy path for byte-identical output.
	// The threshold value is passed through unchanged (0 stays 0).
	if opts.Mode == "" || opts.Mode == RasterThreshold {
		return imageToBitmap(img, opts.Threshold)
	}

	// Dither modes: an unset threshold means the standard quantization midpoint.
	if opts.Threshold == 0 {
		opts.Threshold = 128
	}
	gray := luminanceGrid(img)
	applyTone(gray, opts.Gamma, opts.Brightness, opts.Contrast)

	var black [][]bool
	switch opts.Mode {
	case RasterThreshold:
		// Unreachable: handled by the early return above; listed for exhaustiveness.
		return imageToBitmap(img, opts.Threshold)
	case RasterAtkinson:
		black = errorDiffuse(gray, float64(opts.Threshold), atkinsonKernel)
	case RasterFloyd:
		black = errorDiffuse(gray, float64(opts.Threshold), floydKernel)
	case RasterBayer8:
		black = orderedBayer8(gray)
	default:
		return nil, fmt.Errorf("unknown raster mode %q", opts.Mode)
	}
	return packBoolBits(black), nil
}

// luminanceGrid returns BT.601 luminance in [0,255], matching imageToBitmap's
// scale (RGBA() is 0..65535, divided by 256).
func luminanceGrid(img image.Image) [][]float64 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	g := make([][]float64, h)
	for y := 0; y < h; y++ {
		row := make([]float64, w)
		for x := 0; x < w; x++ {
			r, gg, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			row[x] = (0.299*float64(r) + 0.587*float64(gg) + 0.114*float64(bb)) / 256.0
		}
		g[y] = row
	}
	return g
}

// applyTone reshapes gray in place: v' = ((v^gamma) - 0.5)*contrast + 0.5 + brightness.
func applyTone(gray [][]float64, gamma, brightness, contrast float64) {
	if gamma <= 0 {
		gamma = 1
	}
	if contrast == 0 {
		contrast = 1
	}
	for y := range gray {
		for x := range gray[y] {
			v := gray[y][x] / 255.0
			if gamma != 1 {
				v = math.Pow(clampF(v, 0, 1), gamma)
			}
			v = (v-0.5)*contrast + 0.5 + brightness
			gray[y][x] = clampF(v*255.0, 0, 255)
		}
	}
}

// errorDiffuse quantizes each pixel and pushes the error to future neighbors.
func errorDiffuse(gray [][]float64, threshold float64, k diffusionKernel) [][]bool {
	h := len(gray)
	if h == 0 {
		return nil
	}
	w := len(gray[0])
	work := make([][]float64, h)
	for y := range gray {
		work[y] = append([]float64(nil), gray[y]...)
	}
	black := make([][]bool, h)
	for y := 0; y < h; y++ {
		black[y] = make([]bool, w)
		for x := 0; x < w; x++ {
			old := work[y][x]
			newV := 255.0
			if old < threshold {
				newV = 0
				black[y][x] = true
			}
			err := old - newV
			for _, t := range k.taps {
				nx, ny := x+t.dx, y+t.dy
				if nx >= 0 && nx < w && ny >= 0 && ny < h {
					work[ny][nx] += err * t.w / k.div
				}
			}
		}
	}
	return black
}

// orderedBayer8 tiles the 8x8 Bayer matrix as a per-pixel threshold.
func orderedBayer8(gray [][]float64) [][]bool {
	h := len(gray)
	if h == 0 {
		return nil
	}
	w := len(gray[0])
	black := make([][]bool, h)
	for y := 0; y < h; y++ {
		black[y] = make([]bool, w)
		for x := 0; x < w; x++ {
			t := (bayer8[y%8][x%8] + 0.5) / 64.0 * 255.0
			black[y][x] = gray[y][x] < t
		}
	}
	return black
}

// packBoolBits packs a black/white grid MSB-first, width padded to a multiple
// of 8, matching the firmware GS v 0 contract and imageToBitmap.
func packBoolBits(black [][]bool) *Bitmap {
	h := len(black)
	w := 0
	if h > 0 {
		w = len(black[0])
	}
	wPadded := ((w + 7) / 8) * 8
	bytesPerRow := wPadded / 8
	data := make([]byte, bytesPerRow*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if black[y][x] {
				data[y*bytesPerRow+x/8] |= byte(0x80) >> (x % 8)
			}
		}
	}
	return &Bitmap{Width: wPadded, Height: h, BytesPerRow: bytesPerRow, Data: data}
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
