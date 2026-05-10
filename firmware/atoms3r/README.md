# SToMS3R — AtomS3R Lite Thermal Printer Console Firmware

SToMS3R ("Screw This, On My S3R") is an ESP-IDF firmware for the M5Stack
AtomS3R Lite (ESP32-S3-PICO-1-N8R8) that drives the M5Stack K118 thermal
printer kit through an interactive `esp_console` REPL over USB Serial/JTAG.

## Hardware

| Component | Details |
|-----------|---------|
| Controller | M5Stack AtomS3R Lite (ESP32-S3, 8MB Flash, 8MB PSRAM) |
| Printer | M5Stack K118 Thermal Printer Kit (58mm, 203dpi) |
| Connection | UART1 9600 8N1, K118 header pins TX=GPIO8, RX=GPIO7, CTS=GPIO6 (software TX/RX swap defaults on) |
| Console | USB Serial/JTAG (`/dev/ttyACM0`) |
| Power | USB-C for logic, 12V/2.5A for printer mechanism |

## Console Commands

### Printer
- `printer_init` — Reset printer
- `printer_text <text>` — Print a line of text
- `printer_feed [n]` — Feed n lines (default 3)
- `printer_size <0-7>` — Set font size
- `printer_bold <on|off>` — Bold on/off
- `printer_align <0|1|2>` — Left/center/right
- `printer_barcode <type> <data>` — Print barcode (CODE128, EAN13, ...)
- `printer_qr <text>` — Print QR code
- `printer_bitmap_test` — Print test pattern
- `printer_probe` — Query status and diagnose wiring
- `printer_swap <on|off>` — Toggle software TX/RX crossover
- `printer_baud <rate>` — Change ESP32 UART baud only (recovery)
- `set_baudrate <rate>` — Send K118 baud-rate command, then switch ESP32 UART
  - Supported in firmware: 9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600
  - K118 docs explicitly show 9600 and 115200; higher rates are experimental
- `printer_status` — Read 4-byte printer status (buffer full, cover, paper, overheat)
- `printer_temp` — Read printer temperature
- `printer_get_baud` — Read printer-side baud rate
- `printer_density <0-39>` — Set darkness/current draw
- `printer_speed <speed>` — Set mechanism speed (25..220 table values)
- `printer_graphics_mode <30|31|32>` — Set graphics mode (BLE/adaptive/constant)
- `printer_settings_save <baud> <density> <speed> <mode>` — Save startup printer settings to NVS
- `printer_settings_show` — Show saved startup printer settings
- `printer_settings_apply` — Apply saved startup printer settings now
- `printer_settings_clear` — Clear saved startup printer settings

Recommended currently verified startup profile:

```text
set_baudrate 460800
printer_density 30
printer_speed 80
printer_graphics_mode 31
printer_settings_save 460800 30 80 31
```

On boot, saved baud sets the ESP32 UART side directly and assumes the K118 printer-side baud has already been set/persisted by `set_baudrate`. If the printer is power-cycled back to 9600 or communication is lost, use `printer_baud 9600` for recovery or `printer_settings_clear` to return to defaults.

### WiFi
- `wifi_scan` — Scan for access points
- `wifi_connect --ssid <s> --pass <p>` — Connect and save credentials
- `wifi_status` — Show connection state
- `wifi_disconnect` — Disconnect
- `wifi_forget` — Erase saved credentials

## Build / Flash / Monitor

```bash
./build.sh build
./build.sh /dev/ttyACM0 flash-monitor
```

Or manually:

```bash
source ~/esp/v5.4.x/esp-idf/export.sh
idf.py build
idf.py -p /dev/ttyACM0 flash monitor
```

Exit monitor: `Ctrl+]`
