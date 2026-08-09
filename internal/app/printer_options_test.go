package app

import (
	"encoding/json"
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
	})
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
	})
	if err == nil {
		t.Fatal("expected density-setting failure")
	}
	if bitmapRequests != 0 {
		t.Fatalf("bitmap sent despite setting failure: %d requests", bitmapRequests)
	}
}
