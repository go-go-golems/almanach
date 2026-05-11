---
Title: Investigation Diary
Ticket: ALMANACH-PRINTER-UART
Status: active
Topics:
    - firmware
    - printer
    - uart
    - hardware-diagnostics
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/main/printer_drv.c
      Note: Old firmware comparison baseline
    - Path: firmware/atoms3r/main/app_main.c
      Note: Saved settings boot application evidence
    - Path: firmware/atoms3r/main/printer_cmd.c
      Note: Commands used during hardware validation
    - Path: firmware/atoms3r/main/printer_drv.c
      Note: Current UART driver validated at 460800
    - Path: internal/web/embed/public/almanach-bundle.js
      Note: Regenerated embedded web bundle
    - Path: web/dist/almanach-bundle.js
      Note: Regenerated web dist bundle
    - Path: web/src/almanach-studio.jsx
      Note: Light thermal image tone implementation
ExternalSources: []
Summary: Chronological investigation diary for the AtomS3R/K118 UART serial interface comparison between old working firmware and copied Almanach firmware.
LastUpdated: 2026-05-10T22:04:00-04:00
WhatFor: Use this diary to resume UART serial debugging, understand what was tested, and avoid mistaking TX/RX echo for printer protocol replies.
WhenToUse: Use when comparing old stoms3r firmware, sdkconfig, ESP-IDF versions, printer UART pins, CTS flow control, or serial probe behavior.
---



# Diary

## Goal

Capture the investigation into the AtomS3R Lite to K118 printer UART serial interface, especially the difference between the old `esp32-s3-m5/stoms3r` firmware that appears to communicate and the copied Almanach firmware that currently times out on status probes.

## Step 1: Create a dedicated UART ticket and verify the old flashed firmware behavior

The user clarified that this work is specifically about the serial interface and asked for a new ticket with a detailed intern-oriented analysis/design/implementation guide, plus a detailed diary. I created `ALMANACH-PRINTER-UART` so this UART investigation is separate from the broader setup/provisioning and print-pipeline tickets.

I then attached a monitor from the old firmware directory that the user identified as currently flashed/working. The old firmware does report `PRINTER RESPONDED` from `printer_probe`, but the raw byte evidence is suspicious: arbitrary raw bytes are echoed back exactly, which is more consistent with TX/RX echo or loopback than a real K118 protocol response.

### Prompt Context

**User prompt (verbatim):**

```text
NHo this is about the serial interface. In fact, create a new ticket for that, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


The firmware in the other esp32-s3-m5 works in that it can communicate iwht the printer (it's currently flashed, if you want to verify). 

As you investigate, keep a detailed diary
```

**Assistant interpretation:** Start a new docmgr ticket focused on the printer UART serial interface, investigate the old working firmware versus the copied Almanach firmware, document the system deeply for a new intern, keep a diary, and upload the final guide to reMarkable.

**Inferred user intent:** Build enough technical understanding and evidence to reliably fix the copied firmware's serial communication with the K118 printer instead of guessing from pinout or provisioning changes.

**Commit (code):** N/A — investigation and documentation only.

### What I did
- Created docmgr ticket `ALMANACH-PRINTER-UART`.
- Created this diary:
  - `almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md`
- Created the design guide placeholder:
  - `almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/design-doc/01-atoms3r-k118-printer-uart-serial-interface-analysis-design-and-implementation-guide.md`
- Restarted the serial monitor from the old firmware directory:
  - `cd /home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r && ./build.sh /dev/ttyACM0 monitor`
- Confirmed the booted firmware identifies as:
  - project `stoms3r`
  - app version `b1e12e3-dirty`
  - ESP-IDF `v5.4.1`
- Ran:
  - `printer_probe`
  - `printer_status`
  - `printer_get_baud`
  - `printer_raw A55A`
  - `printer_raw 01020304`

### Why
- The old firmware is the baseline the user trusts for communication, so it must be treated as primary evidence.
- Running direct raw probes helps distinguish real printer replies from local echo/loopback.
- The design guide needs real observed evidence, not only source comparisons.

### What worked
- The old firmware monitor started and showed the expected old boot sequence.
- `printer_probe` reported `=== Result: PRINTER RESPONDED ===`.
- The old firmware received bytes on UART RX after UART TX.
- The old boot uses the same apparent printer mapping:
  - nominal `TX=8 RX=7 CTS=6`
  - effective swapped pins `TX=GPIO7 RX=GPIO8 CTS=GPIO6`

### What didn't work
- `printer_status` did not return a valid 4-byte K118 status packet. It logged:
  - `TX 3 bytes: 1D 61 00`
  - `GS a status: got 3 bytes`
  - `Error: ESP_ERR_TIMEOUT`
- `printer_get_baud` returned:
  - `esp32_baud=9600 printer_baud=7 raw=g7`
  This looks like the command suffix `67 37` (`g7`) being echoed, not a structured baud-rate response.
- `printer_raw A55A` received exactly `A5 5A`.
- `printer_raw 01020304` received exactly `01 02 03 04`.
- Those raw results strongly suggest the old firmware's "communication" may be TX/RX echo rather than a real printer response.

### What I learned
- The old firmware communicates with *something* on the UART RX line, but the data matches the transmitted bytes too exactly to trust as printer protocol replies.
- The existing `printer_probe` can produce a false positive if the RX path echoes the transmitted command. It treats the first byte of `10 04 n` as a status response (`0x10`) and later drains the remaining echoed bytes (`04 n`).
- The old firmware is still valuable because it exposes a behavioral difference: old flashed firmware sees echo/readback, while the copied Almanach firmware recently saw no bytes at all for equivalent probes.

### What was tricky to build
- The user-visible term "communication works" can mean either "the printer sent a meaningful protocol reply" or merely "UART RX received bytes after TX." The evidence shows the old firmware satisfies the second definition, but not yet the first.
- K118 status commands are not self-framing in this driver. A single echoed TX byte can be mistaken for the expected one-byte status result unless the probe explicitly rejects echoes.
- `printer_get_baud` parsing is particularly misleading because the raw echo contains ASCII-ish bytes (`g7`) that are easy to mistake for a printer response string.

### What warrants a second pair of eyes
- Confirm with physical observation whether the printer paper moves or prints during old-firmware `printer_text` / `printer_feed` commands.
- Confirm with a logic analyzer whether GPIO7 TX activity appears on the K118 RX pin and whether GPIO8 receives any non-echo printer-originated data.
- Review whether the K118 board electrically echoes TX onto RX under some power/cable states, e.g. unpowered carrier, shared connector, or loopback through the module.

### What should be done in the future
- Update `printer_probe` so echoed commands are classified as `UART_LOOPBACK_OR_ECHO`, not `PRINTER RESPONDED`.
- Add raw diagnostics that report whether RX bytes equal the just-transmitted TX bytes.
- Compare current copied firmware against old firmware under the same ESP-IDF version and with display/BLE disabled to isolate integration effects.

### Code review instructions
- Start with old firmware driver:
  - `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/main/printer_drv.c`
- Then current copied driver:
  - `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/printer_drv.c`
- Pay special attention to:
  - `printer_drv_query_status`
  - `printer_drv_drain_rx`
  - `printer_drv_send_raw`
  - `printer_drv_swap_pins`
- Validate by running raw echo probes and checking whether arbitrary bytes echo back.

### Technical details
- Old observed boot:
  - `ESP-IDF: v5.4.1`
  - `Printer UART1 ready: TX=8 RX=7 CTS=6 baud=9600`
  - `Swapping TX/RX pins: TX=GPIO7 RX=GPIO8 CTS=GPIO6 (SWAPPED) baud=9600`
- Old observed echo evidence:
  - `printer_raw A55A` -> RX `A5 5A`
  - `printer_raw 01020304` -> RX `01 02 03 04`
  - `printer_get_baud` -> raw `g7`, matching command bytes `1D 67 37` suffix.

