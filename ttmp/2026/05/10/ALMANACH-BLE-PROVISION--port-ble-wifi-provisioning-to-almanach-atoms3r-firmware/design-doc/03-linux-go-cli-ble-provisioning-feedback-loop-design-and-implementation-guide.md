---
Title: Linux Go CLI BLE Provisioning Feedback Loop Design and Implementation Guide
Ticket: ALMANACH-BLE-PROVISION
Status: active
Topics:
    - almanach
    - firmware
    - esp-idf
    - ble
    - wifi-provisioning
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/app/cmd_ble_provision.go
      Note: Linux BLE provisioning Glazed command implementation and ESP-IDF esp_prov.py wrapper.
    - Path: internal/app/cmd_root.go
      Note: Registers the ble-provision Glazed verb in the Almanach binary.
    - Path: firmware/atoms3r/main/provisioning_mgr.c
      Note: AtomS3R firmware BLE provisioning manager, service name, PoP, status, and reset API.
    - Path: firmware/atoms3r/main/provisioning_cmd.c
      Note: Serial console commands for prov_status, prov_start, and prov_reset.
    - Path: firmware/atoms3r/main/app_main.c
      Note: Boot flow that starts BLE provisioning when no credentials are available.
    - Path: firmware/atoms3r/sdkconfig.defaults
      Note: BLE/NimBLE/protocomm configuration defaults required by the firmware.
    - Path: /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py
      Note: Espressif reference host provisioning client used by the Almanach Go wrapper.
ExternalSources:
    - 'ESP-IDF esp_prov tool README: /home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/README.md'
    - 'ESP-IDF WiFi provisioning manager API: /home/manuel/esp/esp-idf-5.4.2/components/wifi_provisioning/include/wifi_provisioning/manager.h'
    - 'ESP-IDF protocomm BLE transport header: /home/manuel/esp/esp-idf-5.4.2/components/protocomm/include/transports/protocomm_ble.h'
Summary: Intern-facing technical guide for the Linux Go/Glazed ble-provision command that provisions Almanach AtomS3R firmware over ESP-IDF BLE provisioning.
LastUpdated: 2026-05-10T19:00:00-04:00
WhatFor: Use this when testing AtomS3R BLE WiFi provisioning from a Linux laptop without using the phone app or browser Web Bluetooth.
WhenToUse: Read before modifying ble-provision, troubleshooting Linux BLE permissions, changing firmware provisioning endpoints, or validating the provisioning feedback loop.
---

# Linux Go CLI BLE Provisioning Feedback Loop Design and Implementation Guide

## Executive Summary

The Almanach AtomS3R firmware now starts ESP-IDF BLE WiFi provisioning when it boots without WiFi credentials. That means a fresh or erased device advertises a BLE provisioning service such as `ALM_0F2320`, accepts Security 1 proof-of-possession such as `alm-0f2320`, receives WiFi credentials over protocomm, stores them in ESP-IDF WiFi/NVS state, connects to the LAN, and then starts the existing Almanach web/print API.

For fast development we need a local Linux feedback loop. The phone app is useful for end-user validation, but it is slow for firmware iteration because the developer already has a terminal, `idf.py monitor`, and the Almanach Go binary open. The new `ble-provision` Glazed verb puts provisioning into the same CLI that already renders, inspects, serves, and prints Almanach layouts:

```text
almanach-render-service ble-provision \
  --action provision \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --ssid YOUR_WIFI \
  --passphrase YOUR_PASSWORD
```

The first implementation intentionally wraps Espressif's maintained `esp_prov.py` instead of reimplementing BLE GATT, protobuf framing, Security 1, X25519 key exchange, AES-CTR encryption, and WiFi provisioning protobuf messages in Go. The command is still a Go/Glazed command: it validates inputs, resolves ESP-IDF paths, redacts secrets in structured output, streams the provisioning tool output, returns Glazed rows, and supports a protocol version smoke test.

## Why This Exists

### The problem

