package app

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ProvisionedDevice struct {
	ServiceName string `json:"serviceName"`
	IP          string `json:"ip"`
	SSID        string `json:"ssid,omitempty"`
	Source      string `json:"source"`
	SeenAt      string `json:"seenAt"`
}

type setupDeviceStore struct {
	mu     sync.RWMutex
	device *ProvisionedDevice
}

func (s *setupDeviceStore) set(device ProvisionedDevice) ProvisionedDevice {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := device
	s.device = &copy
	return copy
}

func (s *setupDeviceStore) get() (*ProvisionedDevice, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.device == nil {
		return nil, false
	}
	copy := *s.device
	return &copy, true
}

func (s *Server) effectivePrinterIP() string {
	if s.cfg.PrinterIP != "" {
		return s.cfg.PrinterIP
	}
	if s.setupDevices != nil {
		if device, ok := s.setupDevices.get(); ok {
			return device.IP
		}
	}
	return ""
}

func (s *Server) handleProvisionedDevice(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		device, ok := s.setupDevices.get()
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "device": nil})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "device": device})
	case http.MethodPost:
		var req ProvisionedDevice
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON body"})
			return
		}
		req.IP = strings.TrimSpace(req.IP)
		req.ServiceName = strings.TrimSpace(req.ServiceName)
		req.SSID = strings.TrimSpace(req.SSID)
		req.Source = strings.TrimSpace(req.Source)
		if req.IP == "" || net.ParseIP(req.IP) == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "valid ip is required"})
			return
		}
		if req.Source == "" {
			req.Source = "unknown"
		}
		if req.SeenAt == "" {
			req.SeenAt = time.Now().UTC().Format(time.RFC3339)
		}
		device := s.setupDevices.set(req)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "device": device})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "use GET or POST"})
	}
}
