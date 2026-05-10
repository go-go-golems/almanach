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

## Step 4: Phase 3 long-press pairing button

This step added physical button handling for pairing mode. The firmware now configures the AtomS3R button on GPIO41, queues GPIO edge events from an ISR, and runs the pairing/reset state machine in a FreeRTOS task.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue implementing the ticket phases after display status rendering.

**Inferred user intent:** Make the device enter pairing mode from the physical button, with safe reset behavior for already-provisioned devices.

**Commit (code):** pending — this step will be committed after diary/changelog update.

### What I did

- Added `firmware/atoms3r/main/button_input.h`.
- Added `firmware/atoms3r/main/button_input.c`.
- Added `button_input.c` to `main/CMakeLists.txt`.
- Included `button_input.h` from `app_main.c`.
- Started `button_input_start()` during boot after the display status task setup.
- Implemented GPIO41 active-low button handling with:
  - GPIO interrupt on any edge
  - ISR-to-queue delivery
  - task-context debounce and hold-duration logic
  - pairing hold threshold from `CONFIG_ALMANACH_ATOMS3R_PAIRING_HOLD_MS`
  - reset-confirm threshold from `CONFIG_ALMANACH_ATOMS3R_PAIRING_RESET_HOLD_MS`
- On pairing threshold, the task calls `provisioning_mgr_start_if_needed()` and shows pairing status.
- On reset-confirm threshold, the task follows the same semantics as `prov_reset`:
  - disconnect WiFi
  - erase explicit console WiFi credentials
  - reset ESP-IDF provisioning state
  - reboot
- Built firmware successfully:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Why

- Long-press detection belongs in task context, not interrupt context. The ISR should only enqueue minimal edge data.
- The physical button reset path should match serial `prov_reset` so the device has one clear definition of "return to pairing mode".

### What worked

- Firmware built successfully.
- Output image remains within partition limits:
  - `stoms3r.bin binary size 0x17cbf0 bytes`
  - free app partition space: `0x283410 bytes (63%)`

### What didn't work

- This has not yet been flashed and physically validated. Button edge polarity, display feedback, and reset behavior still need hardware testing.

### What I learned

- The Kconfig values from the design are sufficient for the first button implementation: GPIO41, active-low, 50 ms debounce, 2500 ms pairing hold, and 5000 ms reset hold.
- Keeping reset semantics in one helper inside `button_input.c` makes it easy to compare against `provisioning_cmd.c` during review.

### What was tricky to build

- The task must continue polling while the button is held, even if no new GPIO edge arrives. The implementation uses a 100 ms queue receive timeout so hold thresholds are detected before release.
- Calling `provisioning_mgr_start_if_needed()` on a provisioned device should be harmless; destructive reset is delayed until the longer reset-confirm threshold.

### What warrants a second pair of eyes

- Review whether the physical button should ever erase credentials automatically, or whether the first hardware implementation should stop at non-destructive pairing start.
- Review if `GPIO_INTR_ANYEDGE` behaves reliably on this AtomS3R button or if the task should poll level without relying on both edges.

### What should be done in the future

- Flash and test short press, pairing hold, release-to-cancel, and reset-confirm hold.
- If the LCD is too slow/noisy during hold progress updates, reduce update frequency or move hold-progress display into the display status task.

### Code review instructions

- Review `button_input.c` for ISR/task separation.
- Compare `reset_provisioning_and_reboot()` with `provisioning_cmd.c` `prov_reset` semantics.
- Validate with:
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

## Step 5: Phase 4 first hardware flash, I2C conflict fix, and monitor validation

This step flashed the display/button firmware to the physical AtomS3R and caught a runtime-only I2C driver conflict. The first flashed build entered a reboot loop before `app_main()` because the backlight helper used the legacy `driver/i2c.h` driver while the managed M5GFX component pulled in the newer I2C driver stack. ESP-IDF aborts when both old and new I2C drivers are linked.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Flash when needed and keep a detailed diary of what worked and what failed.

**Inferred user intent:** Validate the display and button implementation on real hardware instead of stopping at compile success.

**Commit (code):** pending — this step includes the I2C driver fix.

### What I did

- Flashed the firmware to `/dev/ttyACM0`:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 flash`
- Started monitor in tmux:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 monitor`
- Observed a boot-loop abort before `app_main()`:
  - `E i2c: CONFLICT! driver_ng is not allowed to be used with this old driver`
  - abort at `check_i2c_driver_conflict` in ESP-IDF's legacy I2C driver.
