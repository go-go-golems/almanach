package native

import (
	"context"
	"fmt"

	"github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf"
	"google.golang.org/protobuf/proto"
)

type WiFiCtrlAction string

const (
	WiFiCtrlActionReset  WiFiCtrlAction = "reset"
	WiFiCtrlActionReprov WiFiCtrlAction = "reprov"
)

func (c *Client) ResetWiFi(ctx context.Context) (espidf.Status, error) {
	if c.Security == nil {
		return espidf.Status_InvalidSession, fmt.Errorf("security1 session is not established")
	}
	return SendWiFiControl(ctx, c.Transport, c.Security, WiFiCtrlActionReset)
}

func (c *Client) ReprovisionWiFi(ctx context.Context) (espidf.Status, error) {
	if c.Security == nil {
		return espidf.Status_InvalidSession, fmt.Errorf("security1 session is not established")
	}
	return SendWiFiControl(ctx, c.Transport, c.Security, WiFiCtrlActionReprov)
}

func SendWiFiControl(ctx context.Context, t Transport, sec *Security1Session, action WiFiCtrlAction) (espidf.Status, error) {
	payload, wantMsg, err := wifiCtrlPayload(action)
	if err != nil {
		return espidf.Status_InvalidArgument, err
	}
	resp, err := sendEncryptedControl(ctx, t, sec, payload)
	if err != nil {
		return espidf.Status_InternalError, err
	}
	if resp.GetMsg() != wantMsg {
		return espidf.Status_InvalidProto, fmt.Errorf("unexpected %s response message: got %s want %s", action, resp.GetMsg(), wantMsg)
	}
	return resp.GetStatus(), nil
}

func wifiCtrlPayload(action WiFiCtrlAction) (*espidf.WiFiCtrlPayload, espidf.WiFiCtrlMsgType, error) {
	switch action {
	case WiFiCtrlActionReset:
		return &espidf.WiFiCtrlPayload{
			Msg: espidf.WiFiCtrlMsgType_TypeCmdCtrlReset,
		}, espidf.WiFiCtrlMsgType_TypeRespCtrlReset, nil
	case WiFiCtrlActionReprov:
		return &espidf.WiFiCtrlPayload{
			Msg: espidf.WiFiCtrlMsgType_TypeCmdCtrlReprov,
		}, espidf.WiFiCtrlMsgType_TypeRespCtrlReprov, nil
	default:
		return nil, espidf.WiFiCtrlMsgType_TypeCtrlReserved, fmt.Errorf("unsupported WiFi control action %q", action)
	}
}

func sendEncryptedControl(ctx context.Context, t Transport, sec *Security1Session, payload *espidf.WiFiCtrlPayload) (*espidf.WiFiCtrlPayload, error) {
	if t == nil {
		return nil, fmt.Errorf("transport is nil")
	}
	if sec == nil || !sec.Established() {
		return nil, fmt.Errorf("security1 session is not established")
	}
	plain, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal WiFi control request: %w", err)
	}
	encrypted, err := sec.Encrypt(plain)
	if err != nil {
		return nil, err
	}
	respEncrypted, err := t.Send(ctx, EndpointProvCtrl, encrypted)
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", EndpointProvCtrl, err)
	}
	respPlain, err := sec.Decrypt(respEncrypted)
	if err != nil {
		return nil, err
	}
	var resp espidf.WiFiCtrlPayload
	if err := proto.Unmarshal(respPlain, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal WiFi control response: %w", err)
	}
	return &resp, nil
}
