package native

import "context"

const (
	EndpointProtoVer    = "proto-ver"
	EndpointProvSession = "prov-session"
	EndpointProvConfig  = "prov-config"
	EndpointProvScan    = "prov-scan"
	EndpointProvCtrl    = "prov-ctrl"
)

// EndpointInfo describes one ESP-IDF protocomm endpoint discovered on the BLE
// provisioning service. For BLE transports, UUID is the characteristic UUID.
type EndpointInfo struct {
	Name string
	UUID string
}

// Transport is the byte-oriented endpoint transport used by the native
// provisioning protocol implementation.
//
// ESP-IDF BLE provisioning maps logical endpoints such as proto-ver and
// prov-config to GATT characteristics. Higher protocol layers should not know
// about BLE handles, descriptors, or BlueZ objects; they only send bytes to an
// endpoint and receive response bytes.
type Transport interface {
	Connect(ctx context.Context, serviceName string) error
	Disconnect(ctx context.Context) error
	Endpoints() map[string]EndpointInfo
	Send(ctx context.Context, endpoint string, request []byte) ([]byte, error)
}
