---
Title: Investigation Diary
Ticket: ALMANACH-DISPLAY-PAIRING
Status: active
Topics:
    - almanach
    - firmware
    - esp-idf
    - ble
    - wifi-provisioning
    - atoms3r
    - display
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-05-08/extract-almanach/esp32-s3-m5/0013-atoms3r-gif-console/main/hello_world_main.cpp
      Note: GIF console donor inspected for display, button, and GIF playback architecture.
    - Path: /home/manuel/workspaces/2026-05-08/extract-almanach/esp32-s3-m5/0017-atoms3r-web-ui/main/display_app.cpp
      Note: Display wrapper donor inspected for target module shape.
    - Path: /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c
      Note: Target boot flow inspected for display/status integration points.
ExternalSources: []
Summary: Chronological investigation diary for AtomS3R display status and long-press BLE pairing design.
LastUpdated: 2026-05-10T19:45:00-04:00
WhatFor: Use this diary to understand what was inspected and why the design recommends a staged display/button/GIF port.
WhenToUse: Read before implementing display status or physical-button pairing behavior in Almanach AtomS3R firmware.
---

# Investigation Diary

## Goal

This diary records the analysis and documentation work for adding AtomS3R display status screens and long-press BLE pairing behavior to the Almanach firmware. The target behavior is that the device can show provisioning, WiFi/IP, and pairing/reset state on its built-in display, and that holding the physical button enters BLE pairing mode.

## Step 1: Ticket setup and donor firmware analysis

I created ticket `ALMANACH-DISPLAY-PAIRING` and inspected the AtomS3R display/button/GIF firmware examples in `./esp32-s3-m5`. The most useful donors are `0013-atoms3r-gif-console` for GIF playback and queue-based controls, and `0017-atoms3r-web-ui` for a simpler display wrapper and button input task.

### Prompt Context

**User prompt (verbatim):** "analyze in ./esp32-s3-m5 for firmware for the atoms3r with display that shows gif animations and such. What I want is that if you hold down the button for a while, it enters BLE pairing mode. It also should display information about the status and IP and provisioning status and all that on the display. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Analyze donor AtomS3R display/GIF/button firmware in `esp32-s3-m5`, connect it to the current Almanach BLE provisioning firmware, create a detailed intern-facing design guide, store it in a docmgr ticket, and upload it to reMarkable.

**Inferred user intent:** Prepare an implementation handoff for making the physical AtomS3R user interface useful during BLE provisioning and WiFi connection, not just the serial console and Linux CLI.

**Commit (code):** N/A — documentation and analysis only.

### What I inspected

Donor display/GIF/button firmware:

- `esp32-s3-m5/0013-atoms3r-gif-console/main/hello_world_main.cpp`
- `esp32-s3-m5/0013-atoms3r-gif-console/main/control_plane.cpp`
- `esp32-s3-m5/0013-atoms3r-gif-console/main/control_plane.h`
- `esp32-s3-m5/0013-atoms3r-gif-console/main/display_hal.cpp`
- `esp32-s3-m5/0013-atoms3r-gif-console/main/Kconfig.projbuild`
- `esp32-s3-m5/0013-atoms3r-gif-console/main/CMakeLists.txt`
- `esp32-s3-m5/components/echo_gif/src/gif_player.cpp`
- `esp32-s3-m5/0014-atoms3r-animatedgif-single/main/hello_world_main.cpp`
- `esp32-s3-m5/0017-atoms3r-web-ui/main/display_app.cpp`
- `esp32-s3-m5/0017-atoms3r-web-ui/main/button_input.cpp`
- `esp32-s3-m5/0017-atoms3r-web-ui/main/hello_world_main.cpp`
- `esp32-s3-m5/0017-atoms3r-web-ui/main/wifi_app.cpp`
- `esp32-s3-m5/0017-atoms3r-web-ui/main/http_server.cpp`
- `esp32-s3-m5/0017-atoms3r-web-ui/main/Kconfig.projbuild`

Target Almanach firmware:

- `almanach/firmware/atoms3r/main/app_main.c`
- `almanach/firmware/atoms3r/main/provisioning_mgr.c`
- `almanach/firmware/atoms3r/main/provisioning_mgr.h`
- `almanach/firmware/atoms3r/main/provisioning_cmd.c`
- `almanach/firmware/atoms3r/main/wifi_mgr.c`
- `almanach/firmware/atoms3r/main/wifi_mgr.h`
- `almanach/firmware/atoms3r/main/CMakeLists.txt`
- `almanach/firmware/atoms3r/sdkconfig.defaults`

### What I found

