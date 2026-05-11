package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestProvisionedDeviceAPIStoresPrinterIP(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	server := &Server{cfg: Config{StateFile: stateFile}}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"serviceName":"ALM_0F2320","ip":"192.168.1.242","ssid":"LabNet","source":"web-bluetooth"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/setup/provisioned-device", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST status: got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := server.effectivePrinterIP(); got != "192.168.1.242" {
		t.Fatalf("effective printer IP = %q, want 192.168.1.242", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/setup/provisioned-device", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET status: got %d body=%s", rr.Code, rr.Body.String())
	}
	var got struct {
		OK     bool               `json:"ok"`
		Device *ProvisionedDevice `json:"device"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.OK || got.Device == nil || got.Device.IP != "192.168.1.242" || got.Device.ServiceName != "ALM_0F2320" {
		t.Fatalf("unexpected response: %+v", got)
	}

	loaded := &Server{cfg: Config{StateFile: stateFile}}
	mux = http.NewServeMux()
	loaded.RegisterRoutes(mux)
	if got := loaded.effectivePrinterIP(); got != "192.168.1.242" {
		t.Fatalf("loaded effective printer IP = %q, want 192.168.1.242", got)
	}
}

func TestProvisionedDeviceAPIDeleteClearsPersistedPrinterIP(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	server := &Server{cfg: Config{StateFile: stateFile}}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"serviceName":"ALM_0F2320","ip":"192.168.1.242","source":"web-bluetooth"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/setup/provisioned-device", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST status: got %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/setup/provisioned-device", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE status: got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := server.effectivePrinterIP(); got != "" {
		t.Fatalf("effective printer IP after delete = %q, want empty", got)
	}

	loaded := &Server{cfg: Config{StateFile: stateFile}}
	mux = http.NewServeMux()
	loaded.RegisterRoutes(mux)
	if got := loaded.effectivePrinterIP(); got != "" {
		t.Fatalf("loaded effective printer IP after delete = %q, want empty", got)
	}
}

func TestProvisionedDeviceAPIDoesNotOverrideConfiguredPrinterIP(t *testing.T) {
	server := &Server{cfg: Config{PrinterIP: "10.0.0.55"}}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"serviceName":"ALM_0F2320","ip":"192.168.1.242","source":"web-bluetooth"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/setup/provisioned-device", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST status: got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := server.effectivePrinterIP(); got != "10.0.0.55" {
		t.Fatalf("effective printer IP = %q, want configured 10.0.0.55", got)
	}
}

func TestProvisionedDeviceAPIRejectsInvalidIP(t *testing.T) {
	server := &Server{}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	body := bytes.NewBufferString(`{"serviceName":"ALM_0F2320","ip":"not-an-ip","source":"web-bluetooth"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/setup/provisioned-device", body)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("POST status: got %d want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}