- Fixed `backlight.cpp` to use ESP-IDF's new I2C master API:
  - `driver/i2c_master.h`
  - `i2c_new_master_bus()`
  - `i2c_master_bus_add_device()`
  - `i2c_master_transmit()`
- Rebuilt and reflashed successfully.
- Re-ran monitor and confirmed the firmware boots through display initialization, BLE provisioning, and button setup.

### What worked after the fix

Monitor showed:

```text
I display_app: display boot; free_heap=8673128 dma_free=279180
I display_backlight: backlight i2c init: port=0 scl=0 sda=45 addr=0x30 reg=0x0e
I display_hal: m5gfx init: pclk=40000000Hz gap=(0,32)
I display_app: canvas ok: 32768 bytes
I stoms3r: No saved WiFi credentials — starting BLE provisioning
I wifi_prov_mgr: Provisioning started with service name : ALM_0F2320
I button_input: button init: gpio=41 active_low=1 debounce_ms=50 pair_hold_ms=2500 reset_hold_ms=5000
```

This validates that:

- The M5GFX component initializes without crashing.
- The AtomS3R display path creates the 128x128 canvas.
- BLE provisioning still starts.
- The GPIO41 button handler starts.
- Serial console still starts.

### What didn't work / what remains unvalidated

- I cannot confirm visual LCD readability from the serial transcript alone. The monitor proves display init and canvas allocation, but a human should still inspect the screen.
- I did not physically press and hold the button in this step. Long-press behavior is built and booted, but not physically validated.
- Because I did not erase flash in this step, the validation reflects the current no-saved-WiFi state, not a full erase/flash cycle.

### What I learned

- The donor comment about legacy I2C being safer is not valid with the managed M5GFX component selected here. The component manager version links the new I2C driver stack, so any legacy I2C use causes an early abort.
- The display path must use the new I2C master API or disable direct backlight I2C entirely.

### What was tricky to build

- The failure happens before normal application logs, during global constructor/driver conflict checking. It is invisible to compile-time validation and only appears after flashing.
- The fix needed to preserve the no-GPIO7-gate strategy while changing only the I2C backend.

### What warrants a second pair of eyes

- Review whether direct backlight I2C control is necessary if M5GFX can own the AtomS3R light device.
- Review whether the I2C bus conflict handling should try to attach to an existing bus or simply skip direct backlight control when the bus already exists.

### What should be done in the future

- Visually inspect the LCD for boot/status text orientation, contrast, and clipping.
- Physically test button hold thresholds.
- If the display is dark, resolve the GPIO7 backlight gate versus printer UART conflict before enabling gate control.

### Code review instructions

- Review `backlight.cpp` for new I2C driver usage.
- Validate on hardware with:
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 build && ./build.sh /dev/ttyACM0 flash`
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 monitor`


## Step 6: Full BLE WiFi provisioning and reboot autoconnect validation

This step ran the requested end-to-end hardware test with `idf.py flash monitor` in tmux and the Almanach Go/Glazed BLE provisioning command in a second tmux pane. The important result is that the firmware can be provisioned over BLE with real WiFi credentials, persists ESP-IDF provisioning state, reconnects after reboot, starts the web server on the provisioned boot path, and serves `/api/status`.

The test also exposed one timing gap: immediately after first-time provisioning, the app's startup path had already given up waiting for WiFi after 30 seconds, so the web server did not start until the device was rebooted. Reboot autoconnect is successful and starts the web server quickly, but the initial post-provisioning path needs either an event-driven web-server start or a longer/renewed wait.

### Prompt Context

**User prompt (verbatim):** "can you run a real test after flashing the firmware, use idf.py flash monitor and the go 

in a tmux.

(keep a diary, read skills)"

**Assistant interpretation:** Flash the current AtomS3R firmware, keep `idf.py flash monitor` running in tmux, run the Go BLE provisioning command from another tmux pane, and document the exact validation outcome.

**Inferred user intent:** Prove the firmware works on real hardware end-to-end, not only through builds or partial BLE protocol checks.

**Additional user credential prompt:** The SSID was `Verizon_9DNVB9`; the passphrase was intentionally redacted from this diary and logs.

### What I did

