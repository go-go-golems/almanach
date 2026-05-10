# Tasks

## TODO

- [x] Create ALMANACH-NATIVE-PROVISION ticket workspace
- [x] Inspect current Go/Glazed ble-provision wrapper and identify Python delegation boundary
- [x] Inspect ESP-IDF esp_prov.py provisioning flow
- [x] Inspect Security 1 Python implementation and protobuf schemas
- [x] Inspect firmware/browser evidence needed for native Go compatibility
- [x] Write intern-facing native Go provisioning design guide
- [x] Write investigation diary
- [x] Phase 1: add Go package layout for native provisioning transport/protocol modules
  - [x] Create `internal/provisioning/native` package with transport interfaces and high-level client skeleton
  - [x] Add fake transport test helper for protocol unit tests without BLE hardware
  - [x] Implement native `proto-ver` parsing/verification against fake transport
  - [x] Run `go test ./...`
  - [x] Update diary/changelog and commit Phase 1
- [ ] Phase 2: generate or hand-maintain Go protobuf bindings from ESP-IDF schemas
- [ ] Phase 3: implement BLE transport using a Linux BlueZ-capable Go BLE library
- [ ] Phase 4: implement proto-ver and endpoint discovery parity with browser/Linux Python client
- [ ] Phase 5: implement Security 1 X25519 + PoP + AES-CTR session in Go
- [ ] Phase 6: implement encrypted WiFi config set/apply/status polling
- [ ] Phase 7: add native mode to ble-provision behind a flag while keeping Python fallback
- [ ] Phase 8: hardware validate native Go provisioning against AtomS3R
- [ ] Phase 9: use native Go implementation as reference for future browser JavaScript port
- [x] Upload native Go provisioning design package to reMarkable
