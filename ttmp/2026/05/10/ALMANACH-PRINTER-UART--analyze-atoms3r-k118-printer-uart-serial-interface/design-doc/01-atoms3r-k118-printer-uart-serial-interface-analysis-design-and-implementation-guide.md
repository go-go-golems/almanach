---
Title: AtomS3R K118 Printer UART Serial Interface Analysis Design and Implementation Guide
Ticket: ALMANACH-PRINTER-UART
Status: active
Topics:
    - firmware
    - printer
    - uart
    - hardware-diagnostics
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/main/printer_drv.c
      Note: Old firmware UART behavior baseline
    - Path: firmware/atoms3r/main/app_main.c
      Note: Boot-time saved printer settings application
    - Path: firmware/atoms3r/main/nvs_store.c
      Note: Printer settings persistence
    - Path: firmware/atoms3r/main/printer_cmd.c
      Note: Serial console diagnostics and saved printer settings
    - Path: firmware/atoms3r/main/printer_drv.c
      Note: Current UART driver validated at 460800
ExternalSources: []
Summary: Intern-oriented technical guide to the AtomS3R Lite to K118 printer UART serial interface, old-vs-current firmware comparison, diagnostics, false-positive echo detection, and implementation plan.
LastUpdated: 2026-05-10T22:10:00-04:00
WhatFor: Use this guide to understand and fix the Almanach AtomS3R/K118 printer UART serial path.
WhenToUse: Use before changing printer pin mappings, sdkconfig, UART flow control, serial diagnostics, or firmware init order.
---


# AtomS3R K118 Printer UART Serial Interface Analysis Design and Implementation Guide

## Executive Summary

This guide explains the serial path between an M5Stack AtomS3R Lite and the M5Stack K118 thermal printer carrier used by the Almanach/SToMS3R firmware. It is written for a new intern who needs to understand the hardware, firmware files, ESP-IDF UART APIs, diagnostic commands, old-vs-current firmware differences, and the next implementation steps.

The most important finding is that the old firmware currently flashed on the device does receive bytes after transmitting UART data, but the observed bytes look like a TX/RX echo, not a meaningful printer response. For example, sending arbitrary raw bytes `A5 5A` receives exactly `A5 5A`; sending `01 02 03 04` receives exactly `01 02 03 04`. The old `printer_probe` interprets this echo as `PRINTER RESPONDED`, because it reads the first echoed command byte as if it were the printer's status byte.

A second major finding is that the printer's own settings page reports a baud rate of `460800`, while both the old and copied firmware boot at `9600` unless saved startup settings override it. If the K118's TTL UART is really persisted at `460800`, then text/feed at `9600` will not work. Changing only the ESP32 side with `printer_baud 460800` is the correct recovery test; it was run on the old flashed firmware and still produced echo-like RX bytes, so baud alignment and echo classification must both be handled.

The copied Almanach firmware uses the same historical printer pin mapping and largely the same driver code, but it was recently observed to receive no bytes at all on the same status probes. This suggests the problem may not be simply "wrong pinout." The investigation should separate three phenomena that are easy to confuse:

- **UART TX works**: bytes leave the ESP32 UART peripheral.
- **UART RX sees echo/readback**: bytes appear on RX but may just be the transmitted bytes looped back by wiring, cable, module behavior, or an unpowered board.
- **Printer protocol works**: the K118 returns semantically valid responses or physically feeds/prints paper.

The immediate operational fix is to run the printer UART at `460800` and persist that baud in firmware NVS settings. The copied Almanach firmware has now been hardware-validated at 460800, including `printer_status`, `printer_get_baud`, `printer_probe`, text output, and feed output. The remaining engineering recommendation is to upgrade diagnostics so echo/loopback is classified separately from real printer replies, because the old 9600-baud probe produced misleading echo-like false positives.

## Problem Statement

The copied Almanach firmware needs reliable serial control of the K118 printer. The user reports that the old firmware in:

```text
/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r
```

works at least for communication with the printer, while feed/text did not physically do anything. The copied firmware in:

```text
/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r
```

recently timed out on serial status probes. The user asked whether `sdkconfig` or another serial-interface detail might explain the difference.

We need to answer:

1. What is the intended hardware pinout?
2. What does the old firmware actually prove?
3. Which parts of `sdkconfig` can influence the serial interface?
4. Which files implement the UART path?
5. How should diagnostics distinguish echo from printer responses?
6. What implementation steps should be done next?