- Started `idf.py -D IDF_TARGET=esp32s3 -p /dev/ttyACM0 flash monitor` in tmux session `alm-real-test`.
- Confirmed firmware boot with display, BLE provisioning, button task, and console prompt.
- First attempted provisioning against the wrong SSID (`CoxWiFi`) because it was the host laptop's active SSID; BLE/protocomm worked but WiFi failed with AP-not-found.
- Ran `prov_reset` through the monitor console to clear the bad provisioning state and reboot.
- Re-ran provisioning through the Almanach Go/Glazed wrapper with SSID `Verizon_9DNVB9`, reading the passphrase from stdin so it was not passed as a command-line flag.
- Verified the Go command reported success.
- Verified firmware logs showed association, DHCP, successful provisioning, and provisioning manager teardown.
- Rebooted the device from `idf.py monitor` and verified persisted credentials autoconnected.
- Verified `/api/status` over HTTP after reboot:
  - `curl http://192.168.1.242/api/status`

### What worked

Go provisioning output, sanitized:

```text
==== Starting Session ====
==== Session Established ====
==== Sending Wi-Fi Credentials to Target ====
==== Wi-Fi Credentials sent successfully ====
==== Applying Wi-Fi Config to Target ====
==== Apply config sent successfully ====
==== Wi-Fi connection state  ====
++++ WiFi state: Connecting... ++++
==== Wi-Fi connection state  ====
==== WiFi state: Connected ====
==== Provisioning was successful ====
exit_code: 0
service_name: ALM_0F2320
ssid: Verizon_9DNVB9
```

Firmware monitor evidence:

```text
I provisioning: Received WiFi credentials for SSID 'Verizon_9DNVB9'
I wifi:connected with Verizon_9DNVB9, aid = 1, channel 11, 40D
I esp_netif_handlers: sta ip: 192.168.1.242, mask: 255.255.255.0, gw: 192.168.1.1
I wifi_mgr: Got IP: 192.168.1.242
I wifi_prov_mgr: STA Got IP
I provisioning: Provisioned WiFi credentials connected successfully
I wifi_prov_mgr: Provisioning stopped
I provisioning: BLE WiFi provisioning ended
I provisioning: WiFi provisioning manager deinitialized
```

Reboot/autoconnect evidence:

```text
I stoms3r: Provisioned WiFi found — starting station mode
I wifi:connected with Verizon_9DNVB9, aid = 1, channel 11, 40D
I wifi_mgr: Got IP: 192.168.1.242
I stoms3r: WiFi connected — starting web server
I web_server: HTTP server started on port 80
```

HTTP validation:

```json
{"ok":true,"wifi":{"connected":true,"ip":"192.168.1.242"},"printer":{"baud":9600,"swapped":true}}
```

### What didn't work

- The initial attempt used the wrong SSID (`CoxWiFi`) and failed as expected:

```text
Failure reason: Incorrect SSID
wifi_prov_mgr: STA AP Not found
provisioning: Provisioned WiFi connection failed: AP not found
```

- A second attempt with the right SSID failed before reset because the firmware was still in the previous bad connection retry loop:

```text
RuntimeError: Error in apply Wi-Fi config
```

- After a clean `prov_reset`, the real provisioning flow succeeded.
- The first successful provisioning connected after the initial `app_main` 30-second web-server wait had already expired:

```text
W stoms3r: WiFi not connected after 30s — web server not started
```

  The device was reachable by ping after provisioning, but HTTP port 80 was not open until reboot.

### What I learned

- BLE transport, Security 1, credential transfer, and ESP-IDF provisioning persistence all work on hardware.
- `prov_reset` is necessary and effective after a failed provisioning attempt if the station retry loop is still active.
- The web server startup should not depend only on a one-time boot wait. Provisioning can succeed after that wait, especially with BLE/protocomm overhead and AP association latency.

### What was tricky to build

- The Go command can exit successfully only after the ESP-IDF provisioning status reports connected, but the firmware's application-level web server is controlled by a separate startup wait. This creates a split-brain success state: provisioning/WiFi are successful, but HTTP is unavailable until reboot.
- The monitor contains repeated low-level WiFi BA add/delete noise, so the validation had to focus on high-signal lines: credential receipt, `STA Got IP`, `Provisioning stopped`, reboot autoconnect, and `HTTP server started`.

### What warrants a second pair of eyes

- Review `app_main.c` web-server startup semantics. It should probably start the web server when WiFi gets an IP, not only if WiFi is connected within a fixed startup timeout.
- Review whether failed provisioning attempts should automatically stop/restart provisioning more cleanly, or whether the documented recovery path should remain `prov_reset`.
- Review whether storing logs in `/tmp` is enough for validation artifacts or whether sanitized excerpts should be copied into ticket artifacts.

### What should be done in the future

- Fix the initial provisioning web-server startup gap with an event-driven or polling-on-connected path.
- Physically validate the GPIO41 long-press pairing and reset-confirm behavior.
- Keep using stdin or another non-argv secret path for provisioning passwords.

