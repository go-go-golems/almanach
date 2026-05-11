# Original ATOM-PRINTER Arduino Firmware Command Inventory

This document inventories the printer commands used by the original M5Stack Arduino firmware and library located at:

```text
0090-m5printer-research/source/ATOM-PRINTER/
```

The relevant files are:

```text
src/ATOM_PRINTER_CMD.h
src/ATOM_PRINTER.cpp
src/ATOM_PRINTER.h
examples/PRINTER_FW/PRINTER_FW.ino
examples/PRINTER_FW/ATOM_PRINTER_WEB.cpp
examples/PRINTER_FW/ATOM_PRINTER_MQTT.cpp
```

## Executive summary

The original Arduino library uses a **small subset** of the printer command manual:

- reset / initialize
- absolute print position
- font size
- raw ASCII text
- line feed
- barcode HRI position
- barcode enable/disable
- barcode print
- QR error correction setting
- QR data store
- QR print
- raster bitmap print

It does **not** use the richer vendor-specific diagnostic/tuning commands we found in the Chinese spec:

- no `GS a n` 4-byte status query
- no `GS g 6` temperature query
- no `GS g 7` baud query
- no `ESC ## SBDR` baud setting
- no `ESC ## STDP` density setting
- no `ESC ## STSP` speed setting
- no `ESC ## SPSM` graphics mode setting
- no `ESC ## SFFC` software flow-control setting

This is important: the original firmware is minimal. It proves the basic print command sequences, but it does not contain hidden fixes for bitmap striping via density/speed/flow-control commands.

## Command constants in `ATOM_PRINTER_CMD.h`

```cpp
const uint8_t INIT_PRINTER_CMD[] = {0x1B, 0x40};
const uint8_t PRINT_POS_CMD[] = {0x1B, 0x24};
const uint8_t FONT_SIZE_CMD[] = {0x1D, 0x21};
const uint8_t SET_BAR_CODE_POS_CMD[] = {0x1D, 0x48};
const uint8_t ENABLE_BAR_CODE_MODE_CMD[] = {0x1D, 0x45, 0x43, 0xff};
const uint8_t PRINTER_BAR_CODE_CMD[] = {0x1D, 0x6B, 0x41, 0xff};
const uint8_t PRINTER_QRCODE_CMD[] = {0x1D, 0x28, 0x6B, 0x03, 0x00,
                                      0x31, 0x51, 0x30, 0x00};
const uint8_t SET_QRCODE_CMD[] = {0x1D, 0x28, 0x6B, 0xff,
                                  0Xff, 0x31, 0x50, 0x30};
const uint8_t SET_QRCODE_ECL_CODE_CMD[] = {0x1D, 0x28, 0x6B, 0x03,
                                           0X00, 0x31, 0x45};
const uint8_t PRINTER_BMP_CMD[] = {0x1D, 0x76, 0x30, 0xff,
                                   0xff, 0xff, 0xff, 0xff};
```

## Library API and backing commands

| Arduino API | Bytes / command | Meaning | SToMS3R equivalent |
|---|---|---|---|
| `begin(serial, baud, RX, TX)` | `serial->begin(baud, SERIAL_8N1, RX, TX)` | Configure UART only | `printer_drv_init()` |
| `init()` | `1B 40` | Initialize printer | `printer_init` |
| `printPos(posx)` | `1B 24 nL nH` | Absolute print position | not exposed yet |
| `fontSize(font_size)` | `1D 21 n` | Character size | `printer_size` |
| `newLine(count)` | repeated `0A` | Line feed(s) | `printer_feed` is similar but uses `ESC d n`; text also sends LF |
| `printASCII(data)` | raw string | Print raw text | `printer_text` |
| `setBarCodeHRI(pos)` | `1D 48 pos` | Barcode HRI print position | not separately exposed |
| `enableBarCode(state)` | `1D 45 43 state` | Enable/disable barcode mode | internal to Arduino barcode path; not in SToMS3R |
| `printBarCode(type, data)` | `1D 6B type len data 00` plus enable/disable | Print barcode | `printer_barcode` |
| `setQRCodeECL(level)` | `1D 28 6B 03 00 31 45 level` | Set QR error correction | not separately exposed; current QR path uses default enum parameter internally |
| `printQRCode(qrcode)` | store + print QR commands | Print QR code | `printer_qr` |
| `printBMP(mode, xdot, ydot, buffer)` | `1D 76 30 m xL xH yL yH data...` | Print raster bitmap | `/api/print/bitmap`, `printer_bitmap_test` |
| `WriteCMD(buff, size)` | arbitrary bytes | Raw command write | `printer_raw` |

## Exact behavior by function

### `begin()`

```cpp
_serial->begin(baud, SERIAL_8N1, RX, TX);
```

Defaults in `ATOM_PRINTER.h`:

```cpp
void begin(HardwareSerial *serial = &Serial2, int baud = 9600,
           uint8_t RX = 33, uint8_t TX = 23, bool debug = false);
```

This configures ATOM Lite `Serial2` at 9600 baud with RX=GPIO33, TX=GPIO23. The original code does not configure CTS.