## System Overview

### Components

The system has five layers:

```text
┌─────────────────────────────────────────────────────────────┐
│ Human / test operator                                       │
│ - serial REPL over USB Serial/JTAG                          │
│ - commands: printer_probe, printer_raw, printer_text, ...    │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ ESP-IDF console layer                                       │
│ - esp_console command registration                          │
│ - USB Serial/JTAG console, not UART0/1                      │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ Printer command layer: printer_cmd.c                        │
│ - parses commands                                           │
│ - calls printer driver API                                  │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ Printer driver layer: printer_drv.c/.h                      │
│ - installs UART1 driver                                     │
│ - maps pins through ESP-IDF pin matrix                      │
│ - sends ESC/POS/K118 command bytes                          │
│ - reads UART RX bytes                                       │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ Hardware layer                                               │
│ - AtomS3R Lite bottom-header pins                           │
│ - K118 cable/carrier board                                  │
│ - thermal printer controller and mechanism                  │
└─────────────────────────────────────────────────────────────┘
```

The key point is that the serial console and the printer UART are separate. The operator types commands over USB Serial/JTAG. The printer driver talks to the K118 over UART1 on external GPIO pins.

### Printer UART Data Path

At runtime, the firmware default is:

```text
USB Serial/JTAG console        -> operator shell, prompt `stoms3r>`
UART1 TX effective GPIO7       -> bytes sent toward K118 printer RX
UART1 RX effective GPIO8       -> bytes read from K118 printer TX or echo path
UART1 CTS GPIO6                -> optional printer busy / clear-to-send input
Default firmware baud          -> 9600 8N1 unless saved settings override it
Reported printer settings baud -> 460800 on the physical printer settings page
```

The code defines nominal header positions as:

```c
#define PRINTER_UART_NUM   UART_NUM_1
#define PRINTER_TX_GPIO    8
#define PRINTER_RX_GPIO    7
#define PRINTER_CTS_GPIO   6
#define PRINTER_BAUD       9600
```

Then it swaps TX/RX on boot:

```c
printer_drv_swap_pins(true);
```

So the effective runtime mapping becomes:

```text
UART TX = PRINTER_RX_GPIO = GPIO7
UART RX = PRINTER_TX_GPIO = GPIO8
CTS     = PRINTER_CTS_GPIO = GPIO6
```

This naming is confusing but historically intentional. `PRINTER_TX_GPIO` means "the physical header position historically labeled TX," not necessarily the current ESP32 UART TX after software swapping.

## Hardware Pinout History

The K118 kit was designed for the older ATOM Lite bottom header:

```text
ATOM Lite / K118 original physical positions:
  TX  = GPIO23
  RX  = GPIO33
  CTS = GPIO19
```

On AtomS3R Lite, the same physical bottom-header positions map to different ESP32-S3 GPIOs:

```text
AtomS3R Lite same physical positions:
  TX-position  = GPIO8
  RX-position  = GPIO7
  CTS-position = GPIO6
```

The historical ticket diary in:

```text
/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/ttmp/2026/04/28/STOMS3R-001--stoms3r-atoms3r-lite-thermal-printer-console-firmware/reference/01-diary.md
```

records that an earlier implementation incorrectly assumed the AtomS3R HY2.0-4P/Grove-style pins `GPIO5/GPIO6`. That was corrected to `GPIO8/GPIO7/GPIO6` after the K118 bottom-header mapping was identified.

### Why not GPIO5/GPIO6?

The AtomS3R Lite has multiple physical connector/pin groups. The K118 cable uses the bottom-header positions that match the ATOM Lite printer kit, not the separate HY2.0-4P/Grove-style connector.

```text
Wrong early assumption:
  AtomS3R HY2.0-4P port -> GPIO5/GPIO6

Corrected K118 path:
  AtomS3R bottom-header positions -> GPIO8/GPIO7/GPIO6
```

Therefore, `GPIO8/GPIO7/GPIO6` is not arbitrary; it is the historically corrected mapping for the K118 on AtomS3R Lite.

## Firmware File Map

### Old baseline firmware

```text
/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/
```

Key files:

```text
main/printer_drv.h       Pin constants and printer driver API.
main/printer_drv.c       UART install, pin mapping, TX/RX helpers, ESC/POS commands.
main/printer_cmd.c       Console commands for printer diagnostics and actions.
main/app_main.c          Init order: NVS, WiFi, printer, console.
sdkconfig.defaults       Project defaults for ESP-IDF configuration.
sdkconfig                Generated resolved config used by the build.
build.sh                 Sources ESP-IDF 5.4.1 via repo .envrc.
```

