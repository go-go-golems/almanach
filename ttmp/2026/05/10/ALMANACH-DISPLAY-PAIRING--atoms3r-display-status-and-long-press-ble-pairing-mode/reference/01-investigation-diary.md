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
