//go:build linux

package native

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

const defaultBLEReadBufferSize = 512

// TinyGoTransport is a Linux BLE transport backed by tinygo.org/x/bluetooth.
// It uses fallback endpoint UUIDs because tinygo's central API does not expose
// GATT user-description descriptor reads on all platforms.
type TinyGoTransport struct {
	adapter     *bluetooth.Adapter
	serviceUUID bluetooth.UUID
	device      bluetooth.Device
	connected   bool
	endpoints   map[string]EndpointInfo
	chars       map[string]bluetooth.DeviceCharacteristic
	readSize    int
}

func NewTinyGoTransport() (*TinyGoTransport, error) {
	serviceUUID, err := bluetooth.ParseUUID(ProvisioningServiceUUID)
	if err != nil {
		return nil, fmt.Errorf("parse provisioning service UUID: %w", err)
	}
	return &TinyGoTransport{
		adapter:     bluetooth.DefaultAdapter,
		serviceUUID: serviceUUID,
		endpoints:   map[string]EndpointInfo{},
		chars:       map[string]bluetooth.DeviceCharacteristic{},
		readSize:    defaultBLEReadBufferSize,
	}, nil
}

func (t *TinyGoTransport) Connect(ctx context.Context, serviceName string) error {
	if t.adapter == nil {
		return fmt.Errorf("bluetooth adapter is nil")
	}
	if err := t.adapter.Enable(); err != nil {
		return fmt.Errorf("enable bluetooth adapter: %w", err)
	}

	result, err := t.scanForDevice(ctx, serviceName)
	if err != nil {
		return err
	}

	device, err := t.adapter.Connect(result.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return fmt.Errorf("connect to %s (%s): %w", serviceName, result.Address.String(), err)
	}
	t.device = device
	t.connected = true

	services, err := device.DiscoverServices([]bluetooth.UUID{t.serviceUUID})
	if err != nil {
		_ = t.Disconnect(context.Background())
		return fmt.Errorf("discover provisioning service %s: %w", ProvisioningServiceUUID, err)
	}
	if len(services) == 0 {
		_ = t.Disconnect(context.Background())
		return fmt.Errorf("provisioning service %s not found", ProvisioningServiceUUID)
	}

	wanted, err := parseEndpointUUIDs()
	if err != nil {
		_ = t.Disconnect(context.Background())
		return err
	}

	uuids := make([]bluetooth.UUID, 0, len(wanted))
	for _, uuid := range wanted {
		uuids = append(uuids, uuid)
	}
	chars, err := services[0].DiscoverCharacteristics(uuids)
	if err != nil {
		_ = t.Disconnect(context.Background())
		return fmt.Errorf("discover provisioning characteristics: %w", err)
	}

	byUUID := map[string]bluetooth.DeviceCharacteristic{}
	for _, ch := range chars {
		byUUID[strings.ToLower(ch.UUID().String())] = ch
	}

	t.endpoints = map[string]EndpointInfo{}
	t.chars = map[string]bluetooth.DeviceCharacteristic{}
	for name, uuidText := range FallbackEndpointUUIDs {
		ch, ok := byUUID[strings.ToLower(uuidText)]
		if !ok {
			_ = t.Disconnect(context.Background())
			return fmt.Errorf("endpoint %s characteristic %s not found", name, uuidText)
		}
		t.endpoints[name] = EndpointInfo{Name: name, UUID: uuidText}
		t.chars[name] = ch
	}
	return nil
}

func (t *TinyGoTransport) Disconnect(ctx context.Context) error {
	if !t.connected {
		return nil
	}
	err := t.device.Disconnect()
	t.connected = false
	return err
}

func (t *TinyGoTransport) Endpoints() map[string]EndpointInfo {
	out := make(map[string]EndpointInfo, len(t.endpoints))
	for k, v := range t.endpoints {
		out[k] = v
	}
	return out
}

func (t *TinyGoTransport) Send(ctx context.Context, endpoint string, request []byte) ([]byte, error) {
	ch, ok := t.chars[endpoint]
	if !ok {
		return nil, fmt.Errorf("endpoint %q not discovered", endpoint)
	}
	if _, err := ch.Write(request); err != nil {
		return nil, fmt.Errorf("write endpoint %s: %w", endpoint, err)
	}

	buf := make([]byte, t.readSize)
	n, err := ch.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read endpoint %s: %w", endpoint, err)
	}
	return append([]byte(nil), buf[:n]...), nil
}

func (t *TinyGoTransport) scanForDevice(ctx context.Context, serviceName string) (bluetooth.ScanResult, error) {
	if serviceName == "" {
		return bluetooth.ScanResult{}, fmt.Errorf("service name is required")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	found := make(chan bluetooth.ScanResult, 1)
	var stopOnce sync.Once
	stop := func() { _ = t.adapter.StopScan() }

	go func() {
		<-ctx.Done()
		stopOnce.Do(stop)
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- t.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			if result.LocalName() != serviceName {
				return
			}
			select {
			case found <- result:
			default:
			}
			stopOnce.Do(stop)
		})
	}()

	select {
	case result := <-found:
		cancel()
		return result, nil
	case err := <-errCh:
		if err != nil {
			return bluetooth.ScanResult{}, fmt.Errorf("scan for %s: %w", serviceName, err)
		}
		return bluetooth.ScanResult{}, fmt.Errorf("scan stopped before finding %s", serviceName)
	case <-ctx.Done():
		return bluetooth.ScanResult{}, fmt.Errorf("scan for %s timed out: %w", serviceName, ctx.Err())
	case <-time.After(30 * time.Second):
		return bluetooth.ScanResult{}, fmt.Errorf("scan for %s timed out", serviceName)
	}
}

func parseEndpointUUIDs() (map[string]bluetooth.UUID, error) {
	out := make(map[string]bluetooth.UUID, len(FallbackEndpointUUIDs))
	for name, uuidText := range FallbackEndpointUUIDs {
		uuid, err := bluetooth.ParseUUID(uuidText)
		if err != nil {
			return nil, fmt.Errorf("parse endpoint %s UUID %s: %w", name, uuidText, err)
		}
		out[name] = uuid
	}
	return out, nil
}