### Current copied Almanach firmware

```text
/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/
```

Key files:

```text
main/printer_drv.h       Same printer pins plus temporary diagnostic flow API.
main/printer_drv.c       Same driver plus runtime printer_flow support in current working tree.
main/printer_cmd.c       Console command registration, including diagnostic commands.
main/app_main.c          Adds display, BLE provisioning, button handling, new WiFi onboarding.
main/display_app.cpp     Display UI; initialized before printer in current app_main.c.
main/display_hal.cpp     LCD SPI pin setup: CS 14, SCK 15, MOSI 21, DC 42, RST 48.
main/backlight.cpp       Backlight I2C setup; GPIO7 gate disabled by default.
sdkconfig.defaults       Adds BLE provisioning and button timing defaults.
sdkconfig                Generated resolved config with display/BLE symbols.
build.sh                 Prefers ESP-IDF 5.4.2 if no IDF is already active.
```

### Historical ticket evidence

```text
/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/ttmp/2026/04/28/STOMS3R-001--stoms3r-atoms3r-lite-thermal-printer-console-firmware/reference/01-diary.md
```

Important historical steps:

- Step 3: corrected printer GPIO pin mapping from `GPIO5/GPIO6` to `GPIO8/GPIO7/GPIO6`.
- Step 5: enabled CTS flow control after banding experiments.
- Later steps: added baud/status/density/speed diagnostics.

## ESP-IDF UART APIs Used

The driver uses standard ESP-IDF UART APIs.

### Install the driver

```c
uart_driver_install(PRINTER_UART_NUM, 1024, 2048, 0, NULL, 0);
```

Meaning:

- TX/RX ring buffers are allocated.
- `PRINTER_UART_NUM` is `UART_NUM_1`.
- RX buffer is required for status/probe reads.

### Configure UART parameters

```c
uart_config_t uart_config = {
    .baud_rate  = PRINTER_BAUD,
    .data_bits  = UART_DATA_8_BITS,
    .parity     = UART_PARITY_DISABLE,
    .stop_bits  = UART_STOP_BITS_1,
    .flow_ctrl  = UART_HW_FLOWCTRL_CTS,
    .source_clk = UART_SCLK_DEFAULT,
};

uart_param_config(PRINTER_UART_NUM, &uart_config);
```

Meaning:

- 9600 baud.
- 8 data bits.
- no parity.
- 1 stop bit.
- CTS flow control in the current/old final state.

### Map signals to pins

```c
uart_set_pin(PRINTER_UART_NUM,
             tx_gpio,
             rx_gpio,
             UART_PIN_NO_CHANGE,
             cts_gpio);
```

Meaning:

- ESP-IDF's pin matrix maps UART TX and RX signals to arbitrary valid GPIOs.
- RTS is not used.
- CTS may be GPIO6 or detached with `UART_PIN_NO_CHANGE` in diagnostic builds.

### Write bytes

```c
int written = uart_write_bytes(PRINTER_UART_NUM, data, len);
uart_wait_tx_done(PRINTER_UART_NUM, pdMS_TO_TICKS(timeout_ms));
```

Meaning:

- Bytes are queued for UART transmission.
- `uart_wait_tx_done` waits until the hardware has shifted them out.
- At 9600 baud, large bitmaps take a long time; the code uses length-based timeouts.

### Read bytes

```c
int n = uart_read_bytes(PRINTER_UART_NUM, buf, sizeof(buf), pdMS_TO_TICKS(timeout_ms));
```

Meaning:

- Reads from the UART RX ring buffer.
- Any received bytes could be real printer replies, stale bytes, echoed TX bytes, noise, or loopback.
- Existing diagnostics did not sufficiently distinguish those cases.

## Serial Console Commands

The command layer lives in `printer_cmd.c`. Important commands:

```text
printer_probe              Query status bytes and send ESC @.
printer_raw <hex>          Send arbitrary raw bytes, then drain RX.
printer_status             Query K118 4-byte status packet using GS a n.
printer_get_baud           Query printer-side baud using GS g 7.
printer_text <text>        Send plain text plus line feed.
printer_feed [n]           Send ESC d n feed command.
printer_swap <on|off>      Toggle software TX/RX pin swap.
printer_flow <cts|off>     New diagnostic command in current working tree.
```

