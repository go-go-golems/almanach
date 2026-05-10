---
Title: Investigation Diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py
      Note: Reference provisioning control flow inspected.
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/tools/esp_prov/prov/wifi_prov.py
      Note: WiFi config request/status flow inspected.
    - Path: ../../../../../../../../../../esp/esp-idf-5.4.2/tools/esp_prov/security/security1.py
      Note: Security 1 algorithm inspected.
    - Path: internal/app/cmd_ble_provision.go
      Note: |-
        Existing Go command inspected to identify Python delegation boundary.
        Implementation selector and native passphrase handling for Phase 7
    - Path: internal/app/cmd_ble_provision_native.go
      Note: Native CLI provisioning path for Phase 7
    - Path: internal/provisioning/native/security1.go
      Note: Native Security 1 implementation for Phase 5
    - Path: internal/provisioning/native/wifi_config.go
      Note: Encrypted WiFi config flow for Phase 6
ExternalSources: []
Summary: Chronological investigation diary for native Go ESP-IDF BLE provisioning client design.
LastUpdated: 2026-05-10T20:20:00-04:00
WhatFor: Use this to understand what was inspected and why the native Go design proposes the phased implementation plan.
WhenToUse: Read before implementing the ALMANACH-NATIVE-PROVISION ticket.
---


# Diary

## Goal

This diary records the research and documentation work for creating a native Go implementation of ESP-IDF BLE WiFi provisioning. The goal is to replace the current Python subprocess dependency only after a native implementation can prove parity with the existing `esp_prov.py` path.

## Step 1: Ticket setup and native Go provisioning design

This step created the `ALMANACH-NATIVE-PROVISION` ticket and mapped the current provisioning stack from Go CLI wrapper to ESP-IDF Python client to firmware service. The key finding is that the current Go command owns CLI workflow, while Espressif's Python code owns all protocol-heavy work: BLE endpoint transport, Security 1, protobuf message construction, encryption, WiFi config, and status polling.

The design therefore recommends a staged port. The first native Go implementation should not attempt full credential provisioning immediately. It should add package boundaries and a testable `proto-ver` path, then add protobuf generation, Security 1, BLE transport, encrypted WiFi config, and CLI integration in separate phases.

### Prompt Context

**User prompt (verbatim):** "ok, let's create a new ticket where we are porting all the python stuff to go first (we will do javascript later on, once we can make sure that the go version works, since we can more easily exercise it automatically on the CLI rather than testing in the browser).

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket for a native Go port of Espressif's Python BLE provisioning stack, inspect the current Go/Python/firmware/browser implementation, write a detailed intern-facing design and implementation guide, keep a diary, and upload the documentation bundle to reMarkable.

**Inferred user intent:** Build confidence in the ESP-IDF provisioning protocol outside Python before attempting the more constrained browser JavaScript credential-transfer port.

**Commit (code):** N/A — documentation and ticket setup only.

### What I did

- Created ticket `ALMANACH-NATIVE-PROVISION` with title `Native Go ESP-IDF BLE Provisioning Client`.
- Created the primary design document:
  - `design-doc/01-native-go-esp-idf-ble-provisioning-client-analysis-design-and-implementation-guide.md`
- Created this investigation diary:
  - `reference/01-investigation-diary.md`
- Updated `tasks.md` with research-complete items and future implementation phases.
- Copied evidence excerpts into ticket sources:
  - `sources/01-code-evidence-excerpts.txt`
  - `sources/02-protobuf-schema-evidence.txt`
- Inspected Almanach repo files:
  - `internal/app/cmd_ble_provision.go`
  - `firmware/atoms3r/main/provisioning_mgr.c`
  - `web/src/provisioning/espidf-client.js`
- Inspected ESP-IDF Python/protobuf reference files:
  - `/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py`
  - `/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/security/security1.py`
  - `/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/prov/wifi_prov.py`
  - `/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/transport/transport_ble.py`
  - `/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/transport/ble_cli.py`
  - `/home/manuel/esp/esp-idf-5.4.2/components/protocomm/proto/session.proto`
  - `/home/manuel/esp/esp-idf-5.4.2/components/protocomm/proto/sec1.proto`
  - `/home/manuel/esp/esp-idf-5.4.2/components/protocomm/proto/constants.proto`
  - `/home/manuel/esp/esp-idf-5.4.2/components/wifi_provisioning/proto/wifi_config.proto`
  - `/home/manuel/esp/esp-idf-5.4.2/components/wifi_provisioning/proto/wifi_constants.proto`
  - `/home/manuel/esp/esp-idf-5.4.2/components/wifi_provisioning/proto/wifi_ctrl.proto`

### Why

- The browser has now validated Web Bluetooth through `proto-ver`, but credential transfer requires the same Security 1/protobuf protocol that Python already implements.
- Go is easier to test automatically from the CLI than JavaScript in an interactive browser, so a native Go port is the safer next implementation layer.
- The existing Go wrapper already has a good CLI surface, so the native work can focus on replacing the implementation behind that surface rather than redesigning user commands.

### What worked

- `docmgr ticket create-ticket` created the ticket workspace successfully.
- `docmgr doc add` created both the design document and diary.
- Source inspection found exact Python functions to port:
  - `establish_session()`
  - `send_wifi_config()`
  - `apply_wifi_config()`
  - `get_wifi_config()`
  - `wait_wifi_connected()`