## Step 2: Compare sdkconfig and source-level differences relevant to UART

I compared the old working firmware tree and the current copied Almanach firmware tree. The top-level sdkconfig defaults are similar for the serial console and printer-relevant baseline: both use USB Serial/JTAG console so UART1 is free for the printer, 8 MB flash, PSRAM, a custom partition table, and 240 MHz CPU.

The main differences are not ordinary UART console settings. The current Almanach firmware adds BLE/NimBLE provisioning, display/backlight support, button handling, and it is being built with ESP-IDF 5.4.2, while the old currently-flashed baseline was built with ESP-IDF 5.4.1.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Compare old and copied firmware configuration and code to find serial-interface-relevant differences.

**Inferred user intent:** Identify whether sdkconfig, ESP-IDF version, or integration changes explain why copied firmware sees no printer UART bytes while the old firmware sees echo/readback.

**Commit (code):** N/A — investigation and documentation only.

### What I did
- Compared old and current `sdkconfig.defaults`.
- Compared filtered `sdkconfig` symbols containing UART, console, USB, GPIO, PSRAM, WiFi, BT, and partition terms.
- Compared old and current `build.sh`.
- Compared old and current `printer_drv.c` / `printer_drv.h`.
- Compared old and current `app_main.c` for init-order and integration differences.
- Inspected current display/backlight code for GPIO7 conflicts.

### Why
- User specifically asked whether this could be `sdkconfig` related.
- Serial interface failures often come from console stealing a UART, pin matrix changes, flow control settings, or another peripheral reconfiguring the same GPIOs.

### What worked
- The old and current `sdkconfig.defaults` both set:
  - `CONFIG_ESP_CONSOLE_USB_SERIAL_JTAG=y`
  - `CONFIG_SPIRAM=y`
  - `CONFIG_SPIRAM_MODE_OCT=y`
  - custom partition table
  - 240 MHz CPU
- The old firmware build script sources:
  - `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/.envrc`
  - which sources `~/esp/esp-idf-5.4.1/export.sh`
- The current firmware build script prefers:
  - `~/esp/esp-idf-5.4.2/export.sh`
- Current display/backlight code intentionally avoids GPIO7 backlight gate by default:
  - `# CONFIG_ALMANACH_ATOMS3R_BACKLIGHT_GATE_ENABLE is not set`
  - backlight I2C uses SCL GPIO0 and SDA GPIO45.

### What didn't work
- No smoking-gun `sdkconfig` option directly reassigns UART1 pins; the printer pins are hard-coded in `printer_drv.h` and applied by `uart_set_pin()`.
- The current diagnostic changes add `printer_flow`, but the previous no-response behavior was seen even when effective pins matched old firmware.

### What I learned
- The likely sdkconfig/configuration suspects are indirect:
  - ESP-IDF 5.4.1 vs 5.4.2 UART driver behavior or pin-matrix behavior.
  - current firmware enables BLE/NimBLE, which changes memory/task/coexistence behavior but should not directly steal UART1 pins.
  - current firmware initializes display before printer, but display pins are SPI pins 14/15/21/42/48 and backlight I2C pins 0/45, not GPIO7/8/6.
- The old firmware's apparent communication is likely echo. The current firmware's no-response may mean it does not see the same echo under ESP-IDF 5.4.2 or after current integration init order.

### What was tricky to build
- `sdkconfig` contains Bluetooth HCI UART symbols such as `CONFIG_BT_NIMBLE_HCI_UART_CTS_PIN=23`, but current Bluetooth controller mode is VHCI, not UART H4. Those symbols are easy to misread as stealing UART pins, but they are not the active host-controller transport in this build.
- Display folklore around AtomS3R GPIO7 backlight conflicts with the current code: current code disables the GPIO7 backlight gate by default specifically to avoid conflicting with printer UART.

### What warrants a second pair of eyes
- Verify with `idf.py size-components` or map files whether ESP-IDF 5.4.2 pulls in different UART driver code paths that could affect RX echo/loopback behavior.
- Build the current copied firmware under ESP-IDF 5.4.1 as an isolation experiment.
- Build old firmware under ESP-IDF 5.4.2 as the reciprocal isolation experiment if feasible.

### What should be done in the future
- Add a reproducible matrix:
  - old source + IDF 5.4.1
  - old source + IDF 5.4.2
  - current source + IDF 5.4.1
  - current source + IDF 5.4.2
- For each, run:
  - `printer_raw A55A`
  - `printer_probe`
  - `printer_status`
  - `printer_text HELLO`
  - `printer_feed 3`
- Record whether each test shows echo, real structured status, and/or physical paper movement.

### Code review instructions
- Compare:
  - `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/sdkconfig.defaults`
  - `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/sdkconfig.defaults`
  - `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/build.sh`
  - `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/build.sh`
  - both `main/app_main.c` files.

### Technical details
- Current integration adds:
  - BLE provisioning: `CONFIG_BT_ENABLED=y`, `CONFIG_BT_NIMBLE_ENABLED=y`.
  - display startup before printer init.
  - button handler.
  - provisioning manager and WiFi onboarding changes.
- Current display pins:
  - LCD CS 14, SCK 15, MOSI 21, DC 42, RST 48.
  - Backlight I2C SCL 0, SDA 45.
  - GPIO7 gate disabled by default.

## Step 3: Incorporate printer settings page baud rate and test ESP32 at 460800

The user reported that the printer's own settings page says its baud rate is `460800`. That is a major clue because both old and current firmware boot with the ESP32 UART at `9600` unless a saved startup baud is applied. If the K118 is truly configured to `460800`, then text/feed commands sent at `9600` would not be understood by the printer.

I changed only the ESP32 UART side to `460800` in the old currently-flashed firmware and reran the probes. The result still looked like exact TX echo rather than a real K118 response, but this test was still useful: it shows that the old firmware's echo-like receive behavior survives the baud change and therefore cannot be treated as proof that the printer protocol is working.

### Prompt Context

**User prompt (verbatim):** "baudrate 460800 when I print out the settings page on the printer"

**Assistant interpretation:** The physical printer reports that its serial baud rate is 460800, so the firmware's default 9600 baud is likely wrong for actual printer commands.

**Inferred user intent:** Use the printer's self-reported settings to explain why text/feed do not work and adjust the UART investigation around the actual configured printer baud.

**Commit (code):** N/A — investigation and documentation only.

### What I did
- On the old currently-flashed firmware, ran:
  - `printer_baud 460800`
  - `printer_probe`
  - `printer_status`
  - `printer_get_baud`
  - `printer_text BAUD_460800_TEST`
  - `printer_feed 3`
- Captured the serial monitor output.

### Why
- `printer_baud 460800` changes only the ESP32 UART side and is the correct recovery/test command when the printer-side baud is already known to be 460800.
- This avoids sending `set_baudrate`, which would command the printer to change settings. The user already has the printer-side setting from the printed settings page.

### What worked
- The ESP32 UART side changed successfully:
  - `Changing ESP32 UART baud rate: 9600 -> 460800`
  - `ESP32 UART baud rate set to 460800 (printer was NOT commanded)`
- The old firmware continued to receive bytes after sending commands at 460800.
- The command sequence executed without crashing the monitor or firmware.

### What didn't work
- The RX pattern at 460800 still matched echo-like behavior:
  - `DLE EOT` commands still reported first byte `0x10`, then drained the remaining `04 n` bytes.
  - `printer_status` still got exactly 3 bytes for the 3-byte command and timed out waiting for a 4-byte status packet.
  - `printer_get_baud` still returned `raw=g7`, matching the suffix of the transmitted command `1D 67 37`.
- I still cannot see whether `printer_text BAUD_460800_TEST` or `printer_feed 3` physically printed/fed paper from the terminal.