### Current weakness in `printer_probe`

The old/current probe logic is vulnerable to echo false positives.

Simplified old logic:

```pseudocode
for n in [1, 2, 3, 4]:
    drain_rx()
    send_bytes([0x10, 0x04, n])
    resp = read_one_byte(timeout=500ms)
    if got resp:
        mark any_response = true
        print "Status n response: resp"
```

If the RX line receives an echo of `[0x10, 0x04, n]`, the first byte read is `0x10`. The probe classifies `0x10` as a printer status response even though it is just the first transmitted byte. Later, `drain_rx()` reads the remaining echoed bytes `0x04 n`.

Observed old firmware evidence:

```text
TX: 10 04 01
read response: 10
drain later: 04 01
```

That sequence is exactly the transmitted command split across reads.

## Observed Evidence

### Old currently-flashed firmware boot

From `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r` monitor:

```text
Project name:     stoms3r
App version:      b1e12e3-dirty
ESP-IDF:          v5.4.1
Printer UART1 ready: TX=8 RX=7 CTS=6 baud=9600
Swapping TX/RX pins: TX=GPIO7 RX=GPIO8 CTS=GPIO6 (SWAPPED) baud=9600
```

### Old firmware `printer_probe`

Observed:

```text
printer_probe
Draining RX buffer...
RX 2 bytes: 1B 40
Drained 2 stale bytes

TX 3 bytes: 10 04 01
Status query n=1 response: 0x10
RX 2 bytes: 04 01

TX 3 bytes: 10 04 02
Status query n=2 response: 0x10
RX 2 bytes: 04 02

TX 3 bytes: 10 04 03
Status query n=3 response: 0x10
RX 2 bytes: 04 03

TX 3 bytes: 10 04 04
Status query n=4 response: 0x10

Sending ESC @ (init)...
TX 2 bytes: 1B 40
RX 4 bytes: 04 04 1B 40
Got 4 bytes after init (printer is alive!)

=== Result: PRINTER RESPONDED ===
```

Interpretation:

- `1B 40` after boot is the echo of the boot reset command.
- `10 04 n` status commands are echoed as `10 04 n`.
- The probe reads the first echoed byte `10` as if it were status.
- Later drains see the remaining bytes.
- `PRINTER RESPONDED` is likely a false positive.

### Old firmware raw echo probes

Observed:

```text
printer_raw A55A
TX 2 bytes: A5 5A
RX 2 bytes: A5 5A

printer_raw 01020304
TX 4 bytes: 01 02 03 04
RX 4 bytes: 01 02 03 04
```

A real thermal printer controller is not expected to echo arbitrary binary bytes. This is strong evidence for echo/loopback rather than protocol-level printer communication.

### Old firmware `printer_status`

Observed:

```text
printer_status
TX 3 bytes: 1D 61 00
GS a status: got 3 bytes
Error: ESP_ERR_TIMEOUT
```

Interpretation:

- The command length is 3 bytes.
- The driver expected 4 bytes.
- It received 3 bytes, likely the echoed command itself.

### Old firmware `printer_get_baud`

Observed at 9600:

```text
printer_get_baud
TX 3 bytes: 1D 67 37
esp32_baud=9600 printer_baud=7 raw=g7
```

Observed after setting the ESP32 side to 460800:

```text
printer_baud 460800
ESP32 UART baud rate set to 460800 (printer was NOT commanded)

printer_get_baud
TX 3 bytes: 1D 67 37
esp32_baud=460800 printer_baud=7 raw=g7
```

Interpretation:

- `0x67 0x37` is ASCII `g7`.
- The raw value `g7` appears to be the suffix of the command itself, not a meaningful baud response.
- The echo-like pattern occurs at both 9600 and 460800, so RX bytes alone are not proof of printer protocol communication.

### Physical printer settings page baud

The user reported that the printer's own settings page says:

```text
baudrate 460800
```

This matters because the firmware boot default remains 9600 when no saved startup printer settings are present. If the printer persists 460800 internally, then these commands are different:

```text
printer_baud 460800
```

Changes only the ESP32 UART side. Use this when the printer is already known to be at 460800.

```text
set_baudrate 460800
```

Sends the K118 baud-rate command to the printer at the current ESP32 baud, then changes the ESP32 side. Use this only when the printer currently understands the existing baud and you intentionally want to change the printer's persisted baud.

