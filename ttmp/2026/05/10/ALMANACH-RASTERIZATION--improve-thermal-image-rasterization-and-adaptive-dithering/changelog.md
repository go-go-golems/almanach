# Changelog

## 2026-05-10

- Initial workspace created


## 2026-05-10

Created a self-contained macOS 1-style monochrome dithering lab HTML page with embedded cat portrait presets, drag/drop image loading, multiple algorithms, and live tuning controls.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/reference/01-investigation-diary.md — Step 2 lab creation diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html — Standalone browser dithering playground


## 2026-05-10

Added a Print to Printer button to the standalone dithering lab, including printer URL/feed controls, browser-side MSB-first bit packing, feed rows, and a 90 KiB guard.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/reference/01-investigation-diary.md — Step 3 print button diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html — Print-to-printer UI and browser-side bitmap packing


## 2026-05-10

Added a localhost server/proxy for the dithering lab so browser print requests go to same-origin /api/print/bitmap and are forwarded to the printer, avoiding direct file:// or cross-origin printer POSTs.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/reference/01-investigation-diary.md — Step 4 CORS/proxy diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/scripts/01-serve-raster-dither-lab.py — Localhost lab server and bitmap print proxy
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html — Default printer URL changed to same-origin proxy


## 2026-05-10

Split the raster lab JavaScript into raster-dither-lab.js and changed browser printing to include a monochrome settings/algorithm header above the dithered image.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/reference/01-investigation-diary.md — Step 5 lab refactor diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html — HTML shell now loads external lab JavaScript
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.js — Print settings header

