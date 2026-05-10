# Tasks

## TODO

- [x] Analyze AtomS3R display/GIF/button donor firmware in esp32-s3-m5
- [x] Create intern-facing display status and long-press BLE pairing design guide
- [x] Upload display pairing design guide to reMarkable

## Implementation plan

- [x] Phase 0: Resolve implementation constraints before code
  - [x] Confirm whether standalone Almanach should vendor display dependencies or use ESP-IDF component manager
  - [x] Confirm GPIO7 conflict between AtomS3R backlight gate and printer UART mapping
  - [x] Decide initial backlight strategy: I2C brightness only with disabled GPIO gate
- [x] Phase 1: Add text-only display infrastructure
  - [x] Add display/backlight Kconfig with Almanach-specific symbols
  - [x] Add M5GFX/display dependency without coupling to esp32-s3-m5 donor paths
  - [x] Add `display_hal.cpp/.h` for AtomS3R GC9107 initialization
  - [x] Add `display_app.cpp/.h` for boot/status/error screens
  - [x] Wire display initialization into `app_main.c`
  - [x] Build firmware and commit text-only display bring-up
- [x] Phase 2: Add display status model
  - [x] Add a polling UI status task
  - [x] Read `provisioning_mgr_get_status()` into display status
  - [x] Read `wifi_mgr_is_connected()` and `wifi_mgr_get_ip()` into display status
  - [x] Show BLE service name, PoP, client/security state, WiFi state, IP, and web readiness
  - [x] Build firmware and commit display status screens
- [x] Phase 3: Add button long-press pairing mode
  - [x] Add `button_input` ISR plus task state machine for GPIO41 active-low
  - [x] Detect short press, pairing hold, and reset-confirm hold outside ISR context
  - [x] On pairing hold, call `provisioning_mgr_start_if_needed()` and show pairing screen
  - [x] On reset-confirm hold, erase console WiFi and provisioning state consistently with `prov_reset`
  - [x] Build firmware and commit long-press pairing behavior
- [ ] Phase 4: Hardware validation
  - [ ] Flash AtomS3R and validate boot/status screen
  - [ ] Validate erased-flash BLE pairing screen
  - [ ] Validate long-press pairing behavior
  - [ ] Validate reset-confirm behavior only after on-screen feedback is visible
  - [ ] Update diary/changelog with monitor/display observations
- [ ] Phase 5: Optional GIF animation layer
  - [ ] Vendor or add `echo_gif` and AnimatedGIF dependencies
  - [ ] Add storage/asset strategy for GIF files
  - [ ] Add GIF background/screensaver mode that never blocks status rendering
  - [ ] Validate heap stability and status preemption during provisioning/errors