- Source inspection found the Security 1 implementation details in `security1.py`: X25519, SHA256(PoP) XOR, AES-CTR, and session proofs.
- Protobuf schema inspection identified the minimum required schemas for a native client.

### What didn't work

- No code implementation was attempted in this step.
- No reMarkable upload was performed yet at the moment this diary entry was written; upload is the next delivery action after `docmgr doctor` validation.

### What I learned

- The Go command is currently a workflow wrapper, not a protocol implementation. It preserves good operator behavior such as passphrase-from-stdin and command timeouts.
- ESP-IDF's Python client is cleanly layered enough to port: transport, security, and provisioning message helpers are separable.
- Security 1's AES-CTR stream continuity is the most important implementation detail to preserve in Go. Python uses one cipher context and repeatedly calls `cipher.update()`.

### What was tricky to build

- The design had to separate what should be ported from what should remain as fallback. Removing the Python path immediately would create unnecessary risk. The guide recommends a `--implementation native|python` selector so the native client can be validated without losing the known-good path.
- The protobuf generation plan needs care because ESP-IDF schemas were not written primarily for Go package generation. The implementation may need a small schema-copy or `Mfile=package` mapping strategy.

### What warrants a second pair of eyes

- Review the proposed Go BLE library selection criteria before adding a dependency.
- Review the AES-CTR stream handling design before coding Security 1.
- Review whether native Go should initially support only provisioning/version or also reset/reprov commands.

### What should be done in the future

- Run `docmgr doctor --ticket ALMANACH-NATIVE-PROVISION --stale-after 30`.
- Upload the design package to reMarkable.
- Start Phase 1 with native package interfaces and fake transport tests.

### Code review instructions

- Start with the design doc executive summary and current-state architecture.
- Then inspect `internal/app/cmd_ble_provision.go` to understand the CLI surface to preserve.
- Then inspect ESP-IDF Python files in this order:
  - `esp_prov.py`
  - `security/security1.py`
  - `prov/wifi_prov.py`
  - `transport/transport_ble.py`
- Validate future implementation with Python/native parity tests against the AtomS3R.

### Technical details

Commands run during setup and inspection included:

```bash
cd almanach
docmgr status --summary-only
docmgr ticket create-ticket --ticket ALMANACH-NATIVE-PROVISION --title "Native Go ESP-IDF BLE Provisioning Client" --topics almanach,go,ble,esp-idf,protocomm,wifi-provisioning,cli
docmgr doc add --ticket ALMANACH-NATIVE-PROVISION --doc-type design-doc --title "Native Go ESP-IDF BLE Provisioning Client Analysis Design and Implementation Guide"
docmgr doc add --ticket ALMANACH-NATIVE-PROVISION --doc-type reference --title "Investigation Diary"
```

Evidence was gathered with `nl -ba` over the Go wrapper, firmware provisioning manager, browser Web Bluetooth client, ESP-IDF Python provisioning client, Security 1 implementation, WiFi provisioning helpers, and ESP-IDF protobuf schemas.

## Step 2: Phase 1 native package skeleton and proto-ver tests

This step started implementation by adding the smallest useful native Go protocol slice. The new `internal/provisioning/native` package defines the transport boundary, endpoint metadata, a high-level client skeleton, and a native `proto-ver` verifier. The implementation uses a fake transport in tests, so it can validate protocol parsing and call sequencing without BLE hardware.

This is intentionally not a BLE implementation yet. The goal is to establish package shape and a testable protocol seam before adding BlueZ, protobuf generation, or Security 1. The code mirrors the `proto-ver` behavior already validated in Python and Chrome: accept either a bare `v1.1` string or a JSON capability object with `prov.ver` set to `v1.1`.

### Prompt Context

**User prompt (verbatim):** "Do ahead, add tasks to the ticket, and then go through them one by one, committing at appropriate intervals, keeping a diary."

**Assistant interpretation:** Add detailed implementation tasks to the native Go provisioning ticket, begin executing the first phase, validate it, commit at a clean boundary, and record the work in the diary.

**Inferred user intent:** Move from design into implementation while keeping changes reviewable and documented.

**Commit (code):** e3645d0147b5c8e365ab6ac364a55ea281ead466 — "Add native provisioning proto-ver skeleton"

### What I did

- Expanded Phase 1 tasks with explicit subtasks for package creation, fake transport, proto-ver verification, tests, and diary/changelog.
- Added `internal/provisioning/native/transport.go`:
  - endpoint constants for `proto-ver`, `prov-session`, `prov-config`, `prov-scan`, and `prov-ctrl`
  - `EndpointInfo`
  - `Transport` interface
- Added `internal/provisioning/native/client.go`:
  - `Client` wrapper
  - `NewClient()`
  - `VerifyVersion()`
- Added `internal/provisioning/native/protover.go`:
  - `DefaultProtoVersion`
  - `ProtoInfo`
  - `VerifyProtoVersion()`
- Added fake transport test helper in `fake_transport_test.go`.
- Added `protover_test.go` covering:
  - bare string response
  - JSON response with `sec_ver`, `sec_patch_ver`, and capabilities
  - default version behavior
  - version mismatch
  - transport error wrapping
  - high-level client forwarding
- Ran `gofmt` and `go test ./...`.

### Why

- `proto-ver` is the lowest-risk native protocol slice because it is plaintext and has already been validated through Python and Chrome.
- A fake transport lets the project test native protocol behavior before choosing a BLE library.
- The `Transport` interface protects future protocol code from BLE implementation details.

