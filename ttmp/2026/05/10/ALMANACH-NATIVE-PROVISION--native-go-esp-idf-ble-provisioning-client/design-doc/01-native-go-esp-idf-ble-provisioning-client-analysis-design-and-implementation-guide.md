---
Title: Native Go ESP-IDF BLE Provisioning Client Analysis Design and Implementation Guide
Ticket: ALMANACH-NATIVE-PROVISION
Status: active
Topics:
    - almanach
    - go
    - ble
    - esp-idf
    - protocomm
    - wifi-provisioning
    - cli
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/components/protocomm/proto/sec1.proto
      Note: Security 1 protobuf schema.
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/components/protocomm/proto/session.proto
      Note: Session envelope schema.
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/components/wifi_provisioning/proto/wifi_config.proto
      Note: WiFi credential/status protobuf schema.
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py
      Note: |-
        Reference provisioning workflow to port.
        Reference Python provisioning workflow to port
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/tools/esp_prov/prov/wifi_prov.py
      Note: |-
        Reference WiFi config protobuf request/response helpers.
        Reference Python WiFi config protobuf helpers
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/tools/esp_prov/security/security1.py
      Note: |-
        Reference Security 1 implementation.
        Reference Python Security 1 state machine and crypto behavior
    - Path: firmware/atoms3r/main/provisioning_mgr.c
      Note: Firmware provisioning service identity
    - Path: internal/app/cmd_ble_provision.go
      Note: |-
        Current Go/Glazed command delegates provisioning to ESP-IDF esp_prov.py.
        Existing Go/Glazed wrapper and CLI surface that the native implementation should preserve
    - Path: web/src/provisioning/espidf-client.js
      Note: |-
        Browser transport evidence for service UUID and endpoint discovery.
        Browser-validated GATT service UUID
ExternalSources: []
Summary: Design guide for replacing the Python esp_prov.py dependency with a native Go ESP-IDF BLE provisioning client while preserving CLI behavior and using the Go version as the future browser protocol reference.
LastUpdated: 2026-05-10T20:20:00-04:00
WhatFor: Use this before implementing the native Go BLE provisioning client.
WhenToUse: Read when porting ESP-IDF Security 1, protobuf provisioning messages, BLE transport, or CLI behavior from Python to Go.
---


# Native Go ESP-IDF BLE Provisioning Client Analysis Design and Implementation Guide

## Executive summary

The current Almanach `ble-provision` command is a Go/Glazed wrapper around Espressif's Python provisioning client, `esp_prov.py`. This was the correct first implementation because it let the project validate firmware BLE provisioning quickly with Espressif's maintained transport, protobuf, and Security 1 code. The browser work has now reached `proto-ver` validation in Chrome, which proves the firmware service UUID, GATT endpoint discovery, and protocol version endpoint. The next hard part is the same in every non-Python client: implement ESP-IDF Security 1, binary protobuf messages, encrypted WiFi configuration, and status polling.

This ticket proposes a native Go provisioning client before the JavaScript browser port. Go is easier to run in automated tests, easier to instrument from the CLI, and can become the reference implementation for the later browser client. The goal is not to replace the existing Python path immediately. The goal is to implement a native path behind a flag, validate it against the same AtomS3R firmware, and keep the Python wrapper as a fallback until the Go implementation reaches parity.

The native Go client must reproduce the successful Python flow:

```text
BLE connect → endpoint discovery → proto-ver → Security 1 session → encrypted set_config → encrypted apply_config → encrypted get_status polling
```

The implementation should be split into packages with clear responsibilities: BLE transport, protocomm endpoint transport, Security 1 session, protobuf WiFi config messages, and CLI integration. Each layer should be testable without hardware where possible.

## Problem statement and scope

The current Go command delegates real provisioning to Python. This creates a dependency on the ESP-IDF Python environment, BlueZ/Bleak Python packages, generated Python protobuf modules, and the exact behavior of `esp_prov.py`. That is acceptable for firmware validation, but it is not ideal as the long-term core provisioning implementation for Almanach.

A native Go client gives the project three advantages:

1. It removes the Python subprocess from the standard Almanach provisioning path after validation.
2. It gives the team a protocol implementation that can be unit-tested and fuzzed inside Go.
3. It creates a clearer bridge to JavaScript because the Go implementation can define the exact byte-level protocol behavior before browser crypto/protobuf constraints are introduced.

