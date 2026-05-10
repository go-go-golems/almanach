package native

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestVerifyProtoVersionBareString(t *testing.T) {
	ft := newFakeTransport()
	ft.handlers[EndpointProtoVer] = func(req []byte) ([]byte, error) {
		if got, want := string(req), "v1.1"; got != want {
			t.Fatalf("request: got %q want %q", got, want)
		}
		return []byte("v1.1"), nil
	}

	info, err := VerifyProtoVersion(context.Background(), ft, "v1.1")
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v1.1" || info.Raw != "v1.1" {
		t.Fatalf("unexpected info: %#v", info)
	}
	if len(ft.calls) != 1 || ft.calls[0].Endpoint != EndpointProtoVer {
		t.Fatalf("unexpected calls: %#v", ft.calls)
	}
}

func TestVerifyProtoVersionJSON(t *testing.T) {
	ft := newFakeTransport()
	ft.handlers[EndpointProtoVer] = func(req []byte) ([]byte, error) {
		return []byte(`{ "prov": { "ver": "v1.1", "sec_ver": 1, "sec_patch_ver": 0, "cap": ["wifi_scan"] } }`), nil
	}

	info, err := VerifyProtoVersion(context.Background(), ft, "v1.1")
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v1.1" || info.SecVersion != 1 || info.SecPatchVer != 0 {
		t.Fatalf("unexpected info: %#v", info)
	}
	if !reflect.DeepEqual(info.Capabilities, []string{"wifi_scan"}) {
		t.Fatalf("capabilities: got %#v", info.Capabilities)
	}
}

func TestVerifyProtoVersionDefaultsVersion(t *testing.T) {
	ft := newFakeTransport()
	ft.handlers[EndpointProtoVer] = func(req []byte) ([]byte, error) {
		if got, want := string(req), DefaultProtoVersion; got != want {
			t.Fatalf("request: got %q want %q", got, want)
		}
		return []byte(DefaultProtoVersion), nil
	}

	if _, err := VerifyProtoVersion(context.Background(), ft, ""); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyProtoVersionMismatch(t *testing.T) {
	ft := newFakeTransport()
	ft.handlers[EndpointProtoVer] = func(req []byte) ([]byte, error) {
		return []byte(`{ "prov": { "ver": "v0.9" } }`), nil
	}

	_, err := VerifyProtoVersion(context.Background(), ft, "v1.1")
	if err == nil || !strings.Contains(err.Error(), "protocol version mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestVerifyProtoVersionTransportError(t *testing.T) {
	ft := newFakeTransport()
	ft.handlers[EndpointProtoVer] = func(req []byte) ([]byte, error) {
		return nil, errors.New("boom")
	}

	_, err := VerifyProtoVersion(context.Background(), ft, "v1.1")
	if err == nil || !strings.Contains(err.Error(), "send proto-ver") {
		t.Fatalf("expected wrapped transport error, got %v", err)
	}
}

func TestClientVerifyVersion(t *testing.T) {
	ft := newFakeTransport()
	ft.handlers[EndpointProtoVer] = func(req []byte) ([]byte, error) { return []byte("v1.1"), nil }

	client := NewClient(ft)
	info, err := client.VerifyVersion(context.Background(), "v1.1")
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v1.1" {
		t.Fatalf("version: got %q", info.Version)
	}
}
