package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

func TestSendBitmapWithOptionsAppliesHeatBeforeBitmap(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	var values []map[string]int
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		paths = append(paths, r.URL.Path)
		if r.URL.Path != "/api/print/bitmap" {
			var value map[string]int
			if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
				t.Errorf("decode setting: %v", err)
			}
			values = append(values, value)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer printer.Close()

	bitmap := &Bitmap{Width: 8, Height: 1, BytesPerRow: 1, Data: []byte{0x55}}
	_, err := sendBitmapWithOptions(printer.URL+"/api/print/bitmap", bitmap, 0, RenderOptions{
		PrinterSpeed:      80,
		PrinterDensity:    20,
		PrinterDensitySet: true,
	}, nil)
	if err != nil {
		t.Fatalf("sendBitmapWithOptions: %v", err)
	}

	wantPaths := []string{"/api/printer/speed", "/api/printer/density", "/api/print/bitmap"}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("request order = %v, want %v", paths, wantPaths)
	}
	wantValues := []map[string]int{{"speed": 80}, {"density": 20}}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("setting payloads = %#v, want %#v", values, wantValues)
	}
}

func TestSendBitmapWithOptionsPreservesMixedDensityRegions(t *testing.T) {
	var paths []string
	var densities []int
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/api/printer/density" {
			var value map[string]int
			if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
				t.Errorf("decode density: %v", err)
			}
			densities = append(densities, value["density"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer printer.Close()

	bitmap := &Bitmap{Width: 8, Height: 3, BytesPerRow: 1, Data: []byte{0x11, 0x22, 0x33}}
	_, err := sendBitmapWithOptions(printer.URL+"/api/print/bitmap", bitmap, 0, RenderOptions{
		PrinterDensity:    30,
		PrinterDensitySet: true,
	}, []HeatRegion{{YStart: 1, YEnd: 2, Density: 20}})
	if err != nil {
		t.Fatalf("sendBitmapWithOptions: %v", err)
	}
	wantPaths := []string{
		"/api/printer/density", "/api/print/bitmap",
		"/api/printer/density", "/api/print/bitmap",
		"/api/printer/density", "/api/print/bitmap",
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("request order = %v, want %v", paths, wantPaths)
	}
	if want := []int{30, 20, 30}; !reflect.DeepEqual(densities, want) {
		t.Fatalf("densities = %v, want %v", densities, want)
	}
}

func TestSendBitmapWithOptionsFailsClosedOnMixedHeatSettingError(t *testing.T) {
	bitmapRequests := 0
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/printer/density" {
			http.Error(w, "density unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/api/print/bitmap" {
			bitmapRequests++
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer printer.Close()

	bitmap := &Bitmap{Width: 8, Height: 1, BytesPerRow: 1, Data: []byte{0x55}}
	_, err := sendBitmapWithOptions(
		printer.URL+"/api/print/bitmap",
		bitmap,
		0,
		RenderOptions{},
		[]HeatRegion{{YStart: 0, YEnd: 1, Density: 20}},
	)
	if err == nil {
		t.Fatal("expected mixed-heat density-setting failure")
	}
	if bitmapRequests != 0 {
		t.Fatalf("bitmap sent despite mixed-heat setting failure: %d requests", bitmapRequests)
	}
}

func TestSegmentedBitmapHandlesNullPrinterResponse(t *testing.T) {
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`null`))
	}))
	defer printer.Close()

	bitmap := &Bitmap{
		Width:       384,
		Height:      800,
		BytesPerRow: 48,
		Data:        make([]byte, 48*800),
	}
	response, err := sendBitmapToPrinter(printer.URL+"/api/print/bitmap", bitmap, 0)
	if err != nil {
		t.Fatalf("sendBitmapToPrinter: %v", err)
	}
	if response["segments"] != 2 {
		t.Fatalf("segments = %#v, want 2", response["segments"])
	}
}

func TestSendBitmapWithOptionsFailsClosedOnSettingError(t *testing.T) {
	bitmapRequests := 0
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/printer/density" {
			http.Error(w, "density unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/api/print/bitmap" {
			bitmapRequests++
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer printer.Close()

	bitmap := &Bitmap{Width: 8, Height: 1, BytesPerRow: 1, Data: []byte{0x55}}
	_, err := sendBitmapWithOptions(printer.URL+"/api/print/bitmap", bitmap, 0, RenderOptions{
		PrinterDensity:    20,
		PrinterDensitySet: true,
	}, nil)
	if err == nil {
		t.Fatal("expected density-setting failure")
	}
	if bitmapRequests != 0 {
		t.Fatalf("bitmap sent despite setting failure: %d requests", bitmapRequests)
	}
}