The scope of this ticket is the native Go client for Linux BLE provisioning. Browser JavaScript credential transfer is explicitly out of scope for this ticket, but this design should make that later port easier. Firmware changes are also out of scope except for bugs discovered during native-client validation.

## Current-state architecture

### Current Go command boundary

The current command lives in `internal/app/cmd_ble_provision.go`. Its help text states that it is a wrapper around ESP-IDF's `esp_prov.py`, reusing Espressif's BLE, protobuf, and Security 1 implementation. The settings structure includes action, service name, SSID, passphrase, PoP, security version, proto version, IDF path, Python path, and timeout. This tells us the command already has the right CLI surface for a native implementation.

The key observed facts are:

- Lines 28-42 define `BLEProvisionSettings`, including `Action`, `ServiceName`, `SSID`, `Passphrase`, `Pop`, `SecVer`, and `ProtoVer`.
- Lines 55-73 describe the command as a Go/Glazed wrapper around `esp_prov.py`.
- Lines 113-129 read passphrase from stdin when omitted, preserving a safer secret path.
- Lines 143-166 execute the Python subprocess with a timeout and `IDF_PATH` environment.
- Lines 186-213 implement the special `version` action by importing `esp_prov` from Python and calling `version_match()`.
- Lines 216-236 build the normal `esp_prov.py` command for `provision`, `reset`, and `reprov`.

The native implementation should keep this user-facing command surface. The migration should add an implementation selector rather than replacing the command shape.

### Firmware provisioning service

The AtomS3R firmware uses ESP-IDF's standard provisioning manager. In `firmware/atoms3r/main/provisioning_mgr.c`, the firmware constructs service identity from the WiFi MAC address, resulting in names like `ALM_0F2320` and PoP values like `alm-0f2320`. It starts provisioning through `wifi_prov_mgr_start_provisioning()` with Security 1.

The firmware exposes the standard ESP-IDF BLE provisioning endpoints:

```text
prov-ctrl
prov-scan
prov-session
prov-config
proto-ver
```

The browser client has already validated the custom service UUID and endpoint mapping. Its constants show the service UUID and characteristic UUIDs that a native Go client should also use as fallback values:

```text
Service UUID: 021a9004-0382-4aea-bff4-6b3f1c5adfb4
prov-ctrl:   021aff4f-0382-4aea-bff4-6b3f1c5adfb4
prov-scan:   021aff50-0382-4aea-bff4-6b3f1c5adfb4
prov-session:021aff51-0382-4aea-bff4-6b3f1c5adfb4
prov-config: 021aff52-0382-4aea-bff4-6b3f1c5adfb4
proto-ver:   021aff53-0382-4aea-bff4-6b3f1c5adfb4
```

The firmware UUID bug found during browser testing is also relevant: ESP-IDF stores the pointer passed to `wifi_prov_scheme_ble_set_service_uuid()`, so the firmware now keeps the UUID in static storage. The native Go client should assume the fixed firmware and should validate service UUID discovery early.

### Python reference implementation

The Python reference lives under the ESP-IDF installation:

```text
/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py
/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/security/security1.py
/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/prov/wifi_prov.py
/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/transport/transport_ble.py
/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/transport/ble_cli.py
```

The Python transport performs BLE discovery, connects by service name, reads characteristic descriptors, and maps endpoint names to characteristic UUIDs. The browser implementation now follows the same strategy. The native Go transport should also prefer endpoint name descriptors and fall back to expected UUIDs.

The Python provisioning workflow is clear:

```python
async def establish_session(tp, sec):
    response = None
    while True:
        request = sec.security_session(response)
        if request is None:
            break
        response = await tp.send_data('prov-session', request)
    return True
```

Then WiFi configuration uses encrypted protobuf messages:

```python
message = prov.config_set_config_request(sec, ssid, passphrase)
response = await tp.send_data('prov-config', message)
prov.config_set_config_response(sec, response)

message = prov.config_apply_config_request(sec)
response = await tp.send_data('prov-config', message)
prov.config_apply_config_response(sec, response)

message = prov.config_get_status_request(sec)
response = await tp.send_data('prov-config', message)
prov.config_get_status_response(sec, response)
```

