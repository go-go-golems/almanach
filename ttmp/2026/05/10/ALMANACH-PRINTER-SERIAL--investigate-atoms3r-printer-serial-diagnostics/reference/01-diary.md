---
Title: Diary
Ticket: ALMANACH-PRINTER-SERIAL
Status: active
Topics:
    - firmware
    - printer
    - hardware-diagnostics
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/app_printer.c
      Note: Simple no-CTS UART baseline for comparison
    - Path: ../../../../../../../esp32-s3-m5/stoms3r/docs/02-bitmap-stripes-flow-control.md
      Note: Old firmware design note for AtomS3R pin swap and CTS behavior
    - Path: ../../../../../../../esp32-s3-m5/stoms3r/main/printer_drv.c
      Note: Old firmware driver baseline
    - Path: ../../../../../../../esp32-s3-m5/ttmp/2026/04/28/STOMS3R-001--stoms3r-atoms3r-lite-thermal-printer-console-firmware/reference/01-diary.md
      Note: Historical pinout correction and CTS evolution evidence
    - Path: firmware/atoms3r/main/printer_cmd.c
      Note: Serial console printer_flow command
    - Path: firmware/atoms3r/main/printer_drv.c
      Note: Runtime CTS/no-CTS diagnostic and AtomS3R UART pin behavior
    - Path: firmware/atoms3r/main/printer_drv.h
      Note: Public diagnostic flow-control API
ExternalSources: []
Summary: Chronological investigation diary for AtomS3R/K118 serial printer diagnostics.
LastUpdated: 2026-05-10T17:50:49.697895828-04:00
WhatFor: Record printer serial debugging steps, firmware comparisons, hardware test results, and review instructions.
WhenToUse: Use when continuing printer UART, CTS, pin mapping, or old-firmware comparison work.
---



# Diary

## Goal

Capture the investigation into why the copied AtomS3R firmware does not reliably drive or query the K118 printer over the serial interface, including comparisons to the old firmware and hardware test outcomes.

## Step 1: Compare firmware printer UART behavior and add runtime CTS diagnostic

The first diagnostic pass checked whether the copied Almanach firmware diverged from the old `stoms3r` printer firmware. The current and old `printer_drv.c` / `printer_drv.h` files matched, so the likely regression was not an accidental copy error in those files.

The meaningful contrast came from the older simple provisioning example: it configured the printer UART with hardware flow control disabled and no CTS pin. To test that hypothesis without permanently removing CTS support, I added a runtime `printer_flow <cts|off>` console command that toggles ESP-IDF UART hardware flow control and detaches/reattaches the CTS pin.

### Prompt Context

**User prompt (verbatim):** "try that, invesetigate, compare to old firmware, in fact feel free to install the old firmware if you want too."

**Assistant interpretation:** Investigate the suspected printer UART/CTS issue by comparing against older firmware and, if useful, flashing alternative firmware or diagnostic changes.

**Inferred user intent:** Restore printer serial behavior by identifying whether the copied firmware differs from a working baseline or whether UART flow-control/pin behavior is the actual problem.

**Commit (code):** N/A — diagnostic change built locally but not committed yet.

### What I did
- Compared current/copied printer driver files against the old `stoms3r` files:
  - `almanach/firmware/atoms3r/main/printer_drv.c`
  - `almanach/firmware/atoms3r/main/printer_drv.h`
  - `esp32-s3-m5/stoms3r/main/printer_drv.c`
  - `esp32-s3-m5/stoms3r/main/printer_drv.h`
- Confirmed those current/old driver files are identical.
- Compared them to the older simple provisioning example:
  - `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/app_printer.c`
  - `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/app_printer.h`
- Observed that the simple example disables hardware flow control and does not attach CTS.
- Ran serial console probes before changing code:
  - `printer_status`
  - `printer_probe`
  - `printer_swap off`
  - `printer_probe`
- Added runtime CTS diagnostics:
  - `printer_drv_set_flow_control(bool enabled)`
  - `printer_drv_is_flow_control_enabled()`
  - `printer_flow <cts|off>` console command.
