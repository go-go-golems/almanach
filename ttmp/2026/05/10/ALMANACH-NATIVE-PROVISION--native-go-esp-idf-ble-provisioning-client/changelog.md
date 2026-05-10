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