The Go implementation should port these functions almost directly, but with Go types, byte slices, and generated protobuf structs.

## ESP-IDF protocol pieces to port

### BLE transport

The BLE transport needs to perform five operations:

1. Discover or connect to a BLE device by name, such as `ALM_0F2320`.
2. Bind to the ESP-IDF provisioning service UUID.
3. Discover characteristics under that service.
4. Read user-description descriptor `0x2901` from characteristics to map endpoint names.
5. Implement endpoint write/read with response.

The transport interface should be small:

```go
type Transport interface {
    Connect(ctx context.Context, serviceName string) error
    Disconnect(ctx context.Context) error
    Endpoints() map[string]EndpointInfo
    Send(ctx context.Context, endpoint string, request []byte) ([]byte, error)
}

type EndpointInfo struct {
    Name string
    UUID string
}
```

The expected behavior of `Send()` is the same as the Python transport: write request bytes to the endpoint characteristic, then read response bytes from the same characteristic.

The main design question is the Go BLE library. Candidate families include BlueZ/D-Bus wrappers and cross-platform BLE packages. The first implementation should prioritize Linux reliability over cross-platform support because the current CLI validation environment is Linux. The library must support:

- scanning by advertised name,
- connecting to a peripheral,
- service discovery,
- characteristic discovery,
- descriptor reads,
- write-with-response,
- characteristic reads.

The implementation should hide library-specific types behind the `Transport` interface. This keeps the protocol code independent of BlueZ.

### Protocol version

The `proto-ver` endpoint is plaintext. It accepts a text request such as `v1.1` and returns either a simple version string or a JSON object. The browser observed this response:

```json
{ "prov": { "ver": "v1.1", "sec_ver": 1, "sec_patch_ver": 0, "cap": ["wifi_scan"] } }
```

The Go function can mirror Python `version_match()`:

```go
func VerifyProtoVersion(ctx context.Context, t Transport, want string) (*ProtoInfo, error) {
    resp, err := t.Send(ctx, "proto-ver", []byte(want))
    if err != nil { return nil, err }

    text := string(resp)
    if strings.EqualFold(text, want) {
        return &ProtoInfo{Version: want}, nil
    }

    var info ProtoInfoEnvelope
    if err := json.Unmarshal(resp, &info); err != nil {
        return nil, fmt.Errorf("unexpected proto-ver response %q: %w", text, err)
    }
    if !strings.EqualFold(info.Prov.Version, want) {
        return nil, fmt.Errorf("protocol version mismatch: got %q want %q", info.Prov.Version, want)
    }
    return &info.Prov, nil
}
```

### Security 1

Security 1 is the most important protocol component. ESP-IDF Security 1 uses X25519 key exchange, proof-of-possession mixing, and AES-CTR encryption. The Python implementation in `security1.py` is the reference.

The session state machine is:

```text
REQUEST1 → RESPONSE1_REQUEST2 → RESPONSE2 → FINISHED
```

The first request sends the client X25519 public key:

```text
SessionData{
  sec_ver: SecScheme1,
  sec1: {
    sc0: { client_pubkey: <32 bytes> }
  }
}
```

The device response contains:

```text
device_pubkey: <32 bytes>
device_random: <16 bytes>
```

The client computes:

```text
shared = X25519(client_private, device_public)
if PoP is present:
    shared = shared XOR SHA256(PoP)
cipher = AES-CTR(shared, device_random)
```

Then the client encrypts the device public key as proof:

```text
client_verify_data = AES_CTR_Encrypt(device_pubkey)
```

The device response contains encrypted device proof data. The Python client decrypts it and checks that it equals the client public key:

```text
AES_CTR_Decrypt(device_verify_data) == client_pubkey
```

A Go API should make the state explicit:

```go
type Security1 struct {
    pop []byte
    privateKey [32]byte
    publicKey [32]byte
    devicePublicKey []byte
    stream *CTRStream
}

func NewSecurity1(pop string, rand io.Reader) (*Security1, error)
func (s *Security1) Next(response []byte) (request []byte, done bool, err error)
func (s *Security1) Encrypt(data []byte) ([]byte, error)
func (s *Security1) Decrypt(data []byte) ([]byte, error)
```

