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