### What I learned
- The printer settings page baud of `460800` is likely why text/feed at the firmware's default 9600 do nothing on the physical printer.
- However, simply seeing bytes on RX at either 9600 or 460800 is not enough to prove real printer communication, because the observed bytes still match transmitted bytes.
- If paper moved after the 460800 test, then TX at 460800 is likely correct and the remaining issue is RX/status semantics. If paper still did not move, the echo is probably unrelated to actual printer input.

### What was tricky to build
- There are two baud states:
  - ESP32 UART baud (`printer_baud` changes this only).
  - Printer's persisted baud (`set_baudrate` changes printer then ESP32).
- The old firmware boot log always starts at 9600 unless saved settings are present, so a printer persisted at 460800 will be out of sync immediately after boot.
- Echo/loopback can happen at both baud rates, so baud alignment alone does not disambiguate real K118 replies.

### What warrants a second pair of eyes
- Confirm physically whether the printer reacted to `printer_text BAUD_460800_TEST` or `printer_feed 3` after `printer_baud 460800`.
- Confirm whether the K118 settings page's `460800` refers to the TTL UART baud used by this connector and not another interface/profile.
- Review whether saved printer settings should default the ESP32 side to 460800 in the Almanach firmware when this particular printer has been configured that way.

### What should be done in the future
- If 460800 physically works, run `printer_settings_save 460800 <density> <speed> <mode>` or equivalent once the desired density/speed/mode are known, so firmware boots at the printer's persisted baud.
- Add documentation warning that `printer_baud` is recovery-only and must match the printer's persisted baud after power cycling.
- Add an echo-aware probe before trusting `printer_get_baud` or `printer_status` outputs.

### Code review instructions
- Review old/current implementations of:
  - `do_printer_baud`
  - `printer_drv_set_baud`
  - `do_set_baudrate`
  - `printer_drv_set_printer_baudrate`
  - `printer_settings_save` / `apply_saved_printer_settings`
- Confirm boot-time saved settings apply the ESP32 UART baud before any printer commands that depend on the printer's persisted baud.

### Technical details
- Command used:
  - `printer_baud 460800`
- Important output:
  - `Warning: rates above 115200 are experimental on K118.`
  - `ESP32 UART baud rate set to 460800 (printer was NOT commanded)`
- Follow-up output still echo-like:
  - `printer_get_baud` -> `esp32_baud=460800 printer_baud=7 raw=g7`


## Step 4: Validate copied Almanach firmware at 460800 and persist startup settings

After confirming the old firmware worked physically once the ESP32 UART was switched to 460800, I flashed the copied Almanach firmware again and repeated the same test. This time the new firmware also communicated correctly with the printer once its ESP32 UART baud matched the printer settings page.

The decisive fix was not changing the pinout or ESP-IDF SDK configuration. The printer was persisted at 460800 baud, while both firmware trees booted at 9600 without saved startup settings. After switching the new firmware to 460800, the K118 returned real status/baud responses, text/feed worked, and saved startup settings made the baud survive reboot.

### Prompt Context

**User prompt (verbatim):** "ok, continue, keep a printer"

**Assistant interpretation:** Continue the printer UART investigation on the copied/new firmware, keep tracking the work in the diary, and validate the 460800 baud finding.

**Inferred user intent:** Confirm whether the new Almanach firmware can drive the printer the same way as the old firmware once configured correctly, then make the working state persistent.

**Commit (code):** N/A — hardware validation and NVS configuration only.

### What I did
- Flashed the copied/new Almanach firmware from:
  - `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r`
- Confirmed boot identity:
  - `App version: d3b5d2e-dirty`
  - `ESP-IDF: v5.4.2`
- Ran the successful manual sequence:
  - `printer_baud 460800`
  - `printer_flow off`
  - `printer_probe`
  - `printer_text NEW_FW_460800_NOCTS`
  - `printer_feed 3`
  - `printer_status`
  - `printer_get_baud`
  - `printer_flow cts`
  - `printer_probe`
- User confirmed the physical printer output worked.
- Saved startup settings in NVS:
  - `printer_settings_save 460800 20 80 31`
- Restarted the copied/new firmware monitor and confirmed boot applied saved settings:
  - `Applying saved printer settings: baud=460800 density=20 speed=80 graphics_mode=31`
- Revalidated after restart:
  - `printer_probe`
  - `printer_get_baud`
  - `printer_status`

### Why
- The printer settings page reported 460800 baud, so the ESP32 side needed to match that persisted printer-side speed.
- Saving settings avoids requiring a manual `printer_baud 460800` after every reboot.
- Testing both no-CTS and CTS modes checks whether CTS remains safe after the baud fix.

### What worked
- New firmware at 460800 returned real status bytes:
  - `Status n=1 (printer): 0x16`
  - `Status n=2 (offline): 0x12`
  - `Status n=3 (error): 0x12`
  - `Status n=4 (paper): 0x12`
- `printer_status` returned a real 4-byte status packet:
  - `raw: 14 00 00 0F`
  - `buffer_full=no cover_open=no feed_key=no cutter_error=no auto_error=no overheated=no paper_near_end=no paper_out=no`
- `printer_get_baud` returned a real K118 response:
  - `esp32_baud=460800 printer_baud=460800 raw=uart baudrate: 460800`
- User confirmed physical printing worked.
- After saving startup settings and restarting, the new firmware booted, applied baud 460800, and all serial diagnostics still worked without manual baud setup.

### What didn't work
- Boot still sends an initial `ESC @` at 9600 before saved settings are applied because `printer_drv_init()` sends reset before `apply_saved_printer_settings()` changes baud. This did not block the later settings application, but it is a small ordering wart.

### What I learned
- The copied/new firmware is not fundamentally broken for printer UART.
- The main issue was baud mismatch: printer persisted at 460800, firmware default at 9600.
- The old firmware's earlier echo-like behavior at 9600 was a symptom of being at the wrong baud and should not be interpreted as valid communication.
- Once baud matches, the same pins and current ESP-IDF 5.4.2 build produce valid K118 responses.

### What was tricky to build
- The old `printer_probe` false-positive problem made the first investigation confusing: at 9600 it could still print `PRINTER RESPONDED` based on echo-like bytes, while the printer did not physically act.
- The real proof came from `printer_status` and `printer_get_baud` returning structured responses after switching to 460800, plus physical output confirmation.
- Persisting settings required selecting density/speed/graphics-mode values too; I used `density=20`, `speed=80`, `graphics_mode=31 adaptive`, consistent with the existing safe diagnostic recommendations.

### What warrants a second pair of eyes
- Review whether `printer_drv_init()` should avoid sending `ESC @` until after saved baud is applied, or whether `apply_saved_printer_settings()` should be integrated into the driver initialization path.
- Confirm `density=20 speed=80 graphics_mode=31` are the desired long-term defaults for this physical printer.
- Decide whether the project should keep NVS-only baud persistence or add a documented default baud override for Almanach devices known to be configured at 460800.

### What should be done in the future
- Improve `printer_probe` to reject echo/loopback false positives.
- Consider changing boot order so saved baud is applied before any printer command is sent.
- Document the recovery sequence:
  - `printer_baud 460800`
  - `printer_get_baud`
  - `printer_status`
  - `printer_settings_save 460800 20 80 31`
- Re-run a small HTTP print through the render path now that the firmware boots at the correct printer baud.

### Code review instructions
- Review boot ordering in:
  - `almanach/firmware/atoms3r/main/app_main.c`
  - `almanach/firmware/atoms3r/main/printer_drv.c`
- Review saved settings in:
  - `almanach/firmware/atoms3r/main/printer_cmd.c`
  - `almanach/firmware/atoms3r/main/nvs_store.c`
- Validate on hardware with:
  - `printer_settings_show`
  - `printer_probe`
  - `printer_get_baud`
  - `printer_status`

### Technical details
- Saved startup settings now present on the device:
  - `baud=460800`
  - `density=20`
  - `speed=80`
  - `graphics_mode=31 (adaptive)`