### What worked

- `go test ./...` passed.
- New package tests passed:
  - `ok github.com/go-go-golems/almanach/internal/provisioning/native`
- The package now has a clear seam for Phase 2 protobuf and Phase 3/4 BLE transport work.

### What didn't work

- No failure occurred in this phase.
- No hardware test was attempted because this phase intentionally avoids BLE.

### What I learned

- The current command and browser work gave a stable enough target to implement `proto-ver` parsing without guessing.
- Keeping fake transport in `_test.go` is enough for unit tests now; a reusable fake can be promoted later if integration tests need it from another package.

### What was tricky to build

- The native package needs to be useful before BLE exists. That means the first abstractions must be byte-oriented endpoint abstractions, not BLE-specific abstractions. The `Transport.Send(ctx, endpoint, []byte)` shape keeps later Security 1 and WiFi config code independent of BlueZ.
- `proto-ver` has two valid response shapes. Treating the JSON response as the only shape would diverge from Espressif's Python behavior, which also accepts a bare version string.

### What warrants a second pair of eyes

- Review whether `Transport.Connect()` belongs in the same interface as `Send()` or whether future tests would benefit from splitting connected endpoint transport from connection management.
- Review whether `ProtoInfo` should preserve the full JSON envelope instead of flattening only `prov` fields.

### What should be done in the future

- Phase 2 should add protobuf bindings or a repeatable generation path for ESP-IDF schemas.
- Phase 3 should implement Security 1 independently of BLE using fake request/response fixtures before real transport work.
- Phase 4 should choose the Linux BLE library and implement the real transport behind the current interface.

### Code review instructions

- Start with `internal/provisioning/native/transport.go` to understand the intended boundary.
- Then inspect `internal/provisioning/native/protover.go` and `protover_test.go`.
- Validate with:
  - `go test ./...`

### Technical details

Commands run:

```bash
cd almanach
gofmt -w internal/provisioning/native
go test ./...
git commit -m "Add native provisioning proto-ver skeleton"
```

## Step 3: Phase 2 Buf-generated ESP-IDF protobuf bindings

This step added the protobuf schema layer needed before implementing Security 1 and WiFi config messages. The native Go client now vendors the relevant ESP-IDF protocomm and WiFi provisioning `.proto` files, adds Go package options, and generates Go bindings using Buf with the local `protoc-gen-go` plugin.

The first attempt used `protoc` directly. The user corrected the workflow to use Buf, which is the preferred project convention for protobuf generation. I replaced the direct invocation with `buf.yaml`, `buf.gen.yaml`, and a `go:generate` hook so future regeneration is repeatable with `go generate ./internal/provisioning/native/proto`.

### Prompt Context

**User prompt (verbatim):** "use buf btw"

**Assistant interpretation:** Switch protobuf generation for the native provisioning package from ad hoc `protoc` commands to a Buf-based workflow.

**Inferred user intent:** Keep protobuf generation aligned with the repository's preferred schema/codegen practice and make future regeneration reproducible.

**Commit (code):** eb1161cdcd7fe34f9e8d3c21dec1097a76304430 — "Add ESP-IDF provisioning protobuf bindings"

### What I did

- Added Buf configuration under `internal/provisioning/native/proto/`:
  - `buf.yaml`
  - `buf.gen.yaml`
  - `generate.go` with `//go:generate buf generate`
- Vendored ESP-IDF protocomm schemas into `internal/provisioning/native/proto/espidf/`:
  - `constants.proto`
  - `sec0.proto`
  - `sec1.proto`
  - `sec2.proto`
  - `session.proto`
- Vendored ESP-IDF WiFi provisioning schemas into the same module:
  - `wifi_constants.proto`
  - `wifi_config.proto`
  - `wifi_scan.proto`
  - `wifi_ctrl.proto`
- Added `option go_package = "github.com/go-go-golems/almanach/internal/provisioning/native/proto/espidf;espidf";` to the vendored schemas.
- Generated Go bindings with Buf and local `protoc-gen-go`.
- Added `internal/provisioning/native/proto_roundtrip_test.go` to verify generated types compile and round-trip:
  - `SessionData` containing Security 1 `SessionCmd0`
  - `WiFiConfigPayload` containing `CmdSetConfig`
- Ran:
  - `go generate ./internal/provisioning/native/proto`
  - `go test ./...`

### Why

- Security 1 and WiFi config messages are protobuf messages. Implementing those phases without generated bindings would either duplicate schema logic manually or increase the chance of field-number mistakes.
- Buf makes the code generation path explicit and repeatable.
- Round-trip tests prove that the generated bindings can represent the exact message shapes needed in the next implementation phases.

### What worked

- `buf generate` generated nine Go protobuf files.
- `go generate ./internal/provisioning/native/proto` completed successfully.
- `go test ./...` passed, including the new protobuf round-trip tests.

### What didn't work

- The initial direct `protoc` generation attempt put output in the wrong location when run from the repository root.
- A second root-level `protoc` attempt hit duplicate/import confusion because ESP-IDF imports like `constants.proto` expect the schema working directory layout. Running generation through Buf from `internal/provisioning/native/proto` avoided the ad hoc command issues.

### What I learned

- ESP-IDF's protobuf schemas can be vendored with minimal edits: adding `go_package` is enough for Go generation.
- Keeping all ESP-IDF provisioning schemas in one generated Go package avoids import/package friction between protocomm and WiFi provisioning messages.

