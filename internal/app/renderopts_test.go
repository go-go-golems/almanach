package app

import "testing"

func TestParseRenderOptions_TypedFields(t *testing.T) {
	m := map[string]interface{}{
		"selector":         ".paper-body",
		"threshold":        float64(200), // JSON numbers decode as float64
		"supersampleScale": float64(2),
		"rasterMode":       "RASTER_MODE_ATKINSON",
		"gamma":            0.8,
		"printerDensity":   float64(38),
	}
	p, err := parseRenderOptions(m)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.GetSelector() != ".paper-body" {
		t.Errorf("selector = %q", p.GetSelector())
	}
	if p.GetThreshold() != 200 {
		t.Errorf("threshold = %d", p.GetThreshold())
	}
	if rasterModeString(p.GetRasterMode()) != "atkinson" {
		t.Errorf("rasterMode = %v", p.GetRasterMode())
	}
	if p.GetPrinterDensity() != 38 {
		t.Errorf("printerDensity = %d", p.GetPrinterDensity())
	}
}

func TestParseRenderOptions_Empty(t *testing.T) {
	p, err := parseRenderOptions(nil)
	if err != nil {
		t.Fatalf("parse nil: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil for empty options, got %v", p)
	}
}

func TestParseRenderOptions_Validation(t *testing.T) {
	cases := []map[string]interface{}{
		{"threshold": float64(300)},
		{"supersampleScale": float64(0)},
		{"viewportWidth": float64(-1)},
		{"printerDensity": float64(-5)},
		// The firmware accepts density 0..39 only — 100 used to pass validation,
		// reach the printer, error there, and print at the old density.
		{"printerDensity": float64(100)},
		// The firmware accepts a discrete speed set; 45 is not in it.
		{"printerSpeed": float64(45)},
		{"printerSpeed": float64(0)},
	}
	for i, m := range cases {
		if _, err := parseRenderOptions(m); err == nil {
			t.Errorf("case %d: expected validation error, got nil", i)
		}
	}
	// Boundary and supported values pass.
	valid := []map[string]interface{}{
		{"printerDensity": float64(39)},
		{"printerSpeed": float64(50)},
		{"printerSpeed": float64(220)},
	}
	for i, m := range valid {
		if _, err := parseRenderOptions(m); err != nil {
			t.Errorf("valid case %d: unexpected error: %v", i, err)
		}
	}
}

// applyRenderOptions overlays only the set fields; unset fields keep the base.
func TestApplyRenderOptions_Overlay(t *testing.T) {
	base := RenderOptions{
		Selector:         ".base",
		Threshold:        128,
		SupersampleScale: 1,
		ViewportWidth:    384,
	}
	p, err := parseRenderOptions(map[string]interface{}{
		"threshold":      float64(38),
		"printerDensity": float64(38),
		"rasterMode":     "RASTER_MODE_THRESHOLD",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := applyRenderOptions(base, p)
	if got.Selector != ".base" {
		t.Errorf("selector should keep base, got %q", got.Selector)
	}
	if got.Threshold != 38 {
		t.Errorf("threshold should be overlaid, got %d", got.Threshold)
	}
	if got.ViewportWidth != 384 {
		t.Errorf("viewportWidth should keep base, got %d", got.ViewportWidth)
	}
	if got.PrinterDensity != 38 || !got.PrinterDensitySet {
		t.Errorf("printer density = %d, set = %t", got.PrinterDensity, got.PrinterDensitySet)
	}
	if got.RasterMode != "threshold" {
		t.Errorf("rasterMode = %q", got.RasterMode)
	}
}

func TestApplyRenderOptions_PreservesExplicitZeroThresholdThroughDefaults(t *testing.T) {
	p, err := parseRenderOptions(map[string]interface{}{"threshold": float64(0)})
	if err != nil {
		t.Fatal(err)
	}
	got := applyRenderOptions(RenderOptions{}, p).withDefaults()
	if !got.ThresholdSet || got.Threshold != 0 {
		t.Fatalf("threshold = %d, set = %t; want explicit zero", got.Threshold, got.ThresholdSet)
	}
}

func TestApplyRenderOptions_ExplicitZeroDensity(t *testing.T) {
	base := RenderOptions{PrinterDensity: 38, PrinterDensitySet: true}
	p, err := parseRenderOptions(map[string]interface{}{"printerDensity": float64(0)})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := applyRenderOptions(base, p)
	if got.PrinterDensity != 0 || !got.PrinterDensitySet {
		t.Errorf("printer density = %d, set = %t; want explicit zero", got.PrinterDensity, got.PrinterDensitySet)
	}
}

func TestApplyRenderOptions_NilKeepsBase(t *testing.T) {
	base := RenderOptions{Selector: ".x", Threshold: 100}
	got := applyRenderOptions(base, nil)
	if got.Selector != base.Selector || got.Threshold != base.Threshold {
		t.Errorf("nil overlay changed base: %+v", got)
	}
}

func TestPerBlockRenderOptions(t *testing.T) {
	layout := `{
	  "blocks": [
	    {"id": "text-1", "type": "quote", "render": {"printerDensity": 38, "rasterMode": "RASTER_MODE_THRESHOLD"}},
	    {"id": "img-1", "type": "image", "render": {"rasterMode": "RASTER_MODE_ATKINSON", "gamma": 0.8, "printerDensity": 20}},
	    {"id": "plain", "type": "note"}
	  ]
	}`
	m, err := perBlockRenderOptions(layout)
	if err != nil {
		t.Fatalf("perBlock: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 blocks with render, got %d", len(m))
	}
	if m["text-1"].GetPrinterDensity() != 38 {
		t.Errorf("text-1 density = %d", m["text-1"].GetPrinterDensity())
	}
	if rasterModeString(m["img-1"].GetRasterMode()) != "atkinson" {
		t.Errorf("img-1 rasterMode = %v", m["img-1"].GetRasterMode())
	}
	if _, ok := m["plain"]; ok {
		t.Error("block without render should be omitted")
	}
}