### Code review instructions

- Start with `firmware/atoms3r/main/app_main.c` and find the one-time WiFi wait before `web_server_start()`.
- Review `firmware/atoms3r/main/provisioning_mgr.c` event handling around `WIFI_PROV_CRED_SUCCESS`, `WIFI_PROV_END`, and manager deinit.
- Reproduce with:
  - `idf.py -D IDF_TARGET=esp32s3 -p /dev/ttyACM0 flash monitor`
  - `go run ./cmd/almanach-render-service ble-provision --action provision --service-name ALM_0F2320 --pop alm-0f2320 --ssid <ssid> --timeout 180 --output yaml` with passphrase from stdin.
  - Reboot from monitor and `curl http://<device-ip>/api/status`.

### Technical details

Artifacts during this run:

- Monitor log: `/tmp/almanach-idf-monitor.log`
- First wrong-SSID Go log: `/tmp/almanach-go-prov.log`
- Clean successful Go log: `/tmp/almanach-go-prov-verizon-clean.log`
- Tmux session: `alm-real-test`


## Step 7: Keep waiting for WiFi before starting HTTP server

This step fixed the post-provisioning HTTP startup gap found during real hardware validation. The firmware previously waited only 30 seconds after boot for WiFi, then deleted the web-server wait task. First-time BLE provisioning can legitimately take longer than that, so WiFi could connect successfully while port 80 stayed closed until a reboot.

The fix keeps the background task alive after the 30-second mark. It logs that it is still waiting, then starts the idempotent HTTP server as soon as `wifi_mgr_is_connected()` becomes true. A clean reset/provisioning retest confirmed `/api/status` is available immediately after first-time provisioning, without reboot.

### Prompt Context

**User prompt (verbatim):** "what's the display resolutoin on the atoms3r

Then, fix web server startup after provisioning"

**Assistant interpretation:** Answer the display resolution question first, then patch and validate firmware so HTTP starts after provisioning succeeds.

**Inferred user intent:** Remove the remaining reboot requirement from the BLE provisioning happy path.

**Follow-up prompt (verbatim):** "ok continue"

**Commit (code):** a183a90 — "Start web server after delayed provisioning"

### What I did

- Updated `firmware/atoms3r/main/app_main.c` so `web_server_task()` no longer exits after 30 seconds.
- Kept the warning log at 30 seconds, but changed it to:
  - `WiFi not connected after 30s — still waiting to start web server`
- Preserved the existing `web_server_start()` idempotency and added error logging if the HTTP server start fails.
- Built firmware with:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`
- Flashed and monitored with tmux:
  - `idf.py -D IDF_TARGET=esp32s3 -p /dev/ttyACM0 flash monitor`
- Ran `prov_reset`, provisioned again with the Go/Glazed `ble-provision` command, and tested HTTP without reboot.

### What worked

Build passed:

```text
stoms3r.bin binary size 0x17bf00 bytes. Smallest app partition is 0x400000 bytes. 0x284100 bytes (63%) free.
Project build complete.
```

The first-time provisioning path now logs the expected delayed wait and then starts HTTP after WiFi obtains an IP:

```text
W stoms3r: WiFi not connected after 30s — still waiting to start web server
I provisioning: Received WiFi credentials for SSID 'Verizon_9DNVB9'
I wifi_mgr: Got IP: 192.168.1.242
I wifi_prov_mgr: STA Got IP
I provisioning: Provisioned WiFi credentials connected successfully
I stoms3r: WiFi connected — starting web server
I web_server: HTTP server started on port 80
```

HTTP validation succeeded without reboot:

```json
{"ok":true,"wifi":{"connected":true,"ip":"192.168.1.242"},"printer":{"baud":9600,"swapped":true}}
```

### What didn't work

- No new firmware failure was observed during this fix.
- The tmux/log workflow still produces noisy WiFi BA add/delete logs, so validation excerpts should focus on the `Got IP`, web-server, and `/api/status` lines.

### What I learned

- The original 30-second wait was too strict for the first provisioning path because BLE provisioning and AP association can happen after `app_main()` has already moved on.
- A small polling task is sufficient for this firmware because `web_server_start()` is already idempotent and WiFi status is exposed through `wifi_mgr_is_connected()`.

### What was tricky to build

- The task needed to keep waiting only until the first successful HTTP startup. It should not start multiple HTTP server instances, and it should not spam warnings forever while the device is waiting for provisioning.
- The validation had to clear existing provisioned credentials first; otherwise reboot autoconnect would test the already-provisioned path rather than the fixed first-time provisioning path.

### What warrants a second pair of eyes

- Review whether a pure event-driven `IP_EVENT_STA_GOT_IP` hook should replace polling in a future cleanup.
- Review whether HTTP should stop on WiFi disconnect or remain running while disconnected. This patch preserves the previous behavior: start once, then leave the server running.

### What should be done in the future

- Physically validate GPIO41 long-press pairing/reset behavior.
- Consider reducing WiFi reconnect log noise if it makes field diagnostics hard.

### Code review instructions

- Start in `firmware/atoms3r/main/app_main.c`, function `web_server_task()`.
- Validate with a clean provisioning cycle:
  - flash/monitor firmware
  - `prov_reset`
  - run `ble-provision --action provision`
  - confirm `HTTP server started on port 80` before reboot
  - `curl http://<device-ip>/api/status`