### What was tricky to build

- ESP-IDF schemas import files by basename, for example `import "constants.proto"`. That means the generation module needs to present the schema directory as the import root.
- Buf lint needed exceptions because the upstream ESP-IDF schemas do not define packages and use enum names that do not follow Buf's default style rules. The project should preserve upstream schema shape rather than rewriting it aggressively.

### What warrants a second pair of eyes

- Review whether the vendored `.proto` files should stay as edited copies with `go_package` or whether a future script should copy them from `IDF_PATH` and patch package options.
- Review whether all generated schemas are needed immediately. `wifi_scan` and `wifi_ctrl` are included for parity, but Phase 3/4 can initially use only session/sec1/wifi_config.

### What should be done in the future

- Phase 3 should implement Security 1 using the generated `SessionData` and `Sec1Payload` types.
- Add golden tests for serialized protobuf bytes once native Security 1 fixtures are available.

### Code review instructions

- Start with `internal/provisioning/native/proto/buf.yaml` and `buf.gen.yaml`.
- Then inspect `internal/provisioning/native/proto/espidf/session.proto`, `sec1.proto`, and `wifi_config.proto`.
- Finally inspect `internal/provisioning/native/proto_roundtrip_test.go`.
- Validate with:
  - `go generate ./internal/provisioning/native/proto`
  - `go test ./...`

### Technical details

Commands run:

```bash
cd almanach/internal/provisioning/native/proto
buf generate
cd /home/manuel/workspaces/2026-05-08/extract-almanach/almanach
go generate ./internal/provisioning/native/proto
go test ./...
git commit -m "Add ESP-IDF provisioning protobuf bindings"
```

## Step 4: Phase 3 Linux BLE transport skeleton

This step added the first native BLE transport implementation behind the existing `Transport` interface. The transport uses `tinygo.org/x/bluetooth` because it provides Linux BLE central support through a Go API that can scan, connect, discover services, discover characteristics, write bytes, and read bytes.

The implementation is still a skeleton in the sense that it has not yet been hardware-tested through the CLI, but it is real transport code rather than a fake. It scans for the advertised service name, connects to the device, discovers the Almanach ESP-IDF provisioning service UUID, maps the five provisioning endpoint UUIDs, and implements byte-oriented endpoint write/read. This prepares Phase 4, where the existing native `proto-ver` verifier can be exercised against the real AtomS3R instead of the fake transport.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue executing the ticket implementation phases with focused commits and diary updates.

**Inferred user intent:** Progress from fake protocol tests toward a real native Go BLE client that can be validated automatically from the CLI.

**Commit (code):** 5b13f0871788fbbfda00582ba9c3bf65e76f7ee5 — "Add native BLE transport skeleton"

### What I did

- Added `tinygo.org/x/bluetooth v0.15.0` as the BLE dependency.
- Added `internal/provisioning/native/uuid.go` with:
  - provisioning service UUID `021a9004-0382-4aea-bff4-6b3f1c5adfb4`
  - fallback endpoint characteristic UUIDs for `prov-ctrl`, `prov-scan`, `prov-session`, `prov-config`, and `proto-ver`
- Added `internal/provisioning/native/tinygo_transport.go` behind `//go:build linux`.
- Implemented `NewTinyGoTransport()`.
- Implemented `Connect()` to:
  - enable the Bluetooth adapter
  - scan for the exact service name such as `ALM_0F2320`
  - connect to the device
  - discover the provisioning service
  - discover known endpoint characteristics
- Implemented `Send()` as write request bytes then read response bytes from the endpoint characteristic.
- Ran `go test ./...`.

### Why

- The protocol code should not depend on BlueZ details. The `Transport` interface lets the native provisioning protocol call `Send(endpoint, bytes)` while the BLE transport owns scanning and GATT details.
- `tinygo.org/x/bluetooth` compiles in the repo and exposes the operations needed for the first native hardware test.
- This transport uses fallback endpoint UUIDs because the TinyGo central API does not expose user-description descriptor reads in the same way Chrome and Bleak do. The fallback UUIDs are already validated by the browser trace.

### What worked

- `go get tinygo.org/x/bluetooth@v0.15.0` succeeded.
- `go doc` confirmed the needed API surface: `Adapter.Scan`, `Adapter.Connect`, `Device.DiscoverServices`, `DeviceService.DiscoverCharacteristics`, `DeviceCharacteristic.Write`, and `DeviceCharacteristic.Read`.
- `go test ./...` passed.

### What didn't work

- Descriptor-based endpoint discovery is not implemented in this transport because the chosen TinyGo API does not expose descriptor reads through the documented `DeviceCharacteristic` API.
- No hardware test was run in this step. Phase 4 should wire the transport into a native version action and test it against `ALM_0F2320`.

### What I learned

- The browser's fallback endpoint UUID map is directly useful for native Go. ESP-IDF derives endpoint characteristics by replacing the service UUID's 16-bit component with `0xFF4F` through `0xFF53`.
- The transport can be byte-oriented from the beginning, which is compatible with future Security 1 and protobuf payloads.

### What was tricky to build

- BLE scanning is callback-driven. The transport wraps scanning with context cancellation and calls `StopScan()` once the requested local name is found.
- There is a short-term tradeoff between descriptor discovery and implementation progress. Python and Chrome can read descriptor `0x2901`; TinyGo's documented central API does not show descriptor reads, so fallback UUIDs are the practical first implementation.

