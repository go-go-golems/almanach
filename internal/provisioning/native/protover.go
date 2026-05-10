package native

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const DefaultProtoVersion = "v1.1"

// ProtoInfo is the parsed response from ESP-IDF's plaintext proto-ver endpoint.
type ProtoInfo struct {
	Version      string
	SecVersion   int
	SecPatchVer  int
	Capabilities []string
	Raw          string
}

type protoVerEnvelope struct {
	Prov struct {
		Version      string   `json:"ver"`
		SecVersion   int      `json:"sec_ver"`
		SecPatchVer  int      `json:"sec_patch_ver"`
		Capabilities []string `json:"cap"`
	} `json:"prov"`
}

// VerifyProtoVersion sends the expected protocol version to ESP-IDF's proto-ver
// endpoint and accepts both response shapes supported by esp_prov.py:
//
//   - a bare version string such as "v1.1"
//   - a JSON capability object such as
//     {"prov":{"ver":"v1.1","sec_ver":1,"sec_patch_ver":0,"cap":["wifi_scan"]}}
func VerifyProtoVersion(ctx context.Context, t Transport, want string) (*ProtoInfo, error) {
	if t == nil {
		return nil, fmt.Errorf("transport is nil")
	}
	if want == "" {
		want = DefaultProtoVersion
	}

	resp, err := t.Send(ctx, EndpointProtoVer, []byte(want))
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", EndpointProtoVer, err)
	}

	text := strings.TrimSpace(string(resp))
	if strings.EqualFold(text, want) {
		return &ProtoInfo{Version: want, Raw: text}, nil
	}

	var envelope protoVerEnvelope
	if err := json.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("unexpected proto-ver response %q: %w", text, err)
	}
	if !strings.EqualFold(envelope.Prov.Version, want) {
		return nil, fmt.Errorf("protocol version mismatch: got %q want %q", envelope.Prov.Version, want)
	}

	return &ProtoInfo{
		Version:      envelope.Prov.Version,
		SecVersion:   envelope.Prov.SecVersion,
		SecPatchVer:  envelope.Prov.SecPatchVer,
		Capabilities: append([]string(nil), envelope.Prov.Capabilities...),
		Raw:          text,
	}, nil
}
