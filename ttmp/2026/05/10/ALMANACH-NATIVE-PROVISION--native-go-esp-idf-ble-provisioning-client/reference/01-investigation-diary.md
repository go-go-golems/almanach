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
  - Path: /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_ble_provision.go
    Note: Existing Go command inspected to identify Python delegation boundary.
  - Path: /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py
    Note: Reference provisioning control flow inspected.
  - Path: /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/security/security1.py
    Note: Security 1 algorithm inspected.
  - Path: /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/prov/wifi_prov.py
    Note: WiFi config request/status flow inspected.
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