- `0013-atoms3r-gif-console` has the richest GIF playback implementation. It uses M5GFX, a full-screen `M5Canvas`, FATFS storage under `/storage/gifs`, the `echo_gif` component, and AnimatedGIF callbacks that stream from files rather than loading GIFs into memory.
- `0013` also has a queue-based control-plane pattern: a GPIO ISR sends a small `CtrlEvent` to a FreeRTOS queue, and the main loop consumes events to change GIF playback state.
- `0017-atoms3r-web-ui` has a cleaner display wrapper for boot screens and PNG rendering. This is a better first shape for Almanach than letting GIF playback own the main application loop.
- Both `0013` and `0017` configure the AtomS3R button as GPIO41, active-low, with a 50 ms debounce default.
- The target Almanach firmware already exposes the provisioning and WiFi state needed for display status through `provisioning_mgr_get_status()`, `wifi_mgr_is_connected()`, and `wifi_mgr_get_ip()`.
- There is a critical GPIO conflict to resolve: donor display code uses GPIO7 as a backlight gate, while the Almanach printer firmware has used GPIO7 in printer UART pin mapping. The design guide calls this out explicitly and recommends disabling GPIO7 backlight gate control until the hardware mapping is confirmed.

### Why the guide recommends staged implementation

The requested feature crosses several subsystems: SPI display, backlight, C++ M5GFX code, button ISR/task handling, BLE provisioning state, WiFi/IP state, printer UART pins, optional FATFS GIF playback, and reset semantics. Implementing all of that at once would make failures hard to localize.

The guide recommends this order:

1. Port static display bring-up and boot/status text.
2. Add periodic provisioning/IP status rendering.
3. Add long-press button detection and non-destructive pairing start.
4. Add destructive reset behavior only after on-screen hold feedback works.
5. Add GIF playback as a background/screensaver feature after status and pairing are validated.

### What warrants a second pair of eyes

- Review the GPIO7 backlight/printer conflict before any code is copied into Almanach firmware.
- Review whether long press should immediately reset provisioned devices or use a two-stage hold threshold.
- Review whether PoP should be displayed on screen for development only or hidden in production.
- Review whether GIF playback should be included in the first implementation or deferred until the status display is stable.

### Output

Created design guide:

```text
almanach/ttmp/2026/05/10/ALMANACH-DISPLAY-PAIRING--atoms3r-display-status-and-long-press-ble-pairing-mode/design/01-atoms3r-display-status-and-long-press-ble-pairing-design-guide.md
```

## Step 2: Phase 0 and Phase 1 text-only display bring-up

This step moved from design into implementation. I resolved the first implementation constraints, added a managed M5GFX dependency to the standalone Almanach firmware, ported the AtomS3R display/backlight/display-app shape, and wired a boot screen into `app_main.c`.

### Prompt Context

**User prompt (verbatim):** "alright, add tasks to the ticket, then implement them one by one, ccomitting at at appropriate intervals, keeping  a diary (see skill)"

**Assistant interpretation:** Expand the display-pairing ticket tasks into concrete implementation phases, implement them incrementally, build/validate between phases, commit logical milestones, and keep this diary current.

**Inferred user intent:** Start the physical-display and long-press pairing implementation while preserving reviewability and hardware-validation discipline.

**Commit (code):** pending — this step will be committed after diary/changelog update.

### What I did

- Rewrote `tasks.md` into a phased checklist for constraints, text display, status display, long-press pairing, hardware validation, and optional GIF playback.
- Added `firmware/atoms3r/main/idf_component.yml` with `m5stack/M5GFX` as an ESP-IDF managed component dependency.
- Added `firmware/atoms3r/dependencies.lock` generated by ESP-IDF component manager.
- Added `managed_components/` to `firmware/atoms3r/.gitignore` so downloaded managed components are not committed.
- Added `firmware/atoms3r/main/Kconfig.projbuild` with Almanach-specific display/backlight/button symbols.
- Set the backlight GPIO gate default to disabled because donor firmware uses GPIO7 for the backlight gate, while the Almanach printer mapping has used GPIO7.
- Added text-only display files:
  - `backlight.cpp/.h`
  - `display_hal.cpp/.h`
  - `display_app.cpp/.h`
- Updated `firmware/atoms3r/main/CMakeLists.txt` for C++ display sources, SPI/I2C driver dependencies, and the M5GFX component.
- Updated `app_main.c` to initialize the display early and show `ALMANACH / AtomS3R / Booting...` on a best-effort basis.
- Built firmware successfully:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Why

