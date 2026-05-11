# Changelog

## 2026-05-10

- Initial workspace created


## 2026-05-10

Recorded printer settings page baud 460800 and tested old flashed firmware with ESP32 UART switched to 460800; RX still looked like echo, so baud alignment and echo-aware diagnostics are both required.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/design-doc/01-atoms3r-k118-printer-uart-serial-interface-analysis-design-and-implementation-guide.md — Updated baud guidance
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Baud test diary


## 2026-05-10

Validated copied Almanach firmware with K118 at 460800 baud, saved startup printer settings, restarted, and confirmed probe/status/baud diagnostics work after boot.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/design-doc/01-atoms3r-k118-printer-uart-serial-interface-analysis-design-and-implementation-guide.md — Validated resolution section
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Step 4 hardware validation


## 2026-05-10

Re-tested localhost server render-and-print after the 460800 baud fix; tiny and one-image cat layouts printed successfully through /api/render-and-print.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/printer.go — printer HTTP client path under validation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/server.go — render-and-print path under validation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Server print validation


## 2026-05-10

Tested foo-cat ZIP layout bundles: 4, 5, and 6 images printed successfully; 7 and 8 images failed around ~100KB bitmap POST body before UART raster printing, pointing to firmware HTTP receive limits rather than ZIP or UART baud.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/web_server.c — Firmware bitmap HTTP receive path
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/layout_bundle.go — Host ZIP layout support
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/printer.go — Host bitmap POST path
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — ZIP bundle print threshold results


## 2026-05-10

Added host-side light thermal image tone for image blocks and validated a 6-image foo-cat ZIP print through the render service.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/web/embed/public/almanach-bundle.js — Regenerated embedded studio bundle
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Option 2 implementation diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/almanach-studio.jsx — Image block thermalTone UI and render filter


## 2026-05-10

Created GitHub issue #1 for segmented/chunked print endpoint, added 90 KiB bitmap rejection, saved bundle-generation scripts, corrected the 4x4 cat portrait split, and printed a large cat-owner story successfully.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/web_server.c — Firmware 90 KiB bitmap guard
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/printer.go — Host-side oversized bitmap rejection
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Step 8 diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/scripts/02-create-cat-owner-story-bundle.py — Corrected 4x4 cat portrait split and story bundle generator


## 2026-05-10

Adjusted cat-owner story generator for larger font and borderless minimal portraits; printed the large-font bundle successfully at 91,920 bytes, just under the 90 KiB guard.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Step 9 print validation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/scripts/02-create-cat-owner-story-bundle.py — Larger font and borderless minimal portrait layout


## 2026-05-10

Adjusted cat-owner story to use three larger borderless portraits and printed successfully at 82,752 bytes.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Step 10 large portrait validation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/scripts/02-create-cat-owner-story-bundle.py — Three larger portrait layout


## 2026-05-10

Increased cat-owner story body scale to 1.52, adjusted portrait height to 196, and printed successfully at 85,584 bytes.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Step 11 bigger font validation
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/scripts/02-create-cat-owner-story-bundle.py — Bigger font layout defaults


## 2026-05-10

Recovered after crash: rebuilt embedded web assets, restarted localhost setup server on port 18299, verified persisted printer IP and direct printer status.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/web/embed/public/almanach-bundle.js — Regenerated embedded studio bundle
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Crash recovery diary step
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/web/src/almanach-studio.jsx — Default sample text cleanup before rebuild


## 2026-05-10

Re-extracted the 4x4 cat collage from the refreshed clipboard image into ticket assets, generated a contact sheet/manifest for verification, updated the single-cat print script to use ticket-local assets, and printed one large cat portrait successfully.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/contact-sheet.png — Visual verification sheet for corrected 4x4 cat crops
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/manifest.md — Crop manifest and dimensions
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Step 13 extraction and print diary
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/scripts/03-create-single-large-cat-bundle.py — Single large cat print bundle generator now uses ticket assets
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/scripts/04-extract-cat-portraits.py — Reproducible ticket-local cat portrait extraction


## 2026-05-10

Printed four additional large single-cat portraits from ticket-local assets, all successful at 24,528 bytes each.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/portraits/cat-portrait-r01-c01.png — Printed large portrait
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/portraits/cat-portrait-r01-c03.png — Printed large portrait
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/portraits/cat-portrait-r04-c03.png — Printed large portrait
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/portraits/cat-portrait-r04-c04.png — Printed large portrait
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/reference/01-investigation-diary.md — Step 14 batch portrait print diary