- Confirmed restart log:
  - `Applying saved printer settings: baud=460800 density=20 speed=80 graphics_mode=31`
- Confirmed after restart:
  - `printer_get_baud` -> `esp32_baud=460800 printer_baud=460800 raw=uart baudrate: 460800`
  - `printer_status` -> `raw: 14 00 00 0F`


## Step 5: Re-test Almanach server render-and-print path after UART baud fix

With the copied firmware booting at the printer's persisted 460800 baud, I re-tested the higher-level Almanach server path instead of only the serial console commands. This validates the end-to-end chain from localhost render server through the printer HTTP endpoint down to the K118 UART driver.

Both the tiny text-focused cat layout and the one-image cat layout printed successfully through `/api/render-and-print`. This confirms the earlier HTTP print failures were at least partly caused by the firmware/printer baud mismatch, not by the localhost setup rendezvous or render server itself.

### Prompt Context

**User prompt (verbatim):** "Can we try printing an almanach test again, through the server?"

**Assistant interpretation:** Send test Almanach layouts through the running localhost render server to the newly fixed printer firmware.

**Inferred user intent:** Validate that the user-facing server print workflow works after fixing the underlying printer UART baud configuration.

**Commit (code):** N/A — hardware/server validation only.

### What I did
- Checked the running setup/render server:
  - `curl -sS http://localhost:18299/health`
  - returned `{"ok":true,"printer":"192.168.1.242","version":"dev"}`
- Confirmed persisted setup-discovered printer state:
  - `curl -sS http://localhost:18299/api/setup/provisioned-device`
  - returned `ALM_0F2320 -> 192.168.1.242`
- Sent the tiny cat layout through the server:
  - `curl -sS -X POST http://localhost:18299/api/render-and-print -H 'Content-Type: application/json' --data-binary @/tmp/cat-tiny.json`
- Sent the one-image cat layout through the server:
  - `curl -sS -X POST http://localhost:18299/api/render-and-print -H 'Content-Type: application/json' --data-binary @/tmp/cat-one-image.json`

### Why
- Serial console success proves the firmware UART path works, but the real product path is: layout JSON -> localhost render server -> bitmap POST to printer HTTP endpoint -> firmware UART -> K118.
- Testing through `/api/render-and-print` validates that the persisted printer IP and HTTP print endpoint still work after the baud fix.

### What worked
- Tiny cat layout response:
  - `height=298`
  - `width=384`
  - `ok=true`
  - `printed=true`
  - `printerResponse={"ok":true}`
- One-image cat layout response:
  - `height=627`
  - `width=384`
  - `ok=true`
  - `printed=true`
  - `printerResponse={"ok":true}`
- The server used the persisted printer IP `192.168.1.242` correctly.

### What didn't work
- N/A for these two test layouts. Larger multi-image layouts may still need separate testing because they previously exposed long-duration HTTP/bitmap limitations.

### What I learned
- The server path is healthy for small and medium test prints once the printer UART baud is correct.
- The one-image layout that previously timed out now returned success, which strongly implicates the baud mismatch as the major blocker for prior printer HTTP behavior.

### What was tricky to build
- This validation spans multiple layers, so success depends on both the render server's persisted printer state and the firmware's persisted printer baud settings.
- The setup server was already running from earlier work; it did not need a restart for this test because the printer IP persistence was already loaded and correct.

### What warrants a second pair of eyes
- Re-test the largest image-heavy Almanach layouts before declaring the whole image pipeline fixed.
- Review whether the printer HTTP handler waits until UART printing is complete before responding; large bitmap jobs may still need respond-before-print or longer client timeouts.

### What should be done in the future
- Try the larger `/tmp/cat-interesting-facts-almanach.json` or six-image layout after confirming the printer is ready.
- If larger layouts still fail, investigate HTTP handler timeout/streaming separately from UART baud.
- Add a short playbook command sequence for server print validation after printer baud recovery.

### Code review instructions
- Server path starts in:
  - `almanach/internal/app/server.go`, `handleRenderAndPrint`
- Printer HTTP client path:
  - `almanach/internal/app/printer.go`, `sendBitmapToPrinter`
- Firmware endpoint path:
  - `almanach/firmware/atoms3r/main/web_server.c`
  - `almanach/firmware/atoms3r/main/printer_drv.c`

### Technical details
- Successful tiny response saved at:
  - `/tmp/almanach-server-print-cat-tiny-response.json`
- Successful one-image response saved at:
  - `/tmp/almanach-server-print-cat-one-image-response.json`


## Step 6: Test longer foo-cat ZIP bundles and isolate HTTP body-size threshold

The user asked to use the `foo-` images because they are actual cat images, and asked whether longer failures mean we need bitmap streaming. I created ZIP layout bundles that reference `images/foo-cat-NN.png` assets instead of embedding data URLs, then printed them with the CLI `print --layout bundle.zip` path.

The result is that ZIP layout input works, and medium-length prints now succeed. However, very long rendered bitmaps still fail around the firmware HTTP receive stage before UART bitmap printing begins. This is not a ZIP input problem and not an immediate UART streaming problem; it is most likely a firmware HTTP request-body receive/connection-size issue for large bitmap POST bodies.

### Prompt Context

**User prompt (verbatim):** "try sending as zip"

**Assistant interpretation:** Create a ZIP layout bundle with external image files and print that, rather than embedding all images as data URLs in JSON.

**Inferred user intent:** Reduce layout payload bloat and test whether ZIP-based image handling avoids long-print failures.

**Commit (code):** N/A — hardware/server validation only.

### What I did
- Built ZIP bundles under `/tmp` using real `foo-banner-*.png` cat images from:
  - `/home/manuel/workspaces/2026-05-08/extract-almanach/esp32-s3-m5/ttmp/2026/05/08/ALMANACH-IMAGE-BLOCKS--add-almanach-image-blocks-and-upload-support/various/grid-banners/foo`
- Created and tested:
  - `/tmp/foo-cat-long-bundle.zip` with 8 images
  - `/tmp/foo-cat-4-bundle.zip` with 4 images
  - `/tmp/foo-cat-5-bundle.zip` with 5 images
  - `/tmp/foo-cat-6-bundle.zip` with 6 images
  - `/tmp/foo-cat-7-bundle.zip` with 7 images
- Printed using:
  - `go run ./cmd/almanach-render-service print --layout /tmp/foo-cat-N-bundle.zip --printer-ip 192.168.1.242 --output json`

### Why
- ZIP bundles avoid putting base64 image data in the author-facing layout file; the CLI unpacks/loads the images and inlines them for the browser render step.
- This tests the existing ZIP layout support without changing the HTTP printer endpoint.

### What worked
- 4-image ZIP succeeded:
  - rendered bitmap `384x1221`
  - sent bitmap `height=1293`
  - `bytes=62064`
  - `printer_ok=true`
- 5-image ZIP succeeded:
  - rendered bitmap `384x1485`
  - sent bitmap `height=1557`
  - `bytes=74736`
  - `printer_ok=true`
- 6-image ZIP succeeded:
  - rendered bitmap `384x1748`
  - sent bitmap `height=1820`
  - `bytes=87360`
  - `printer_ok=true`

### What didn't work
- 8-image ZIP failed:
  - rendered bitmap `384x2374`
  - bitmap body `113952 bytes`
  - error: `use of closed network connection`
- 7-image ZIP failed:
  - rendered bitmap `384x2012`
  - bitmap body about `100032 bytes` after feed rows
  - error: `use of closed network connection`
- Firmware logs for the failed long request showed only the pre-read reset:
  - `TX 2 bytes: 1B 40`
  - `httpd_uri: uri handler execution failed`
- Firmware logs for successful requests showed the raster header and bitmap bytes:
  - 4-image: `TX 8 bytes: 1D 76 30 00 30 00 0D 05`; `TX 62064 bytes ...`
  - 5-image: `TX 8 bytes: 1D 76 30 00 30 00 15 06`; `TX 74736 bytes ...`
  - 6-image: `TX 8 bytes: ...`; `TX 87360 bytes ...`

