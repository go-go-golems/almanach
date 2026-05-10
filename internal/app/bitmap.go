package app

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
)

// PngToBitmap decodes a PNG image and converts it to a 1-bit monochrome bitmap.
// Pixels with gray < threshold become black (1), else white (0).
func PngToBitmap(pngData []byte, threshold uint8) (*Bitmap, error) {
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("decode PNG: %w", err)
	}
	return imageToBitmap(img, threshold)
}

// imageToBitmap converts any image.Image to a 1-bit packed bitmap (MSB first).
func imageToBitmap(img image.Image, threshold uint8) (*Bitmap, error) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Pad width to next multiple of 8 for the printer
	wPadded := ((w + 7) / 8) * 8
	bytesPerRow := wPadded / 8
	data := make([]byte, bytesPerRow*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// Convert to grayscale using ITU-R BT.601 luminance weights
			gray := uint8((0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256.0)

			if gray < threshold {
				// Black pixel — set the bit (MSB first)
				data[y*bytesPerRow+x/8] |= byte(0x80) >> (x % 8)
			}
		}
	}

	return &Bitmap{
		Width:       wPadded,
		Height:      h,
		BytesPerRow: bytesPerRow,
		Data:        data,
	}, nil
}
