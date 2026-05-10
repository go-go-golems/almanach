package native

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf"
)

func TestClientProvisionWiFiUsesEncryptedConfigFlow(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	transport.statusQueue = []*espidf.RespGetStatus{
		{Status: espidf.Status_Success, StaState: espidf.WifiStationState_Connecting},
		{Status: espidf.Status_Success, StaState: espidf.WifiStationState_Connected},
	}

	client := NewClient(transport)
	client.Security = NewSecurity1SessionWithReader("alm-0f2320", bytes.NewReader(bytes.Repeat([]byte{0x55}, 64)))
	if err := client.Security.Establish(ctx, transport); err != nil {
		t.Fatalf("establish: %v", err)
	}

	status, err := client.ProvisionWiFi(ctx, "LabNet", "correct horse battery staple", time.Nanosecond)
	if err != nil {
		t.Fatalf("provision wifi: %v", err)
	}
	if status.State != espidf.WifiStationState_Connected || status.StateText() != "connected" {
		t.Fatalf("unexpected final status: %+v", status)
	}
	if transport.setSSID != "LabNet" || transport.setPassphrase != "correct horse battery staple" {
		t.Fatalf("unexpected credentials: ssid=%q pass=%q", transport.setSSID, transport.setPassphrase)
	}
	if transport.applyCount != 1 {
		t.Fatalf("apply count = %d, want 1", transport.applyCount)
	}
}

func TestClientProvisionWiFiStopsOnFailedStatus(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	transport.statusQueue = []*espidf.RespGetStatus{
		{
			Status:   espidf.Status_Success,
			StaState: espidf.WifiStationState_ConnectionFailed,
			State:    &espidf.RespGetStatus_FailReason{FailReason: espidf.WifiConnectFailedReason_AuthError},
		},
	}

	client := NewClient(transport)
	client.Security = NewSecurity1SessionWithReader("alm-0f2320", bytes.NewReader(bytes.Repeat([]byte{0x66}, 64)))
	if err := client.Security.Establish(ctx, transport); err != nil {
		t.Fatalf("establish: %v", err)
	}

	status, err := client.ProvisionWiFi(ctx, "LabNet", "wrong", time.Nanosecond)
	if err != nil {
		t.Fatalf("provision wifi returned error instead of failed status: %v", err)
	}
	if status.StateText() != "failed" {
		t.Fatalf("state text = %q, want failed", status.StateText())
	}
	if status.FailReason != espidf.WifiConnectFailedReason_AuthError {
		t.Fatalf("fail reason = %s, want AuthError", status.FailReason)
	}
}

func TestWiFiConfigRequiresEstablishedSecurity(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	client := NewClient(transport)
	if _, err := client.SetWiFiConfig(ctx, "LabNet", "secret"); err == nil {
		t.Fatalf("expected missing security session error")
	}
}
