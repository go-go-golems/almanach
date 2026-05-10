# Changelog

## 2026-05-10

- Initial workspace created


## 2026-05-10

Created native Go ESP-IDF BLE provisioning research ticket, design guide, investigation diary, tasks, and evidence excerpts for porting the current Python esp_prov.py provisioning path to Go.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_ble_provision.go — Existing Go/Glazed wrapper and CLI surface to preserve
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — Firmware provisioning identity, service UUID, and Security 1 setup
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-client.js — Browser-validated service UUID, endpoint mapping, and proto-ver behavior
- /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py — Python workflow reference for native Go port
- /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/security/security1.py — Python Security 1 reference
- /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/prov/wifi_prov.py — Python WiFi config protobuf reference
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/design-doc/01-native-go-esp-idf-ble-provisioning-client-analysis-design-and-implementation-guide.md — Primary design guide
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/reference/01-investigation-diary.md — Investigation diary

## 2026-05-10

Validated the ALMANACH-NATIVE-PROVISION ticket with docmgr doctor and uploaded the design guide plus diary bundle to reMarkable at `/ai/2026/05/10/ALMANACH-NATIVE-PROVISION`.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/design-doc/01-native-go-esp-idf-ble-provisioning-client-analysis-design-and-implementation-guide.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/reference/01-investigation-diary.md — Included in uploaded bundle

## 2026-05-10

Phase 1 implementation: added the native Go provisioning package skeleton with endpoint transport interfaces, high-level client wrapper, fake transport tests, and plaintext `proto-ver` verification.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/transport.go — Native endpoint transport interface and endpoint constants
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/client.go — High-level native client skeleton
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/protover.go — Native proto-ver parsing and validation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/fake_transport_test.go — Fake transport for protocol unit tests
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/protover_test.go — Unit tests for proto-ver behavior
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/tasks.md — Phase 1 task completion
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/reference/01-investigation-diary.md — Phase 1 diary entry

## 2026-05-10

Phase 2 implementation: added Buf-based generation for vendored ESP-IDF protocomm and WiFi provisioning protobuf schemas, generated Go bindings, and added round-trip tests for session and WiFi config payloads.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/proto/buf.yaml — Buf module and lint configuration for ESP-IDF schemas
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/proto/buf.gen.yaml — Local protoc-gen-go generation config
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/proto/generate.go — go:generate hook for Buf generation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/proto/espidf — Vendored ESP-IDF protobuf schemas and generated Go bindings
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/proto_roundtrip_test.go — Generated protobuf round-trip tests
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/tasks.md — Phase 2 task completion
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-NATIVE-PROVISION--native-go-esp-idf-ble-provisioning-client/reference/01-investigation-diary.md — Phase 2 diary entry