### Technical details

Validation artifacts:

- Monitor log: `/tmp/almanach-webwait-monitor.log`
- Go provisioning log: `/tmp/almanach-webwait-go.log`
- Tmux session: `alm-webwait-test`


## Step 8: Button hold UX adjustment after physical press testing

This step interpreted the button logs from physical presses and adjusted the button UX. The presses were reaching the pairing threshold around 2.6 seconds, but because the device was already provisioned the old button path called `provisioning_mgr_start_if_needed()` and reported `Device already provisioned; BLE provisioning not started`. That made the 3-second action feel ambiguous: the hold was detected, but it did not actually open pairing for a configured device.

The reset threshold was also too close to the pairing threshold. The firmware now uses a 3-second pairing hold and a 10-second destructive reset hold. The pairing action now calls the force-start provisioning path so a provisioned device can explicitly re-enter BLE pairing/reprovisioning mode from the button instead of silently doing nothing.

### Prompt Context

**User prompt (verbatim):** "I tried to press, read the logs, a couple of time. I think 3 vs 5 is too small a difference, it should be 3 and 10. I don't know if the 3 seconds thing worked"

**Assistant interpretation:** Inspect monitor logs from the user's physical button presses, determine whether the pairing hold was detected, then change the UX to 3 seconds for pairing and 10 seconds for reset.

**Inferred user intent:** Make the physical button interaction less error-prone and make pairing visibly/actually happen when holding for the shorter threshold.

**Commit (code):** pending — button threshold and pairing behavior update.

### What I did

- Read `/tmp/almanach-webwait-monitor.log` and the active tmux monitor pane.
- Confirmed several physical button presses were captured.
- Confirmed the prior threshold was effectively reached around 2.6 seconds, but the provisioned-device path did not start BLE advertising.
- Updated `firmware/atoms3r/main/Kconfig.projbuild`:
  - reset hold default: `10000` ms
- Updated `firmware/atoms3r/sdkconfig.defaults`:
  - pairing hold: `3000` ms
  - reset hold: `10000` ms
- Updated `firmware/atoms3r/main/button_input.c` so the pairing hold calls `provisioning_mgr_start_force()` instead of `provisioning_mgr_start_if_needed()`.
- Built and flashed the firmware.
- Started a new monitor session and confirmed boot logs show:
  - `pair_hold_ms=3000 reset_hold_ms=10000`

### What worked

The user's prior physical presses were visible in the monitor:

```text
I button_input: button press started
I button_input: button pairing hold reached: 2592 ms
I provisioning: Device already provisioned; BLE provisioning not started
I button_input: button released after 3047 ms
```

This means the button GPIO and hold detector worked. The confusing part was the policy: because the device was already provisioned, the old non-force start path refused to start BLE provisioning.

After the change and flash, monitor showed the new thresholds:

```text
I button_input: button init: gpio=41 active_low=1 debounce_ms=50 pair_hold_ms=3000 reset_hold_ms=10000
```

### What didn't work

- The old 3-ish second pairing action did not visibly start pairing on an already provisioned device because it used `start_if_needed`.
- I have not yet observed a post-change physical 3-second hold from the user in the new monitor log, so the force-start behavior still needs one more physical press validation.

### What I learned

- The hardware button itself is working: short presses, releases, and long-hold timing are logged correctly.
- The UX issue was a combination of timing and semantics. A pairing button should enter pairing mode even when already provisioned; otherwise users cannot tell whether anything happened.

### What was tricky to build

- There are two separate concepts: `pairing` and `reset`. Pairing should be non-destructive and easy to trigger. Reset is destructive and needs a much larger hold gap.
- ESP-IDF provisioning state made the earlier button action look like a no-op on provisioned devices. Switching the button path to `provisioning_mgr_start_force()` better matches user expectations for a physical pairing button.

