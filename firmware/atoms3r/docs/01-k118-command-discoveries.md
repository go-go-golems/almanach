# K118 / ATOM Printer Command Discoveries

This document collects the practical discoveries from the M5Stack ATOM Printer command manual and the SToMS3R debugging sessions. It is firmware-local documentation for the `stoms3r/` project; the source reference lives in the research ticket:

- Original PDF: `../0090-m5printer-research/docs/ATOM_PRINTER_CMD_v1.06.pdf`
- Extracted text: `../0090-m5printer-research/docs/ATOM_PRINTER_CMD_v1.06.txt`
- English reference: `../0090-m5printer-research/docs/ATOM_PRINTER_CMD_v1.06.en.md`

## The high-value discovery

The Chinese spec is not just a generic ESC/POS command table. It includes a set of vendor-specific `ESC ## ...` commands and status queries that expose the exact knobs we need for bitmap debugging:

- UART baud rate setting and query
- 4-byte status query with buffer-full and overheat bits
- print density
- mechanism print speed
- graphics print mode, including adaptive-speed graphics printing
- software flow control enable/disable
- temperature query
- print-head voltage query
- self-test and function-list print commands

The standard ESC/POS commands are still useful, but the vendor-specific commands are the most important part of the manual for SToMS3R.

## Notation

| Symbol | Byte |
|---|---:|
| `ESC` | `0x1B` |
| `GS` | `0x1D` |
| `FS` | `0x1C` |
| `DLE` | `0x10` |
| `NUL` | `0x00` |
| `##` | two literal `#` bytes: `0x23 0x23` |

Most multi-byte values in the `ESC ## ...` vendor commands are little-endian.

## Baud-rate control

### Set serial baud rate

Command:

```text
ESC ## SBDR baudrate
Hex: 1B 23 23 53 42 44 52 <baudrate-le32>
```

`baudrate` is a 4-byte little-endian integer.

Examples:

| Baud | Integer | Payload bytes |
|---:|---:|---|
| 9600 | `0x00002580` | `80 25 00 00` |
| 115200 | `0x0001C200` | `00 C2 01 00` |

Full command for 115200:

```text
1B 23 23 53 42 44 52 00 C2 01 00
```

SToMS3R command:

```text
set_baudrate 115200
```

This sends the printer-side command at the current UART baud, waits for TX to drain, delays briefly, then switches the ESP32 UART to the same baud.

Recovery command:

```text
printer_baud <rate>
```

This changes only the ESP32 UART side. Use it if the ESP32 and printer become out of sync.

### Query printer-side baud rate

Command:

```text
GS g 7
Hex: 1D 67 37
```

Returns a text string like:

```text
uart baudrate: 115200
```

SToMS3R command:

```text
printer_get_baud
```

Web API:

```text
GET /api/printer/baud
```

## Status queries

### Real-time one-byte status

Command:

```text
DLE EOT n
Hex: 10 04 n
Range: 1 <= n <= 4
```

This is what `printer_probe` uses. It gives quick one-byte status responses:

| `n` | Meaning |
|---:|---|
| 1 | printer status / online-offline |
| 2 | offline status / cover / feed key / paper / error |
| 3 | error status / cutter / unrecoverable / temperature-voltage |
| 4 | paper sensor status |

This is useful for connectivity probing because it is simple and real-time.

### Four-byte status with buffer-full bit

Command:

```text
GS a n
Hex: 1D 61 n
```

Returns 4 bytes. The most important field is byte 1 bit 3:

```text
byte 1, bit 3 = 0: printer has buffer space, can accept data
byte 1, bit 3 = 1: printer buffer full, cannot accept data
```

Other useful fields:

| Byte | Bit | Meaning |
|---:|---:|---|
| 1 | 3 | buffer full |
| 1 | 5 | cover open |
| 1 | 6 | feed key active |
| 2 | 3 | cutter error |
| 2 | 5 | auto-recoverable error |
| 2 | 6 | overheated |
| 3 | 0-1 | paper near end |
| 3 | 2-3 | paper out |

SToMS3R command:

```text
printer_status
```

Web API:

```text
GET /api/printer/status
```

