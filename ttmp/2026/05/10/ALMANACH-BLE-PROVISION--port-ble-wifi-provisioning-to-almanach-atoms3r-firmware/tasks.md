# Tasks

## TODO

- [x] Add tasks here

- [ ] Implement provisioning_mgr BLE WiFi provisioning subsystem
- [ ] Preserve and validate esp_console WiFi save/status/forget behavior
- [ ] Add provisioning console commands and reset flow
- [ ] Validate provisioning with Espressif app on AtomS3R hardware
- [x] Phase 1: add BLE/NimBLE/protocomm sdkconfig defaults and CMake dependencies, then clean-build firmware
- [ ] Phase 2: add provisioning_mgr.c/.h with service name, MAC-derived PoP, status, event handling, start/reset APIs
- [ ] Phase 3: integrate provisioning decision into app_main boot flow without breaking saved console WiFi autoconnect
- [ ] Phase 4: add provisioning_cmd.c/.h for prov_status, prov_start, prov_reset and register commands
- [ ] Phase 5: update wifi_forget/reset semantics to clear both explicit NVS WiFi and provisioning manager state
- [ ] Phase 6: build, flash, monitor, and validate console plus BLE provisioning behavior on AtomS3R