### What warrants a second pair of eyes

- Review whether `tinygo.org/x/bluetooth` is the right long-term BLE library for Linux provisioning, especially if descriptor reads or pairing control become necessary.
- Review the scan timeout behavior. The code has a fixed 30-second fallback timeout in addition to caller context cancellation.

### What should be done in the future

- Phase 4 should add a native CLI path for `--action version` using `TinyGoTransport` and the existing `VerifyProtoVersion()` function.
- Hardware-test the transport against the AtomS3R and compare output with Python and browser traces.

### Code review instructions

- Start with `internal/provisioning/native/tinygo_transport.go`.
- Check that all BLE-specific behavior stays behind the `Transport` interface.
- Validate with:
  - `go test ./...`

### Technical details

Commands run:

```bash
go get tinygo.org/x/bluetooth@v0.15.0
go doc tinygo.org/x/bluetooth.Adapter
go doc tinygo.org/x/bluetooth.Device
go doc tinygo.org/x/bluetooth.DeviceService
go doc tinygo.org/x/bluetooth.DeviceCharacteristic
gofmt -w internal/provisioning/native/tinygo_transport.go internal/provisioning/native/uuid.go
go test ./...
git commit -m "Add native BLE transport skeleton"
```

## Step 5: Phase 4 native proto-ver CLI and hardware validation

This step connected the native transport and native `proto-ver` verifier to the existing `ble-provision` command. The command now accepts `--implementation python|native`, defaults to the existing Python implementation, and routes `--implementation native --action version` through the Go BLE transport and `VerifyProtoVersion()`.

The hardware test succeeded against the AtomS3R advertising as `ALM_0F2320`. This is the first end-to-end native Go BLE/protocomm validation: the Go process scanned, connected over BLE, discovered the ESP-IDF provisioning service and five endpoint characteristics, wrote `v1.1` to `proto-ver`, read the JSON response, parsed it, and emitted a Glazed row.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue executing ticket phases and validate the native Go path on real hardware when the code reaches that point.

**Inferred user intent:** Replace the Python provisioning dependency incrementally while preserving known-good Python behavior as fallback.

**Commit (code):** 9320f58a6f1ecf662d1c98360bd5b1af3494e396 — "Wire native BLE proto-ver command"

### What I did

- Added `implementation` flag to `ble-provision`:
  - choices: `python`, `native`
  - default: `python`
- Added `internal/app/cmd_ble_provision_native.go`.
- Routed native `--action version` through:
  - `native.NewTinyGoTransport()`
  - `transport.Connect(ctx, serviceName)`
  - `native.NewClient(transport).VerifyVersion(ctx, protoVer)`
- Added structured Glazed output fields for native version checks:
  - `implementation`
  - `proto_ver`
  - `sec_ver`
  - `sec_patch_ver`
  - `capabilities`
  - `raw_response`
  - `duration_ms`
  - `endpoint_count`
- Returned explicit not-implemented errors for native actions other than `version`.
- Ran `go test ./...`.
- Ran a hardware native version check against `ALM_0F2320`.

### Why

- The native package was only unit-tested through fake transport before this step. The project needed a CLI path to exercise real BLE automatically outside the browser.
- Keeping Python as the default preserves the known-good provisioning path while native support is incomplete.
- Version probing is the right first hardware action because it is plaintext and already validated through Python and Chrome.

### What worked

`go test ./...` passed.

The native hardware run succeeded:

```bash
go run ./cmd/almanach-render-service ble-provision \
  --implementation native \
  --action version \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --timeout 45 \
  --output yaml
```

Output:

```yaml
action: version
capabilities:
    - wifi_scan
duration_ms: 154
endpoint_count: 5
implementation: native
pop: alm-0f2320
proto_ver: v1.1
raw_response: |-
    {
    	"prov":	{
    		"ver":	"v1.1",
    		"sec_ver":	1,
    		"sec_patch_ver":	0,
    		"cap":	["wifi_scan"]
    	}
    }
sec_patch_ver: 0
sec_ver: 1
service_name: ALM_0F2320
```

### What didn't work

- Native `provision`, `reset`, and `reprov` intentionally return not-implemented errors for now.
- The native transport uses fallback endpoint UUIDs rather than descriptor names because the selected TinyGo BLE API does not expose descriptor reads in the documented central API.

### What I learned

- The TinyGo BLE transport is sufficient for the native `proto-ver` path on the current Linux host.
- The native endpoint UUID mapping agrees with Chrome and ESP-IDF: five endpoints are available and `proto-ver` returns the expected JSON capability response.
- The native Go implementation has now reached parity with the first Python wrapper milestone: `--action version`.

### What was tricky to build

- The existing command validates passphrase and builds Python command arguments before execution. Native routing must happen before Python argument construction, but after shared validation for service name and timeout.
- The command must preserve the Python default while allowing native testing. A new `implementation` field is less disruptive than introducing a separate command.

### What warrants a second pair of eyes

- Review whether the native implementation selector should be exposed in help now or hidden until native provisioning is complete.
- Review whether native version should include endpoint UUIDs in output for debugging.

### What should be done in the future

- Phase 5 should implement Security 1 over `prov-session` using generated protobuf bindings.
- Once Security 1 exists, the native command can support real `provision`.

### Code review instructions