Why it matters: this is the direct way to observe whether bitmap printing is overrunning the printer input buffer or hitting an overheat state.

## Raster bitmap printing

Command:

```text
GS v 0 m xL xH yL yH d1...dk
Hex: 1D 76 30 m xL xH yL yH d1...dk
```

Parameters:

```text
x = xL + xH * 256      // width in bytes
y = yL + yH * 256      // height in dots
k = x * y              // payload byte count
```

For the K118 58mm mechanism:

```text
width = 384 dots
bytes_per_row = 384 / 8 = 48
xL = 48
xH = 0
```

Modes:

| `m` | Meaning | Vertical resolution | Horizontal resolution |
|---:|---|---:|---:|
| 0 / 48 | normal | 200 DPI | 200 DPI |
| 1 / 49 | double height | 200 DPI | 100 DPI |
| 2 / 50 | double width | 100 DPI | 200 DPI |
| 3 / 51 | double height + double width | 100 DPI | 100 DPI |

Important spec notes:

- Bit value `1` prints a dot; bit value `0` leaves the dot blank.
- `ESC a` alignment affects raster bitmaps.
- Text modes such as bold, underline, inverse, and character scaling do not affect raster bitmaps.
- In standard mode, the command is valid only when the printer buffer is empty.

SToMS3R uses browser-side bit packing, MSB first, then sends one complete raster command through the UART driver.

## Print density

Command:

```text
ESC ## STDP n
Hex: 1B 23 23 53 54 44 50 n
```

Range:

```text
0 <= n <= 39
39 = darkest / highest density
```

SToMS3R command:

```text
printer_density 20
```

Web API:

```text
POST /api/printer/density
{"density":20}
```

Why it matters: density changes both darkness and current draw. Dense black output at high density is a power and thermal stress test. If bitmap stripes correlate with power sag or heat, lower density may improve reliability.

Useful test values:

```text
printer_density 12
printer_density 20
printer_density 28
printer_density 39
```

## Print speed

Command:

```text
ESC ## STSP n
Hex: 1B 23 23 53 54 53 50 n
```

Valid values from the manual:

```text
25, 30, 37, 50, 56, 62, 70, 80, 90, 100, 120, 150, 180, 200, 220
```

The manual describes these as mm/s values.

SToMS3R command:

```text
printer_speed 80
```

Web API:

```text
POST /api/printer/speed
{"speed":80}
```

Why it matters: speed controls mechanism timing. Slower speeds may allow more stable heating; faster speeds reduce total print time but can increase current/thermal stress.

## Graphics print mode

Command:

```text
ESC ## SPSM n
Hex: 1B 23 23 53 50 53 4D n
```

Values:

| `n` | Meaning |
|---:|---|
| 30 | BLE graphics printing |
| 31 | adaptive-speed graphics printing |
| 32 | constant-speed graphics printing |

SToMS3R command:

```text
printer_graphics_mode 31
```

Web API:

```text
POST /api/printer/graphics-mode
{"mode":31}
```

Why it matters: this is probably the most interesting vendor command for bitmap stripe/pause behavior. `31` is explicitly an adaptive-speed graphics mode, which sounds designed for dense graphics output. `32` gives a constant-speed comparison point.

Recommended bitmap experiment:

```text
printer_graphics_mode 31
printer_density 20
printer_speed 80
# print graylevels / diagonal / dithered image
```

Then compare:

```text
printer_graphics_mode 32
# print the same image
```

## Temperature query

Command:

```text
GS g 6
Hex: 1D 67 36
```

Returns a text string like:

```text
temp:42
```

SToMS3R command:

```text
printer_temp
```

Web API:

```text
GET /api/printer/temp
```

Why it matters: this lets us observe whether dense bitmap pauses correlate with thermal behavior.

## Software flow control

Command:

```text
ESC ## SFFC n
Hex: 1B 23 23 53 46 46 43 n
```

Values:

| `n` | Meaning |
|---:|---|
| 0 | disable software flow control |
| 1 | enable software flow control |

The manual says only “software flow control,” but in serial-printer terminology this usually means XON/XOFF:

```text
XOFF = 0x13  // stop sending
XON  = 0x11  // resume sending
```

How it would work:

