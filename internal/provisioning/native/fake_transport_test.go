package native

import (
	"context"
	"fmt"
)

type fakeTransport struct {
	endpoints map[string]EndpointInfo
	handlers  map[string]func([]byte) ([]byte, error)
	calls     []fakeCall
	connected bool
}

type fakeCall struct {
	Endpoint string
	Request  string
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{
		endpoints: map[string]EndpointInfo{
			EndpointProtoVer: {Name: EndpointProtoVer, UUID: "021aff53-0382-4aea-bff4-6b3f1c5adfb4"},
		},
		handlers: map[string]func([]byte) ([]byte, error){},
	}
}

func (f *fakeTransport) Connect(ctx context.Context, serviceName string) error {
	f.connected = true
	return nil
}

func (f *fakeTransport) Disconnect(ctx context.Context) error {
	f.connected = false
	return nil
}

func (f *fakeTransport) Endpoints() map[string]EndpointInfo {
	out := make(map[string]EndpointInfo, len(f.endpoints))
	for k, v := range f.endpoints {
		out[k] = v
	}
	return out
}

func (f *fakeTransport) Send(ctx context.Context, endpoint string, request []byte) ([]byte, error) {
	f.calls = append(f.calls, fakeCall{Endpoint: endpoint, Request: string(request)})
	h, ok := f.handlers[endpoint]
	if !ok {
		return nil, fmt.Errorf("no handler for endpoint %q", endpoint)
	}
	return h(request)
}
