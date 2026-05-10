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