### What warrants a second pair of eyes

- Confirm `provisioning_mgr_start_force()` is the correct long-term API for provisioned-device pairing/reprovisioning mode while WiFi remains connected.
- Confirm whether the display should show a stronger countdown between 3s and 10s, e.g. `Hold to reset: 7...6...5`.

### What should be done in the future

- Ask the user to press and hold for about 3.5 seconds on the newly flashed firmware and verify BLE provisioning starts.
- Ask the user to only test the 10-second reset when ready to reprovision WiFi.
- Consider adding a short-press display/status refresh action later.

### Code review instructions

- Start with `firmware/atoms3r/main/button_input.c`, function `maybe_start_pairing()`.
- Then review `firmware/atoms3r/main/Kconfig.projbuild` and `firmware/atoms3r/sdkconfig.defaults` for threshold defaults.
- Validate with monitor:
  - 3.5-second hold should log `button pairing hold reached` and start/keep BLE provisioning.
  - 10-second hold should log reset and reboot.

### Technical details

Commands run:

```bash
cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build
cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 flash
cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 monitor
```


## Step 9: Confirm 3-second button pairing mode on hardware

This step confirmed the revised 3-second pairing behavior after the user reported the device was in pair mode. The monitor logs show a 3.383-second hold, the new 3-second threshold firing, and BLE provisioning advertising as `ALM_0F2320` with PoP `alm-0f2320`.

This closes the uncertainty from the previous physical test: the button GPIO and long-hold detector work, and the provisioned-device pairing path now actually starts BLE provisioning instead of returning `Device already provisioned; BLE provisioning not started`.

### Prompt Context

**User prompt (verbatim):** "it's in pair mode now"

**Assistant interpretation:** Read the live monitor logs and confirm whether the updated button behavior put the device into BLE pairing mode.

**Inferred user intent:** Verify that the newly flashed 3-second button behavior is working on real hardware.

**Commit (code):** 56d57b7 — "Tune AtomS3R button pairing and reset holds"

### What I did

- Read the active tmux monitor session `alm-button-test`.
- Filtered `/tmp/almanach-button-monitor.log` for button, BLE, protocomm, and provisioning lines.
- Confirmed the 3-second hold crossed the threshold and started BLE advertising.
- Marked long-press pairing validation complete in the display pairing task list.

### What worked

Monitor evidence:

```text
I button_input: button init: gpio=41 active_low=1 debounce_ms=50 pair_hold_ms=3000 reset_hold_ms=10000
I button_input: button press started
I button_input: button pairing hold reached: 3093 ms
I BLE_INIT: Bluetooth MAC: 98:88:e0:0f:23:22
I protocomm_nimble: BLE Host Task Started
I NimBLE: GAP procedure initiated: advertise;
I wifi_prov_mgr: Provisioning started with service name : ALM_0F2320
I provisioning: BLE WiFi provisioning started
I provisioning:   Device    : ALM_0F2320
I provisioning:   PoP       : alm-0f2320
I button_input: button released after 3383 ms
```

This confirms the revised 3-second pairing action works on a provisioned device.

### What didn't work

- No failure was observed in the 3-second pairing validation.
- Reset-confirm remains intentionally untested because it is destructive and should only be tested when ready to reprovision.

### What I learned

- `provisioning_mgr_start_force()` is the right behavior for the physical pairing button: users expect a provisioned device to become discoverable again.
- A 3-second hold is reachable and logged clearly; the 10-second reset gap should reduce accidental credential wipes.

### What was tricky to build

- Entering provisioning mode while already connected to WiFi causes WiFi to disconnect as ESP-IDF provisioning takes over station/provisioning flow. That is expected but worth documenting because pairing mode is not just a passive advertisement.

### What warrants a second pair of eyes

- Confirm whether pairing mode should disconnect current WiFi immediately, or whether the long-term UX should present a confirmation/countdown before entering provisioning.
- Confirm whether the display should explicitly show `Pairing: ALM_0F2320` while advertising.

### What should be done in the future

- Test the 10-second reset-confirm behavior when ready to reprovision.
- Consider adding a timeout to forced pairing mode if no client connects.

### Code review instructions

- Review `button_input.c` around `maybe_start_pairing()` and the hold-threshold loop.
- Validate with a 3.5-second press and monitor for `Provisioning started with service name`.

### Technical details

Relevant runtime log file:

- `/tmp/almanach-button-monitor.log`


## Step 10: Improve long-hold reset countdown visibility

