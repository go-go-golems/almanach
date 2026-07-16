# Changelog

## 2026-07-16

- Initial workspace created


## 2026-07-16

Step 1: Investigated rasterization pipeline; created ticket, diary, and intern guide; established lab-and-paper co-evolution workflow (docs-only)

### Related Files

- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/bitmap.go — Documented as the lever-2 rasterizer seam


## 2026-07-16

Step 2: Added scripts/01-thermal-lab.py experiment harness (synthetic mixed card, tone curve, dither algorithms, density control, MSB-first print client); validated via dry-run previews

### Related Files

- /home/manuel/code/wesen/go-go-golems/almanach/ttmp/2026/07/16/ALMANACH-RASTER-LAB--thermal-rasterization-tuning-lab-lab-and-paper-co-evolution/scripts/01-thermal-lab.py — Experiment harness


## 2026-07-16

Step 3: First paper batch (4 algos @ d20). Atkinson best photo, Floyd 2nd, Bayer8 gridded, threshold collapses. Text survives dithering; ramp crushes in shadows -> test gamma<1. Printer is 192.168.0.126.


## 2026-07-16

Step 4: Cat locks at Atkinson gamma~0.8 (user feedback). Added text card + speed control to harness; printed text density sweep {16,24,32,39}. Noted per-segment density enables mixed-page text/photo heat split with no firmware change.


## 2026-07-16

Step 5: Text sweet spot ~d28-32 (hotter than photo). Within-line tone variation = power droop, test slower speed. Printed text (3x3 density/speed) and photo (3x2) matrices.