- Start in `internal/app/cmd_ble_provision.go` and find the `implementation` flag and native routing point.
- Then inspect `internal/app/cmd_ble_provision_native.go`.
- Validate with:
  - `go test ./...`
  - `go run ./cmd/almanach-render-service ble-provision --implementation native --action version --service-name ALM_0F2320 --timeout 45 --output yaml`

### Technical details

Commands run:

```bash
go test ./...
go run ./cmd/almanach-render-service ble-provision --implementation native --action version --service-name ALM_0F2320 --pop alm-0f2320 --timeout 45 --output yaml
git commit -m "Wire native BLE proto-ver command"
```

## Step 6: Phase 5 native Security 1 session

This step ported ESP-IDF protocomm Security 1 from the Python reference into native Go. The new `Security1Session` performs the two-message `prov-session` handshake, derives the X25519 shared secret, applies the optional SHA-256(PoP) XOR adjustment, and initializes a stateful AES-256-CTR stream that matches Espressif's Python implementation.

The most important outcome is that the native protocol stack can now create the encrypted session required before sending WiFi credentials. Tests include a fake Security 1 device so the client and device streams advance together through setup0, setup1, client-encrypted data, and device-encrypted responses.

### Prompt Context

**User prompt (verbatim):** "Implement the next tasks, keep a diary (read docmgr and diary skills), commit at appropriate intervals."

**Assistant interpretation:** Continue the native Go provisioning ticket, read/follow the docmgr and diary workflows, implement the next task phases, test them, and commit code/docs at reasonable boundaries.

**Inferred user intent:** Move the native Go BLE provisioning client from proto-ver proof toward real encrypted credential provisioning while preserving traceable project documentation.

**Commit (code):** 87df9f9e3ba3811b8a13647295609754203d900b — "Implement native Security 1 session"

### What I did

- Read the docmgr and diary skills before continuing.
- Added `internal/provisioning/native/security1.go`.
- Implemented Security 1 setup0 request generation with a generated X25519 client key.
- Implemented setup0 response parsing for device public key and device random IV.
- Implemented X25519 shared-secret derivation.
- Implemented SHA-256(PoP) XOR shared-key adjustment.
- Implemented AES-256-CTR stream initialization and continuous stream encrypt/decrypt helpers.
- Implemented setup1 client proof and device proof verification.
- Added `Client.EstablishSecurity1()`.
- Added fake-device tests in `internal/provisioning/native/security1_test.go`.
- Ran `go test ./...`.

### Why

- ESP-IDF WiFi provisioning requires Security 1 before `prov-config` accepts encrypted WiFi config messages.
- Porting Security 1 into Go removes the biggest dependency on `esp_prov.py` and gives the future browser JavaScript port a clearer reference implementation.

### What worked

- The generated protobuf bindings were sufficient for `SessionData` / `Sec1Payload` setup messages.
- The fake device test proved successful PoP handshake and encrypted bidirectional payload flow.
- `go test ./...` passed after fixing the wrong-PoP expectation.

### What didn't work

- The first wrong-PoP test expected the client-side error `failed to verify device`, but the fake device rejected the setup1 proof earlier with `client failed to prove possession of shared key`. I updated the test to assert the relevant shared-key failure instead.

### What I learned

- The Python client uses one AES-CTR encryptor for both encryption and decryption. The Go implementation must preserve that continuous stream behavior exactly; separate encrypt/decrypt streams would drift.
- The PoP does not participate directly in protobuf messages. It only mutates the shared secret via SHA-256(PoP) XOR.

### What was tricky to build

- CTR stream ordering is the sharp edge. Setup1 encrypts the device public key first, then setup1 response decrypts the device proof using the next bytes of the same keystream. Application payloads continue from there.
- The fake device had to mirror the same stream-consumption order to catch drift that simple protobuf round-trips would miss.

### What warrants a second pair of eyes

- Review the continuous AES-CTR stream behavior in `security1.go`; this is the most compatibility-critical part.
- Review whether comparing proof bytes via `string(...)` should be replaced with `bytes.Equal` or constant-time comparison before wider use.

### What should be done in the future

- Hardware-test Security 1 as part of the native provisioning command.
- Add fixtures from a Python run if we want golden-byte comparisons for setup messages.

### Code review instructions

- Start with `internal/provisioning/native/security1.go`, especially `handleSetup0Response`, `setup1Request`, and `handleSetup1Response`.
- Then inspect `internal/provisioning/native/security1_test.go` and the fake device's stream ordering.
- Validate with `go test ./...`.

### Technical details

Reference implementation:

```text
/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/security/security1.py
```

Validation command:

```bash
go test ./...
```

## Step 7: Phase 6 encrypted WiFi config flow

This step added the encrypted `prov-config` commands used after Security 1 completes. The native package can now send SetConfig, ApplyConfig, and GetStatus requests by serializing ESP-IDF WiFi protobuf payloads, encrypting them with the active Security 1 stream, sending them over the transport, decrypting responses, and parsing the response protobufs.

A higher-level `ProvisionWiFi` helper now sequences SetConfig, ApplyConfig, and status polling until the device reports a terminal state. This is still fake-transport tested rather than live-provisioned, but it exercises the same encrypted stream continuity that the real device uses.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Continue implementing the next native provisioning phases with tests and commits.

**Inferred user intent:** Build the native Go stack far enough to send WiFi credentials without the Python fallback.

**Commit (code):** fdfe70b327d8e2bf64e6beede28135c651fe0a55 — "Implement native encrypted WiFi config flow"

### What I did