### `init()`

```cpp
_serial->write(INIT_PRINTER_CMD, sizeof(INIT_PRINTER_CMD));
```

Command:

```text
ESC @
1B 40
```

### `printPos(uint16_t posx)`

```cpp
memcpy(buffer, PRINT_POS_CMD, sizeof(PRINT_POS_CMD));
buffer[2] = posx & 0xff;
buffer[3] = (posx >> 8) & 0xff;
_serial->write(buffer, 4);
```

Command:

```text
ESC $ nL nH
1B 24 nL nH
```

This sets the absolute print position. The MQTT path uses it for text messages:

```cpp
printer.printPos(posx);
printer.fontSize(fonts);
printer.printASCII(...);
```

SToMS3R does not currently expose absolute position. It may be useful for advanced layout, but not urgent for bitmap debugging.

### `fontSize(uint8_t font_size)`

```cpp
if (font_size > 7) font_size = 7;
memcpy(buffer, FONT_SIZE_CMD, sizeof(FONT_SIZE_CMD));
buffer[2] = (font_size | (font_size << 4)) & 0xff;
_serial->write(buffer, 3);
```

Command:

```text
GS ! n
1D 21 n
```

The code sets both width and height magnification to the same value.

### `newLine(uint8_t count)`

```cpp
for (uint8_t i = 0; i < count; i++) {
    _serial->write(0x0A);
}
```

This sends raw LF bytes, not `ESC d n`.

SToMS3R has `printer_feed`, which uses:

```text
ESC d n
1B 64 n
```

and `printer_text` sends text followed by LF.

### `printASCII(String data)`

```cpp
_serial->print(data);
```

No ESC/POS framing. It simply writes text bytes to the printer.

### `setBarCodeHRI(BarCodePos_t pos)`

```cpp
_serial->write(SET_BAR_CODE_POS_CMD, sizeof(SET_BAR_CODE_POS_CMD));
_serial->write(pos);
```

Command:

```text
GS H n
1D 48 n
```

Enum values:

```cpp
typedef enum { HIDE = 0x00, ABOVE, BELOW, BOTH } BarCodePos_t;
```

The original web/MQTT paths set HRI to hidden:

```cpp
printer.setBarCodeHRI(HIDE);
```

### `enableBarCode(bool state)`

```cpp
memcpy(buffer, ENABLE_BAR_CODE_MODE_CMD, sizeof(ENABLE_BAR_CODE_MODE_CMD));
buffer[3] = state;
_serial->write(buffer, sizeof(ENABLE_BAR_CODE_MODE_CMD));
```

Command bytes:

```text
GS E C state
1D 45 43 state
```

This is not the usual minimal ESC/POS barcode command; it is a K118/manual-specific barcode on/off command (`GS E C` appears in the translated manual as barcode switch command). The original barcode path enables barcode mode before printing and disables it afterward.

### `printBarCode(BarCode_t type, String barcode)`

```cpp
enableBarCode(1);
memcpy(buffer, PRINTER_BAR_CODE_CMD, sizeof(PRINTER_BAR_CODE_CMD));
uint8_t len = barcode.length();
buffer[2]   = type;
buffer[3]   = len;
_serial->write(buffer, sizeof(PRINTER_BAR_CODE_CMD));
_serial->print(barcode);
_serial->write(0x00);
enableBarCode(0);
```

Header constant before patching:

```text
1D 6B 41 FF
```

Actual bytes after patching:

```text
GS k type len data... NUL
1D 6B type len data... 00
```

Barcode enum:

```cpp
typedef enum {
    UPC_A = 0x41,
    UPC_E,
    JAN13_EAN13,
    JAN8_EAN8,
    CODE39,
    ITF,
    CODABAR,
    CODE93,
    CODE128
} BarCode_t;
```

Original firmware uses only:

```cpp
printer.printBarCode(CODE128, ...)
```

### `setQRCodeECL(QRCode_EC_Level_t level)`

```cpp
_serial->write(SET_QRCODE_ECL_CODE_CMD, sizeof(SET_QRCODE_ECL_CODE_CMD));
_serial->write(level);
```

Command:

```text
GS ( k 03 00 31 45 level
1D 28 6B 03 00 31 45 level
```

Enum:

```cpp
typedef enum { LEVEL_L = 0x48, LEVEL_M, LEVEL_Q, LEVEL_H } QRCode_EC_Level_t;
```

The public function exists, but the example firmware does not appear to call it. The default printer QR error correction level is used unless set elsewhere.

### `printQRCode(String qrcode)`

The original code stores QR data and prints it:

```cpp
len = qrcode.length();
len += 3;
nL = len & 0x00ff;
nH = (len >> 8) & 0x00ff;
memcpy(buffer, SET_QRCODE_CMD, sizeof(SET_QRCODE_CMD));
buffer[3] = nL;
buffer[4] = nH;
_serial->write(buffer, sizeof(SET_QRCODE_CMD));
_serial->print(qrcode);
_serial->write(0x00);

memcpy(buffer, PRINTER_QRCODE_CMD, sizeof(PRINTER_QRCODE_CMD));
_serial->write(buffer, sizeof(PRINTER_QRCODE_CMD));
```

