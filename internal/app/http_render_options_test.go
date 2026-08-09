package app

import "testing"

func TestLayoutJSONFromRawPreservesFlatPageRenderOptions(t *testing.T) {
	s := &Server{}
	layout, render, err := s.layoutJSONFromRaw([]byte(`{
		"paperWidth": 384,
		"render": {
			"rasterMode": "RASTER_MODE_ATKINSON",
			"gamma": 0.8,
			"printerDensity": 20,
			"printerSpeed": 80
		},
		"blocks": [{"id":"photo","type":"image","data":{}}]
	}`))
	if err != nil {
		t.Fatalf("layoutJSONFromRaw: %v", err)
	}
	if render["printerDensity"] != float64(20) || render["printerSpeed"] != float64(80) {
		t.Fatalf("page render settings lost: %#v", render)
	}
	if render["rasterMode"] != "RASTER_MODE_ATKINSON" || render["gamma"] != float64(0.8) {
		t.Fatalf("page raster settings lost: %#v", render)
	}
	if layout == "" {
		t.Fatal("normalized layout is empty")
	}

	typed, err := parseRenderOptions(render)
	if err != nil {
		t.Fatalf("parseRenderOptions: %v", err)
	}
	opts := applyRenderOptions(defaultRenderOptions("http://example.test"), typed)
	if opts.RasterMode != "atkinson" || opts.Gamma != 0.8 {
		t.Fatalf("effective raster options = %#v", opts)
	}
	if !opts.PrinterDensitySet || opts.PrinterDensity != 20 || opts.PrinterSpeed != 80 {
		t.Fatalf("effective printer options = %#v", opts)
	}
}

func TestLayoutJSONFromRawPreservesWrappedPageRenderOptions(t *testing.T) {
	s := &Server{}
	_, render, err := s.layoutJSONFromRaw([]byte(`{
		"layout": {"blocks": []},
		"render": {"printerDensity": 20, "printerSpeed": 80}
	}`))
	if err != nil {
		t.Fatalf("layoutJSONFromRaw: %v", err)
	}
	if render["printerDensity"] != float64(20) || render["printerSpeed"] != float64(80) {
		t.Fatalf("wrapped render settings lost: %#v", render)
	}
}