```text
ESP32 TX  -> printer RX: bitmap bytes
ESP32 RX  <- printer TX: XOFF / XON bytes
```

The host must watch incoming bytes while transmitting. Enabling the printer-side setting is not enough. The ESP32 must stop queueing bytes when it sees XOFF and resume when it sees XON.

A software-flow-aware sender must not call `uart_write_bytes()` once with a huge bitmap, because too much data can already be queued before XOFF is observed. Instead it should send small chunks and poll RX between chunks:

```c
while (off < len) {
    poll_rx_for_xon_xoff();
    while (xoff_seen) {
        vTaskDelay(pdMS_TO_TICKS(5));
        poll_rx_for_xon_xoff();
    }
    uart_write_bytes(UART_NUM_1, data + off, chunk_len);
    off += chunk_len;
}
```

Recommended implementation path:

1. Add `printer_softflow on|off` that only sends `ESC ## SFFC n`.
2. During bitmap prints, log any incoming `0x11` / `0x13` bytes.
3. If the printer actually emits XON/XOFF, add a chunked XON/XOFF-aware write path.
4. Compare modes: CTS only, software flow only, both, neither.

## Print-head voltage query

The manual lists:

```text
GS g l
```

as returning the print-head voltage value.

This may be useful because the K118 docs say print quality depends directly on having a 12V supply rated at 2.5A or higher. If this command returns a meaningful voltage string, it could help detect power sag during dense bitmap prints.

Suggested future command:

```text
printer_voltage
```

## Self-test and info print commands

Useful vendor commands that print information directly on paper:

| Command | Meaning |
|---|---|
| `ESC ## SELF` | print self-test information |
| `ESC # V` | print software version |
| `ESC # F` | print function list |
| `ESC # K` | print density level |
| `ESC # L` | print print-head temperature |
| `ESC # M` | print print speed |

These are useful for hardware discovery. They print to paper rather than returning structured data over UART.

Suggested future console commands:

```text
printer_selftest
printer_print_version
printer_print_functions
printer_print_density
printer_print_temp
printer_print_speed
```

## Low-priority command families

The manual also contains commands for:

- cutter control
- cash drawer control
- black-mark label/ticket paper
- Ethernet/network/cloud settings
- Bluetooth settings
- MD5/security features
- page mode / TSC mode

These are likely shared across multiple printer-controller products and may not apply to the K118 as a simple TTL serial 58mm thermal printer. They should not be prioritized unless hardware testing shows they are relevant.

## Current SToMS3R coverage

Implemented now:

| Area | Console/API |
|---|---|
| text | `printer_text`, `/api/print/text` |
| feed | `printer_feed` |
| style | `printer_size`, `printer_bold`, `printer_align` |
| barcode | `printer_barcode` |
| QR | `printer_qr` |
| bitmap | `printer_bitmap_test`, `/api/print/bitmap` |
| status quick | `printer_probe` |
| status rich | `printer_status`, `/api/printer/status` |
| temperature | `printer_temp`, `/api/printer/temp` |
| baud set/query | `set_baudrate`, `printer_get_baud`, `/api/printer/baud` |
| density | `printer_density`, `/api/printer/density` |
| speed | `printer_speed`, `/api/printer/speed` |
| graphics mode | `printer_graphics_mode`, `/api/printer/graphics-mode` |
| saved startup settings | `printer_settings_save`, `printer_settings_show`, `printer_settings_apply`, `printer_settings_clear` |
| raw debug | `printer_raw` |
| wiring debug | `printer_swap` |

Saved startup settings now live in NVS namespace `printer`. The currently useful profile is:

```text
set_baudrate 460800
printer_density 30
printer_speed 80
printer_graphics_mode 31
printer_settings_save 460800 30 80 31
```

On boot, the firmware applies the saved ESP32 UART baud first, then sends density, speed, and graphics-mode commands at that baud. This assumes the K118 printer-side baud has already been changed/persisted by `set_baudrate`. If the printer comes back at 9600, recover with `printer_baud 9600` or clear saved settings.

Missing but interesting:

```text
printer_softflow on|off
printer_voltage
printer_selftest
printer_print_functions
printer_print_version
printer_flow_mode none|cts|xonxoff|both
```