### What I learned
- The host/layout ZIP is not the same thing as streaming the printer bitmap.
- Current pipeline:
  - ZIP layout assets are read by the Go CLI and converted into browser-renderable layout JSON.
  - Browser/Chrome renders a PNG.
  - Go converts PNG to a 1-bit bitmap.
  - Go sends one HTTP POST body to firmware `/api/print/bitmap`.
  - Firmware reads the full HTTP body into RAM before UART printing.
  - Firmware sends one complete `GS v 0` raster command to UART.
- We are intentionally **not** streaming HTTP chunks directly to UART, because previous notes indicate network gaps inside a raster command can create stripes/corruption.
- The current failure threshold appears to be somewhere between about `87 KB` and `100 KB` of bitmap POST body.

### What was tricky to build
- ZIP reduces source layout size and keeps image assets cleaner, but after rendering the printer still receives a raw 1-bit bitmap. Therefore ZIP does not reduce the final `/api/print/bitmap` body if the rendered paper height is the same.
- The firmware failure occurs before `printer_drv_print_bitmap()` logs the raster header for failed large jobs, so this is probably HTTP receive/body handling, not UART print duration.

### What warrants a second pair of eyes
- Inspect `almanach/firmware/atoms3r/main/web_server.c` `read_body()` and `api_print_bitmap_post()` for large-body receive behavior.
- Consider increasing `httpd_config_t.recv_wait_timeout` and improving `httpd_req_recv()` error handling/logging.
- Consider whether ESP-IDF httpd or lwIP has a request-body or socket-buffer behavior causing closes around 100 KB.

### What should be done in the future
- Add detailed logging in `read_body()` for offset, expected length, `httpd_req_recv()` return values, and `errno`/timeout classification.
- Consider splitting long Almanach bitmaps into multiple HTTP requests at safe page boundaries or raster bands, while keeping each individual raster body fully buffered before UART.
- If streaming is implemented, stream from HTTP to RAM chunks/files first, not directly into the middle of a printer raster command.

### Code review instructions
- Host ZIP layout support:
  - `almanach/internal/app/layout_bundle.go`
  - `almanach/internal/app/cmd_print.go`
- Host printer POST:
  - `almanach/internal/app/printer.go`
- Firmware HTTP endpoint:
  - `almanach/firmware/atoms3r/main/web_server.c`, `api_print_bitmap_post()` and `read_body()`
- Firmware UART raster send:
  - `almanach/firmware/atoms3r/main/printer_drv.c`, `printer_drv_print_bitmap()`

### Technical details
- Successful response artifacts:
  - `/tmp/foo-cat-4-bundle-print-response.json`
  - `/tmp/foo-cat-5-bundle-print-response.json`
  - `/tmp/foo-cat-6-bundle-print-response.json`
- Failed response artifacts/commands:
  - `/tmp/foo-cat-long-bundle-print-response.json` was attempted but command exited nonzero for 8 images.
  - `/tmp/foo-cat-7-bundle-print-response.json` was attempted but command exited nonzero for 7 images.


## Step 7: Add light thermal tone for image blocks and validate foo-cat print

The user chose the simpler Option 2: keep one bitmap/raster pipeline, but make images lighter during host-side rendering so text can remain high contrast while photos avoid becoming too dark on thermal paper. I added a per-image `thermalTone` setting to Almanach Studio and used CSS filters during browser render to lighten image blocks before the PNG-to-1-bit conversion.

This does not change printer density and does not reduce the final bitmap byte size. It changes the rendered pixels before thresholding, which means fewer image pixels cross the black threshold. The text stays crisp because the filter applies only to image block `<img>` elements.

### Prompt Context

**User prompt (verbatim):** "ok, let's try option 2. Add tasks to the ticket and commit at appropriate intervals. keep a diary."

**Assistant interpretation:** Implement the host-side lighter image rendering approach, track it in the UART ticket tasks/diary, validate with a foo-cat print, and commit the work.

**Inferred user intent:** Improve image print quality without implementing segmented density switching yet.

**Commit (code):** N/A at diary write time; code commit follows this validation step.

### What I did
- Added ticket tasks for the lighter image option and validation.
- Updated `web/src/almanach-studio.jsx`:
  - added default `thermalTone: "normal"` for image blocks,
  - added `imageThermalFilter(data, preview)` helper,
  - added `thermalTone === "light"` CSS filter,
  - added editor buttons for `Normal` vs `Light image`.
- Rebuilt frontend assets:
  - `pnpm --prefix web run build`
  - `BUILD_WEB_LOCAL=1 go run ./cmd/build-web`
- Generated a 6-image foo-cat ZIP bundle using `thermalTone: "light"`:
  - `/tmp/foo-cat-6-light-bundle.zip`
- Printed it through the CLI/server path:
  - `go run ./cmd/almanach-render-service print --layout /tmp/foo-cat-6-light-bundle.zip --printer-ip 192.168.1.242 --output json`
- Ran:
  - `go test ./...`
- Checked tasks 2 and 3.

### Why
- Text and images have different thermal needs, but one full-page bitmap cannot carry density commands inside pixel data.
- Applying a lighter CSS filter to images before bitmap thresholding is the lowest-risk first improvement: it requires no firmware protocol change and preserves the existing one-bitmap print path.

### What worked
- Frontend build succeeded.
- Embedded web asset rebuild succeeded.
- Go tests passed.
- The light-tone foo-cat 6-image print succeeded:
  - rendered bitmap `384x1748`
  - sent bitmap `height=1820`
  - sent bytes `87360`
  - `printer_ok=true`
  - `printer_response={"ok":true}`

### What didn't work
- This option does not address the ~100 KB firmware HTTP body threshold observed with 7+ images. The byte size of the light-tone 6-image bitmap is the same as the normal-tone 6-image bitmap because the dimensions did not change.
- Visual darkness improvement still needs human assessment on the physical paper output.

### What I learned
- The render pipeline can support per-image thermal tuning entirely in the React layout layer.
- ZIP bundles can carry the new `thermalTone` field normally because it is just image block data in `layout.json`.
- Option 2 is complementary to, not a replacement for, segmented printing if we later need per-region printer density or long-page chunking.

### What was tricky to build
- The capture/export path had prior CSS that removes some filters for export, but the server-side capture CSS does not globally remove image filters. Keeping the filter inline on the `<img>` made it apply during Chrome screenshot capture.
- The filter has to lighten images without making them disappear after thresholding. I used `brightness(1.28) contrast(0.82)` for print and slightly milder values for editor preview.

### What warrants a second pair of eyes
- The exact filter values are subjective and should be tuned against real paper output.
- The editor naming (`Light image`) should be reviewed for clarity; maybe `Normal thermal` / `Lighter thermal` would be clearer.
- If image quality remains too dark, consider adding a slider rather than a two-state toggle.

### What should be done in the future
- Compare normal vs light foo-cat prints side by side.
- Add optional `thermalTone: "extraLight"` or numeric image thermal strength if needed.
- Implement segmented printing later if text density and image density still need true printer-side differences.

### Code review instructions
- Review:
  - `web/src/almanach-studio.jsx`, `imageThermalFilter`, `ImageBlock`, and `ImageEditor`.
  - `internal/web/embed/public/almanach-bundle.js` regenerated by `cmd/build-web`.
- Validate with:
  - `pnpm --prefix web run build`
  - `BUILD_WEB_LOCAL=1 go run ./cmd/build-web`
  - `go test ./...`
  - print `/tmp/foo-cat-6-light-bundle.zip`.

### Technical details
- Light image filter:
  - preview: `grayscale(100%) brightness(1.22) contrast(0.88)`
  - print/render: `grayscale(100%) brightness(1.28) contrast(0.82)`
- Successful print artifact:
  - `/tmp/foo-cat-6-light-bundle-print-response.json`