- Built the firmware with:
  - `cd /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Why
- The current firmware default uses `.flow_ctrl = UART_HW_FLOWCTRL_CTS` and maps CTS to GPIO6.
- The older simple example uses `.flow_ctrl = UART_HW_FLOWCTRL_DISABLE` and `UART_PIN_NO_CHANGE` for RTS/CTS.
- A runtime toggle is safer than immediately replacing the driver because it allows testing all combinations on the same flashed binary:
  - swapped pins + CTS
  - swapped pins + no CTS
  - normal pins + CTS
  - normal pins + no CTS.

### What worked
- The current and old `stoms3r` printer driver files compared cleanly; no driver-copy drift was found.
- The diagnostic firmware built successfully.
- The serial console confirmed the printer command set includes useful diagnostics such as `printer_probe`, `printer_status`, `printer_feed`, `printer_text`, `printer_swap`, and now `printer_flow` after flashing the diagnostic build.

### What didn't work
- Serial status probing timed out before the diagnostic change:
  - `printer_status` sent `1D 61 00` and received no 4-byte status response.
  - `printer_probe` sent `10 04 01` through `10 04 04` and received no real-time status response.
- Both swapped and normal pin modes failed to get a response before the CTS toggle existed.
- One monitor session reported:
  - `--- Error: device reports readiness to read but returned no data (device disconnected or multiple access on port?)`
  - The monitor later reconnected.

### What I learned
- The copied firmware matches the old `stoms3r` firmware for the printer driver, so the next suspect is not a bad copy of that driver.
- The old design document in `esp32-s3-m5/stoms3r/docs/02-bitmap-stripes-flow-control.md` explicitly says the AtomS3R header mapping should default to TX/RX swapped with CTS on GPIO6.
- The older simple provisioning firmware is not an AtomS3R-equivalent baseline; it targets Atom Lite pins and UART2, but it is still valuable because it proves a no-CTS serial configuration existed.

### What was tricky to build
- `printer_swap` previously owned the `uart_set_pin()` call directly. Adding a flow-control toggle without duplicating pin logic required factoring pin application into an internal helper that respects both current swap state and current CTS mode.
- Disabling flow control must do two things together: call `uart_set_hw_flow_ctrl(... UART_HW_FLOWCTRL_DISABLE ...)` and detach the CTS pin with `UART_PIN_NO_CHANGE`. Only changing one side could leave the UART peripheral or pin matrix in a misleading state.
- The current `printer_probe` display prints nominal header pins plus a `(SWAPPED)` marker; the actual effective UART TX pin in swapped mode is GPIO7. That is easy to misread during hardware debugging.

### What warrants a second pair of eyes
- Confirm that `uart_set_hw_flow_ctrl(PRINTER_UART_NUM, UART_HW_FLOWCTRL_CTS, 0)` uses an acceptable threshold argument for ESP-IDF when enabling CTS-only flow control.
- Review whether runtime pin remapping while the UART driver is installed needs additional flushing or delays before probe commands.
- Check whether the K118 CTS polarity matches ESP-IDF UART CTS expectations; if polarity is inverted, CTS mode could block or corrupt transmission.

### What should be done in the future
- Flash the diagnostic firmware.
- Run the four matrix tests:
  - `printer_flow cts`; `printer_swap on`; `printer_probe`; `printer_text hello`; `printer_feed 3`
  - `printer_flow off`; `printer_swap on`; `printer_probe`; `printer_text hello`; `printer_feed 3`
  - `printer_flow cts`; `printer_swap off`; `printer_probe`; `printer_text hello`; `printer_feed 3`
  - `printer_flow off`; `printer_swap off`; `printer_probe`; `printer_text hello`; `printer_feed 3`
- If no combination works, consider flashing the old `stoms3r` firmware as a baseline or temporarily porting the simple no-CTS Atom Lite-style UART setup to AtomS3R pins.

### Code review instructions
- Start with `almanach/firmware/atoms3r/main/printer_drv.c`:
  - `apply_uart_pins`
  - `printer_drv_swap_pins`
  - `printer_drv_set_flow_control`
- Then review `almanach/firmware/atoms3r/main/printer_cmd.c`:
  - `do_printer_flow`
  - `register_printer_commands`
- Validate with:
  - `cd /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`
  - flash and run the serial matrix above.

### Technical details
- Current/copied and old `stoms3r` driver default:
  - UART: `UART_NUM_1`
  - nominal TX: GPIO8
  - nominal RX: GPIO7
  - CTS: GPIO6
  - baud: 9600
  - flow control: `UART_HW_FLOWCTRL_CTS`
  - boot default: `printer_drv_swap_pins(true)`
- Older simple provisioning example:
  - UART: `UART_NUM_2`
  - TX: GPIO23
  - RX: GPIO33
  - CTS: GPIO19 in the header, but not used in `uart_set_pin()`
  - baud: 9600
  - flow control: `UART_HW_FLOWCTRL_DISABLE`

## Step 2: Flash diagnostic firmware and test the pin/CTS matrix

After adding the runtime flow-control command, I flashed the diagnostic firmware and exercised the UART matrix from the serial console. The flashed firmware booted correctly and exposed the new `printer_flow` command, so the diagnostic build itself is viable.

All four UART combinations still produced no status response from the K118. That does not conclusively prove text is not physically printed, because I cannot see the paper output through the terminal, but it does prove the ESP32 did not receive any status bytes over the configured RX path in any combination tested.

### Prompt Context

**User prompt (verbatim):** "is it the right pinout?"

**Assistant interpretation:** Re-evaluate whether GPIO8/GPIO7/GPIO6 is really the correct AtomS3R-to-K118 mapping, given the serial diagnostics are not receiving responses.

**Inferred user intent:** Decide whether to keep debugging software/flow-control or shift attention to physical pin mapping, cable orientation, or a different hardware baseline.

**Commit (code):** N/A — diagnostic firmware is still uncommitted.

### What I did
- Flashed the diagnostic build with:
  - `cd /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 flash`
- Restarted the serial monitor in tmux:
  - `tmux new-session -d -s alm-button-test './build.sh /dev/ttyACM0 monitor'`
- Confirmed boot logs:
  - `Printer UART1 ready: TX=8 RX=7 CTS=6 flow=cts baud=9600`
  - `Applying printer UART pins: TX=GPIO7 RX=GPIO8 CTS=GPIO6 (SWAPPED) flow=cts baud=9600`
- Ran the tested matrix:
  - `printer_flow cts`; `printer_swap on`; `printer_probe`; `printer_text CTS_SWAPPED_HELLO`; `printer_feed 3`
  - `printer_flow off`; `printer_swap on`; `printer_probe`; `printer_text NOCTS_SWAPPED_HELLO`; `printer_feed 3`
  - `printer_flow cts`; `printer_swap off`; `printer_probe`; `printer_text CTS_NORMAL_HELLO`; `printer_feed 3`
  - `printer_flow off`; `printer_swap off`; `printer_probe`; `printer_text NOCTS_NORMAL_HELLO`; `printer_feed 3`

### Why
- The existing repo documentation says GPIO8/GPIO7/GPIO6 are the AtomS3R physical header positions corresponding to the K118 Atom Lite TX/RX/CTS pins.
- The same documentation says the K118 cable appears straight-through, so firmware defaults to swapped TX/RX: effective UART TX on GPIO7 and RX on GPIO8.
- Testing all four combinations checks whether the issue is swap direction or CTS flow-control, before assuming the documented physical pinout is wrong.

### What worked
- Diagnostic firmware flashed successfully.
- The device booted and rejoined WiFi.
- Runtime flow-control toggling worked at the console and logs showed CTS being detached for `printer_flow off`.
- Runtime TX/RX swap toggling worked at the console and logs showed the effective TX/RX pins changing.

### What didn't work
- Every `printer_probe` run timed out on all four `DLE EOT` status queries.
- The exact repeated failure pattern was:
  - `Status query n=1: no response (got 0 bytes)`
  - `Status query n=2: no response (got 0 bytes)`
  - `Status query n=3: no response (got 0 bytes)`
  - `Status query n=4: no response (got 0 bytes)`
  - `=== Result: NO RESPONSE (check wiring/power) ===`
- Since the tests were run remotely via serial monitor, I could not visually confirm whether the `printer_text` or `printer_feed` commands physically moved paper.

### What I learned
- The pinout in the current firmware is internally consistent with the old `stoms3r` docs and old `stoms3r` code:
  - Atom Lite K118 header: TX GPIO23, RX GPIO33, CTS GPIO19.
  - AtomS3R same header positions: TX position GPIO8, RX position GPIO7, CTS position GPIO6.
  - Because the cable is believed straight-through, effective swapped UART is TX GPIO7 and RX GPIO8.
- However, the live hardware did not return bytes in either swapped or normal mode, with or without CTS.
- Therefore the most accurate answer is: the pinout is probably right according to the repo's prior AtomS3R investigation, but the current evidence does not prove the physical unit is wired that way today.

### What was tricky to build
- The labels are easy to confuse: `PRINTER_TX_GPIO` means the board header's TX-position GPIO, but in swapped mode the actual ESP32 UART TX is `PRINTER_RX_GPIO` / GPIO7.
- `printer_probe` still prints the nominal header mapping plus `(SWAPPED)` rather than the effective pins, which can make diagnostics look contradictory.
- A no-response status probe may mean RX is wrong, the printer does not implement/respond to that status command in the current state, the printer lacks power, or the cable is misoriented; it is not a single-cause signal.

### What warrants a second pair of eyes
- Verify the actual physical connector orientation against the AtomS3R Lite silkscreen and the K118 cable, not just the historical repo documentation.
- Confirm whether the K118 status commands used by `printer_probe` are expected to return bytes on this exact printer module/firmware.
- If a logic analyzer is available, observe GPIO7 and GPIO8 during `printer_text` and `printer_probe` to confirm TX activity and any RX response.

### What should be done in the future
- Ask the operator to confirm whether paper moved or text printed during the four `printer_text` / `printer_feed` tests.
- If paper moved, then TX pinout is correct and only RX/status is suspect.
- If paper did not move, verify 12V printer power and cable orientation first.
- If still inconclusive, flash the old `esp32-s3-m5/stoms3r` firmware as a baseline; because the copied driver is identical, this mostly tests non-driver integration and build/runtime environment.

### Code review instructions
- Review `almanach/firmware/atoms3r/docs/02-bitmap-stripes-flow-control.md` for the historical pinout rationale.
- Review `almanach/firmware/atoms3r/main/printer_drv.h` for nominal pin definitions.
- Review `almanach/firmware/atoms3r/main/printer_drv.c` for effective swapped runtime pin application.
- Validate with serial monitor commands listed above and, if possible, physical paper observation.

### Technical details
- Historical documented AtomS3R mapping:
  - TX header position -> GPIO8
  - RX header position -> GPIO7
  - CTS header position -> GPIO6
- Effective default runtime mapping after `printer_drv_swap_pins(true)`:
  - UART TX -> GPIO7
  - UART RX -> GPIO8
  - UART CTS -> GPIO6 when flow is `cts`
- No-CTS diagnostic mapping:
  - `printer_flow off` calls `uart_set_hw_flow_ctrl(... UART_HW_FLOWCTRL_DISABLE ...)`
  - `apply_uart_pins(...)` passes `UART_PIN_NO_CHANGE` for CTS.

## Step 3: Mine the old `esp32-s3-m5` ticket diary for pinout provenance

The user pointed out that the old firmware lives in `esp32-s3-m5/stoms3r/` and that the old ticket history lives under `esp32-s3-m5/ttmp/`. I searched those ticket diaries for the printer pinout history instead of relying only on the copied source tree.

That old diary gives the clearest provenance so far: the first `stoms3r` implementation originally assumed the AtomS3R HY2.0-4P/Grove-style pins `GPIO5/GPIO6`, then the user corrected it to the K118 bottom-header mapping `GPIO8/GPIO7/GPIO6`. It also records that CTS was initially wired but flow control disabled, then later enabled after bitmap banding experiments.

### Prompt Context

**User prompt (verbatim):** "the old firmware is in @esp32-s3-m5/stoms3r/ btw, in case you didn't know, and there's a bunch of diaries and tickets in esp32-s3-m5/ttmp/"

**Assistant interpretation:** Use the old firmware tree and historical docmgr-style ticket records as evidence for the pinout and printer-driver evolution.

**Inferred user intent:** Avoid guessing from the copied firmware alone; reconstruct the known-good or previously-debugged hardware assumptions from the historical investigation record.

**Commit (code):** N/A — documentation/investigation only.

### What I did
- Searched `esp32-s3-m5/ttmp` for K118/pinout references.
- Read the key old diary section:
  - `esp32-s3-m5/ttmp/2026/04/28/STOMS3R-001--stoms3r-atoms3r-lite-thermal-printer-console-firmware/reference/01-diary.md`
- Confirmed the old firmware path:
  - `esp32-s3-m5/stoms3r/`
- Found the pinout correction history and CTS evolution in the diary.

### Why
- The old source file alone only shows the final state; the diary explains how the project got there and which assumptions were corrected by hardware observation.
- The current question is about whether the pinout is right, so provenance matters more than just the present `#define`s.

