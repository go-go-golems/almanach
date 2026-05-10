package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Server holds the shared state for the HTTP server.
type Server struct {
	cfg           Config
	allocatorCtx  context.Context
	allocatorDone context.CancelFunc
	setupDevices  *setupDeviceStore
}

// RegisterRoutes wires all HTTP handlers onto the given mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// SPA static files (loaded by Chrome headless)
	registerStaticRoutes(mux, s.cfg.WebDir)

	if s.setupDevices == nil {
		s.setupDevices = &setupDeviceStore{}
	}

	// Setup rendezvous API
	mux.HandleFunc("/api/setup/provisioned-device", s.handleProvisionedDevice)

	// Render API
	mux.HandleFunc("/api/render", s.handleRender)
	mux.HandleFunc("/api/render-and-print", s.handleRenderAndPrint)

	// Schedule API
	mux.HandleFunc("/api/schedule", s.handleSchedule)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": Version,
		"printer": s.effectivePrinterIP(),
	})
}

// handleRender renders an almanac page and returns the bitmap.
// POST with optional JSON body to override layout data.
func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "use POST"})
		return
	}

	result, err := s.render(r.Context(), r.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	// Check Accept header to determine response format
	accept := r.Header.Get("Accept")
	switch {
	case contains(accept, "application/octet-stream"):
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Width", fmt.Sprintf("%d", result.Bitmap.Width))
		w.Header().Set("X-Height", fmt.Sprintf("%d", result.Bitmap.Height))
		w.Write(result.Bitmap.Data)
	case contains(accept, "image/png"):
		w.Header().Set("Content-Type", "image/png")
		w.Write(result.PNG)
	default:
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"width":      result.Bitmap.Width,
			"height":     result.Bitmap.Height,
			"theme":      result.Theme,
			"renderedAt": result.RenderedAt,
		})
	}
}

// handleRenderAndPrint renders and forwards to the ESP32 printer.
func (s *Server) handleRenderAndPrint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "use POST"})
		return
	}

	printerIP := s.effectivePrinterIP()
	if printerIP == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "ALMANACH_PRINTER_IP not configured",
		})
		return
	}

	result, err := s.render(r.Context(), r.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	// Forward to ESP32
	printerURL := fmt.Sprintf("http://%s/api/print/bitmap", printerIP)
	printResp, err := sendBitmapToPrinter(printerURL, result.Bitmap, s.cfg.FeedLines)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":         false,
			"error":      fmt.Sprintf("printer failed: %v", err),
			"width":      result.Bitmap.Width,
			"height":     result.Bitmap.Height,
			"renderedAt": result.RenderedAt,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"width":           result.Bitmap.Width,
		"height":          result.Bitmap.Height,
		"printed":         true,
		"printerResponse": printResp,
		"renderedAt":      result.RenderedAt,
	})
}

// handleSchedule is a stub for the cron scheduler (Phase 5).
func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "schedule": nil, "message": "not yet implemented"})
	case http.MethodPost:
		writeJSON(w, http.StatusNotImplemented, map[string]any{"ok": false, "error": "scheduler not yet implemented"})
	case http.MethodDelete:
		writeJSON(w, http.StatusNotImplemented, map[string]any{"ok": false, "error": "scheduler not yet implemented"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "use GET, POST, or DELETE"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