The AES-CTR detail needs careful handling. Python uses one cipher context and calls `cipher.update()` repeatedly for proof, encrypted requests, and decrypted responses. A Go implementation must preserve the same stream progression. The simplest reliable approach is to create an `cipher.Stream` with `cipher.NewCTR(block, deviceRandom)` and use `XORKeyStream()` sequentially on every plaintext/ciphertext in the same order as Python. Do not create a new CTR stream for each message unless the ESP-IDF protocol explicitly resets the counter. The Python code does not reset it.

### Protobuf schemas

The necessary schemas are already present in ESP-IDF:

```text
components/protocomm/proto/session.proto
components/protocomm/proto/sec1.proto
components/protocomm/proto/constants.proto
components/wifi_provisioning/proto/wifi_config.proto
components/wifi_provisioning/proto/wifi_constants.proto
```

The design recommends generating Go protobuf bindings from the upstream `.proto` files into an internal package, for example:

```text
internal/provisioning/proto/sessionpb
internal/provisioning/proto/wificonfigpb
```

The generation path must handle import names cleanly because both protocomm and WiFi provisioning have a `constants.proto` concept. If generation friction is high, a short-term internal hand-written encoder is possible, but it is not the preferred path. Protobuf generation is safer and easier to review.

The schema facts that matter for credential provisioning are:

- `session.proto` defines `SessionData` and `SecScheme1`.
- `sec1.proto` defines `SessionCmd0`, `SessionResp0`, `SessionCmd1`, and `SessionResp1`.
- `wifi_config.proto` defines `CmdSetConfig`, `CmdApplyConfig`, `CmdGetStatus`, and the corresponding responses.
- `wifi_constants.proto` defines station states: `Connected`, `Connecting`, `Disconnected`, and `ConnectionFailed`.
- `wifi_constants.proto` defines failure reasons: `AuthError` and `NetworkNotFound`.

### WiFi config flow

Once Security 1 is established, every WiFi config request is serialized protobuf, encrypted with the Security 1 stream, sent to `prov-config`, decrypted on response, and parsed.

The minimum API should be:

```go
type WiFiConfigClient struct {
    transport Transport
    security  *Security1
}

func (c *WiFiConfigClient) SetConfig(ctx context.Context, ssid, passphrase string) error
func (c *WiFiConfigClient) ApplyConfig(ctx context.Context) error
func (c *WiFiConfigClient) GetStatus(ctx context.Context) (WiFiStatus, error)
func (c *WiFiConfigClient) WaitConnected(ctx context.Context, pollInterval time.Duration) (WiFiStatus, error)
```

The pseudocode is:

```go
func (c *WiFiConfigClient) SetConfig(ctx context.Context, ssid, pass string) error {
    payload := &wificonfigpb.WiFiConfigPayload{
        Msg: wificonfigpb.WiFiConfigMsgType_TypeCmdSetConfig,
        Payload: &wificonfigpb.WiFiConfigPayload_CmdSetConfig{
            CmdSetConfig: &wificonfigpb.CmdSetConfig{
                Ssid: []byte(ssid),
                Passphrase: []byte(pass),
            },
        },
    }
    plain, _ := proto.Marshal(payload)
    encrypted, _ := c.security.Encrypt(plain)
    respEncrypted, _ := c.transport.Send(ctx, "prov-config", encrypted)
    respPlain, _ := c.security.Decrypt(respEncrypted)
    resp := parse RespSetConfig
    return statusToError(resp.Status)
}
```

`ApplyConfig` and `GetStatus` follow the same pattern.

## Proposed Go package layout

A clean package layout keeps transport, crypto, protobuf, and CLI behavior separate:

```text
internal/provisioning/native/
  client.go              high-level provisioning workflow
  transport.go           transport interface and endpoint metadata
  protover.go            proto-ver parsing and validation
  security1.go           X25519 + PoP + AES-CTR Security 1
  wifi_config.go         encrypted WiFi config messages and polling
  errors.go              typed errors for auth, AP not found, BLE, protocol
  bluez_transport.go     Linux BLE implementation
  testdata/              captured protocol fixtures when available

internal/provisioning/native/proto/
  ... generated Go protobuf files ...
```

The high-level client should be small:

```go
type Client struct {
    Transport Transport
    Security  *Security1
}

func (c *Client) Version(ctx context.Context, serviceName string) (*ProtoInfo, error)
func (c *Client) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
func (c *Client) Reset(ctx context.Context, serviceName string) error
```

