package native

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf"
)

func TestClientResetWiFiUsesEncryptedControlEndpoint(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	client := NewClient(transport)
	client.Security = NewSecurity1SessionWithReader("alm-0f2320", bytes.NewReader(bytes.Repeat([]byte{0x77}, 64)))
	if err := client.Security.Establish(ctx, transport); err != nil {
		t.Fatalf("establish: %v", err)
	}

	status, err := client.ResetWiFi(ctx)
	if err != nil {
		t.Fatalf("reset wifi: %v", err)
	}
	if status != espidf.Status_Success {
		t.Fatalf("status = %s, want Success", status)
	}
	if transport.resetCount != 1 {
		t.Fatalf("reset count = %d, want 1", transport.resetCount)
	}
}

func TestClientReprovisionWiFiUsesEncryptedControlEndpoint(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	client := NewClient(transport)
	client.Security = NewSecurity1SessionWithReader("alm-0f2320", bytes.NewReader(bytes.Repeat([]byte{0x88}, 64)))
	if err := client.Security.Establish(ctx, transport); err != nil {
		t.Fatalf("establish: %v", err)
	}

	status, err := client.ReprovisionWiFi(ctx)
	if err != nil {
		t.Fatalf("reprovision wifi: %v", err)
	}
	if status != espidf.Status_Success {
		t.Fatalf("status = %s, want Success", status)
	}
	if transport.reprovCount != 1 {
		t.Fatalf("reprov count = %d, want 1", transport.reprovCount)
	}
}

func TestWiFiControlRequiresEstablishedSecurity(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	client := NewClient(transport)
	if _, err := client.ResetWiFi(ctx); err == nil {
		t.Fatalf("expected missing security session error")
	}
	if _, err := client.ReprovisionWiFi(ctx); err == nil {
		t.Fatalf("expected missing security session error")
	}
}
