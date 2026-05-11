# Changelog

## 2026-05-10

- Initial workspace created


## 2026-05-10

Implemented persistent setup-discovered printer endpoint state. The setup/render server now loads a provisioned printer endpoint from a JSON state file, persists browser-reported endpoints atomically, supports `DELETE /api/setup/provisioned-device`, keeps explicit `ALMANACH_PRINTER_IP` precedence, exposes `ALMANACH_STATE_FILE` / `--state-file`, and documents the behavior in the provisioning help guide.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/setup_device.go — Persistent setup-device store and API
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/setup_device_test.go — State-file load/save/delete and precedence tests
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/config.go — State-file configuration
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_setup.go — Setup command state-file flag
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/cmd_serve.go — Serve command state-file flag and logging
- /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/internal/app/doc/provisioning-printer-user-guide.md — User-facing persistence docs