The CLI should not know about X25519, AES-CTR, descriptors, or protobuf messages. It should call this package and report rows.

## CLI migration strategy

Do not remove the Python wrapper immediately. Add a mode flag:

```text
--implementation python|native
```

Default can remain `python` until the native implementation passes hardware validation. Once native Go reaches parity, the default can change to `native` with `python` kept as fallback.

Recommended command behavior:

```bash
# current behavior, explicit
almanach-render-service ble-provision --implementation python --action provision ...

# new native behavior
almanach-render-service ble-provision --implementation native --action version --service-name ALM_0F2320
almanach-render-service ble-provision --implementation native --action provision --service-name ALM_0F2320 --pop alm-0f2320 --ssid Verizon_9DNVB9
```

The command should preserve the existing stdin passphrase path:

```text
if action=provision and --passphrase is omitted, read one line from stdin
```

The command should also preserve structured output fields:

```text
action
service_name
ssid
pop
implementation
exit_code or ok
duration_ms
```

## Implementation phases

### Phase 1: package skeleton and interfaces

Create `internal/provisioning/native` with interfaces, request/result types, and a fake transport for tests. Implement `proto-ver` parsing against the fake transport. This phase needs no BLE hardware.

Deliverables:

- `transport.go`
- `client.go`
- `protover.go`
- unit tests for simple `v1.1` and JSON proto-ver responses

### Phase 2: protobuf generation

Add a repeatable generation step for ESP-IDF protobuf schemas. Prefer a checked-in `go:generate` script that copies or references the schemas and runs `protoc` with `protoc-gen-go`.

Deliverables:

- generated Go protobuf packages
- `go generate` instructions
- tests that construct and round-trip `SessionData` and `WiFiConfigPayload`

### Phase 3: Security 1 implementation

Port `security/security1.py` to Go. Test it with deterministic keys if possible. If deterministic device responses are hard to construct initially, test helper functions independently:

- PoP SHA256 and XOR,
- AES-CTR stream progression,
- protobuf command construction,
- state-machine transitions.

Deliverables:

- `security1.go`
- tests for state machine and crypto helpers
- documentation explaining CTR stream continuity

### Phase 4: BLE transport

Choose and integrate a Go BLE library for Linux. Implement scan/connect/service/characteristic/descriptor/write/read. This is the first hardware-facing native phase.

Deliverables:

- `bluez_transport.go`
- native `version` action working against `ALM_0F2320`
- comparison output against Python `--action version`

### Phase 5: WiFi config messages

Implement encrypted `SetConfig`, `ApplyConfig`, and `GetStatus`. Use the fake transport for unit tests and hardware for end-to-end validation.

Deliverables:

- `wifi_config.go`
- typed statuses and errors
- native provisioning run against AtomS3R

### Phase 6: CLI integration

Add `--implementation native|python` to the existing command and route to native code when selected. Preserve Python fallback and current behavior.

Deliverables:

- updated `cmd_ble_provision.go`
- tests for argument validation and implementation selection
- documentation updates

### Phase 7: hardware validation and parity checklist

Validate the native client against the same scenarios already proven with Python:

1. `version` action succeeds.
2. provisioning succeeds with real WiFi credentials.
3. wrong SSID reports AP-not-found.
4. wrong password reports auth failure.
5. reboot autoconnect works.
6. `/api/status` is reachable after provisioning.
7. reset/reprovision path works.

## Testing strategy

### Unit tests

Unit tests should cover every protocol component that does not require BLE hardware:

- proto-ver JSON parsing,
- endpoint fallback mapping,
- protobuf marshal/unmarshal,
- Security 1 state progression,
- PoP digest XOR,
- AES-CTR stream continuity,
- WiFi status response parsing.

### Fake transport tests

A fake transport can model endpoint request/response behavior:

```go
type FakeTransport struct {
    Handlers map[string]func([]byte) ([]byte, error)
}
```

Use it to test that the high-level client sends messages to the correct endpoints in the correct order.

### Hardware tests

Hardware tests should run manually first, then move toward scripted validation:

```bash
almanach-render-service ble-provision --implementation native --action version --service-name ALM_0F2320 --pop alm-0f2320
almanach-render-service ble-provision --implementation native --action provision --service-name ALM_0F2320 --pop alm-0f2320 --ssid <ssid>
```