This step investigated the user's report that holding the button for a long time did not visibly change the screen much. The logs show the button reached the 3-second pairing threshold again while BLE provisioning was already running, but there was no 10-second reset log and no reboot. The existing screen feedback was also too subtle: the hold screen still said `PAIRING` and printed small millisecond text, which is hard to distinguish from the normal pairing screen on a 128x128 display.

The firmware now renders a much clearer hold screen after pairing starts: `PAIR ON`, `Release OK`, and a larger `RESET Ns` countdown that turns red near the reset threshold. The button task also logs hold progress once per second after the pairing threshold so future hardware tests can show whether the hold is still being sampled and how long remains before reset.

### Prompt Context

**User prompt (verbatim):** "i did for a long time but i din't see much change on the screen."

**Assistant interpretation:** Inspect the button logs after a longer hold and improve the on-device reset countdown feedback because the screen state was not obvious.

**Inferred user intent:** Make the button UX understandable without needing serial logs, especially while distinguishing safe pairing from destructive reset.

**Commit (code):** pending — display countdown and hold-progress logging update.

### What I did

- Read `/tmp/almanach-button-monitor.log` and the active `alm-button-test` monitor pane.
- Confirmed a later hold reached the pairing threshold while provisioning was already running:
  - `button pairing hold reached: 3094 ms`
  - `provisioning: BLE provisioning already running`
- Did not see a 10-second reset log or reboot after that hold.
- Updated `display_app_show_pairing_hold()` to use clearer large text:
  - before pairing: `PAIR in Ns`
  - after pairing: `PAIR ON`, `Release OK`, `RESET Ns`, `keep hold`
  - countdown turns red for the final 3 seconds.
- Updated `button_input.c` to log hold progress once per second after the pairing threshold:
  - `button held: <held_ms> ms (reset in <remaining_ms> ms)`
- Built firmware successfully.

### What worked

Build passed after the feedback changes:

```text
stoms3r.bin binary size 0x17c0c0 bytes. Smallest app partition is 0x400000 bytes. 0x283f40 bytes (63%) free.
Project build complete.
```

The prior log evidence explains why the screen looked ambiguous:

```text
I button_input: button pairing hold reached: 3094 ms
I provisioning: BLE provisioning already running
```

That was a valid pairing state, but not a visible reset-progress state.

### What didn't work

- The 10-second reset path is still not validated. I did not see `button reset hold reached; clearing WiFi/provisioning state` or a reboot in the latest log excerpt.
- The improved display feedback has been built but still needs flashing and another physical hold test.

### What I learned

- The display copy needs to distinguish safe pairing from destructive reset. `PAIRING` plus millisecond text is not enough on a tiny 128x128 screen.
- Serial logs should include countdown progress for long holds so we can tell the difference between a user releasing early, a sampling problem, or a display-only visibility problem.

### What was tricky to build

- The same button hold has two phases: non-destructive pairing after 3 seconds and destructive reset at 10 seconds. The screen must tell the user that releasing after 3 seconds is safe, but continuing to hold will reset credentials.
- The display status task may redraw normal status once per second, so the hold screen needs to be visually bold enough to notice while the button task refreshes it.

### What warrants a second pair of eyes

- Review whether display writes from the status task and button task should be serialized with a mutex or routed through one UI task.
- Review whether destructive reset should require releasing after 10 seconds instead of firing immediately while held.

### What should be done in the future

- Flash this improved feedback firmware.
- Hold the button through 10 seconds and watch for both screen countdown and serial progress logs.
- If reset still does not fire, inspect button level sampling while held and display/task concurrency.

### Code review instructions

- Start with `firmware/atoms3r/main/display_app.cpp`, `display_app_show_pairing_hold()`.
- Then review `firmware/atoms3r/main/button_input.c` for once-per-second hold progress logging.
- Validate by flashing and holding the button for 11 seconds.

### Technical details

Relevant files:

- `/tmp/almanach-button-monitor.log`
- `firmware/atoms3r/main/display_app.cpp`
- `firmware/atoms3r/main/button_input.c`


## Step 11: Make reset reachable from the already-pairing screen

This step addressed a UX/state-model problem found in the live button logs. The user held the button while the device was already in BLE pairing mode and expected to be able to continue into reset from that screen. The logs showed the button task reaching the 3-second threshold again and reporting `BLE provisioning already running`, but no reset transition was logged. The implementation was not a formal state machine; it used per-press booleans such as `pairing_announced` and `reset_triggered`, which made the already-pairing case too implicit.

