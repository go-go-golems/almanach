# Changelog

## 2026-05-14

- Initial workspace created


## 2026-05-14

Paired printer via native BLE (ALM_0F2320 on yolobolo at 192.168.0.126), printed test layouts via CLI and Python, deployed almanach-render-service to crib-k3s (https://almanach.crib.scapegoat.dev), verified remote printing works, created SKILL.md and helper scripts


## 2026-05-14

Fixed large bitmap printing: 1) Use X-Feed header instead of baking feed rows (saves ~3.4KB per print), 2) Segmented printing for bitmaps >36KB, 3) Connection:close + DisableKeepAlives on HTTP transport. Both local CLI and remote crib.k3s service now print daily briefing successfully. Commit ab6fd22.

### Related Files

- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/cmd_print.go — Removed baked feed rows
- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/printer.go — Segmented printing + X-Feed + Connection:close fix (commit ab6fd22)