The next intern should treat 460800 as the likely target baud for physical text/feed tests on this device, but should still use echo-aware diagnostics because echo can appear at either baud.

## sdkconfig and Build Comparison

### Similar defaults

Both old and current defaults configure:

```text
CONFIG_ESP_CONSOLE_USB_SERIAL_JTAG=y
CONFIG_SPIRAM=y
CONFIG_SPIRAM_MODE_OCT=y
CONFIG_SPIRAM_USE_CAPS_ALLOC=y
CONFIG_SPIRAM_MALLOC_ALWAYSINTERNAL=16384
CONFIG_PARTITION_TABLE_CUSTOM=y
CONFIG_PARTITION_TABLE_CUSTOM_FILENAME="partitions.csv"
CONFIG_ESP_DEFAULT_CPU_FREQ_MHZ_240=y
CONFIG_ESP_DEFAULT_CPU_FREQ_MHZ=240
```

The most serial-relevant item is USB Serial/JTAG console. This is correct because it prevents the interactive REPL from occupying UART1.

### Important differences

Old `build.sh` sources:

```bash
source /home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/.envrc
```

That `.envrc` contains:

```bash
source ~/esp/esp-idf-5.4.1/export.sh
```

Current `build.sh` prefers:

```bash
~/esp/esp-idf-5.4.2/export.sh
```

Current firmware adds:

```text
CONFIG_BT_ENABLED=y
CONFIG_BT_NIMBLE_ENABLED=y
CONFIG_ESP_PROTOCOMM_SUPPORT_SECURITY_VERSION_1=y
CONFIG_ALMANACH_ATOMS3R_DISPLAY_ENABLE=y
CONFIG_ALMANACH_ATOMS3R_BACKLIGHT_I2C_ENABLE=y
```

Current firmware also adds app integration:

```text
display_app_init()
button_input_start()
provisioning_mgr_init()
provisioning_cmd_register()
BLE provisioning stack
```

### What sdkconfig probably does *not* explain directly

- Printer pins are not selected by sdkconfig; they are hard-coded in `printer_drv.h`.
- The console is USB Serial/JTAG in both builds; it is not stealing UART1.
- Current display pins do not use GPIO7/8/6 by default:
  - LCD CS 14
  - LCD SCK 15
  - LCD MOSI 21
  - LCD DC 42
  - LCD RST 48
  - backlight I2C SCL 0
  - backlight I2C SDA 45
  - GPIO7 backlight gate disabled.
- Bluetooth HCI UART config symbols in `sdkconfig` are not necessarily active because the controller transport is VHCI, not UART H4, in the current BLE provisioning build.

### What sdkconfig/build may explain indirectly

- ESP-IDF 5.4.1 vs 5.4.2 could change UART driver behavior, pin matrix defaults, or flow-control corner cases.
- Additional BLE/display tasks can change timing and memory, but they should not directly prevent UART RX.
- Current app initializes display before printer; if a future display/backlight config enables GPIO7 gate, it could conflict, but the present config disables that path.

## Proposed Diagnostic Classifier

The next firmware diagnostic should classify RX bytes into categories.

### Categories

```text
NO_RX
  No bytes received after TX.

ECHO_EXACT
  RX bytes exactly match TX bytes.

ECHO_PREFIX
  RX begins with TX bytes or a suffix/prefix of TX bytes.

POSSIBLE_PROTOCOL_REPLY
  RX does not match TX and has plausible length/value for the command.

VALID_K118_STATUS
  RX matches the expected structure for a specific K118 status command.

PHYSICAL_PRINT_ONLY
  No meaningful RX, but printer visibly feeds/prints.
```

### Raw diagnostic pseudocode

```pseudocode
function probe_raw_echo(tx_bytes):
    drain_rx_until_quiet()
    write(tx_bytes)
    wait_tx_done()
    rx = read_all_for(300ms)

    if rx.length == 0:
        return {class: NO_RX, tx, rx}

    if rx == tx_bytes:
        return {class: ECHO_EXACT, tx, rx}

    if rx is prefix/suffix/substring of tx_bytes:
        return {class: ECHO_PARTIAL, tx, rx}

    return {class: NON_ECHO_RX, tx, rx}
```

### Safer status probe pseudocode