A freshly erased AtomS3R does not know the local WiFi network. The firmware cannot serve `/api/status`, `/almanach`, or print endpoints until it joins WiFi. Before this work, testing provisioning required one of these paths:

- USB serial console commands such as `wifi_connect`.
- Espressif's phone app.
- Direct manual use of `/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py`.

Those paths work, but they split the workflow. The developer must remember service names, PoP derivation, Python environment details, BLE dependency setup, and exact `esp_prov.py` flags.

### The goal

The goal is a local Linux loop:

1. Build and flash firmware.
2. Watch serial monitor until the firmware prints the BLE provisioning identity.
3. Run one Almanach CLI command from Linux to provision WiFi.
4. Watch firmware logs transition to WiFi connected and web server started.
5. Hit `/api/status` or run print/render workflows.
6. Reset and repeat.

### Non-goals for this phase

This phase does **not** create a pure-Go ESP protocomm implementation. That remains possible later, but it is a larger task. This phase also does not replace the future browser/Web Bluetooth onboarding UI. It gives developers a reliable terminal-driven tool now.

## Current Implementation Status

Implemented files:

- `internal/app/cmd_ble_provision.go`
  - Defines the `BLEProvisionCommand` Glazed verb.
  - Resolves ESP-IDF and Python paths.
  - Builds safe `esp_prov.py` invocations.
  - Provides `provision`, `reset`, `reprov`, and `version` actions.
  - Redacts `--passphrase` in dry-run and structured output.
- `internal/app/cmd_root.go`
  - Registers `ble-provision` as a Glazed command next to `render`, `inspect`, and `print`.

Validated commands:

```text
go test ./...
go run ./cmd/almanach-render-service ble-provision --action version --service-name ALM_0F2320 --pop alm-0f2320 --timeout 30 --output yaml
```

Observed Linux BLE/protocol version check:

```text
Discovering...
Connecting...
Getting Services...
proto-ver response :  {
    "prov": {
        "ver": "v1.1",
        "sec_ver": 1,
        "sec_patch_ver": 0,
        "cap": ["wifi_scan"]
    }
}
==== Verified protocol version successfully ====
Disconnecting...
```

This proves that the Linux host can discover the AtomS3R BLE service, connect, find the provisioning characteristics, call the `proto-ver` endpoint, and decode the ESP-IDF provisioning manager response.

## System Overview

The full system has four layers:

```text
┌────────────────────────────────────────────────────────────────────┐
│ Linux developer shell                                               │
│                                                                    │
│  almanach-render-service ble-provision                              │
│    - Glazed flags and output                                        │
│    - path resolution                                                │
│    - subprocess supervision                                         │
│    - structured result row                                          │
└───────────────────────────────┬────────────────────────────────────┘
                                │ exec + IDF_PATH
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│ ESP-IDF host provisioning client                                    │
│                                                                    │
│  tools/esp_prov/esp_prov.py                                         │
│    - bleak BLE client                                               │
│    - protobuf request/response models                               │
│    - protocomm Security 1                                           │
│    - WiFi scan/config/apply/status                                  │
└───────────────────────────────┬────────────────────────────────────┘
                                │ BLE GATT writes/notifications
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│ AtomS3R ESP-IDF provisioning transport                              │
│                                                                    │
│  wifi_prov_scheme_ble + protocomm                                   │
│    - service UUID                                                   │
│    - endpoint characteristics                                       │
│    - prov-session, proto-ver, prov-config, prov-scan, prov-ctrl      │
└───────────────────────────────┬────────────────────────────────────┘
                                │ calls into WiFi provisioning manager
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│ Almanach firmware application                                       │
│                                                                    │
│  provisioning_mgr.c + app_main.c + wifi_mgr.c                       │
│    - starts BLE when no credentials exist                           │
│    - stores provisioned credentials via ESP-IDF WiFi/NVS             │
│    - connects station interface                                     │
│    - starts web server after WiFi connects                           │
└────────────────────────────────────────────────────────────────────┘
```