The firmware now treats a press that begins while provisioning is already running as a reset-countdown press from the start. It shows a dedicated reset hold screen immediately, logs once-per-second progress from the beginning of that hold, and still triggers the same 10-second reset path. I also added a display mutex so the status task and button task do not draw to the shared M5Canvas concurrently.

### Prompt Context

**User prompt (verbatim):** "it should be possible to get there from within the pairing screen too. Are we using a state machine? Look at my logs..."

**Assistant interpretation:** Inspect the logs for the long-hold behavior while already in pairing mode, explain whether the button code is currently a state machine, and make reset reachable from the pairing screen.

**Inferred user intent:** The physical UX should be: once the device is on the pairing screen, continuing/starting a long hold should visibly progress toward reset rather than re-triggering pairing or appearing stuck.

**Commit (code):** pending — pairing-screen reset countdown fix.

### What I did

- Read `/tmp/almanach-button-monitor.log` and `/tmp/almanach-button-monitor2.log`.
- Confirmed old logs showed repeated pairing-threshold events without reset:
  - `button pairing hold reached: 3094 ms`
  - `provisioning: BLE provisioning already running`
- Confirmed newer countdown logs progressed to 5 seconds, but did not show reset or release in the captured excerpt:
  - `button held: 3094 ms (reset in 6906 ms)`
  - `button held: 4044 ms (reset in 5956 ms)`
  - `button held: 5044 ms (reset in 4956 ms)`
- Updated `button_input.c` so when a press starts while `provisioning_mgr_get_status()` reports `running=true`, the task sets a `reset_countdown_only` state and shows reset countdown immediately.
- Added `display_app_show_reset_hold()` for an explicit reset screen.
- Added a FreeRTOS mutex around display drawing in `display_app.cpp` to serialize display writes from the status task and button task.
- Built firmware successfully.

### What worked

The build passed:

```text
stoms3r.bin binary size 0x17c3a0 bytes. Smallest app partition is 0x400000 bytes. 0x283c60 bytes (63%) free.
Project build complete.
```

The code now has an explicit state for the already-pairing press path:

```c
if (provisioning_mgr_get_status(&st) == ESP_OK && st.running) {
    pairing_announced = true;
    reset_countdown_only = true;
    ESP_LOGI(TAG, "button press started while pairing is already active; reset countdown armed");
    display_app_show_reset_hold(0, CONFIG_ALMANACH_ATOMS3R_PAIRING_RESET_HOLD_MS);
}
```

### What didn't work

- This is built but not yet flashed/tested on hardware in this step.
- The code is still not a full table-driven state machine; it is a small explicit per-press state model with booleans. That may be sufficient, but a real enum-based state machine would be clearer if button/display behavior grows further.

### What I learned

- The logs showed the problem clearly: the button task was re-entering the pairing threshold while the device was already in pairing mode. It needed to switch into reset countdown semantics instead.
- The display path had another risk: both the display status task and button task can draw. That can make screen feedback appear to flicker or fail to update. A mutex is the first step toward making the UI deterministic.

### What was tricky to build

- Reset has to be reachable from two entry points: a fresh normal screen hold and an already-pairing screen hold. Those two paths should share the same destructive reset function but use different display copy.
- The already-pairing case should not wait another 3 seconds before showing reset progress. It should immediately tell the user that reset countdown is armed.

### What warrants a second pair of eyes

- Review whether the button model should be refactored into an explicit enum state machine, for example `IDLE`, `HOLD_FOR_PAIR`, `PAIRING_ACTIVE`, `HOLD_FOR_RESET`, `RESETTING`.
- Review whether the display mutex should become a proper single UI task/message queue instead of locking around M5Canvas calls.

### What should be done in the future

- Flash this build and validate: from the pairing screen, press and hold for 10 seconds and verify reset logs/reboot.
- If the behavior remains confusing, move to a formal state machine and require release-after-countdown confirmation before destructive reset.

### Code review instructions

- Start in `firmware/atoms3r/main/button_input.c`, especially `reset_countdown_only` and the press-start branch.
- Then review `firmware/atoms3r/main/display_app.cpp` for display mutex and `display_app_show_reset_hold()`.
- Validate on hardware by entering pairing mode, then pressing/holding again until reset.

### Technical details

Relevant observed logs:

```text
I button_input: button pairing hold reached: 3094 ms
I provisioning: BLE provisioning already running
I button_input: button held: 3094 ms (reset in 6906 ms)
I button_input: button held: 4044 ms (reset in 5956 ms)
I button_input: button held: 5044 ms (reset in 4956 ms)
```