- Added `internal/provisioning/native/wifi_config.go`.
- Implemented encrypted `SetWiFiConfig`.
- Implemented encrypted `ApplyWiFiConfig`.
- Implemented encrypted `GetWiFiStatus`.
- Added `WiFiStatus` with state text and terminal-state helpers.
- Added `Client.ProvisionWiFi()` to set credentials, apply them, and poll status.
- Extended the fake Security 1 transport to decrypt and respond to `prov-config` messages.
- Added `internal/provisioning/native/wifi_config_test.go` covering success, failed WiFi status, and missing security session.
- Ran `go test ./...`.

### Why

- Security 1 alone only establishes encryption. Actual provisioning requires the WiFi config protobuf flow over `prov-config`.
- Keeping this in the native package lets the CLI remain thin and keeps protocol sequencing testable without hardware.

### What worked

- The generated ESP-IDF WiFi protobuf bindings mapped cleanly to SetConfig, ApplyConfig, and GetStatus.
- Fake encrypted tests verified that credentials are transmitted through the encrypted path and that polling stops on connected or failed states.
- `go test ./...` passed.

### What didn't work

- No hardware provisioning run was attempted in this step because live provisioning requires known WiFi credentials and changes device state.

### What I learned

- ESP-IDF's WiFi provisioning status response carries both a generic protocomm `Status` and a station state enum. The station state is the useful high-level value for CLI output.
- Failed WiFi status should be returned as a terminal status rather than necessarily as a Go error; the command can then report the device's failure reason.

### What was tricky to build

- The same Security 1 stream is used for config request encryption and response decryption. The tests had to make the fake device decrypt requests and encrypt responses with the peer stream in exactly the same sequence.
- `ProvisionWiFi` needs to avoid busy-looping while still making tests fast, so it accepts a polling interval and normalizes non-positive values to one second.

### What warrants a second pair of eyes

- Review whether `Disconnected` should be treated as terminal in all cases or whether a short retry window should be allowed after ApplyConfig.
- Review CLI semantics for failed status: returning a row with `wifi_state=failed` may be better than returning a process error, but callers may expect non-zero exit for failed provisioning.

### What should be done in the future

- Hardware-test `ProvisionWiFi` through the CLI.
- Add richer output for connected-state details if the firmware returns IP/auth/SSID metadata.

### Code review instructions

- Start with `internal/provisioning/native/wifi_config.go`.
- Then inspect the `EndpointProvConfig` branch in `internal/provisioning/native/security1_test.go` and the tests in `wifi_config_test.go`.
- Validate with `go test ./...`.

### Technical details

Reference implementation:

```text
/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/prov/wifi_prov.py
```

Validation command:

```bash
go test ./...
```

## Step 8: Phase 7 native provisioning CLI wiring

This step connected the native encrypted provisioning flow to the existing `ble-provision` command. Python remains the default implementation, while `--implementation native --action provision` now runs proto-ver verification, Security 1 establishment, encrypted SetConfig, ApplyConfig, and status polling.

The command also preserves the existing stdin passphrase prompt for native provisioning and keeps reset/reprov unsupported in native mode with explicit errors. This gives the project a real hardware-testable native provisioning path without removing the Python fallback.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Continue from the native protocol package into CLI integration while keeping fallback behavior safe.

**Inferred user intent:** Make native Go provisioning usable from the same operator command that previously delegated to Python.

**Commit (code):** 4a3130ec051b02a36ecb6d204d6c679be70bf425 — "Wire native BLE provisioning command"

### What I did

- Updated `ble-provision` help for native support.
- Moved passphrase resolution before implementation dispatch so native provision also supports stdin passphrase prompting.
- Updated `runNativeBLEProvision` to support `version` and `provision`.
- Added native dry-run output.
- Routed native provision through:
  - BLE connect
  - proto-ver verification
  - Security 1 establishment
  - encrypted WiFi provisioning and status polling
- Added structured Glazed output fields for native provisioning status.
- Ran `go test ./...`.

### Why

- A protocol package is useful, but operators need to exercise it through the Almanach binary.
- Keeping `--implementation python` as the default avoids breaking the known fallback path while native support matures.

### What worked

- The command compiled and all tests passed.
- The native version path remains available, and native provision is now reachable behind an explicit flag.

### What didn't work

- I did not run live native provisioning because the task needs real WiFi credentials and changes the device's provisioning state.
- Native reset/reprov remain unsupported.

### What I learned

- Dispatch placement matters: native routing originally happened before passphrase prompting, which would have made native provision fail whenever `--passphrase` was omitted. Moving dispatch after passphrase resolution keeps behavior aligned with Python mode.

### What was tricky to build

- The same command needs to support two implementations with different maturity levels. The safest shape is an explicit selector with Python default and native-only support for actions that are implemented.
- Native provision should still verify proto-ver first, because that gives a clear plaintext compatibility check before entering the encrypted session.

### What warrants a second pair of eyes

- Review whether live provisioning failures should produce non-zero exit codes even when the command successfully reports `wifi_state=failed`.
- Review whether native dry-run should avoid passphrase prompting; current behavior matches the Python branch, which prompts before dry-run for `action=provision`.

### What should be done in the future

- Phase 8 should hardware-validate native provisioning with known WiFi credentials.
- Add reset/reprov native actions after provisioning parity is proven.

### Code review instructions

