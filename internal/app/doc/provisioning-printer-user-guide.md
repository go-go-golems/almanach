---
Title: "Provisioning an Almanach Printer"
Slug: "provisioning-printer-user-guide"
Short: "Set up an AtomS3R printer over BLE, hand its WiFi IP back to the local render server, and recover with reset paths."
Topics:
- provisioning
- bluetooth
- wifi
- setup
- printer
Commands:
- setup
- ble-provision
Flags:
- implementation
- action
- service-name
- pop
- ssid
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This guide walks through first-time WiFi provisioning for an Almanach AtomS3R printer. The setup page is served by the local `almanach-render-service` binary, while Chrome talks to the printer over Web Bluetooth. After the printer joins WiFi, the browser reports the printer IP back to the local server so the same server can render pages and send them to the newly provisioned device.

Use this guide when the printer is new, after you reset WiFi settings, or when you want to verify that the local render server knows which printer IP to use.

## What Happens During Provisioning

Provisioning crosses two networks. The browser starts on localhost, then temporarily talks to the printer over BLE. The printer receives WiFi credentials, joins the access point, reports its connected status, and then the browser posts the result back to the local server.

```text
localhost setup page -> Chrome Web Bluetooth -> AtomS3R BLE provisioning
AtomS3R joins WiFi -> browser receives connected status -> POST result to localhost server
```

This shape matters because the printer cannot serve its own setup page before it has WiFi. The local server is the rendezvous point: it serves the setup UI before provisioning and remembers the device endpoint after provisioning.

## Prerequisites

Before starting, make sure you have:

- Chrome or Edge with Web Bluetooth support.
- The Almanach binary running on the same machine as the browser.
- The printer in BLE provisioning mode, advertising as `ALM_...`.
- The proof of possession printed by firmware logs or the device status screen, for example `alm-0f2320`.
- A 2.4 GHz WiFi SSID and password.

If the printer is already provisioned, BLE provisioning may have stopped. Use the physical long-hold reset or serial `prov_reset` command to return it to setup mode.

## Start the Local Setup Server

Run the setup server from the repository or from an installed binary:

```bash
almanach-render-service setup --port 18299
```

Open the printed URL:

```text
http://localhost:18299/setup
```

`localhost` is important. Chrome treats localhost as a secure context, which allows Web Bluetooth without HTTPS certificates.

## Provision from the Browser

On the setup page:

1. Enter the proof of possession, such as `alm-0f2320`.
2. Enter the WiFi SSID and password.
3. Click **Find BLE printer**.
4. Select the `ALM_...` device in Chrome's Bluetooth picker.
5. Wait for the setup page to verify `proto-ver`.
6. Click **Continue provisioning**.

A successful progress log looks like this:

```text
Verified ESP-IDF provisioning protocol v1.1
Security 1 session established
Sending encrypted WiFi credentials for SSID ...
Applying encrypted WiFi configuration
WiFi provisioning status: connecting
WiFi provisioning status: connected
Printer connected to WiFi successfully.
Reported provisioned printer ... to local setup server.
Disconnected from ALM_...
```

The disconnect at the end is expected. ESP-IDF stops BLE provisioning after WiFi succeeds.

## Verify the Local Server Learned the Printer IP

The setup page reports the provisioned device to the same local server that served the page. The server persists that record in a local JSON state file so it survives restarts. By default the file is:

```text
~/.config/almanach/render-service/state.json
```

Override it with `ALMANACH_STATE_FILE` or the `--state-file` flag. You can inspect the remembered device with:

```bash
curl http://localhost:18299/api/setup/provisioned-device
```

A successful response looks like:

```json
{
  "ok": true,
  "device": {
    "serviceName": "ALM_0F2320",
    "ip": "192.168.1.242",
    "ssid": "Verizon_9DNVB9",
    "source": "web-bluetooth",
    "seenAt": "2026-05-10T20:56:18Z"
  }
}
```

The server uses this IP as the active printer endpoint when `ALMANACH_PRINTER_IP` is not explicitly configured. This lets the setup flow hand off naturally into rendering and printing. Explicit configuration still wins: if `ALMANACH_PRINTER_IP` is set, the persisted setup-discovered IP is ignored.

To forget the persisted device:

```bash
curl -X DELETE http://localhost:18299/api/setup/provisioned-device
```

## Provision from the Native CLI

The browser is the recommended interactive path. The native CLI is useful for debugging or automation:

```bash
almanach-render-service ble-provision \
  --implementation native \
  --action provision \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --ssid YOUR_WIFI
```

If you omit `--passphrase`, the command reads it from stdin so it does not appear in shell history.

Use `--action version` to test BLE transport and `proto-ver` without changing WiFi state:

```bash
almanach-render-service ble-provision \
  --implementation native \
  --action version \
  --service-name ALM_0F2320 \
  --pop alm-0f2320
```

## Reset and Recovery

There are three recovery paths.

| Situation | Recommended action |
|---|---|
| BLE provisioning is active and the setup page is connected. | Use **Reset printer WiFi** or **Reprovision** in the setup page. |
| BLE provisioning is active and you want a CLI test. | Use `ble-provision --implementation native --action reset` or `--action reprov`. |
| The printer is already provisioned and BLE has stopped. | Use the physical long-hold reset or serial `prov_reset`. |

The physical reset path is the most reliable fallback because it does not require BLE or WiFi. Hold the device button until the reset threshold is reached. The firmware clears WiFi/provisioning state and reboots into the normal first-boot setup path.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Chrome says Web Bluetooth is unavailable. | The page is not running in a secure context or the browser does not support Web Bluetooth. | Open `http://localhost:<port>/setup` in Chrome or Edge. |
| No `ALM_...` device appears. | The printer is not in BLE provisioning mode. | Use physical reset or serial `prov_reset`, then reload the setup page. |
| `proto-ver` fails. | The browser selected the wrong device or service discovery failed. | Reconnect, check the service name, and confirm firmware is advertising ESP-IDF provisioning. |
| Security 1 fails. | The PoP is wrong or the session stream is out of sync. | Re-enter the PoP exactly as shown by firmware/device status and retry from a fresh connection. |
| WiFi status stays `connecting`. | The AP is slow, out of range, or credentials are wrong. | Wait briefly, then reset/reprovision with known-good 2.4 GHz credentials. |
| The local server has no printer IP after success. | The browser could not report the connected result back to localhost. | Check `/api/setup/provisioned-device`, keep the setup page on the same localhost server, and retry provisioning. |
| The server uses an old printer IP after restart. | The setup-discovered endpoint is persisted in the local state file. | Delete it with `curl -X DELETE http://localhost:18299/api/setup/provisioned-device`, remove the state file, or set `ALMANACH_PRINTER_IP` explicitly. |

## See Also

- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help layout-dsl-reference`
- `almanach-render-service ble-provision --help`
- `almanach-render-service setup --help`