## Step 8: Create chunked-print issue, add 90 KiB rejection, and print large corrected cat portraits

The user asked to capture the chunked/segmented endpoint work as a GitHub issue, add immediate rejection for oversized single bitmap jobs, and then split a cat collage into individual portraits for a larger portrait story print. I created the GitHub issue, implemented the 90 KiB guard, saved the helper scripts in the ticket `scripts/` directory, and printed the corrected large portrait story successfully.

The first portrait split was wrong because I assumed a 3x3 grid. Looking at the image showed it is a 4x4 collage with red guide lines. I corrected the splitter to use a 4x4 grid, crop inside each cell to remove the red borders, and produce square portraits.

### Prompt Context

**User prompt (verbatim):** "Create a gh issue with all the details, so we can address the chunked endpoint later.

Then do the reject at 90kb.

THen, split the image /tmp/pi-clipboard-684daa14-7016-4dfe-bdf6-432e3fc0d1b1.png into individual square cat portraits and then print a little story about cats and their owners, using portraits."

**Assistant interpretation:** Track the future long-print protocol on GitHub, add a short-term guardrail for oversized bitmap requests, and create/print a cat-owner story layout using cropped portraits from the supplied collage.

**Inferred user intent:** Avoid losing the chunked endpoint design, make current large-print failures deterministic, and produce a nice physical cat story print using the supplied image.

**Commit (code):** N/A at diary write time; commit follows this step.

### What I did
- Created GitHub issue:
  - `https://github.com/go-go-golems/almanach/issues/1`
  - title: `Add chunked or segmented bitmap print endpoint for long Almanachs`
- Added 90 KiB guards:
  - firmware `/api/print/bitmap` rejects expected bitmap bodies over `90 * 1024` bytes,
  - host `sendBitmapToPrinter` rejects oversized bitmap bodies before POSTing.
- Built and flashed firmware with the rejection change.
- Verified host-side rejection on `/tmp/foo-cat-7-bundle.zip`:
  - `bitmap too large: got 100032 bytes, max 92160 bytes; use a shorter layout or future segmented print endpoint`
- Saved helper scripts in the ticket scripts directory:
  - `scripts/01-create-foo-cat-bundles.py`
  - `scripts/02-create-cat-owner-story-bundle.py`
- Inspected the supplied collage and corrected the splitter from 3x3 to 4x4.
- Generated corrected portraits:
  - `/tmp/pi-cat-portraits/cat-portrait-R-C.png`
  - contact sheet: `/tmp/pi-cat-portraits-contact.png`
- Printed the corrected large portrait story bundle:
  - `/tmp/cat-owner-story-bundle.zip`

### Why
- A future segmented/chunked endpoint is the right long-term answer for long Almanachs, but the current endpoint should fail clearly instead of resetting the connection.
- The host guard avoids sending a body that the current firmware cannot reliably accept.
- The corrected 4x4 crop produces actual individual cat portraits without mixing adjacent cells.

### What worked
- GitHub issue created successfully.
- Firmware build and flash succeeded.
- Host rejection works deterministically for 7-image long jobs.
- Corrected portrait split produced 16 clean square portraits, e.g. `cat-portrait-2-2.png` looked good when inspected.
- Corrected large portrait story printed successfully:
  - rendered bitmap `384x1806`
  - sent bitmap `height=1878`
  - sent bytes `90144`, under the 90 KiB limit (`92160`)
  - `printer_ok=true`
  - `printer_response={"ok":true}`

### What didn't work
- Firmware-side rejection was added and flashed, but the first live oversized print attempt still closed the connection before returning the firmware JSON. Host-side rejection now prevents that path for CLI/server calls that use `sendBitmapToPrinter`.
- The first collage split assumed 3x3 and was visually wrong; it had to be corrected after inspecting the image.

### What I learned
- The supplied collage is 4x4, 1254x1254, with thin red guides; cropping with an inset removes guide lines and keeps each portrait square.
- The practical safe body limit is tight: the successful large portrait story was `90144` bytes, just below `92160`.
- Host-side rejection is important even if firmware also rejects, because it avoids WiFi/httpd connection behavior for known-too-large jobs.

### What was tricky to build
- The ZIP file can be large due to source PNG portraits, but the relevant print limit is the final rendered 1-bit bitmap body, not the ZIP size.
- Making portraits “large” increases page height quickly, so the layout must balance portrait size against the 90 KiB single-bitmap limit.

### What warrants a second pair of eyes
- Firmware rejection currently returns the same `send_json_error` shape as other errors; review whether it should set HTTP 413 Payload Too Large.
- Confirm whether `MAX_BITMAP_BODY_BYTES` should be 90 KiB exactly or slightly lower for safety.
- Review the issue body and acceptance criteria for the future segmented endpoint.

### What should be done in the future
- Implement GitHub issue #1 with segmented or buffered chunk upload.
- Add automated tests for host-side max bitmap body rejection.
- Add a visible UI warning when rendered bitmap size exceeds the single-request limit.

### Code review instructions
- Review:
  - `firmware/atoms3r/main/web_server.c` for `MAX_BITMAP_BODY_BYTES` and early reject.
  - `internal/app/printer.go` for host-side oversized bitmap rejection.
  - ticket scripts `01-create-foo-cat-bundles.py` and `02-create-cat-owner-story-bundle.py`.
- Validate with:
  - `go test ./...`
  - firmware build/flash
  - `/tmp/foo-cat-7-bundle.zip` should reject on host side
  - `/tmp/cat-owner-story-bundle.zip` should print successfully.

### Technical details
- GitHub issue:
  - `https://github.com/go-go-golems/almanach/issues/1`
- Corrected collage input:
  - `/tmp/pi-clipboard-684daa14-7016-4dfe-bdf6-432e3fc0d1b1.png`
- Generated portraits:
  - `/tmp/pi-cat-portraits/`
- Generated story bundle:
  - `/tmp/cat-owner-story-bundle.zip`
- Successful print response:
  - `/tmp/cat-owner-story-large-print-response.json`


## Step 9: Reprint cat-owner story with larger font and borderless minimal portraits

The user asked for a larger font and a minimal layout without image borders. I adjusted the cat-owner story generator to keep the minimal theme, remove portrait borders, increase body scale, and slightly reduce portrait height so the final single bitmap still stays under the current 90 KiB guard.

The updated story printed successfully, but it is very close to the current single-request limit. This is a good example of why the future segmented endpoint remains useful even after the immediate 90 KiB guard.

### Prompt Context

**User prompt (verbatim):** "make the font larger, use minimal layout for no borders"

**Assistant interpretation:** Adjust the generated cat-owner portrait story to prioritize readability and a cleaner minimal aesthetic, then print it again.

**Inferred user intent:** Improve the physical print composition based on the first output: larger text and less visual clutter around portraits.

**Commit (code):** N/A at diary write time; commit follows this step.

### What I did
- Updated `scripts/02-create-cat-owner-story-bundle.py`:
  - image `border: false`,
  - `bodyScale: 1.32`,
  - portrait height adjusted to `154` to stay under the 90 KiB bitmap limit.
- Regenerated bundle:
  - `/tmp/cat-owner-story-largefont-bundle.zip`
- Printed it with:
  - `go run ./cmd/almanach-render-service print --layout /tmp/cat-owner-story-largefont-bundle.zip --printer-ip 192.168.1.242 --output json`

### What worked
- Print succeeded:
  - rendered bitmap `384x1843`,
  - sent bitmap `height=1915`,
  - sent bytes `91920`,
  - `printer_ok=true`,
  - `printer_response={"ok":true}`.

### What didn't work
- The output is only `240` bytes below the 90 KiB limit (`92160`), so further font or height increases will likely trip the host-side guard.

### What I learned
- Larger text plus large portraits can still fit in the current single-bitmap endpoint, but only barely.
- Borderless images help the minimal look and save a little visual darkness/clutter, but they do not materially reduce bitmap byte size because byte size is driven by page dimensions.

