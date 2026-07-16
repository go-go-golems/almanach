package app

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
)

// defaultSupersampleScale is the render oversampling factor. The page is drawn at
// scale x device pixels, then box-averaged back down to the target resolution
// before the 1-bit conversion.
//
// Default 1: a font/size/technique matrix on real paper (ALMANACH-PIXELFONT)
// showed that for a well-hinted font at a reasonable size, plain 1x rendering
// with anti-aliasing OFF is as crisp as — often crisper than — supersampling,
// because bytecode hinting is designed for exact-pixel rendering and
// supersampling bypasses it (and 1x is ~3x faster). Supersampling (scale>1, AA
// on) remains available via --supersample as a fallback for delicate,
// lightly-hinted display fonts (e.g. EB Garamond) rendered small.
const defaultSupersampleScale = 1

// downscaleBoxGray reduces img by an integer factor using box averaging, returning
// an 8-bit grayscale image at (w/scale) x (h/scale). Each output pixel is the mean
// BT.601 luminance of the scale x scale source block, matching imageToBitmap's
// luminance so the threshold behaves identically to a 1x render of the same tones.
func downscaleBoxGray(img image.Image, scale int) image.Image {
	if scale <= 1 {
		return img
	}
	b := img.Bounds()
	outW := b.Dx() / scale
	outH := b.Dy() / scale
	if outW <= 0 || outH <= 0 {
		return img
	}

	// Draw into a tightly-packed RGBA buffer once so the inner loop indexes Pix
	// directly (8-bit channels) instead of paying image.At/RGBA interface cost per
	// sample — the difference is seconds at 3x on a full page.
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)

	out := image.NewGray(image.Rect(0, 0, outW, outH))
	// Fixed-point BT.601 weights (×1000) to avoid float per sample.
	area := scale * scale
	for oy := 0; oy < outH; oy++ {
		for ox := 0; ox < outW; ox++ {
			var sum int
			for dy := 0; dy < scale; dy++ {
				row := (oy*scale + dy) * rgba.Stride
				base := row + (ox*scale)*4
				for dx := 0; dx < scale; dx++ {
					p := base + dx*4
					sum += 299*int(rgba.Pix[p]) + 587*int(rgba.Pix[p+1]) + 114*int(rgba.Pix[p+2])
				}
			}
			out.Pix[oy*out.Stride+ox] = uint8(sum / (area * 1000))
		}
	}
	return out
}

// pngToBitmapSupersampled decodes a screenshot PNG rendered at `scale` x the
// target resolution, box-averages it down to the target, and converts to a 1-bit
// bitmap at the given threshold. scale <= 1 is the plain 1x path.
func pngToBitmapSupersampled(pngData []byte, threshold uint8, scale int) (*Bitmap, error) {
	if scale <= 1 {
		return PngToBitmap(pngData, threshold)
	}
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("decode PNG: %w", err)
	}
	return imageToBitmap(downscaleBoxGray(img, scale), threshold)
}
