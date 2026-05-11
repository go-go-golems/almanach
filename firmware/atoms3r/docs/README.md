# SToMS3R Firmware Documentation

This folder contains firmware-local engineering notes for the AtomS3R Lite + K118 thermal printer firmware.

## Documents

- [`01-k118-command-discoveries.md`](01-k118-command-discoveries.md)
  - Practical discoveries from the Chinese `ATOM_PRINTER_CMD_v1.06` spec.
  - Covers baud, status, bitmap, density, speed, graphics mode, temperature, software flow control, voltage, self-test, and current SToMS3R coverage.

- [`02-bitmap-stripes-flow-control.md`](02-bitmap-stripes-flow-control.md)
  - Full record of the bitmap stripe/pause investigation.
  - Covers full-body buffering, why direct HTTP chunk streaming caused stripes, why 5-row banding caused regular seams, CTS, software flow control, and recommended diagnostics.

- [`03-original-arduino-firmware-command-inventory.md`](03-original-arduino-firmware-command-inventory.md)
  - Inventory of every printer command used by the original M5Stack Arduino `ATOM-PRINTER` firmware/library.
  - Shows that the original firmware uses only a small subset of the K118 spec and does not use density/speed/graphics-mode/temperature/status tuning commands.

- [`PROJ-SToMS3R-AtomS3R-Lite-Thermal-Printer-Firmware.md`](PROJ-SToMS3R-AtomS3R-Lite-Thermal-Printer-Firmware.md)
  - Copy of the Obsidian project report for the SToMS3R firmware.
  - Kept here so the firmware tree contains the project narrative, architecture, command surface, and related research links.

## Upstream/source references

The original research material lives outside this firmware folder:

```text
../0090-m5printer-research/
```

Most relevant files:

```text
../0090-m5printer-research/docs/ATOM_PRINTER_CMD_v1.06.pdf
../0090-m5printer-research/docs/ATOM_PRINTER_CMD_v1.06.txt
../0090-m5printer-research/docs/ATOM_PRINTER_CMD_v1.06.en.md
../0090-m5printer-research/source/ATOM-PRINTER/src/ATOM_PRINTER_CMD.h
../0090-m5printer-research/source/ATOM-PRINTER/src/ATOM_PRINTER.cpp
../0090-m5printer-research/source/ATOM-PRINTER/examples/PRINTER_FW/
```
