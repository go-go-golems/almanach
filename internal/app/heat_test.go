package app

import "testing"

// densityBands should cover every row, filling gaps with the default density and
// using the region density inside each heat region.
func TestDensityBandsCoverAndFill(t *testing.T) {
	regions := []HeatRegion{
		{YStart: 50, YEnd: 100, Density: 20}, // a photo, cooler
	}
	bands := densityBands(200, regions, 38) // page default hot
	want := []densityBand{
		{0, 50, 38},
		{50, 100, 20},
		{100, 200, 38},
	}
	if len(bands) != len(want) {
		t.Fatalf("bands=%d, want %d: %+v", len(bands), len(want), bands)
	}
	for i, b := range bands {
		if b != want[i] {
			t.Errorf("band %d = %+v, want %+v", i, b, want[i])
		}
	}
}

// Adjacent regions produce no default gap between them.
func TestDensityBandsAdjacent(t *testing.T) {
	regions := []HeatRegion{
		{YStart: 0, YEnd: 40, Density: 38},
		{YStart: 40, YEnd: 90, Density: 20},
	}
	bands := densityBands(90, regions, 30)
	if len(bands) != 2 {
		t.Fatalf("bands=%d, want 2: %+v", len(bands), bands)
	}
	if bands[0] != (densityBand{0, 40, 38}) || bands[1] != (densityBand{40, 90, 20}) {
		t.Errorf("bands=%+v", bands)
	}
}

// No regions -> a single default band.
func TestDensityBandsEmpty(t *testing.T) {
	bands := densityBands(120, nil, 38)
	if len(bands) != 1 || bands[0] != (densityBand{0, 120, 38}) {
		t.Errorf("bands=%+v", bands)
	}
}

func TestSliceBitmapRows(t *testing.T) {
	bm := &Bitmap{Width: 16, Height: 10, BytesPerRow: 2, Data: make([]byte, 2*10)}
	for i := range bm.Data {
		bm.Data[i] = byte(i)
	}
	s := sliceBitmapRows(bm, 3, 7)
	if s.Height != 4 {
		t.Fatalf("height=%d, want 4", s.Height)
	}
	if len(s.Data) != 4*2 {
		t.Fatalf("data len=%d, want 8", len(s.Data))
	}
	if s.Data[0] != bm.Data[3*2] {
		t.Errorf("slice start byte=%d, want %d", s.Data[0], bm.Data[6])
	}
	// out-of-range clamps to empty
	if got := sliceBitmapRows(bm, 8, 4); got.Height != 0 {
		t.Errorf("inverted range should be empty, got height %d", got.Height)
	}
}

// Gap bands must never print at density 0 once a block override has mutated
// the printer's density state: with no page-level density, the fallback is the
// paper-verified text default, so a photo band at 20 does not leak into the
// text below it (PR #7 follow-up review).
func TestHeatGapDensityFallback(t *testing.T) {
	if got := heatGapDensity(0, false); got != defaultHeatGapDensity {
		t.Errorf("no page density: got %d, want %d", got, defaultHeatGapDensity)
	}
	if got := heatGapDensity(25, true); got != 25 {
		t.Errorf("page density set: got %d, want 25", got)
	}
	if got := heatGapDensity(0, true); got != 0 {
		t.Errorf("explicit zero density: got %d, want 0", got)
	}

	// End to end through densityBands: a photo override at rows [10,20) on a
	// 30-row page with no page density -> gaps print at the text default.
	bands := densityBands(30, []HeatRegion{{YStart: 10, YEnd: 20, Density: 20}}, heatGapDensity(0, false))
	if len(bands) != 3 {
		t.Fatalf("bands = %+v", bands)
	}
	if bands[0].Density != defaultHeatGapDensity || bands[2].Density != defaultHeatGapDensity {
		t.Errorf("gap bands must restore the default density, got %+v", bands)
	}
	if bands[1].Density != 20 {
		t.Errorf("override band density = %d, want 20", bands[1].Density)
	}
}