### What should be done in the future
- Use segmented printing for any longer story or larger portrait set.
- If the user wants still larger text, reduce from four portraits to three or move to segmented print.

### Code review instructions
- Review `scripts/02-create-cat-owner-story-bundle.py` for layout defaults.
- Validate with `/tmp/cat-owner-story-largefont-print-response.json`.


## Step 10: Reprint with fewer, larger portraits

The user clarified that the portraits can be larger. I changed the story generator from four medium portraits to three larger portraits, keeping the minimal/no-border layout and large text. This gives each cat more vertical space while staying comfortably under the current 90 KiB single-bitmap limit.

### Prompt Context

**User prompt (verbatim):** "you can print the portraits larger"

**Assistant interpretation:** Increase the portrait size even if that means using fewer portraits in the story print.

**Inferred user intent:** Improve the visual impact/readability of the cat portrait print rather than maximizing the number of portraits.

**Commit (code):** N/A at diary write time; commit follows this step.

### What I did
- Updated `scripts/02-create-cat-owner-story-bundle.py`:
  - selected three diagonal portraits instead of four,
  - changed subtitle to “three large portraits”,
  - increased image height from `154` to `218`,
  - kept `border: false`, `theme: minimal`, and `bodyScale: 1.32`.
- Generated:
  - `/tmp/cat-owner-story-3-largeportraits-bundle.zip`
- Printed it with:
  - `go run ./cmd/almanach-render-service print --layout /tmp/cat-owner-story-3-largeportraits-bundle.zip --printer-ip 192.168.1.242 --output json`

### What worked
- Print succeeded:
  - rendered bitmap `384x1652`,
  - sent bitmap `height=1724`,
  - sent bytes `82752`,
  - `printer_ok=true`,
  - `printer_response={"ok":true}`.

### What didn't work
- N/A. This version has more safety margin than the four-portrait large-font version.

### What I learned
- Three large portraits is a better fit for the current single-request limit than four medium-large portraits.
- The story still fits comfortably while leaving roughly 9 KiB below the 90 KiB guard.

### What should be done in the future
- If the user wants all 16 portraits large, implement segmented printing from GitHub issue #1.
- If using single-request printing, keep pages below about 82-88 KiB for safety rather than targeting the absolute limit.

### Code review instructions
- Review `scripts/02-create-cat-owner-story-bundle.py` for current defaults.
- Validation artifact:
  - `/tmp/cat-owner-story-3-largeportraits-print-response.json`


## Step 11: Increase story font again while staying under the single-bitmap limit

The user asked for an even bigger font. I increased the body scale again and reduced portrait height slightly to keep the overall bitmap safely below the 90 KiB single-request guard.

### Prompt Context

**User prompt (verbatim):** "bigger font still"

**Assistant interpretation:** Prioritize text readability further, even if portrait size needs a small reduction.

**Inferred user intent:** Make the printed cat-owner story easier to read on thermal paper.

**Commit (code):** N/A at diary write time; commit follows this step.

### What I did
- Updated `scripts/02-create-cat-owner-story-bundle.py`:
  - `bodyScale` from `1.32` to `1.52`,
  - portrait height from `218` to `196`.
- Generated:
  - `/tmp/cat-owner-story-biggerfont-bundle.zip`
- Printed with:
  - `go run ./cmd/almanach-render-service print --layout /tmp/cat-owner-story-biggerfont-bundle.zip --printer-ip 192.168.1.242 --output json`

### What worked
- Print succeeded:
  - rendered bitmap `384x1711`,
  - sent bitmap `height=1783`,
  - sent bytes `85584`,
  - `printer_ok=true`,
  - `printer_response={"ok":true}`.

### What didn't work
- N/A. The print remains below the 90 KiB guard.

### What I learned
- Increasing font size can be balanced against portrait height while remaining under the current single-request limit.
- The three-portrait layout is flexible enough for larger type without needing segmented printing yet.

### What should be done in the future
- If the font still needs to be larger, reduce story text length or move to a two-portrait layout.

### Code review instructions
- Review current defaults in `scripts/02-create-cat-owner-story-bundle.py`.
- Validation artifact:
  - `/tmp/cat-owner-story-biggerfont-print-response.json`


## Step 12: Recover after crash and restart the localhost setup server

The user reported that the session crashed and asked me to restart the server and keep the diary current. I first checked whether the localhost setup server was still alive, confirmed that port `18299` was down, rebuilt the embedded web assets after the default-layout text cleanup, and restarted the setup server.

The restarted server picked up the persisted setup-discovered printer endpoint, so `/health` again reports the printer at `192.168.1.242`. I also verified the printer's direct firmware status endpoint after restart.

### Prompt Context

**User prompt (verbatim):** "we just crashed so you will have to restart the server and all. Keep a diary as you go."

**Assistant interpretation:** Restore the local Almanach setup/print server after the crash, verify it is connected to the known printer, and record the recovery work in the ticket diary.

**Inferred user intent:** Get the working local setup/printing environment back into a known-good state without losing the investigation trail.

**Commit (code):** N/A at diary write time; commit follows after the pending recovery/docs/code changes are reviewed.

### What I did
- Checked repo status and confirmed there are pending changes from the printer/print-layout work.
- Checked for any running `almanach-render-service` process.
- Confirmed `http://localhost:18299/health` was unavailable:
  - `curl: (7) Failed to connect to localhost port 18299 after 0 ms: Couldn't connect to server`
- Rebuilt embedded web assets after removing the Cal Newport sample text from the studio defaults:
  - `BUILD_WEB_LOCAL=1 go run ./cmd/build-web`
- Restarted the setup server:
  - `nohup go run ./cmd/almanach-render-service setup --port 18299 > /tmp/almanach-logs/setup-server-18299.log 2>&1 &`
  - PID file: `/tmp/almanach-logs/setup-server-18299.pid`
- Verified localhost server health:
  - `{"ok":true,"printer":"192.168.1.242","version":"dev"}`
- Verified persisted setup-discovered printer state:
  - service name `ALM_0F2320`
  - IP `192.168.1.242`
  - SSID `Verizon_9DNVB9`
- Verified direct printer firmware status:
  - `{"ok":true,"wifi":{"connected":true,"ip":"192.168.1.242"},"printer":{"baud":460800,"swapped":true}}`
- Confirmed no `Cal Newport` or `Deep Work — Cal Newport` strings remain in `web/src/almanach-studio.jsx` or regenerated `internal/web/embed/public/almanach-bundle.js`.

### Why
- The crash stopped the local setup server, so browser/setup workflows and localhost print routing needed to be restored.
- Rebuilding embedded assets makes the source cleanup visible from the Go-served UI, not just in the source file.
- Verifying both localhost and direct printer endpoints distinguishes server recovery from printer WiFi/firmware health.

### What worked
- The server restarted cleanly on port `18299`.
- Persisted printer endpoint state was loaded successfully.
- Direct printer status remained healthy with the known working K118 settings: baud `460800`, swapped pins enabled.
- The generated embedded studio bundle no longer contains the Cal Newport sample strings.

### What didn't work
- The pre-crash localhost server was gone; there was no live process to reuse.

### What I learned
- The state file recovery path is doing its job: after restart, the server immediately knows about `192.168.1.242` without needing a new BLE provisioning pass.
- Source-only text changes need `BUILD_WEB_LOCAL=1 go run ./cmd/build-web` before the self-contained Go server reflects them.

### What was tricky to build
- The relevant web UI is embedded in `internal/web/embed/public`, so a simple source edit in `web/src/almanach-studio.jsx` is insufficient for the running Go binary path.
- Crash recovery has two health checks: localhost server health and direct printer firmware health.

### What warrants a second pair of eyes
- Review whether generated embedded bundles should be committed with the source edit in this branch.
- Confirm whether the setup server should be supervised by a dev script instead of ad hoc `nohup` during hardware sessions.

