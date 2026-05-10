package app

import (
	"context"
	"fmt"
	"time"

	nativeprov "github.com/go-go-golems/almanach/internal/provisioning/native"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

func runNativeBLEProvision(ctx context.Context, s *BLEProvisionSettings, gp middlewares.Processor, readPassphrase bool) error {
	if s.Action != "version" && s.Action != "provision" {
		return fmt.Errorf("native implementation currently supports only --action version and --action provision; got %q", s.Action)
	}

	protoVer := s.ProtoVer
	if protoVer == "" {
		protoVer = nativeprov.DefaultProtoVersion
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(s.Timeout)*time.Second)
	defer cancel()

	if s.DryRun {
		return gp.AddRow(ctx, types.NewRow(
			types.MRP("action", s.Action),
			types.MRP("service_name", s.ServiceName),
			types.MRP("ssid", s.SSID),
			types.MRP("pop", s.Pop),
			types.MRP("implementation", "native"),
			types.MRP("proto_ver", protoVer),
			types.MRP("dry_run", true),
			types.MRP("read_passphrase_from_stdin", readPassphrase),
		))
	}

	transport, err := nativeprov.NewTinyGoTransport()
	if err != nil {
		return err
	}
	started := time.Now()
	if err := transport.Connect(runCtx, s.ServiceName); err != nil {
		return fmt.Errorf("native BLE connect: %w", err)
	}
	defer transport.Disconnect(context.Background())

	client := nativeprov.NewClient(transport)
	info, err := client.VerifyVersion(runCtx, protoVer)
	if err != nil {
		return fmt.Errorf("native proto-ver: %w", err)
	}

	if s.Action == "version" {
		duration := time.Since(started)
		return gp.AddRow(ctx, types.NewRow(
			types.MRP("action", s.Action),
			types.MRP("service_name", s.ServiceName),
			types.MRP("pop", s.Pop),
			types.MRP("implementation", "native"),
			types.MRP("proto_ver", info.Version),
			types.MRP("sec_ver", info.SecVersion),
			types.MRP("sec_patch_ver", info.SecPatchVer),
			types.MRP("capabilities", info.Capabilities),
			types.MRP("raw_response", info.Raw),
			types.MRP("duration_ms", duration.Milliseconds()),
			types.MRP("endpoint_count", len(transport.Endpoints())),
		))
	}

	if _, err := client.EstablishSecurity1(runCtx, s.Pop); err != nil {
		return fmt.Errorf("native security1: %w", err)
	}
	status, err := client.ProvisionWiFi(runCtx, s.SSID, s.Passphrase, time.Second)
	if err != nil {
		return fmt.Errorf("native provision WiFi: %w", err)
	}
	duration := time.Since(started)

	return gp.AddRow(ctx, types.NewRow(
		types.MRP("action", s.Action),
		types.MRP("service_name", s.ServiceName),
		types.MRP("ssid", s.SSID),
		types.MRP("pop", s.Pop),
		types.MRP("implementation", "native"),
		types.MRP("proto_ver", info.Version),
		types.MRP("sec_ver", info.SecVersion),
		types.MRP("sec_patch_ver", info.SecPatchVer),
		types.MRP("capabilities", info.Capabilities),
		types.MRP("wifi_status", status.Status.String()),
		types.MRP("wifi_state", status.StateText()),
		types.MRP("wifi_fail_reason", status.FailReason.String()),
		types.MRP("wifi_attempts_remaining", status.AttemptsRemaining),
		types.MRP("duration_ms", duration.Milliseconds()),
		types.MRP("endpoint_count", len(transport.Endpoints())),
		types.MRP("read_passphrase_from_stdin", readPassphrase),
	))
}
