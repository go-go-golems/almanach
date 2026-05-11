package native

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf"
	"google.golang.org/protobuf/proto"
)

const (
	security1KeySize    = 32
	security1RandomSize = aes.BlockSize
)

// Security1Session implements ESP-IDF protocomm Security 1.
//
// Security 1 performs an X25519 key exchange over the prov-session endpoint,
// optionally XORs the shared secret with SHA-256(PoP), and then uses the result
// as an AES-256-CTR key. ESP-IDF's Python client uses one continuous CTR stream
// for both encryption and decryption; this type intentionally mirrors that
// stateful behavior.
type Security1Session struct {
	pop []byte
	rnd io.Reader

	privateKey   *ecdh.PrivateKey
	clientPubkey []byte
	devicePubkey []byte
	deviceRandom []byte
	cipherStream cipher.Stream
	established  bool
}

func NewSecurity1Session(pop string) *Security1Session {
	return NewSecurity1SessionWithReader(pop, rand.Reader)
}

func NewSecurity1SessionWithReader(pop string, rnd io.Reader) *Security1Session {
	if rnd == nil {
		rnd = rand.Reader
	}
	return &Security1Session{pop: []byte(pop), rnd: rnd}
}

func (s *Security1Session) Established() bool {
	return s.established
}

func (s *Security1Session) ClientPublicKey() []byte {
	return append([]byte(nil), s.clientPubkey...)
}

func (s *Security1Session) DevicePublicKey() []byte {
	return append([]byte(nil), s.devicePubkey...)
}

func (s *Security1Session) Establish(ctx context.Context, t Transport) error {
	if t == nil {
		return fmt.Errorf("transport is nil")
	}

	setup0, err := s.setup0Request()
	if err != nil {
		return err
	}
	resp0, err := t.Send(ctx, EndpointProvSession, setup0)
	if err != nil {
		return fmt.Errorf("send security1 setup0: %w", err)
	}
	if err := s.handleSetup0Response(resp0); err != nil {
		return err
	}

	setup1, err := s.setup1Request()
	if err != nil {
		return err
	}
	resp1, err := t.Send(ctx, EndpointProvSession, setup1)
	if err != nil {
		return fmt.Errorf("send security1 setup1: %w", err)
	}
	if err := s.handleSetup1Response(resp1); err != nil {
		return err
	}

	s.established = true
	return nil
}

func (s *Security1Session) Encrypt(data []byte) ([]byte, error) {
	return s.crypt(data)
}

func (s *Security1Session) Decrypt(data []byte) ([]byte, error) {
	return s.crypt(data)
}

func (s *Security1Session) crypt(data []byte) ([]byte, error) {
	if !s.established || s.cipherStream == nil {
		return nil, fmt.Errorf("security1 session is not established")
	}
	out := append([]byte(nil), data...)
	s.cipherStream.XORKeyStream(out, out)
	return out, nil
}

func (s *Security1Session) setup0Request() ([]byte, error) {
	privateKey, err := ecdh.X25519().GenerateKey(s.rnd)
	if err != nil {
		return nil, fmt.Errorf("generate X25519 client key: %w", err)
	}
	s.privateKey = privateKey
	s.clientPubkey = append([]byte(nil), privateKey.PublicKey().Bytes()...)

	msg := &espidf.SessionData{
		SecVer: espidf.SecSchemeVersion_SecScheme1,
		Proto: &espidf.SessionData_Sec1{Sec1: &espidf.Sec1Payload{
			Msg: espidf.Sec1MsgType_Session_Command0,
			Payload: &espidf.Sec1Payload_Sc0{Sc0: &espidf.SessionCmd0{
				ClientPubkey: s.clientPubkey,
			}},
		}},
	}
	out, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal security1 setup0 request: %w", err)
	}
	return out, nil
}

