package app

import "testing"

func TestBitmapWithTrailingBlankRows(t *testing.T) {
	bitmap := &Bitmap{
		Width:       16,
		Height:      2,
		BytesPerRow: 2,
		Data:        []byte{0xff, 0x00, 0x80, 0x01},
	}

	got := bitmapWithTrailingBlankRows(bitmap, 3)
	if got == bitmap {
		t.Fatalf("expected copied bitmap with trailing rows")
	}
	if got.Width != 16 {
		t.Fatalf("width: got %d", got.Width)
	}
	wantHeight := 2 + 3*printerFeedLinePixels
	if got.Height != wantHeight {
		t.Fatalf("height: got %d, want %d", got.Height, wantHeight)
	}
	wantLen := 2 * wantHeight
	if len(got.Data) != wantLen {
		t.Fatalf("data len: got %d, want %d", len(got.Data), wantLen)
	}
	for i, b := range bitmap.Data {
		if got.Data[i] != b {
			t.Fatalf("original byte %d changed: got %#02x want %#02x", i, got.Data[i], b)
		}
	}
	for i, b := range got.Data[len(bitmap.Data):] {
		if b != 0x00 {
			t.Fatalf("trailing byte %d: got %#02x want white 0", i, b)
		}
	}
}

func TestBitmapWithTrailingBlankRowsClampsFeed(t *testing.T) {
	bitmap := &Bitmap{Width: 8, Height: 1, BytesPerRow: 1, Data: []byte{0xff}}
	got := bitmapWithTrailingBlankRows(bitmap, 99)
	wantHeight := 1 + 20*printerFeedLinePixels
	if got.Height != wantHeight {
		t.Fatalf("height: got %d, want %d", got.Height, wantHeight)
	}
}

func TestBitmapWithTrailingBlankRowsNoFeed(t *testing.T) {
	bitmap := &Bitmap{Width: 8, Height: 1, BytesPerRow: 1, Data: []byte{0xff}}
	if got := bitmapWithTrailingBlankRows(bitmap, 0); got != bitmap {
		t.Fatalf("feed 0 should return original bitmap")
	}
}
