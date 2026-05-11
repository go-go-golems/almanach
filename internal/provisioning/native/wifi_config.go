package native

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf"
	"google.golang.org/protobuf/proto"
)

type WiFiStatus struct {
	Status            espidf.Status
	State             espidf.WifiStationState
	FailReason        espidf.WifiConnectFailedReason
	HasFailReason     bool
	AttemptsRemaining uint32
}

func (s WiFiStatus) StateText() string {
	switch s.State {
	case espidf.WifiStationState_Connected:
		return "connected"
	case espidf.WifiStationState_Connecting:
		return "connecting"
	case espidf.WifiStationState_Disconnected:
		return "disconnected"
	case espidf.WifiStationState_ConnectionFailed:
		return "failed"
	default:
		return "unknown"
	}
}

func (s WiFiStatus) Terminal() bool {
	return s.State == espidf.WifiStationState_Connected || s.State == espidf.WifiStationState_ConnectionFailed || s.State == espidf.WifiStationState_Disconnected
}

func (s WiFiStatus) FailReasonText() string {
	if !s.HasFailReason {
		return ""
	}
	return s.FailReason.String()
}

func (c *Client) SetWiFiConfig(ctx context.Context, ssid, passphrase string) (espidf.Status, error) {
	if c.Security == nil {
		return espidf.Status_InvalidSession, fmt.Errorf("security1 session is not established")
	}
	return SetWiFiConfig(ctx, c.Transport, c.Security, ssid, passphrase)
}

func (c *Client) ApplyWiFiConfig(ctx context.Context) (espidf.Status, error) {
	if c.Security == nil {
		return espidf.Status_InvalidSession, fmt.Errorf("security1 session is not established")
	}
	return ApplyWiFiConfig(ctx, c.Transport, c.Security)
}

func (c *Client) GetWiFiStatus(ctx context.Context) (*WiFiStatus, error) {
	if c.Security == nil {
		return nil, fmt.Errorf("security1 session is not established")
	}
	return GetWiFiStatus(ctx, c.Transport, c.Security)
}

func (c *Client) ProvisionWiFi(ctx context.Context, ssid, passphrase string, pollInterval time.Duration) (*WiFiStatus, error) {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	if status, err := c.SetWiFiConfig(ctx, ssid, passphrase); err != nil {
		return nil, err
	} else if status != espidf.Status_Success {
		return nil, fmt.Errorf("set WiFi config failed: %s", status)
	}
	if status, err := c.ApplyWiFiConfig(ctx); err != nil {
		return nil, err
	} else if status != espidf.Status_Success {
		return nil, fmt.Errorf("apply WiFi config failed: %s", status)
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		status, err := c.GetWiFiStatus(ctx)
		if err != nil {
			return nil, err
		}
		if status.Terminal() {
			return status, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func SetWiFiConfig(ctx context.Context, t Transport, sec *Security1Session, ssid, passphrase string) (espidf.Status, error) {
	payload := &espidf.WiFiConfigPayload{
		Msg: espidf.WiFiConfigMsgType_TypeCmdSetConfig,
		Payload: &espidf.WiFiConfigPayload_CmdSetConfig{CmdSetConfig: &espidf.CmdSetConfig{
			Ssid:       []byte(ssid),
			Passphrase: []byte(passphrase),
		}},
	}
	resp, err := sendEncryptedWiFiConfig(ctx, t, sec, payload)
	if err != nil {
		return espidf.Status_InternalError, err
	}
	if resp.GetMsg() != espidf.WiFiConfigMsgType_TypeRespSetConfig || resp.GetRespSetConfig() == nil {
		return espidf.Status_InvalidProto, fmt.Errorf("unexpected set-config response message: %s", resp.GetMsg())
	}
	return resp.GetRespSetConfig().GetStatus(), nil
}

func ApplyWiFiConfig(ctx context.Context, t Transport, sec *Security1Session) (espidf.Status, error) {
	payload := &espidf.WiFiConfigPayload{
		Msg:     espidf.WiFiConfigMsgType_TypeCmdApplyConfig,
		Payload: &espidf.WiFiConfigPayload_CmdApplyConfig{CmdApplyConfig: &espidf.CmdApplyConfig{}},
	}
	resp, err := sendEncryptedWiFiConfig(ctx, t, sec, payload)
	if err != nil {
		return espidf.Status_InternalError, err
	}
	if resp.GetMsg() != espidf.WiFiConfigMsgType_TypeRespApplyConfig || resp.GetRespApplyConfig() == nil {
		return espidf.Status_InvalidProto, fmt.Errorf("unexpected apply-config response message: %s", resp.GetMsg())
	}
	return resp.GetRespApplyConfig().GetStatus(), nil
}

func GetWiFiStatus(ctx context.Context, t Transport, sec *Security1Session) (*WiFiStatus, error) {
	payload := &espidf.WiFiConfigPayload{
		Msg:     espidf.WiFiConfigMsgType_TypeCmdGetStatus,
		Payload: &espidf.WiFiConfigPayload_CmdGetStatus{CmdGetStatus: &espidf.CmdGetStatus{}},
	}
	resp, err := sendEncryptedWiFiConfig(ctx, t, sec, payload)
	if err != nil {
		return nil, err
	}
	if resp.GetMsg() != espidf.WiFiConfigMsgType_TypeRespGetStatus || resp.GetRespGetStatus() == nil {
		return nil, fmt.Errorf("unexpected get-status response message: %s", resp.GetMsg())
	}
	status := resp.GetRespGetStatus()
	out := &WiFiStatus{
		Status: status.GetStatus(),
		State:  status.GetStaState(),
	}
	if _, ok := status.GetState().(*espidf.RespGetStatus_FailReason); ok {
		out.FailReason = status.GetFailReason()
		out.HasFailReason = true
	}
	if attempt := status.GetAttemptFailed(); attempt != nil {
		out.AttemptsRemaining = attempt.GetAttemptsRemaining()
	}
	return out, nil
}

func sendEncryptedWiFiConfig(ctx context.Context, t Transport, sec *Security1Session, payload *espidf.WiFiConfigPayload) (*espidf.WiFiConfigPayload, error) {
	if t == nil {
		return nil, fmt.Errorf("transport is nil")
	}
	if sec == nil || !sec.Established() {
		return nil, fmt.Errorf("security1 session is not established")
	}
	plain, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal WiFi config request: %w", err)
	}
	encrypted, err := sec.Encrypt(plain)
	if err != nil {
		return nil, err
	}
	respEncrypted, err := t.Send(ctx, EndpointProvConfig, encrypted)
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", EndpointProvConfig, err)
	}
	respPlain, err := sec.Decrypt(respEncrypted)
	if err != nil {
		return nil, err
	}
	var resp espidf.WiFiConfigPayload
	if err := proto.Unmarshal(respPlain, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal WiFi config response: %w", err)
	}
	return &resp, nil
}