```pseudocode
function query_dle_eot_status(n):
    cmd = [0x10, 0x04, n]
    result = probe_raw_echo(cmd)

    if result.class == NO_RX:
        return {ok: false, reason: "no RX bytes"}

    if result.class starts with ECHO:
        return {
            ok: false,
            reason: "RX echoed transmitted command; not a printer status reply",
            tx: cmd,
            rx: result.rx,
        }

    # If K118 really returns one status byte, validate it here.
    # Do not accept the first byte if it equals cmd[0] and remaining bytes match cmd[1:].
    for each byte in result.rx:
        if plausible_status_byte(byte, n):
            return {ok: true, status: byte, raw: result.rx}

    return {ok: false, reason: "non-echo but invalid status", raw: result.rx}
```

### Physical output diagnostic pseudocode

The firmware cannot observe paper movement directly, so the console should ask the operator.

```pseudocode
function test_physical_tx():
    print "Sending visible feed and text. Watch paper."
    printer_reset()
    printer_text("UART TEST")
    printer_feed(3)
    print "Did paper move or print? yes/no"
```

If adding interactive prompts is too much, document the operator step in the test playbook.

## Validated Resolution

The copied Almanach firmware works with the K118 printer when the ESP32 UART baud matches the printer settings page baud of `460800`. The validated sequence was:

```text
printer_baud 460800
printer_flow off
printer_probe
printer_text NEW_FW_460800_NOCTS
printer_feed 3
printer_status
printer_get_baud
printer_flow cts
printer_probe
```

Then startup settings were saved with:

```text
printer_settings_save 460800 20 80 31
```

After reboot, the copied firmware logged:

```text
Applying saved printer settings: baud=460800 density=20 speed=80 graphics_mode=31
```

and hardware validation returned:

```text
printer_get_baud -> esp32_baud=460800 printer_baud=460800 raw=uart baudrate: 460800
printer_status   -> raw: 14 00 00 0F
```

This resolves the immediate serial communication problem. The pinout and ESP-IDF 5.4.2 are acceptable. The root cause was the printer-side persisted baud being 460800 while firmware booted at 9600 without saved settings.

## Proposed Implementation Plan

### Phase 1: Make diagnostics echo-aware

Modify `printer_drv.c` / `printer_cmd.c` to add:

```text
printer_echo_probe <hex>
printer_probe2
```

`printer_echo_probe` should:

- send arbitrary bytes,
- read all RX bytes for a fixed window,
- compare TX and RX,
- print a classification.

Example output:

```text
TX: A5 5A
RX: A5 5A
classification: ECHO_EXACT
meaning: UART RX saw transmitted bytes; this is not proof of printer protocol response.
```

`printer_probe2` should:

- run raw echo probes first,
- reject DLE EOT echo false positives,
- separately report physical-output commands to try.

### Phase 2: Run source/version isolation matrix

Build and test four combinations:

```text
1. old source     + ESP-IDF 5.4.1
2. old source     + ESP-IDF 5.4.2
3. current source + ESP-IDF 5.4.1
4. current source + ESP-IDF 5.4.2
```

For each combination record:

```text
boot ESP-IDF version
printer UART ready log
printer_raw A55A result at boot baud
printer_raw A55A result after printer_baud 460800
printer_probe result at boot baud
printer_probe result after printer_baud 460800
printer_status result
printer_get_baud result
printer_text physical result after printer_baud 460800
printer_feed physical result after printer_baud 460800
```

Suggested evidence table:

| Source | IDF | Baud | Raw echo? | Probe classification | Status valid? | Physical feed/text? |
|---|---:|---:|---|---|---|---|
| old | 5.4.1 | 9600 | yes/no | echo/real/none | yes/no | yes/no |
| old | 5.4.1 | 460800 | yes/no | echo/real/none | yes/no | yes/no |
| old | 5.4.2 | 9600 | yes/no | echo/real/none | yes/no | yes/no |
| old | 5.4.2 | 460800 | yes/no | echo/real/none | yes/no | yes/no |
| current | 5.4.1 | 9600 | yes/no | echo/real/none | yes/no | yes/no |
| current | 5.4.1 | 460800 | yes/no | echo/real/none | yes/no | yes/no |
| current | 5.4.2 | 9600 | yes/no | echo/real/none | yes/no | yes/no |
| current | 5.4.2 | 460800 | yes/no | echo/real/none | yes/no | yes/no |

### Phase 3: Isolate current integration features

If current source differs from old source even under the same IDF version, test feature toggles:

```text
Current minimal printer firmware:
  display disabled
  BLE disabled
  button disabled
  web optional
  printer + console only

Current full firmware:
  display enabled
  BLE provisioning enabled
  button enabled
  web enabled
```