func (s *Security1Session) handleSetup0Response(data []byte) error {
	var msg espidf.SessionData
	if err := proto.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal security1 setup0 response: %w", err)
	}
	if msg.GetSecVer() != espidf.SecSchemeVersion_SecScheme1 {
		return fmt.Errorf("incorrect security scheme: got %s want %s", msg.GetSecVer(), espidf.SecSchemeVersion_SecScheme1)
	}
	sec1 := msg.GetSec1()
	if sec1 == nil || sec1.GetMsg() != espidf.Sec1MsgType_Session_Response0 || sec1.GetSr0() == nil {
		return fmt.Errorf("unexpected security1 setup0 response message: %v", sec1.GetMsg())
	}
	devicePubkey := sec1.GetSr0().GetDevicePubkey()
	deviceRandom := sec1.GetSr0().GetDeviceRandom()
	if len(devicePubkey) != security1KeySize {
		return fmt.Errorf("invalid device public key length: got %d want %d", len(devicePubkey), security1KeySize)
	}
	if len(deviceRandom) != security1RandomSize {
		return fmt.Errorf("invalid device random length: got %d want %d", len(deviceRandom), security1RandomSize)
	}
	if s.privateKey == nil {
		return fmt.Errorf("client private key is not initialized")
	}

	deviceKey, err := ecdh.X25519().NewPublicKey(devicePubkey)
	if err != nil {
		return fmt.Errorf("parse device X25519 public key: %w", err)
	}
	shared, err := s.privateKey.ECDH(deviceKey)
	if err != nil {
		return fmt.Errorf("derive X25519 shared key: %w", err)
	}
	if len(s.pop) > 0 {
		digest := sha256.Sum256(s.pop)
		xorInPlace(shared, digest[:])
	}

	block, err := aes.NewCipher(shared)
	if err != nil {
		return fmt.Errorf("initialize AES-CTR cipher: %w", err)
	}
	s.devicePubkey = append([]byte(nil), devicePubkey...)
	s.deviceRandom = append([]byte(nil), deviceRandom...)
	s.cipherStream = cipher.NewCTR(block, deviceRandom)
	return nil
}

func (s *Security1Session) setup1Request() ([]byte, error) {
	if s.cipherStream == nil || len(s.devicePubkey) == 0 {
		return nil, fmt.Errorf("security1 setup0 is not complete")
	}
	clientVerify := append([]byte(nil), s.devicePubkey...)
	s.cipherStream.XORKeyStream(clientVerify, clientVerify)

	msg := &espidf.SessionData{
		SecVer: espidf.SecSchemeVersion_SecScheme1,
		Proto: &espidf.SessionData_Sec1{Sec1: &espidf.Sec1Payload{
			Msg: espidf.Sec1MsgType_Session_Command1,
			Payload: &espidf.Sec1Payload_Sc1{Sc1: &espidf.SessionCmd1{
				ClientVerifyData: clientVerify,
			}},
		}},
	}
	out, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal security1 setup1 request: %w", err)
	}
	return out, nil
}

func (s *Security1Session) handleSetup1Response(data []byte) error {
	var msg espidf.SessionData
	if err := proto.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal security1 setup1 response: %w", err)
	}
	if msg.GetSecVer() != espidf.SecSchemeVersion_SecScheme1 {
		return fmt.Errorf("unsupported security protocol: got %s", msg.GetSecVer())
	}
	sec1 := msg.GetSec1()
	if sec1 == nil || sec1.GetMsg() != espidf.Sec1MsgType_Session_Response1 || sec1.GetSr1() == nil {
		return fmt.Errorf("unexpected security1 setup1 response message: %v", sec1.GetMsg())
	}
	deviceVerify := append([]byte(nil), sec1.GetSr1().GetDeviceVerifyData()...)
	if len(deviceVerify) != len(s.clientPubkey) {
		return fmt.Errorf("invalid device verify length: got %d want %d", len(deviceVerify), len(s.clientPubkey))
	}
	s.cipherStream.XORKeyStream(deviceVerify, deviceVerify)
	if string(deviceVerify) != string(s.clientPubkey) {
		return fmt.Errorf("failed to verify device")
	}
	return nil
}

func xorInPlace(dst, mask []byte) {
	for i := 0; i < len(dst) && i < len(mask); i++ {
		dst[i] ^= mask[i]
	}
}