## Firmware Concepts the Intern Must Understand

### Provisioning state versus explicit console credentials

The firmware has two credential paths:

1. ESP-IDF provisioning manager state.
   - Used by BLE provisioning.
   - Stored by ESP-IDF WiFi/NVS internals.
   - Checked with `wifi_prov_mgr_is_provisioned()`.
2. Almanach explicit console credentials.
   - Used by legacy `wifi_connect --ssid ... --pass ...`.
   - Stored by `nvs_store_save_wifi()`.
   - Erased by `nvs_store_erase_wifi()`.

The boot flow prefers ESP-IDF provisioned credentials, then falls back to explicit console credentials, then starts BLE provisioning.

Pseudocode:

```c
app_main() {
    nvs_store_init();
    wifi_mgr_init();
    provisioning_mgr_init();

    if (provisioning_mgr_is_provisioned()) {
        wifi_mgr_start_station();
    } else if (nvs_store_load_wifi(&ssid, &pass) == ESP_OK) {
        wifi_mgr_connect(ssid, pass);
    } else {
        provisioning_mgr_start_if_needed(NULL);
    }

    console_start();
    wait_for_wifi_then_start_web_server();
}
```

### BLE service name and PoP

The firmware derives stable development credentials from the device MAC address. On the tested AtomS3R:

```text
WiFi MAC     : 98:88:e0:0f:23:20
Service name : ALM_0F2320
PoP          : alm-0f2320
```

The user can get the same values from serial monitor:

```text
stoms3r> prov_status
Provisioning manager:
  initialized      : yes
  provisioned      : no
  BLE running      : yes
  service name     : ALM_0F2320
  PoP              : alm-0f2320
```

### Standard ESP-IDF endpoints

The BLE provisioning manager exposes standard protocomm endpoints:

| Endpoint | Purpose |
|---|---|
| `proto-ver` | Returns provisioning protocol version, security version, and capabilities. |
| `prov-session` | Runs Security 1 session setup. |
| `prov-config` | Sends WiFi credentials, applies config, and polls connection status. |
| `prov-scan` | Asks the device to scan nearby WiFi networks. |
| `prov-ctrl` | Resets or reprovisions WiFi state. |

The Go command does not manually encode these endpoints. It delegates them to Espressif's `esp_prov.py`.

## Host-Side Concepts the Intern Must Understand

### Why Linux BLE is special

On Linux, BLE access goes through BlueZ and D-Bus. The Python provisioning client uses `bleak`, which talks to BlueZ. Common failures are environmental rather than firmware bugs:

- Bluetooth service not running.
- User lacks permissions for the adapter.
- Another program is connected to the device.
- Device is already provisioned and no longer advertising.
- Python environment lacks `bleak`, `protobuf`, or `cryptography`.

Useful checks:

```bash
systemctl status bluetooth
bluetoothctl show
bluetoothctl scan on
```

### ESP-IDF Python environment

The command defaults to:

```text
IDF_PATH=/home/manuel/esp/esp-idf-5.4.2
python=/home/manuel/.espressif/python_env/idf5.4_py3.13_env/bin/python
esp_prov=/home/manuel/esp/esp-idf-5.4.2/tools/esp_prov/esp_prov.py
```

If imports fail, install dependencies into that Python environment:

```bash
/home/manuel/.espressif/python_env/idf5.4_py3.13_env/bin/python -m pip install protobuf cryptography bleak
```

This has been done locally during implementation.

## The `ble-provision` Command

### Command shape

```text
almanach-render-service ble-provision [flags]
```

Important flags:

| Flag | Meaning |
|---|---|
| `--action provision` | Send WiFi credentials and apply them. |
| `--action reset` | Reset provisioning state using `prov-ctrl`. |
| `--action reprov` | Ask provisioning manager to reprovision WiFi. |
| `--action version` | Linux BLE/protocol smoke test without sending WiFi credentials. |
| `--service-name` | BLE device name, e.g. `ALM_0F2320`. Required. |
| `--pop` | Security 1 proof-of-possession, e.g. `alm-0f2320`. |
| `--ssid` | WiFi network SSID for provisioning. |
| `--passphrase` | WiFi passphrase. If omitted for provisioning, the wrapper reads one line from stdin. |
| `--idf-path` | ESP-IDF checkout path. |
| `--python` | Python interpreter to run Espressif tooling. |
| `--esp-prov` | Explicit `esp_prov.py` path. |
| `--timeout` | Subprocess timeout in seconds. |
| `--dry-run` | Show resolved command without executing. |
| `--install-hints` | Print Linux/ESP-IDF dependency hints. |

### Dry-run example

```bash
go run ./cmd/almanach-render-service ble-provision \
  --action provision \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --ssid NEMO \
  --passphrase secret \
  --dry-run \
  --output yaml
```

Expected behavior:

- Prints the resolved subprocess command to stderr.
- Emits one Glazed row.
- Redacts the WiFi passphrase in `command` output.
- Does not touch the device.

### Version smoke-test example

```bash
go run ./cmd/almanach-render-service ble-provision \
  --action version \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --timeout 30 \
  --output yaml
```

This action uses a small Python snippet rather than full `esp_prov.py` because the Espressif CLI does not provide a standalone "check version and exit" mode. The snippet imports `esp_prov`, connects over BLE, calls `version_match()`, prints the raw `proto-ver` response in verbose mode, disconnects, and exits.

Pseudocode:

```python
idf = os.environ['IDF_PATH']
sys.path.insert(0, idf + '/components/protocomm/python')
sys.path.insert(1, idf + '/tools/esp_prov')
import esp_prov

tp = await esp_prov.get_transport('ble', service_name)
ok = await esp_prov.version_match(tp, 'v1.1', verbose=True)
await tp.disconnect()
exit(0 if ok else 1)
```

### Provision example

```bash
go run ./cmd/almanach-render-service ble-provision \
  --action provision \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --ssid YOUR_WIFI \
  --passphrase YOUR_PASSWORD \
  --timeout 120 \
  --output yaml
```

Expected Espressif output:

```text
Discovering...
Connecting...
Getting Services...
==== Security Scheme: 1 ====
==== Starting Session ====
==== Session Established ====
==== Sending Wi-Fi Credentials to Target ====
==== Wi-Fi Credentials sent successfully ====
==== Applying Wi-Fi Config to Target ====
==== Apply config sent successfully ====
==== Provisioning was successful ====
Disconnecting...
```

Expected firmware monitor output:

```text
WIFI_PROV_CRED_RECV
WIFI_PROV_CRED_SUCCESS
WIFI_PROV_END
wifi: connected with <ssid>
web_server: starting HTTP server
```

### Reset example

```bash
go run ./cmd/almanach-render-service ble-provision \
  --action reset \
  --service-name ALM_0F2320 \
  --pop alm-0f2320
```

Use reset when provisioning failed and the device is still advertising. If the firmware is already connected and no longer advertising, use the serial console instead:

```text
stoms3r> prov_reset
```

## Implementation Walkthrough

### Registration in the root command

`internal/app/cmd_root.go` follows the existing Almanach pattern:

```go
bleProvisionCmd, err := newBLEProvisionCommand()
if err != nil {
    return nil, err
}
if err := addGlazedCommand(rootCmd, bleProvisionCmd); err != nil {
    return nil, err
}
```

This keeps `ble-provision` consistent with `render`, `inspect`, and `print`. The command participates in Glazed parsing, `--output yaml`, command settings, and logging setup.

### Command description and flags

`internal/app/cmd_ble_provision.go` creates a `cmds.CommandDescription` with `cmds.WithFlags(...)` and the standard Glazed sections:

```go
glazedSection, _ := settings.NewGlazedSchema()
commandSettingsSection, _ := cli.NewCommandSettingsSection()

desc := cmds.NewCommandDescription(
    "ble-provision",
    cmds.WithShort("Provision an Almanach AtomS3R over ESP-IDF BLE provisioning"),
    cmds.WithFlags(bleProvisionFields()...),
    cmds.WithSections(glazedSection, commandSettingsSection),
)
```