### What worked
- The old diary explicitly states:
  - `User reported the K118 pinout: TX=GPIO8, RX=GPIO7, CTS=GPIO6 (not GPIO5/GPIO6 from HY2.0-4P)`
  - The K118 uses the ATOM Lite bottom header positions, not the AtomS3R HY2.0-4P port.
  - The AtomS3R Lite equivalent positions are `GPIO8/GPIO7/GPIO6`.
  - CTS was initially wired but flow control was left disabled, then later enabled in commit `dac86e8` after banding tests.

### What didn't work
- The old design guide contains some stale/contradictory sections that still describe the earlier GPIO5/GPIO6 assumption, so the diary/changelog are more trustworthy for the corrected pin mapping.

### What I learned
- The strongest historical answer is: `GPIO8/GPIO7/GPIO6` was not arbitrary; it was a correction made after `GPIO5/GPIO6` failed or was identified as the wrong connector.
- The old ticket history already anticipated the exact diagnostic we added now: if CTS hangs or is suspect, add a runtime `printer_flow on|off` command.

### What was tricky to build
- The historical records include both stale design-doc text and corrected diary/changelog text. The corrected chronology matters: early docs can be wrong if later diary steps record a hardware correction.
- The names `TX` and `RX` are overloaded between physical header labels, printer-side labels, and ESP32 UART signal direction. This is why the old firmware added `printer_swap` and why the effective runtime TX pin differs from the nominal header-position define.

