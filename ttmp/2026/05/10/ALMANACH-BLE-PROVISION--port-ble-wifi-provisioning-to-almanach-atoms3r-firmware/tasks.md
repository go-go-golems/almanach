# Tasks

## TODO

- [x] Add tasks here

- [x] Implement provisioning_mgr BLE WiFi provisioning subsystem
- [ ] Preserve and validate esp_console WiFi save/status/forget behavior
- [x] Add provisioning console commands and reset flow
- [ ] Validate provisioning with Espressif app on AtomS3R hardware
- [x] Phase 1: add BLE/NimBLE/protocomm sdkconfig defaults and CMake dependencies, then clean-build firmware
- [x] Phase 2: add provisioning_mgr.c/.h with service name, MAC-derived PoP, status, event handling, start/reset APIs
- [x] Phase 3: integrate provisioning decision into app_main boot flow without breaking saved console WiFi autoconnect
- [x] Phase 4: add provisioning_cmd.c/.h for prov_status, prov_start, prov_reset and register commands
- [x] Phase 5: update wifi_forget/reset semantics to clear both explicit NVS WiFi and provisioning manager state
- [ ] Phase 6: build, flash, monitor, and validate console plus BLE provisioning behavior on AtomS3R
- [x] Design Web Bluetooth provisioning UI for top-level Almanach web app
- [x] Setup page Phase 1: add standalone React setup entrypoint and mock provisioning component modules
- [x] Setup page Phase 2: update esbuild to emit setup.html/setup-bundle.js alongside Almanach editor bundle
- [x] Setup page Phase 2b: add Storybook coverage for setup states and capture css-visual-diff screenshots against the main editor
- [ ] Setup page Phase 3: serve /setup and /setup/bundle.js from embedded/local web assets in Go
- [ ] Setup page Phase 4: add almanach-render-service setup command bound to localhost
- [ ] Setup page Phase 5: validate devctl build, setup page load, and mock provisioning flow
- [ ] Setup page Phase 6: update diary/changelog and design docs with implemented route and command names
- [x] Design Linux Go/Glazed BLE provisioning CLI feedback loop
- [x] Implement ble-provision Glazed verb wrapping ESP-IDF esp_prov.py
- [x] Validate ble-provision dry-run and protocol version check against AtomS3R
- [x] Upload Linux CLI provisioning design guide to reMarkable