### Path resolution

The command chooses paths in this order:

1. Explicit flag values.
2. Environment variables such as `IDF_PATH`.
3. Known local defaults:
   - `$HOME/esp/esp-idf-5.4.2`
   - `$HOME/esp/esp-idf-5.4.1`
   - `$HOME/esp/esp-idf`
4. Fallback to `python3` if no ESP-IDF Python env is found.

Pseudocode:

```go
idfPath := flag.IDFPath
if idfPath == "" {
    idfPath = os.Getenv("IDF_PATH")
}
if idfPath == "" {
    idfPath = firstExisting("~/esp/esp-idf-5.4.2", "~/esp/esp-idf-5.4.1", "~/esp/esp-idf")
}

python := firstExisting(
    "~/.espressif/python_env/idf5.4_py3.13_env/bin/python",
    "~/.espressif/python_env/idf5.4_py3.12_env/bin/python",
    "~/.espressif/python_env/idf5.4_py3.11_env/bin/python",
)
```

### Secret handling

The current wrapper passes `--passphrase` to `esp_prov.py` because the upstream tool accepts noninteractive credentials that way. To reduce accidental leaks:

- The command redacts passphrases in `--dry-run` output.
- The command redacts passphrases in the Glazed `command` row.
- If `--passphrase` is omitted, the wrapper reads one line from stdin.

Security caveat: because `esp_prov.py` only accepts `--passphrase` or interactive prompt input, the child process may still briefly contain the passphrase in its argument vector for noninteractive runs. A future pure-Go or patched-Python client can avoid that entirely.

## Validation Runbook

### 1. Build firmware and CLI

```bash
cd almanach
cd firmware/atoms3r && ./build.sh /dev/ttyACM0 build
cd ../..
go test ./...
go build ./cmd/almanach-render-service
```

### 2. Flash a clean device

```bash
cd firmware/atoms3r
./build.sh /dev/ttyACM0 erase-flash
./build.sh /dev/ttyACM0 flash
./build.sh /dev/ttyACM0 monitor
```

Wait for:

```text
No saved WiFi credentials — starting BLE provisioning
Provisioning started with service name : ALM_0F2320
PoP       : alm-0f2320
```

### 3. Smoke-test Linux BLE/protocol connection

In a second terminal:

```bash
cd almanach
go run ./cmd/almanach-render-service ble-provision \
  --action version \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --timeout 30 \
  --output yaml
```

Pass criteria:

- BLE discovery succeeds.
- GATT services are found.
- `proto-ver` response contains `"ver": "v1.1"`.
- Command exits with `exit_code: 0`.

### 4. Provision WiFi

```bash
go run ./cmd/almanach-render-service ble-provision \
  --action provision \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --ssid YOUR_WIFI \
  --passphrase YOUR_PASSWORD \
  --timeout 120 \
  --output yaml
```

Pass criteria:

- Host prints `Session Established`.
- Host prints `Wi-Fi Credentials sent successfully`.
- Host prints `Provisioning was successful`.
- Firmware monitor prints WiFi connected.
- Firmware starts web server.

### 5. Verify Almanach API

Once the firmware prints its IP address:

```bash
curl http://DEVICE_IP/api/status
```

Pass criteria:

- HTTP request succeeds.
- JSON status reports connected WiFi or printer service information.

### 6. Reboot and autoconnect

```bash
# Press reset button or use idf monitor reset.
```

Pass criteria:

- BLE provisioning does not start on reboot.
- Firmware detects provisioned state.
- Firmware starts station mode and reconnects.
- Web server starts again.

### 7. Reset and repeat

Either from serial console:

```text
stoms3r> prov_reset
```

Or while still advertising from BLE:

```bash
go run ./cmd/almanach-render-service ble-provision \
  --action reset \
  --service-name ALM_0F2320 \
  --pop alm-0f2320
```

