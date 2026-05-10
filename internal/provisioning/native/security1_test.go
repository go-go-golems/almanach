package native

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/sha256"
	"fmt"
	"io"
	"testing"

	"github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf"
	"google.golang.org/protobuf/proto"
)

func TestSecurity1EstablishesAndEncryptsWithPoP(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	session := NewSecurity1SessionWithReader("alm-0f2320", bytes.NewReader(bytes.Repeat([]byte{0x42}, 64)))

	if err := session.Establish(ctx, transport); err != nil {
		t.Fatalf("establish security1: %v", err)
	}
	if !session.Established() {
		t.Fatalf("session was not marked established")
	}
	if transport.setup0Count != 1 || transport.setup1Count != 1 {
		t.Fatalf("unexpected setup counts: setup0=%d setup1=%d", transport.setup0Count, transport.setup1Count)
	}

	plaintext := []byte("ssid=LabNet;passphrase=correct horse battery staple")
	ciphertext, err := session.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatalf("ciphertext unexpectedly equals plaintext")
	}
	devicePlaintext := transport.crypt(ciphertext)
	if !bytes.Equal(devicePlaintext, plaintext) {
		t.Fatalf("device decrypted %q, want %q", devicePlaintext, plaintext)
	}

	deviceResponse := []byte("status=success")
	deviceCiphertext := transport.crypt(deviceResponse)
	clientResponse, err := session.Decrypt(deviceCiphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(clientResponse, deviceResponse) {
		t.Fatalf("client decrypted %q, want %q", clientResponse, deviceResponse)
	}
}