- The first useful milestone is a static boot screen. It proves M5GFX dependency resolution, C++ compilation in the current C firmware component, AtomS3R display pin configuration, canvas allocation, and backlight control before adding provisioning/status logic.
- Disabling GPIO7 backlight gate by default avoids a known hardware conflict until the printer/display pin mapping is reviewed on the actual board.

### What worked

- ESP-IDF component manager successfully added and resolved `m5stack/M5GFX`.
- The firmware built with mixed C/C++ sources.
- The generated app still fits comfortably:
  - `stoms3r.bin binary size 0x17b950 bytes`
  - free app partition space: `0x2846b0 bytes (63%)`

### What didn't work

- No compile failures in this phase.
- The display has not yet been flashed and visually validated in this step; this was a build-only milestone.

### What I learned

- The standalone Almanach repo can use ESP-IDF's component manager for M5GFX instead of coupling back to donor paths under `esp32-s3-m5`.
- M5GFX adds a noticeable but acceptable binary-size increase relative to the BLE provisioning firmware.

### What was tricky to build

- The target firmware was C-only before this phase. The display code needs C++ for M5GFX, so headers must expose C-compatible functions and hide M5GFX types except inside C++ implementation files.
- The GPIO7 conflict changes the backlight strategy. The code supports the gate, but Kconfig defaults it off.

### What warrants a second pair of eyes

- Review whether `m5stack/M5GFX ^0.2.8` should remain an ESP-IDF managed dependency or be vendored for release reproducibility.
- Review the default backlight settings on real AtomS3R hardware. If the display stays dark with I2C brightness only, the printer/display GPIO7 conflict must be resolved before enabling the gate.

### What should be done in the future

- Flash to AtomS3R and visually confirm the boot screen.
- Add the periodic display status task that reads provisioning and WiFi state.
- Add long-press button behavior after status rendering is visible.

### Code review instructions

- Review `Kconfig.projbuild` for defaults and especially `ALMANACH_ATOMS3R_BACKLIGHT_GATE_ENABLE=n`.
- Review `display_hal.cpp` for AtomS3R GC9107 pin and offset configuration.
- Review `display_app.cpp` for text-only rendering and C-compatible API boundaries.
- Validate with:
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

## Step 3: Phase 2 polling display status model

This step added the first live status display path. The firmware now starts a `display_status` task when the display initializes successfully. The task polls the existing provisioning and WiFi managers once per second, converts their state into `display_status_t`, and renders the status screen.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the phased implementation after the text-only display bring-up and commit each validated milestone.

**Inferred user intent:** Make the built-in LCD show useful provisioning and IP information before adding button behavior.

**Commit (code):** pending — this step will be committed after diary/changelog update.

### What I did

- Added `display_status_task()` in `app_main.c`.
- The task polls `provisioning_mgr_get_status()` for:
  - provisioned state
  - BLE running state
  - BLE client-connected state
  - protocomm security state
  - service name
  - PoP
- The task polls `wifi_mgr_is_connected()` and `wifi_mgr_get_ip()` for WiFi/IP state.
- The task calls `display_app_show_status()` once per second.
- Built firmware successfully:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Why

- Polling is the simplest safe first implementation. It avoids rendering from ESP event callbacks and avoids adding a new event bus before the display behavior has been hardware-validated.
- The existing managers are already authoritative for provisioning and WiFi state, so the display layer should derive from them rather than storing duplicate state.

### What worked

- Firmware built successfully.
- Output image remains within partition limits:
  - `stoms3r.bin binary size 0x17bd10 bytes`
  - free app partition space: `0x2842f0 bytes (63%)`

### What didn't work

- No compile failures in this phase.
- The task has not yet been visually validated on the LCD; that remains in Phase 4 hardware validation.

### What I learned

- The existing `provisioning_mgr_get_status()` and `wifi_mgr_get_ip()` APIs are enough for a first useful status screen.
- A low-rate polling task keeps the display logic decoupled from provisioning and WiFi event callbacks.

### What was tricky to build

- The display task must start only if `display_app_is_ready()` returns true. Display initialization is best-effort so serial console and printer behavior remain available even if the LCD fails.

### What warrants a second pair of eyes

- Review whether the display should show `Web ready` only after `web_server_start()` succeeds, rather than assuming WiFi connected means web-ready. The current text is a reasonable first approximation but not a distinct web-server state.

### What should be done in the future

- Add event notifications for faster transitions after the polling path is verified.
- Add long-press button behavior.
- Flash and visually check that text fits on the 128x128 display.

### Code review instructions

- Review `display_status_task()` in `app_main.c`.
- Confirm the task reads state but does not mutate provisioning or WiFi behavior.
- Validate with:
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 build`
