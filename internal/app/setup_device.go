package app

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

type serverState struct {
	ProvisionedDevice *ProvisionedDevice `json:"provisionedDevice,omitempty"`
}

type setupDeviceStore struct {
	mu        sync.RWMutex
	device    *ProvisionedDevice
	stateFile string
}

func newSetupDeviceStore(stateFile string) (*setupDeviceStore, error) {
	store := &setupDeviceStore{stateFile: stateFile}
	if stateFile == "" {
		return store, nil
	}
	state, err := loadServerState(stateFile)
	if err != nil {
		return nil, err
	}
	if state.ProvisionedDevice != nil {
		copy := *state.ProvisionedDevice
		store.device = &copy
	}
	return store, nil
}

func (s *setupDeviceStore) set(device ProvisionedDevice) (ProvisionedDevice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := device
	s.device = &copy
	if err := s.saveLocked(); err != nil {
		return copy, err
	}
	return copy, nil
}

func (s *setupDeviceStore) clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.device = nil
	return s.saveLocked()
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

func (s *setupDeviceStore) saveLocked() error {
	if s.stateFile == "" {
		return nil
	}
	state := serverState{}
	if s.device != nil {
		copy := *s.device
		state.ProvisionedDevice = &copy
	}
	return saveServerStateAtomic(s.stateFile, state)
}

func loadServerState(path string) (serverState, error) {
	var state serverState
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, fmt.Errorf("read state file %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("parse state file %s: %w", path, err)
	}
	return state, nil
}

func saveServerStateAtomic(path string, state serverState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp state file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp state file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}
	return nil
}

func (s *Server) ensureSetupDeviceStore() error {
	if s.setupDevices != nil {
		return nil
	}
	store, err := newSetupDeviceStore(s.cfg.StateFile)
	if err != nil {
		return err
	}
	s.setupDevices = store
	return nil
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
		device, err := s.setupDevices.set(req)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "device": device})
	case http.MethodDelete:
		if err := s.setupDevices.clear(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "device": nil})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "use GET, POST, or DELETE"})
	}
}
