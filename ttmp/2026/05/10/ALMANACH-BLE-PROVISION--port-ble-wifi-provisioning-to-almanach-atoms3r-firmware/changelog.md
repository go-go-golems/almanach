# Changelog

## 2026-05-10

- Initial workspace created


## 2026-05-10

Created intern-facing BLE WiFi provisioning port analysis/design/implementation guide and investigation diary.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/design-doc/01-ble-wifi-provisioning-port-analysis-design-and-implementation-guide.md — Primary design guide
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Investigation diary


## 2026-05-10

Uploaded design guide and investigation diary bundle to reMarkable at /ai/2026/05/10/ALMANACH-BLE-PROVISION.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/design-doc/01-ble-wifi-provisioning-port-analysis-design-and-implementation-guide.md — Uploaded in reMarkable bundle
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Uploaded in reMarkable bundle


## 2026-05-10

Phase 1: enabled BLE/NimBLE/protocomm provisioning dependencies and validated a clean AtomS3R firmware build.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/CMakeLists.txt — Provisioning component dependencies
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/sdkconfig.defaults — BLE/NimBLE/protocomm provisioning defaults
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Phase 1 diary entry


## 2026-05-10

Phases 2-3: added provisioning manager, MAC-derived BLE service identity/PoP, provisioning event logging, station start helper, and boot-flow onboarding decision.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c — Boot-flow integration
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — BLE provisioning manager implementation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.h — Provisioning manager API
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/wifi_mgr.c — Stored-credential station start helper
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Phases 2-3 diary entry

