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


## 2026-05-10

Fixed the first-time provisioning HTTP startup gap by keeping the web-server wait task alive after the initial 30-second warning. Hardware validation confirmed that, after `prov_reset` and BLE provisioning, the firmware starts port 80 without reboot and `/api/status` returns WiFi/printer status.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c — `web_server_task()` now keeps waiting until WiFi connects and starts the HTTP server after delayed provisioning.

## 2026-05-10

Added the first real browser Web Bluetooth client slice: Chrome picker for `ALM_` devices, GATT connection, ESP-IDF provisioning service discovery, and a Storybook state for the connected milestone.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-client.js — Web Bluetooth picker/connect/service discovery client
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.jsx — Real BLE vs mock setup flow wiring
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.stories.jsx — Real BLE connected Storybook fixture
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/types.js — Tracks selected client mode
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/web/embed/public/setup-bundle.js — Rebuilt setup bundle
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/tasks.md — Browser BLE implementation tasks
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Browser BLE diary entry

## 2026-05-10

Improved Browser BLE service-discovery diagnostics after the first Chrome hardware attempt; the client now probes both canonical and firmware-order ESP-IDF provisioning UUID candidates and reports service lookup failures distinctly from chooser cancellation.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-client.js — UUID candidate probing and contextual Web Bluetooth errors
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.stories.jsx — Updated real BLE connected log fixture
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/web/embed/public/setup-bundle.js — Rebuilt setup bundle
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Records Chrome hardware attempt and fix

## 2026-05-10

Fixed the firmware BLE provisioning service UUID lifetime bug exposed by Chrome service discovery: the custom UUID now lives in static storage before being passed to ESP-IDF's provisioning scheme.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — Keeps custom BLE service UUID in static storage for ESP-IDF pointer lifetime
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Records Chrome service-discovery debugging and firmware fix

## 2026-05-10

Flashed the AtomS3R with the static BLE service UUID firmware fix and restarted the `alm-button-test` monitor session for browser retesting.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — Firmware fix included in flashed image
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Flash/monitor validation note

## 2026-05-10

Added browser-side ESP-IDF endpoint discovery and `proto-ver` probing after Chrome successfully found the provisioning GATT service.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-client.js — Discovers endpoint descriptors and probes `proto-ver`
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.stories.jsx — Updates real BLE connected fixture with protocol verification logs
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/web/embed/public/setup-bundle.js — Rebuilt setup bundle
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/tasks.md — Marks Browser BLE Phase 2 complete
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/reference/01-investigation-diary.md — Records browser proto-ver implementation

## 2026-05-10

Browser provisioning port: created a textbook-style Obsidian deep dive on the native Go provisioning implementation, then ported the ESP-IDF provisioning protocol layers into the Web Bluetooth client. The browser setup flow now includes Security 1, minimal protobuf helpers, encrypted WiFi SetConfig/ApplyConfig/GetStatus, and status polling. Hardware browser validation remains pending because the device is currently provisioned after native Go validation.

### Related Files

- /home/manuel/code/wesen/obsidian-vault/Projects/2026/05/10/ARTICLE - Almanach BLE Provisioning - Native Go Protocol Deep Dive.md — Textbook-style project report and protocol explanation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/security1.js — Browser Security 1 implementation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-protobuf.js — Minimal ESP-IDF protobuf helpers for browser provisioning
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-client.js — Web Bluetooth client with encrypted provisioning flow
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.jsx — Real-flow UI text and logging updates
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/web/embed/public/setup-bundle.js — Rebuilt embedded setup bundle

## 2026-05-10

Reset/reprovision lifecycle: accepted the user's successful Chrome provisioning trace as browser hardware validation, added native Go encrypted `prov-ctrl` reset/reprov actions, added browser reset/reprov helpers and guarded setup-page buttons, and confirmed the firmware physical long-hold reset path already clears WiFi/provisioning state and reboots.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/wifi_ctrl.go — Native encrypted WiFi control reset/reprov implementation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/provisioning/native/wifi_ctrl_test.go — Fake encrypted transport tests for reset/reprov
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_ble_provision_native.go — Native CLI reset/reprov routing
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-protobuf.js — Browser WiFi control protobuf helpers
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-client.js — Browser reset/reprov methods
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/ProvisioningWizard.jsx — Guarded setup-page reset/reprov buttons
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/button_input.c — Existing physical long-hold reset path

## 2026-05-10

Setup rendezvous: added a Glazed provisioning user guide, added a localhost setup API for reporting the provisioned printer IP, updated the render server to use that remembered IP when `ALMANACH_PRINTER_IP` is unset, and updated the browser provisioning client to decode connected-state IP/SSID and post the result back to the setup server.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/doc/provisioning-printer-user-guide.md — Embedded Glazed help page for printer provisioning
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/setup_device.go — Provisioned-device rendezvous API and in-memory store
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/setup_device_test.go — API and printer-IP selection tests
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/server.go — Registers setup API and uses effective printer IP
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-protobuf.js — Decodes connected IP/SSID from BLE status
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/provisioning/espidf-client.js — Reports provisioned printer IP to localhost setup server
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/web/embed/public/setup-bundle.js — Rebuilt embedded setup bundle