- Start with `internal/app/cmd_ble_provision.go` for validation, passphrase prompting, and implementation dispatch.
- Then inspect `internal/app/cmd_ble_provision_native.go` for native action sequencing.
- Validate with `go test ./...` and then a controlled live provisioning run when credentials are available.

### Technical details

Validation command:

```bash
go test ./...
```

## Step 9: Phase 8 native provisioning hardware validation

This step used the newly wired native Go provisioning path against the real AtomS3R. The native command completed the full BLE provisioning sequence: proto-ver verification, Security 1, encrypted WiFi config, ApplyConfig, and status polling until the device reported a connected state.

The firmware monitor independently confirmed the same path. It logged Security 1 establishment, receipt of the configured SSID, successful WPA2 association, IP assignment, web server startup, provisioning stop, BLE disconnect, and provisioning manager deinitialization. During review of the CLI row, I noticed that the command printed a default `AuthError` fail reason even for a connected state; I fixed that output so fail reasons are only shown when the protobuf oneof actually includes a failure reason.

### Prompt Context

**User prompt (verbatim):** "[WiFi credentials provided for hardware validation; passphrase intentionally omitted from diary to avoid storing secrets]"

**Assistant interpretation:** Use the provided WiFi credentials to run Phase 8 hardware validation of the native Go provisioning command.

**Inferred user intent:** Prove that native Go provisioning works end-to-end on the real AtomS3R and record the result.

**Commit (code):** 89cc5a34b25f621f41441923fd31c2ff8793079b — "Clean native WiFi status output"

### What I did

- Ran native provisioning against `ALM_0F2320` with the provided SSID and stdin-supplied passphrase.
- Confirmed the CLI reported `wifi_state: connected` and `wifi_status: Success`.
- Captured firmware monitor evidence from the `alm-button-test` tmux session.
- Fixed misleading native CLI status output so `wifi_fail_reason` is blank unless `RespGetStatus` actually carries a fail-reason oneof.
- Added `HasFailReason` and `FailReasonText()` to `WiFiStatus`.
- Updated the failed-status unit test to assert explicit fail-reason presence.
- Ran `go test ./...`.

### Why

- Phase 8 needed real hardware validation after the fake-transport tests for Security 1 and encrypted WiFi config.
- The misleading `AuthError` output could confuse operators because enum value zero is also the default when no fail reason is present.

### What worked

The native command completed successfully and reported a connected state:

```yaml
action: provision
capabilities:
    - wifi_scan
duration_ms: 10555
endpoint_count: 5
implementation: native
proto_ver: v1.1
read_passphrase_from_stdin: true
sec_patch_ver: 0
sec_ver: 1
service_name: ALM_0F2320
ssid: Verizon_9DNVB9
wifi_attempts_remaining: 0
wifi_state: connected
wifi_status: Success
```

Firmware monitor evidence included:

```text
provisioning: Provisioning security session established
provisioning: Received WiFi credentials for SSID 'Verizon_9DNVB9'
wifi:connected with Verizon_9DNVB9
esp_netif_handlers: sta ip: 192.168.1.242, mask: 255.255.255.0, gw: 192.168.1.1
wifi_mgr: Got IP: 192.168.1.242
provisioning: Provisioned WiFi credentials connected successfully
stoms3r: WiFi connected — starting web server
web_server: HTTP server started on port 80
wifi_prov_mgr: Provisioning stopped
provisioning: BLE WiFi provisioning ended
wifi_prov_scheme_ble: BTDM memory released
provisioning: WiFi provisioning manager deinitialized
```

`go test ./...` passed after the output cleanup.

### What didn't work

- The first successful hardware row printed `wifi_fail_reason: AuthError` even though `wifi_state: connected`. This happened because `WifiConnectFailedReason_AuthError` is enum value zero, and the code used `GetFailReason()` without checking whether the response's oneof was actually a fail reason.

### What I learned

- For generated proto3 oneof fields, the presence check matters. Getter defaults can look like real values, especially for enum zero.
- The native Go implementation now reaches parity with the Python path for full initial provisioning on this device.

### What was tricky to build

- The hardware run changes device state: after successful provisioning, BLE provisioning stops and the device starts its HTTP server. That means repeated tests require reset/reprovision support or firmware-side credential clearing.
- The status bug was subtle because the command succeeded while carrying contradictory diagnostic data. Checking the oneof discriminant fixed the ambiguity.

### What warrants a second pair of eyes

- Review `WiFiStatus.Terminal()` semantics. Treating `Disconnected` as terminal might be too aggressive after ApplyConfig on slower networks.
- Review whether a connected provisioning status should include IP address in the CLI row if ESP-IDF returns connected-state metadata.

### What should be done in the future

- Implement native reset/reprov support so repeated hardware tests do not require external state clearing.
- Use this Go implementation as the concrete reference for the browser JavaScript provisioning port.

### Code review instructions

- Start with `internal/provisioning/native/wifi_config.go`, especially `GetWiFiStatus()` and `FailReasonText()`.
- Then inspect `internal/app/cmd_ble_provision_native.go` output fields.
- Validate with `go test ./...`.
- Hardware validation command shape:

```bash
printf '[passphrase]\n' | go run ./cmd/almanach-render-service ble-provision \
  --implementation native \
  --action provision \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --ssid Verizon_9DNVB9 \
  --timeout 180 \
  --output yaml
```

### Technical details

The successful hardware validation was followed by a small output correctness fix. The code commit for this step is therefore the status-output cleanup commit; the actual provisioning success happened immediately before that fix.
