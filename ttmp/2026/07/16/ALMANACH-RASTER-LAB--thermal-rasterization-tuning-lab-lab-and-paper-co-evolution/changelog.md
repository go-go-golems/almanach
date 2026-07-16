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


## 2026-07-16

Step 6: Bitmap fonts win small text (6x9/6x10 sweet spot); photo locked at Atkinson g0.8 d20 s80. Bundled PCF fonts, added smalltext card. Recorded paper-verified recipes in guide.

### Related Files

- /home/manuel/code/wesen/go-go-golems/almanach/ttmp/2026/07/16/ALMANACH-RASTER-LAB--thermal-rasterization-tuning-lab-lab-and-paper-co-evolution/scripts/fonts — Bundled X11 PCF bitmap fonts for small-text rendering


## 2026-07-16

Step 7: Mixed-page proof (02-mixed-page.py) prints bitmap-font text hot (d30/s37) + Atkinson cat cool (d20/s80) as per-segment-heat segments in one job; no firmware change.


## 2026-07-16

Step 8: Go port - internal/app/raster.go pluggable rasterizer (threshold byte-identical + Atkinson/Floyd/Bayer + tone curve) and setPrinterHeat density/speed; wired to CLI/render options; lint+tests green (commit d3b79b0)

