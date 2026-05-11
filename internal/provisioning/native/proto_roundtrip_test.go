package native

import (
	"bytes"
	"testing"

	espidf "github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf"
	"google.golang.org/protobuf/proto"
)

func TestGeneratedSessionDataRoundTrip(t *testing.T) {
	clientKey := bytes.Repeat([]byte{0x42}, 32)
	msg := &espidf.SessionData{
		SecVer: espidf.SecSchemeVersion_SecScheme1,
		Proto: &espidf.SessionData_Sec1{
			Sec1: &espidf.Sec1Payload{
				Msg: espidf.Sec1MsgType_Session_Command0,
				Payload: &espidf.Sec1Payload_Sc0{
					Sc0: &espidf.SessionCmd0{ClientPubkey: clientKey},
				},
			},
		},
	}

	encoded, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded espidf.SessionData
	if err := proto.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetSecVer() != espidf.SecSchemeVersion_SecScheme1 {
		t.Fatalf("sec_ver: got %v", decoded.GetSecVer())
	}
	if got := decoded.GetSec1().GetSc0().GetClientPubkey(); !bytes.Equal(got, clientKey) {
		t.Fatalf("client pubkey mismatch: got %x want %x", got, clientKey)
	}
}

func TestGeneratedWiFiConfigPayloadRoundTrip(t *testing.T) {
	msg := &espidf.WiFiConfigPayload{
		Msg: espidf.WiFiConfigMsgType_TypeCmdSetConfig,
		Payload: &espidf.WiFiConfigPayload_CmdSetConfig{
			CmdSetConfig: &espidf.CmdSetConfig{
				Ssid:       []byte("Workshop WiFi"),
				Passphrase: []byte("correct horse battery staple"),
			},
		},
	}

	encoded, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded espidf.WiFiConfigPayload
	if err := proto.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetMsg() != espidf.WiFiConfigMsgType_TypeCmdSetConfig {
		t.Fatalf("msg: got %v", decoded.GetMsg())
	}
	cfg := decoded.GetCmdSetConfig()
	if string(cfg.GetSsid()) != "Workshop WiFi" {
		t.Fatalf("ssid: got %q", cfg.GetSsid())
	}
	if string(cfg.GetPassphrase()) != "correct horse battery staple" {
		t.Fatalf("passphrase: got %q", cfg.GetPassphrase())
	}
}