### What warrants a second pair of eyes
- Reconcile or mark stale any old design-guide section that still says the K118 uses AtomS3R `GPIO5/GPIO6`.
- Physically inspect the current unit to verify it still uses the same bottom-header cable path as the one debugged in April.

### What should be done in the future
- Treat `GPIO8/GPIO7/GPIO6` as the historically corrected K118/AtomS3R mapping unless the physical current unit proves otherwise.
- Continue testing with `printer_flow off` available, because the old diary shows no-CTS was once the safe initial mode and CTS was a later bitmap-quality optimization.

### Code review instructions
- Historical source of truth:
  - `esp32-s3-m5/ttmp/2026/04/28/STOMS3R-001--stoms3r-atoms3r-lite-thermal-printer-console-firmware/reference/01-diary.md`, Step 3 and Step 5.
- Current copied firmware:
  - `almanach/firmware/atoms3r/main/printer_drv.h`
  - `almanach/firmware/atoms3r/main/printer_drv.c`
- Validate by comparing old and copied driver files, then physically observing `printer_text` / `printer_feed` output.

### Technical details
- Old diary Step 3 corrected mapping:
  - Wrong early assumption: AtomS3R HY2.0-4P `GPIO5/GPIO6`.
  - Corrected K118 bottom-header positions: `GPIO8/GPIO7/GPIO6`.
- Old diary Step 5 CTS change:
  - `.flow_ctrl = UART_HW_FLOWCTRL_CTS`
  - `uart_set_pin(..., rts=UART_PIN_NO_CHANGE, cts=GPIO6)`
- The copied Almanach firmware matches the old `esp32-s3-m5/stoms3r` driver state, so the current no-response issue is not explained by source drift in `printer_drv.c` or `printer_drv.h`.