Likely Kconfig toggles to create or use:

```text
CONFIG_ALMANACH_ATOMS3R_DISPLAY_ENABLE=n
CONFIG_BT_ENABLED=n
CONFIG_ALMANACH_ATOMS3R_BUTTON_INPUT_ENABLE=n   # if added
CONFIG_ALMANACH_ATOMS3R_BLE_PROVISIONING_ENABLE=n # if added
```

The current project may not yet have all of these toggles; adding them would make hardware isolation easier.

### Phase 4: Hardware validation

Use a logic analyzer if available.

Probe points:

```text
GPIO7: expected ESP32 UART TX in swapped mode
GPIO8: expected ESP32 UART RX in swapped mode
GPIO6: expected CTS/busy input
GND: shared ground
```

Tests:

```text
printer_raw A55A
printer_raw 55AA
printer_text UART_TEST
printer_feed 3
```

Expected logic analyzer results:

- GPIO7 should show TX waveform for every command.
- GPIO8 should not show the exact same waveform unless there is echo/loopback.
- If printer replies, GPIO8 should show reply timing from the printer, not simultaneous or near-identical command echo.
- GPIO6 should indicate busy/ready if CTS is wired and active.

### Phase 5: Decide final runtime behavior

Once physical behavior is known:

- If only TX matters and printer never returns useful status, make status commands best-effort and do not block print success on them.
- If RX is echo-only, label it clearly and avoid reporting "printer responded."
- If CTS is reliable, keep CTS for bitmap printing.
- If CTS is unreliable, default to no CTS for text/feed and use conservative chunking or operator-selectable flow modes for bitmaps.

## Proposed Code Sketches

### Echo classifier helper

```c
typedef enum {
    PRINTER_RX_NONE,
    PRINTER_RX_ECHO_EXACT,
    PRINTER_RX_ECHO_PARTIAL,
    PRINTER_RX_NON_ECHO,
} printer_rx_class_t;

typedef struct {
    printer_rx_class_t cls;
    uint8_t tx[32];
    size_t tx_len;
    uint8_t rx[64];
    size_t rx_len;
} printer_rx_probe_t;

esp_err_t printer_drv_probe_echo(const uint8_t *tx, size_t tx_len,
                                 printer_rx_probe_t *out)
{
    printer_drv_drain_rx();
    send_bytes(tx, tx_len);
    read_all_available_for_window(out->rx, sizeof(out->rx), &out->rx_len, 300);

    if (out->rx_len == 0) {
        out->cls = PRINTER_RX_NONE;
    } else if (out->rx_len == tx_len && memcmp(out->rx, tx, tx_len) == 0) {
        out->cls = PRINTER_RX_ECHO_EXACT;
    } else if (rx_matches_tx_subsequence(out->rx, out->rx_len, tx, tx_len)) {
        out->cls = PRINTER_RX_ECHO_PARTIAL;
    } else {
        out->cls = PRINTER_RX_NON_ECHO;
    }
    return ESP_OK;
}
```

### Improved command output

```text
stoms3r> printer_echo_probe A55A
TX: A5 5A
RX: A5 5A
classification: ECHO_EXACT
verdict: UART RX sees transmitted bytes; this is not a printer protocol reply.

stoms3r> printer_probe2
DLE EOT 1: ECHO_EXACT (10 04 01) -- rejected
DLE EOT 2: ECHO_EXACT (10 04 02) -- rejected
DLE EOT 3: ECHO_EXACT (10 04 03) -- rejected
DLE EOT 4: ECHO_EXACT (10 04 04) -- rejected
verdict: UART echo/loopback present, no validated K118 status response.
next: visually test printer_text and printer_feed; inspect power/cable.
```

## Common Failure Modes and Interpretations

### Case A: No RX bytes, no paper movement

Likely causes:

- wrong TX pin,
- wrong cable orientation,
- printer not powered,
- no shared ground,
- UART peripheral not mapped to expected pins,
- another peripheral reconfigured the pin.

Next actions:

- logic analyzer on GPIO7,
- physical cable/power check,
- build minimal current firmware with display/BLE disabled.

### Case B: Echo RX bytes, no paper movement

Likely causes:

- TX/RX loopback through cable/module/board,
- printer unpowered but lines electrically coupled,
- UART RX pin wired to TX path instead of printer TX,
- existing probe false positive.

Next actions:

- do not trust `printer_probe` old verdict,
- inspect wiring,
- confirm 12V supply,
- use logic analyzer.

### Case C: No meaningful RX, but paper feeds/prints

Likely causes:

- TX path is correct,
- printer does not return status on this command set,
- printer TX line not connected,
- RX pin/cable wrong but TX is fine.

Next actions:

- treat RX status as optional,
- focus on print reliability and bitmap pacing,
- fix `printer_probe` messaging to say "TX physical output works, RX status unavailable."

### Case D: Valid structured status and physical output

Desired state.

Next actions:

- keep pinout,
- keep or tune CTS,
- write regression playbook,
- update docs.

## Recommended Intern Work Plan

1. Read this guide and the diary.
2. Run old firmware monitor and reproduce raw echo evidence.
3. Modify diagnostics to classify echo.
4. Build old/current firmware under controlled ESP-IDF versions.
5. Run the source/version matrix.
6. Ask operator to observe paper movement for each physical test.
7. If available, capture logic analyzer traces.
8. Decide whether to treat K118 RX/status as reliable, echo-only, or disconnected.
9. Update docs and commit the diagnostic improvements.

## Command Reference

### Monitor old currently-flashed firmware

```bash
cd /home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r
./build.sh /dev/ttyACM0 monitor
```

### Monitor current copied firmware

```bash
cd /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r
./build.sh /dev/ttyACM0 monitor
```

### Build current copied firmware

```bash
cd /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r
./build.sh /dev/ttyACM0 build
```

### Flash current copied firmware

```bash
cd /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r
./build.sh /dev/ttyACM0 flash
```

### Serial diagnostic commands

```text
printer_raw A55A
printer_raw 01020304
printer_probe
printer_status
printer_get_baud
printer_baud 460800  # match this physical printer's reported settings page baud
printer_text HELLO_FROM_UART
printer_feed 3
printer_swap on
printer_swap off
printer_flow cts     # current diagnostic working tree only
printer_flow off     # current diagnostic working tree only
```

## File Reference Index

| File | Why it matters |
|---|---|
| `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/main/printer_drv.c` | Old baseline UART driver and current flashed behavior source. |
| `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/main/printer_drv.h` | Old baseline pin constants. |
| `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/main/printer_cmd.c` | Old baseline serial console printer commands. |
| `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/main/app_main.c` | Old init order and console setup. |
| `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/stoms3r/sdkconfig.defaults` | Old project defaults. |
| `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/.envrc` | Old ESP-IDF version source: 5.4.1. |
| `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/printer_drv.c` | Current copied driver plus diagnostic flow-control changes. |
| `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/printer_drv.h` | Current copied pin constants and driver API. |
| `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/printer_cmd.c` | Current command registration and diagnostics. |
| `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c` | Current init order with display/BLE/button additions. |
| `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/display_hal.cpp` | Confirms current display SPI pins do not use GPIO7/8/6. |
| `/home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/backlight.cpp` | Confirms GPIO7 backlight gate is disabled by default; I2C uses GPIO0/45. |
| `/home/manuel/workspaces/2025-12-21/echo-base-documentation/esp32-s3-m5/ttmp/2026/04/28/STOMS3R-001--stoms3r-atoms3r-lite-thermal-printer-console-firmware/reference/01-diary.md` | Historical pinout correction and CTS evolution evidence. |

## Open Questions

1. Does the printer physically feed or print under old firmware after `printer_baud 460800` when `printer_text` or `printer_feed` is sent?
2. Does the RX echo occur if the K118 is unplugged or unpowered?
3. Does current copied firmware receive echo if built with ESP-IDF 5.4.1?
4. Does old firmware still receive echo if built with ESP-IDF 5.4.2?
5. Is GPIO6 CTS asserted in a polarity compatible with ESP-IDF UART CTS?
6. Does the K118 module ever return useful status bytes on this hardware, or should RX status be considered unsupported?
7. Should this specific device's firmware save ESP32 startup baud as 460800 so boot-time printer commands match the printer's persisted settings?

## Proposed Definition of Done

This investigation should be considered complete when:

- Diagnostics classify echo separately from printer replies.
- Device startup settings persist baud 460800 or the firmware otherwise applies the correct baud before printer commands.
- The operator records physical paper movement results for text/feed tests.
- The final firmware no longer reports `PRINTER RESPONDED` for exact TX echo.
- The guide and diary are updated with final conclusions.
- If code changes are made, they are committed with focused firmware and documentation commits.