Store command:

```text
GS ( k pL pH 31 50 30 data... 00
1D 28 6B pL pH 31 50 30 data... 00
```

Print command:

```text
GS ( k 03 00 31 51 30 00
1D 28 6B 03 00 31 51 30 00
```

Note: The print command constant contains an extra trailing `00` compared with many ESC/POS QR examples. SToMS3R currently uses the common 8-byte print command without the extra zero:

```text
1D 28 6B 03 00 31 51 30
```

This difference is worth testing if QR output fails.

### `printBMP(uint8_t mode, uint16_t xdot, uint16_t ydot, uint8_t *buffer)`

```cpp
if (mode > 3) mode = 3;
memcpy(buffer, PRINTER_BMP_CMD, sizeof(PRINTER_BMP_CMD));
xdot      = xdot / 8;
buffer[3] = mode;
buffer[4] = (uint8_t)(xdot & 0x00ff);
buffer[5] = (uint8_t)((xdot >> 8) & 0x00ff);
buffer[6] = (uint8_t)(ydot & 0x00ff);
buffer[7] = (uint8_t)((ydot >> 8) & 0x00ff);
_serial->write(buffer, sizeof(PRINTER_BMP_CMD));
len = xdot * ydot;
while (len) {
    _serial->write(*buffer++);
    len--;
}
```

Command:

```text
GS v 0 mode xL xH yL yH data...
1D 76 30 mode xL xH yL yH data...
```

The key behavior is that the bitmap data is already in memory before printing. The web upload handler buffers the full upload first:

```cpp
memcpy(bmp_buffer + bmp_data_offset, upload.buf, chunkSize);
...
printer.printBMP(0, bmp_width, bmp_height, bmp_buffer);
```

The original firmware does not stream HTTP chunks directly into the printer.

## Example firmware usage

### Web print handler

`ATOM_PRINTER_WEB.cpp` handles three print types:

```cpp
if (printType == "ASCII") {
    printer.init();
    printer.printASCII(Pdata);
} else if (printType == "QRCode") {
    printer.init();
    printer.printQRCode(QRCode);
} else if (printType == "BarCode") {
    printer.init();
    printer.setBarCodeHRI(HIDE);
    printer.printBarCode(CODE128, BarCode);
}

if (newLine == "on") {
    printer.newLine(1);
}
```

### Web bitmap upload handler

```cpp
if (bmp_width > 0 && bmp_height > 0) {
    printer.printBMP(0, bmp_width, bmp_height, bmp_buffer);
}
```

### MQTT handler

The MQTT payload parser supports:

```text
TEXT<pos>,<font>:<text>
QR:<text>
BAR:<text>
```

For `TEXT`:

```cpp
printer.init();
printer.printPos(posx);
printer.fontSize(fonts);
printer.printASCII(...);
printer.newLine(3);
```

For `QR`:

```cpp
printer.init();
printer.printQRCode(...);
printer.newLine(3);
```

For `BAR`:

```cpp
printer.init();
printer.setBarCodeHRI(HIDE);
printer.printBarCode(CODE128, ...);
printer.newLine(3);
```

## Commands from original firmware not yet mirrored exactly

SToMS3R covers the main functions, but a few original details are not mirrored exactly:

| Original behavior | Difference in SToMS3R | Should we add/change? |
|---|---|---|
| `printPos(posx)` / `ESC $ nL nH` | no console command | Low/medium priority for layout |
| `newLine(count)` sends raw LF | `printer_feed` uses `ESC d n` | Probably fine; maybe add `printer_lf <n>` for exact behavior |
| barcode path toggles `GS E C 1/0` | SToMS3R only sends `GS k` | If barcode fails, add barcode enable/disable around barcode printing |
| QR print command has trailing `00` | SToMS3R omits trailing zero | If QR fails, test adding trailing `00` |
| `setQRCodeECL(level)` exists | no console-level QR EC setting | Add `printer_qr_ex --ec` later |

## Commands the original firmware does not use

The original Arduino firmware does not use the following important K118 commands:

```text
GS a n             // 4-byte rich status
GS g 6             // temperature query
GS g 7             // baud query
ESC ## SBDR        // baud setting
ESC ## STDP        // density setting
ESC ## STSP        // speed setting
ESC ## SPSM        // graphics mode
ESC ## SFFC        // software flow control
GS g l             // print-head voltage
ESC ## SELF        // self-test
ESC # F            // function list
```

This means the original firmware is not a complete guide for tuning. It is a guide for the baseline print protocol.

## Conclusions for SToMS3R

1. The original firmware confirms the basic byte sequences for text, QR, barcode, and bitmap.
2. The original firmware confirms that bitmap uploads should be buffered before printing.
3. The original firmware does not use CTS, software flow control, density, speed, graphics mode, temperature, or buffer-full status.
4. Our ESP-IDF firmware is now more diagnostic-capable than the original Arduino firmware.
5. If QR or barcode output differs, the original library has two details to revisit: QR trailing `00` and barcode `GS E C` enable/disable wrapping.
