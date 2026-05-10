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

