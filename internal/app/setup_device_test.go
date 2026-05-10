package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvisionedDeviceAPIStoresPrinterIP(t *testing.T) {
	server := &Server{cfg: Config{}}
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