### What should be done in the future
- Add a `make setup-server` or devctl command that restarts this exact localhost setup server and tails logs.
- Commit the pending recovery/docs/code changes once reviewed.

### Code review instructions
- Review:
  - `web/src/almanach-studio.jsx` for default sample text cleanup.
  - `internal/web/embed/public/almanach-bundle.js` for regenerated embedded output.
  - this diary step and changelog for recovery notes.
- Validate with:
  - `curl -sS http://localhost:18299/health`
  - `curl -sS http://localhost:18299/api/setup/provisioned-device`
  - `curl -sS http://192.168.1.242/api/status`

### Technical details
- Restarted server PID:
  - `/tmp/almanach-logs/setup-server-18299.pid`
- Restarted server log:
  - `/tmp/almanach-logs/setup-server-18299.log`
- Setup URL:
  - `http://localhost:18299/setup`
- Health response after restart:
  - `{"ok":true,"printer":"192.168.1.242","version":"dev"}`


## Step 13: Re-extract cat portraits into the ticket and print one large portrait

After the crash, the `/tmp` portrait outputs were gone. The user provided the refreshed clipboard image and asked me to extract the cats again, verify the crop visually, and store the results in the ticket so they survive future session resets.

I copied the source collage into the ticket, extracted all sixteen 4x4 cells with a small inset to remove the red guide lines, generated a labeled contact sheet for verification, and then printed one large single-cat portrait from the stored ticket asset.

### Prompt Context

**User prompt (verbatim):** "you will have to extract them again and store them in the ticket, we crashed.
/tmp/pi-clipboard-bcf0fca6-54a3-4349-a272-570e6afe0793.png

Here they are, extract the cats and verify you cut them out correctly and store them in the ticket."

**Assistant interpretation:** Recreate the cat portrait crops from the new clipboard file, persist the source/crops/verification sheet under the ticket, and ensure the crops look correct before using them.

**Inferred user intent:** Avoid losing generated assets to `/tmp` cleanup and make the portrait extraction reproducible and reviewable.

**Commit (code):** N/A at diary write time; commit follows after pending changes are reviewed.

### What I did
- Inspected `/tmp/pi-clipboard-bcf0fca6-54a3-4349-a272-570e6afe0793.png`; it is the same 4x4 cat collage with red guide lines.
- Stored the source image in the ticket:
  - `assets/cat-portraits/source-4x4-cat-collage.png`
- Extracted 16 square portraits to:
  - `assets/cat-portraits/portraits/cat-portrait-rRR-cCC.png`
- Generated verification artifacts:
  - `assets/cat-portraits/contact-sheet.png`
  - `assets/cat-portraits/manifest.md`
- Saved the extraction script:
  - `scripts/04-extract-cat-portraits.py`
- Updated the single-large-cat bundle script to default to the stored ticket asset:
  - `scripts/03-create-single-large-cat-bundle.py`
- Printed one large portrait from `cat-portrait-r02-c02.png`.

### Why
- `/tmp` artifacts were lost after the crash, so the source and derived portrait crops need to live under the ticket.
- A contact sheet makes it easy to verify at a glance that each crop contains exactly one cat and not adjacent cells.
- Updating the print script to use ticket assets avoids depending on ephemeral `/tmp/pi-cat-portraits` paths.

### What worked
- Extraction succeeded:
  - 16 portraits generated.
  - all portraits are `313x313`.
  - contact sheet size is `720x800`.
- Visual verification of `contact-sheet.png` showed the 4x4 crop is correct: each cell contains one cat portrait, and the red grid lines are removed.
- Single large portrait print succeeded:
  - bundle: `/tmp/single-large-cat-bundle.zip`
  - rendered bitmap `384x439`
  - sent bitmap `height=511`
  - sent bytes `24528`
  - `printer_ok=true`
  - `printer_response={"ok":true}`

### What didn't work
- The first single-large-cat attempt failed because it still referenced `/tmp/pi-cat-portraits/cat-portrait-2-2.png`, which was gone after the crash.
- Re-running the old story splitter also failed because the original `/tmp/pi-clipboard-684daa14-7016-4dfe-bdf6-432e3fc0d1b1.png` was gone.

### What I learned
- Any generated asset that matters for follow-up should live under the ticket, not only in `/tmp`.
- The refreshed clipboard image has the same dimensions and grid layout: `1254x1254`, `4x4` cells.

### What was tricky to build
- The crop needs a small inset (`10px`) to remove red guide lines without cutting into the cat illustrations too much.
- The single-large-cat script needed to be made independent of `/tmp` portrait state.

### What warrants a second pair of eyes
- Review `contact-sheet.png` to confirm the chosen inset is visually acceptable.
- Review whether binary assets under the ticket should be committed or whether only scripts/manifest should be committed and source assets stored elsewhere.

### What should be done in the future
- Prefer ticket-local assets for all future hardware print experiments.
- If the assets are too large for git history, move them to a documented artifact store and keep the manifest in the ticket.

### Code review instructions
- Review:
  - `assets/cat-portraits/contact-sheet.png`
  - `assets/cat-portraits/manifest.md`
  - `scripts/04-extract-cat-portraits.py`
  - `scripts/03-create-single-large-cat-bundle.py`
- Validate with:
  - `python scripts/04-extract-cat-portraits.py --input /tmp/pi-clipboard-bcf0fca6-54a3-4349-a272-570e6afe0793.png`
  - `python scripts/03-create-single-large-cat-bundle.py --height 360`
  - `go run ./cmd/almanach-render-service print --layout /tmp/single-large-cat-bundle.zip --printer-ip 192.168.1.242 --output json`

### Technical details
- Stored source:
  - `ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/source-4x4-cat-collage.png`
- Stored portraits:
  - `ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/portraits/`
- Contact sheet:
  - `ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/contact-sheet.png`
- Print response:
  - `/tmp/single-large-cat-print-response.json`


## Step 14: Print additional large cat portraits

The user asked to print some more portraits after the ticket-local extraction was restored. I used the stored ticket assets rather than `/tmp` crops and printed four additional large single-cat portraits.

### Prompt Context

**User prompt (verbatim):** "print out some more portraits"

**Assistant interpretation:** Use the verified ticket-local cat crops and produce additional physical large portrait prints.

**Inferred user intent:** Continue evaluating the visual quality of individual cat portrait prints on the thermal printer.

**Commit (code):** N/A at diary write time; commit follows after pending changes are reviewed.

### What I did
- Used `scripts/03-create-single-large-cat-bundle.py` with `--height 360` for four stored portraits:
  - `cat-portrait-r01-c01.png`
  - `cat-portrait-r01-c03.png`
  - `cat-portrait-r04-c03.png`
  - `cat-portrait-r04-c04.png`
- Generated temporary bundles/responses under `/tmp` for each portrait.
- Printed each bundle to `http://192.168.1.242/api/print/bitmap` via `almanach-render-service print`.

### What worked
- All four additional portraits printed successfully.
- Each rendered as `384x439`, sent as `height=511`, `24528` bytes, with `printer_ok=true` and `printer_response={"ok":true}`.

### What didn't work
- N/A.

### What I learned
- Single 360px portrait prints are comfortably small for the current endpoint and are reliable for quick visual checks.
- The ticket-local asset path now works as intended after crash recovery.

### What should be done in the future
- If many portraits need printing, add a small batch-print script that records selected portrait IDs and response paths automatically.

### Code review instructions
- Review `scripts/03-create-single-large-cat-bundle.py` for the reusable single-portrait print bundle flow.
- Response artifacts:
  - `/tmp/single-large-cat-r01-c01-print-response.json`
  - `/tmp/single-large-cat-r01-c03-print-response.json`
  - `/tmp/single-large-cat-r04-c03-print-response.json`
  - `/tmp/single-large-cat-r04-c04-print-response.json`