Pass criteria:

- Provisioning state is cleared.
- Device reboots or returns to unprovisioned state.
- BLE provisioning advertises again.

## Troubleshooting

### `ModuleNotFoundError: No module named 'google'`

Cause: ESP-IDF Python environment lacks `protobuf`.

Fix:

```bash
/home/manuel/.espressif/python_env/idf5.4_py3.13_env/bin/python -m pip install protobuf
```

### `No module named bleak`

Cause: BLE Python dependency is missing.

Fix:

```bash
/home/manuel/.espressif/python_env/idf5.4_py3.13_env/bin/python -m pip install bleak
```

### BLE discovery cannot find the device

Check:

- Is firmware advertising? Run `prov_status` on serial console.
- Is the device already provisioned? If yes, it may no longer advertise.
- Is Bluetooth enabled on the host?
- Is another app connected to `ALM_0F2320`?

Commands:

```bash
bluetoothctl show
bluetoothctl scan on
```

### Protocol version mismatch

Expected provisioning protocol version for ESP-IDF 5.4 `wifi_prov_mgr` is `v1.1`. If version check fails:

- Confirm firmware actually runs ESP-IDF provisioning manager.
- Confirm the service name points at the Almanach device, not a stale BLE peripheral.
- Confirm the firmware was flashed after BLE provisioning code was added.

### Session establishment fails

Likely causes:

- Wrong PoP.
- Device was reset halfway through provisioning.
- Host connected to old advertisement data.

Fix:

1. Check serial console `prov_status` for service name and PoP.
2. Re-run with correct `--pop`.
3. If still broken, run `prov_reset` from serial console and retry.

## Future Work

### Pure-Go provisioning client

A future implementation could remove the Python dependency by implementing:

- Linux BLE transport with a Go BLE library.
- ESP-IDF endpoint discovery.
- Protobuf messages for:
  - `session.proto`
  - `sec1.proto`
  - `wifi_config.proto`
  - `wifi_scan.proto`
  - `wifi_ctrl.proto`
- Security 1:
  - X25519 key exchange.
  - PoP proof calculation.
  - AES-CTR encrypted protocomm payloads.
- Structured scan results and status polling as native Glazed rows.

That would produce a more self-contained binary, but it is more work and carries protocol compatibility risk. The wrapper is the best first step because it validates the firmware and host BLE environment immediately.

### Better secret handling

Potential improvements:

- Patch or wrap Espressif tooling to accept passphrase from stdin or environment.
- Add an interactive hidden prompt using `golang.org/x/term`.
- Avoid command-line passphrase entirely in a pure-Go implementation.

### More actions

Useful future actions:

- `--action scan` returning WiFi scan rows.
- `--action status` polling provisioning connection state.
- `--action wait-api --printer-ip ...` to wait for `/api/status` after provisioning.
- `--action monitor-loop` to provision, poll API, and optionally print a test page.

## Intern Checklist

Before changing code:

- [ ] Read this document.
- [ ] Read `internal/app/cmd_ble_provision.go`.
- [ ] Read `firmware/atoms3r/main/provisioning_mgr.c`.
- [ ] Flash a clean device and run `prov_status` in serial monitor.
- [ ] Run `ble-provision --action version` successfully.

When changing the command:

- [ ] Keep passphrases redacted in structured output.
- [ ] Preserve `--dry-run` behavior.
- [ ] Keep `go test ./...` passing.
- [ ] Test against a real advertising AtomS3R.
- [ ] Update this design doc and the ticket diary.

When validating end-to-end provisioning:

- [ ] Erase flash.
- [ ] Flash firmware.
- [ ] Confirm BLE advertisement.
- [ ] Run version smoke test.
- [ ] Run provisioning with real WiFi credentials.
- [ ] Confirm firmware connects.
- [ ] Confirm web API responds.
- [ ] Reboot and confirm autoconnect.
- [ ] Reset and confirm BLE advertising returns.
