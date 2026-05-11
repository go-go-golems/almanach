# Changelog

## 2026-05-10

- Initial workspace created


## 2026-05-10

Added diary and runtime printer CTS diagnostic plan after comparing current, old stoms3r, and simple no-CTS firmware.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/printer_cmd.c — printer_flow console command
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/printer_drv.c — Runtime flow-control toggle


## 2026-05-10

Flashed printer serial diagnostic firmware and tested CTS/no-CTS plus swapped/normal pin matrix; all status probes timed out, so pinout remains historically likely but not physically proven.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/printer_drv.c — Effective UART pin and flow-control behavior under test
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-SERIAL--investigate-atoms3r-printer-serial-diagnostics/reference/01-diary.md — Hardware test results


## 2026-05-10

Added old esp32-s3-m5 ticket diary evidence: GPIO8/GPIO7/GPIO6 was the corrected K118 bottom-header mapping after GPIO5/GPIO6 was found to be the wrong connector.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-SERIAL--investigate-atoms3r-printer-serial-diagnostics/reference/01-diary.md — Updated investigation diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/esp32-s3-m5/ttmp/2026/04/28/STOMS3R-001--stoms3r-atoms3r-lite-thermal-printer-console-firmware/reference/01-diary.md — Pinout provenance

