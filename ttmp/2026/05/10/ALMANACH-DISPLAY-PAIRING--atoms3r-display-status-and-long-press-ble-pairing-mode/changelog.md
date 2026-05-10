# Changelog

## 2026-05-10

- Initial workspace created


## 2026-05-10

Created the AtomS3R display status and long-press BLE pairing design guide after analyzing display/GIF/button donor firmware under `esp32-s3-m5` and the current Almanach provisioning firmware.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-DISPLAY-PAIRING--atoms3r-display-status-and-long-press-ble-pairing-mode/design/01-atoms3r-display-status-and-long-press-ble-pairing-design-guide.md — Intern-facing design guide.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-DISPLAY-PAIRING--atoms3r-display-status-and-long-press-ble-pairing-mode/reference/01-investigation-diary.md — Investigation diary.
- /home/manuel/workspaces/2026-05-08/extract-almanach/esp32-s3-m5/0013-atoms3r-gif-console/main/hello_world_main.cpp — GIF playback donor.
- /home/manuel/workspaces/2026-05-08/extract-almanach/esp32-s3-m5/0017-atoms3r-web-ui/main/display_app.cpp — Display wrapper donor.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/provisioning_mgr.c — Target BLE provisioning manager.

## 2026-05-10

Uploaded the AtomS3R display pairing design bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-DISPLAY-PAIRING--atoms3r-display-status-and-long-press-ble-pairing-mode/design/01-atoms3r-display-status-and-long-press-ble-pairing-design-guide.md — Uploaded in bundle.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-DISPLAY-PAIRING--atoms3r-display-status-and-long-press-ble-pairing-mode/reference/01-investigation-diary.md — Uploaded in bundle.
- /ai/2026/05/10/ALMANACH-DISPLAY-PAIRING/ALMANACH_AtomS3R_Display_Pairing_Guide.pdf — reMarkable destination.

## 2026-05-10

Phase 0/1 implementation: added M5GFX managed dependency, Almanach display/backlight/button Kconfig, text-only AtomS3R display bring-up, and best-effort boot screen wiring.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/idf_component.yml — M5GFX managed component dependency.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/dependencies.lock — ESP-IDF component lockfile.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/Kconfig.projbuild — Almanach display/backlight/button configuration.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/display_hal.cpp — AtomS3R GC9107 display bring-up.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/display_app.cpp — Text-only boot/status/error display API.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c — Best-effort display initialization and boot screen.

## 2026-05-10

Phase 2 implementation: added a polling display status task that renders provisioning state, BLE service/PoP, client/security flags, WiFi connection state, and IP address.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c — Added `display_status_task()` and display status task startup.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/display_app.cpp — Existing text status renderer used by the task.

## 2026-05-10

Phase 3 implementation: added GPIO41 button ISR/task handling for long-press BLE pairing mode and reset-confirm credential clearing.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/button_input.c — Button ISR, debounce, pairing hold, and reset-confirm hold implementation.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/button_input.h — Button input startup API.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c — Button task startup during boot.

## 2026-05-10

Phase 4 partial hardware validation: flashed the display/button firmware, found and fixed an ESP-IDF old/new I2C driver conflict, and confirmed monitor logs for display init, BLE provisioning, button setup, and console startup.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/backlight.cpp — Switched backlight I2C writes to the new ESP-IDF I2C master API.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-DISPLAY-PAIRING--atoms3r-display-status-and-long-press-ble-pairing-mode/reference/01-investigation-diary.md — Hardware flash and I2C conflict diary entry.


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

Adjusted AtomS3R button UX after physical press logs showed the hold detector working but the provisioned-device pairing path doing nothing. Pairing is now a 3-second hold, destructive reset is now a 10-second hold, and pairing uses the force-start provisioning path so a configured device can re-enter BLE pairing mode.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/button_input.c — Pairing hold now force-starts BLE provisioning.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/Kconfig.projbuild — Reset hold default increased to 10 seconds.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/sdkconfig.defaults — Committed button timing defaults for fresh builds.


## 2026-05-10

Confirmed the revised 3-second AtomS3R button hold starts BLE pairing mode on hardware. Monitor logs show `button pairing hold reached: 3093 ms`, NimBLE advertising, and `wifi_prov_mgr` starting service `ALM_0F2320`.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/button_input.c — Revised pairing hold behavior validated on hardware.


## 2026-05-10

Improved long-hold reset feedback after hardware testing showed the 3-second pairing threshold was reached but the screen did not clearly communicate the continued hold/reset countdown. The display now shows larger `PAIR ON` and `RESET Ns` text, and the button task logs hold progress once per second after pairing.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/display_app.cpp — Clearer 128x128 pairing/reset countdown UI.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/button_input.c — Once-per-second hold progress logging after pairing threshold.


## 2026-05-10

Updated the button/reset model so reset countdown is reachable from an already-active pairing screen. A press that starts while BLE provisioning is running now immediately arms reset countdown instead of re-triggering pairing. Display drawing is also guarded by a mutex to reduce status-task/button-task canvas contention.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/button_input.c — Adds explicit already-pairing reset-countdown path.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/display_app.cpp — Adds reset hold screen and display draw mutex.
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/display_app.h — Exposes reset hold screen function.
