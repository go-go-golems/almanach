package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWriteRenderErrorIncludesChromeDiagnostics(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeRenderError(recorder, &ChromeRenderError{
		Stage:       "layout-visible",
		Elapsed:     10 * time.Second,
		Timeout:     2 * time.Minute,
		Cause:       errors.New("waiting for function failed: timeout"),
		Diagnostics: []string{"exception: TypeError: object is not iterable"},
	})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["stage"] != "layout-visible" || body["elapsed_ms"] != float64(10000) || body["timeout_ms"] != float64(120000) {
		t.Errorf("unexpected structured fields: %+v", body)
	}
	if diagnostics, ok := body["browser_diagnostics"].([]any); !ok || len(diagnostics) != 1 {
		t.Errorf("browser diagnostics = %#v", body["browser_diagnostics"])
	}
}

func TestWriteRenderErrorKeepsGenericErrorsSimple(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeRenderError(recorder, errors.New("parse layout: invalid YAML"))

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["stage"]; ok {
		t.Errorf("generic error unexpectedly has chrome stage: %+v", body)
	}
}