Use `idf.py monitor` in tmux to confirm firmware events:

```text
BLE provisioning client connected
Provisioning security session established
Received WiFi credentials for SSID '...'
Provisioned WiFi credentials connected successfully
WiFi connected — starting web server
```

### Golden comparison against Python

The native client should be compared against Python during bring-up. For each action, run Python and native implementations against a reset device and compare observable outcomes. Do not require byte-for-byte equality for encrypted payloads because X25519 keys and random values will differ. Compare state transitions and firmware logs.

## Risks and mitigations

### BLE library risk

Go BLE libraries vary in maintenance and BlueZ behavior. Mitigation: isolate BLE behind the `Transport` interface and choose Linux support first. Keep Python fallback until native hardware validation passes.

### AES-CTR stream risk

Security 1 uses a continuous AES-CTR stream in Python. Resetting the stream per message will likely fail after the first proof or encrypted command. Mitigation: implement one stream object and test call ordering carefully.

### Protobuf import risk

ESP-IDF proto files were written for Espressif's build layout. Go generation may require `go_package` mapping. Mitigation: create a small generation script and document exact `protoc` flags. If needed, vendor copies of the schemas into the ticket/repo with package options added.

### Secret handling risk

WiFi passphrases should not appear in process lists or logs. Mitigation: preserve stdin passphrase reading and redact command display strings.

### Behavior drift from Espressif client

A native implementation can diverge from ESP-IDF behavior. Mitigation: keep the Python path, add parity tests, and cite Python reference functions in code comments where helpful.

## Alternatives considered

### Keep Python permanently

This is the lowest implementation effort. It is also less portable as the core Almanach client because it depends on ESP-IDF Python environment setup. It remains valuable as fallback and reference.

### Implement JavaScript first

The browser is the final user-facing target, but JavaScript adds browser crypto and Web Bluetooth constraints. Go is easier to test automatically, easier to run under CLI timeouts, and easier to compare with Python on the same host.

### Implement only browser using an existing JS library

This may still be a good future path, but the project first needs a verified non-Python understanding of the protocol. A native Go implementation gives that reference.

## API references

Key ESP-IDF and Python references:

| Reference | Purpose |
|---|---|
| `tools/esp_prov/esp_prov.py:get_transport()` | Creates BLE transport with service UUID and endpoint lookup. |
| `tools/esp_prov/esp_prov.py:version_match()` | Plaintext `proto-ver` check. |
| `tools/esp_prov/esp_prov.py:establish_session()` | Security session request/response loop. |
| `tools/esp_prov/security/security1.py:Security1` | X25519, PoP, AES-CTR state machine. |
| `tools/esp_prov/prov/wifi_prov.py` | WiFi config protobuf request/response helpers. |
| `components/protocomm/proto/session.proto` | Security session envelope. |
| `components/protocomm/proto/sec1.proto` | Security 1 messages. |
| `components/wifi_provisioning/proto/wifi_config.proto` | WiFi credential and status messages. |
| `components/wifi_provisioning/proto/wifi_constants.proto` | WiFi states and failure reasons. |

## File references in the Almanach repo

| File | Why it matters |
|---|---|
| `internal/app/cmd_ble_provision.go` | Existing CLI surface and Python subprocess wrapper. |
| `firmware/atoms3r/main/provisioning_mgr.c` | Firmware provisioning identity, service UUID, Security 1 setup, event logs. |
| `web/src/provisioning/espidf-client.js` | Browser-proven service UUID, endpoint mapping, and proto-ver logic. |
| `ttmp/2026/05/10/ALMANACH-BLE-PROVISION--.../reference/01-investigation-diary.md` | Hardware validation evidence for firmware, CLI, and browser milestones. |

## Recommended first implementation commit

The first code commit should not touch BLE hardware. It should add the native package skeleton and make `proto-ver` parsing testable. A good first commit is:

```text
Add native provisioning client interfaces
```

It should include:

- `internal/provisioning/native/transport.go`,
- `internal/provisioning/native/protover.go`,
- `internal/provisioning/native/client.go`,
- tests with fake transport,
- no changes to the existing command behavior.

This keeps review small and confirms the package boundary before adding crypto and BLE dependencies.
