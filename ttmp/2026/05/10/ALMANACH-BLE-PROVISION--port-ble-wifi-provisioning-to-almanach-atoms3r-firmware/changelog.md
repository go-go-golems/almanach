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


## 2026-05-10

Phases 4-5: added provisioning console commands and made WiFi forget/reset clear ESP-IDF provisioning state as well as explicit NVS credentials.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c — Provisioning command registration
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_cmd.c — prov_status/prov_start/prov_reset command implementation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_cmd.h — Provisioning command registration API
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/wifi_cmd.c — wifi_forget clears provisioning state
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Phases 4-5 diary entry


## 2026-05-10

Added Web Bluetooth provisioning UI design guide for almanach/web, covering browser constraints, React integration, ESP-IDF BLE protocol adapter, phases, tests, and risks.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — Firmware BLE protocol facts referenced by UI guide
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/design-doc/02-web-bluetooth-provisioning-ui-design-and-implementation-guide.md — New intern-facing web UI provisioning guide
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/almanach-studio.jsx — Primary UI integration target


## 2026-05-10

Hardware smoke-tested BLE provisioning boot path on AtomS3R after erase/flash; verified BLE advertising, prov_status, prov_start idempotence, and wifi_status from serial monitor.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — Fixed provisioning_mgr_start_if_needed to report already-running state correctly
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Hardware smoke-test diary entry


## 2026-05-10

Added Linux Go/Glazed ble-provision command for local ESP-IDF BLE provisioning feedback loops; validated protocol version check against AtomS3R.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_ble_provision.go — Go/Glazed BLE provisioning command wrapping ESP-IDF esp_prov.py
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_root.go — Registers ble-provision verb
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/design-doc/03-linux-go-cli-ble-provisioning-feedback-loop-design-and-implementation-guide.md — Intern-facing design and implementation guide
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Linux CLI provisioning diary entry


## 2026-05-10

Uploaded Linux CLI BLE provisioning guide to reMarkable at /ai/2026/05/10/ALMANACH-BLE-PROVISION.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/design-doc/03-linux-go-cli-ble-provisioning-feedback-loop-design-and-implementation-guide.md — Uploaded as ALMANACH_BLE_Linux_CLI_Provisioning_Guide.pdf


## 2026-05-10

Added Storybook coverage for the localhost setup/provisioning UI and captured css-visual-diff artifacts for setup states plus a main-editor-vs-setup visual reference.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/.storybook/main.js — Storybook React/Vite configuration
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/.storybook/preview.js — Fullscreen layout and Almanach background defaults
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.jsx — Story-friendly state/support/client seams for visual fixtures
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.stories.jsx — Setup page stories for ready, unsupported, WiFi, progress, success, and error states
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/AlmanachStudio.stories.jsx — Main editor reference story
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/artifacts/storybook-visuals — css-visual-diff screenshot and report artifacts
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Storybook/css-visual-diff diary entry

## 2026-05-10

Added a playbook for running Storybook with css-visual-diff and ignored local generated visual-diff artifacts.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/.gitignore — Ignores css-visual-diff output directories and ticket Storybook visual artifacts
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/playbooks/01-storybook-css-visual-diff-playbook.md — Operational Storybook/css-visual-diff capture playbook

## 2026-05-10

Implemented localhost serving for the setup UI: `/setup`, `/setup/bundle.js`, and `almanach-render-service setup` bound to 127.0.0.1.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/static.go — Serves setup HTML and bundle from disk or embedded assets
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_setup.go — Adds localhost-only setup command
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_serve.go — Shares HTTP server startup with explicit listen addresses
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_root.go — Registers setup command
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/static_test.go — Verifies setup and editor static routes
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/tasks.md — Marks setup serving tasks complete
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Records implementation and validation details


## 2026-05-10

Completed real hardware BLE WiFi provisioning validation with `idf.py flash monitor` and the Almanach Go/Glazed `ble-provision` command in tmux. A clean `prov_reset` followed by provisioning to SSID `Verizon_9DNVB9` succeeded, reboot autoconnect succeeded, and `/api/status` returned WiFi/printer status at `192.168.1.242`. The test exposed a remaining web-server startup gap: the first post-provisioning connection can occur after the one-time 30-second boot wait, leaving HTTP unavailable until reboot.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c — Contains the current one-time WiFi wait before web server startup.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — BLE provisioning events and successful credential connection path validated on hardware.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_ble_provision.go — Go/Glazed wrapper used for the successful real WiFi provisioning run.