func TestSecurity1RejectsWrongPoP(t *testing.T) {
	ctx := context.Background()
	transport := newFakeSecurity1Transport(t, "alm-0f2320")
	session := NewSecurity1SessionWithReader("wrong-pop", bytes.NewReader(bytes.Repeat([]byte{0x33}, 64)))

	err := session.Establish(ctx, transport)
	if err == nil {
		t.Fatalf("expected wrong PoP to fail")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("shared key")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSecurity1EncryptBeforeEstablishFails(t *testing.T) {
	session := NewSecurity1Session("alm-0f2320")
	if _, err := session.Encrypt([]byte("hello")); err == nil {
		t.Fatalf("expected encrypt before establish to fail")
	}
}

type fakeSecurity1Transport struct {
	t *testing.T

	pop          []byte
	deviceKey    *ecdh.PrivateKey
	devicePubkey []byte
	deviceRandom []byte
	stream       cipher.Stream

	clientPubkey []byte
	setup0Count  int
	setup1Count  int

	setSSID       string
	setPassphrase string
	applyCount    int
	statusQueue   []*espidf.RespGetStatus
}

func newFakeSecurity1Transport(t *testing.T, pop string) *fakeSecurity1Transport {
	t.Helper()
	devicePrivate := make([]byte, 32)
	for i := range devicePrivate {
		devicePrivate[i] = byte(i + 1)
	}
	deviceKey, err := ecdh.X25519().NewPrivateKey(devicePrivate)
	if err != nil {
		t.Fatalf("device private key: %v", err)
	}
	return &fakeSecurity1Transport{
		t:            t,
		pop:          []byte(pop),
		deviceKey:    deviceKey,
		devicePubkey: append([]byte(nil), deviceKey.PublicKey().Bytes()...),
		deviceRandom: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
	}
}

func (t *fakeSecurity1Transport) Connect(ctx context.Context, serviceName string) error { return nil }
func (t *fakeSecurity1Transport) Disconnect(ctx context.Context) error                  { return nil }
func (t *fakeSecurity1Transport) Endpoints() map[string]EndpointInfo                    { return nil }

func (t *fakeSecurity1Transport) Send(ctx context.Context, endpoint string, request []byte) ([]byte, error) {
	switch endpoint {
	case EndpointProvSession:
		var msg espidf.SessionData
		if err := proto.Unmarshal(request, &msg); err != nil {
			return nil, err
		}
		sec1 := msg.GetSec1()
		if msg.GetSecVer() != espidf.SecSchemeVersion_SecScheme1 || sec1 == nil {
			return nil, fmt.Errorf("unexpected session message: sec_ver=%s sec1=%v", msg.GetSecVer(), sec1)
		}
		switch sec1.GetMsg() {
		case espidf.Sec1MsgType_Session_Command0:
			return t.handleSetup0(sec1.GetSc0())
		case espidf.Sec1MsgType_Session_Command1:
			return t.handleSetup1(sec1.GetSc1())
		default:
			return nil, fmt.Errorf("unexpected security1 message type %s", sec1.GetMsg())
		}
	case EndpointProvConfig:
		return t.handleConfig(request)
	default:
		return nil, fmt.Errorf("unexpected endpoint %s", endpoint)
	}
}

func (t *fakeSecurity1Transport) handleSetup0(cmd *espidf.SessionCmd0) ([]byte, error) {
	if cmd == nil {
		return nil, fmt.Errorf("missing setup0 command")
	}
	t.setup0Count++
	t.clientPubkey = append([]byte(nil), cmd.GetClientPubkey()...)
	clientKey, err := ecdh.X25519().NewPublicKey(t.clientPubkey)
	if err != nil {
		return nil, err
	}
	shared, err := t.deviceKey.ECDH(clientKey)
	if err != nil {
		return nil, err
	}
	if len(t.pop) > 0 {
		digest := sha256.Sum256(t.pop)
		xorInPlace(shared, digest[:])
	}
	block, err := aes.NewCipher(shared)
	if err != nil {
		return nil, err
	}
	t.stream = cipher.NewCTR(block, t.deviceRandom)

	return proto.Marshal(&espidf.SessionData{
		SecVer: espidf.SecSchemeVersion_SecScheme1,
		Proto: &espidf.SessionData_Sec1{Sec1: &espidf.Sec1Payload{
			Msg: espidf.Sec1MsgType_Session_Response0,
			Payload: &espidf.Sec1Payload_Sr0{Sr0: &espidf.SessionResp0{
				DevicePubkey: t.devicePubkey,
				DeviceRandom: t.deviceRandom,
			}},
		}},
	})
}

func (t *fakeSecurity1Transport) handleSetup1(cmd *espidf.SessionCmd1) ([]byte, error) {
	if cmd == nil {
		return nil, fmt.Errorf("missing setup1 command")
	}
	t.setup1Count++
	clientVerify := append([]byte(nil), cmd.GetClientVerifyData()...)
	t.stream.XORKeyStream(clientVerify, clientVerify)
	if !bytes.Equal(clientVerify, t.devicePubkey) {
		return nil, fmt.Errorf("client failed to prove possession of shared key")
	}

	deviceVerify := append([]byte(nil), t.clientPubkey...)
	t.stream.XORKeyStream(deviceVerify, deviceVerify)
	return proto.Marshal(&espidf.SessionData{
		SecVer: espidf.SecSchemeVersion_SecScheme1,
		Proto: &espidf.SessionData_Sec1{Sec1: &espidf.Sec1Payload{
			Msg: espidf.Sec1MsgType_Session_Response1,
			Payload: &espidf.Sec1Payload_Sr1{Sr1: &espidf.SessionResp1{
				DeviceVerifyData: deviceVerify,
			}},
		}},
	})
}

func (t *fakeSecurity1Transport) handleConfig(request []byte) ([]byte, error) {
	plain := t.crypt(request)
	var msg espidf.WiFiConfigPayload
	if err := proto.Unmarshal(plain, &msg); err != nil {
		return nil, err
	}
	var resp *espidf.WiFiConfigPayload
	switch msg.GetMsg() {
	case espidf.WiFiConfigMsgType_TypeCmdSetConfig:
		cmd := msg.GetCmdSetConfig()
		if cmd == nil {
			return nil, fmt.Errorf("missing set-config command")
		}
		t.setSSID = string(cmd.GetSsid())
		t.setPassphrase = string(cmd.GetPassphrase())
		resp = &espidf.WiFiConfigPayload{
			Msg:     espidf.WiFiConfigMsgType_TypeRespSetConfig,
			Payload: &espidf.WiFiConfigPayload_RespSetConfig{RespSetConfig: &espidf.RespSetConfig{Status: espidf.Status_Success}},
		}
	case espidf.WiFiConfigMsgType_TypeCmdApplyConfig:
		t.applyCount++
		resp = &espidf.WiFiConfigPayload{
			Msg:     espidf.WiFiConfigMsgType_TypeRespApplyConfig,
			Payload: &espidf.WiFiConfigPayload_RespApplyConfig{RespApplyConfig: &espidf.RespApplyConfig{Status: espidf.Status_Success}},
		}
	case espidf.WiFiConfigMsgType_TypeCmdGetStatus:
		status := &espidf.RespGetStatus{Status: espidf.Status_Success, StaState: espidf.WifiStationState_Connected}
		if len(t.statusQueue) > 0 {
			status = t.statusQueue[0]
			t.statusQueue = t.statusQueue[1:]
		}
		resp = &espidf.WiFiConfigPayload{
			Msg:     espidf.WiFiConfigMsgType_TypeRespGetStatus,
			Payload: &espidf.WiFiConfigPayload_RespGetStatus{RespGetStatus: status},
		}
	default:
		return nil, fmt.Errorf("unexpected config message type %s", msg.GetMsg())
	}
	respPlain, err := proto.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return t.crypt(respPlain), nil
}

func (t *fakeSecurity1Transport) crypt(data []byte) []byte {
	out := append([]byte(nil), data...)
	t.stream.XORKeyStream(out, out)
	return out
}

var _ io.Reader = (*bytes.Reader)(nil)
