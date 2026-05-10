---
Title: Web Bluetooth Provisioning UI Design and Implementation Guide
Ticket: ALMANACH-BLE-PROVISION
Status: active
Topics:
    - almanach
    - firmware
    - esp-idf
    - ble
    - wifi-provisioning
    - thermal-printer
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: firmware/atoms3r/main/app_main.c
      Note: |-
        Current boot flow, WiFi wait behavior, console registration, and missing web-visible provisioning integration.
        Current onboarding boot flow and provisioning command registration
    - Path: firmware/atoms3r/main/provisioning_mgr.c
      Note: |-
        Firmware BLE provisioning manager, service identity, PoP, custom service UUID, events, and reset/status APIs.
        Firmware BLE provisioning service name
    - Path: firmware/atoms3r/main/provisioning_mgr.h
      Note: |-
        Firmware provisioning status and control API that should be reflected in the UI.
        Firmware provisioning status struct and public API for UI status mapping
    - Path: firmware/atoms3r/main/web_server.c
      Note: |-
        Current embedded HTTP APIs and Almanach Studio asset serving behavior.
        Current firmware HTTP API and embedded Almanach route behavior
    - Path: web/esbuild.mjs
      Note: |-
        Current esbuild bundle entry and generated firmware/render-service HTML shell.
        Frontend build output and generated HTML shell used by render service and firmware asset sync
    - Path: web/package.json
      Note: |-
        Current frontend dependency surface and build scripts.
        Current frontend dependency and build script surface for adding Web Bluetooth/protobuf dependencies
    - Path: web/src/almanach-studio.jsx
      Note: |-
        Current monolithic React Almanach Studio UI, print path, localStorage settings, and headless render API.
        Current React editor
ExternalSources:
    - 'Web Bluetooth API: https://developer.mozilla.org/en-US/docs/Web/API/Web_Bluetooth_API'
    - 'BluetoothRemoteGATTServer: https://developer.mozilla.org/en-US/docs/Web/API/BluetoothRemoteGATTServer'
    - 'ESP-IDF WiFi Provisioning Manager: https://docs.espressif.com/projects/esp-idf/en/latest/esp32/api-reference/provisioning/wifi_provisioning.html'
    - 'ESP-IDF Protocomm: https://docs.espressif.com/projects/esp-idf/en/latest/esp32/api-reference/provisioning/protocomm.html'
    - 'esp-idf-provisioning-web npm package: https://www.npmjs.com/package/esp-idf-provisioning-web'
Summary: Design and implementation guide for adding a browser-based BLE WiFi provisioning flow to the Almanach web app while respecting Web Bluetooth constraints, ESP-IDF protocomm semantics, and the existing AtomS3R firmware provisioning manager.
LastUpdated: 2026-05-10T17:30:00-04:00
WhatFor: Use this as the intern-facing guide for adding a Web Bluetooth provisioning panel and client-side provisioning service to ./almanach/web.
WhenToUse: Read before changing the React Almanach Studio UI, adding Web Bluetooth/protobuf dependencies, modifying the frontend build, or exposing firmware provisioning status to browsers.
---


# Web Bluetooth Provisioning UI Design and Implementation Guide

## Executive Summary

The Almanach web app currently edits, renders, exports, and prints thermal-printer layouts. It does not currently configure a fresh AtomS3R printer's WiFi. The firmware ticket already added the first pieces needed on the device side: a `provisioning_mgr` subsystem that starts ESP-IDF BLE WiFi provisioning, advertises a MAC-derived service name such as `ALM_A1B2C3`, uses Security 1 proof-of-possession, and tracks BLE/protocomm events. The next product step is to add a browser-based provisioning experience to `./almanach/web` so a user can open an Almanach-hosted page, click **Set up printer**, choose the BLE device, enter the proof-of-possession and WiFi credentials, and wait until the printer joins the LAN.

The most important design constraint is that Web Bluetooth is a browser API, not a normal web fetch API. It requires a secure context and a user gesture. In practice, Chrome-based browsers expose `navigator.bluetooth` on `https://` origins and on `localhost`; they do not expose it on ordinary `http://192.168.x.y/almanach` pages served by the firmware. Therefore, the BLE provisioning UI should live in the top-level `almanach/web` app and should be served by the Go render-service on `localhost` during development or by a future HTTPS-hosted Almanach site in production. The firmware-embedded `/almanach` page can still include status and instructions after WiFi is connected, but it should not be the primary Web Bluetooth provisioning entrypoint unless the project later adds HTTPS/TLS for the device UI.

This guide proposes a staged implementation:

1. Extract provisioning UI and protocol code into focused frontend modules instead of growing the 2,594-line `almanach-studio.jsx` monolith further.
2. Add a top-level provisioning panel/modal in Almanach Studio's top bar.
3. Implement a small `web/src/provisioning/` client boundary with a stable state-machine API.
4. Use an existing browser ESP-IDF provisioning package if it proves reliable; otherwise implement the Web Bluetooth transport and call the ESP-IDF protobuf/protocomm operations behind the same boundary.
5. Keep firmware status integration separate from BLE provisioning: after credentials are sent, poll `http://<device-ip>/api/status` only once the user or provisioning result provides a reachable address.
6. Add devctl and build validation so the feature is testable without flashing firmware for every UI change.

The recommendation is to implement the UI around a clean internal adapter interface first. That keeps the React work independent from the exact BLE/protocomm library decision and lets an intern start with mock states, then wire real hardware once the user journey is visible.

## Problem Statement and Scope

### User problem

A new Almanach printer starts with no WiFi credentials. The firmware can print and serve a web UI once it is connected, but a first-time user still needs a way to put credentials onto the device. The current developer path is either USB console commands or Espressif's mobile app. That is not a polished Almanach onboarding path.

The target experience is:

```text
Open Almanach setup page
  -> click Set up printer
  -> browser asks to select BLE device named ALM_xxxxxx
  -> enter/check PoP shown by firmware docs, label, serial console, or QR
  -> enter WiFi SSID/password
  -> browser sends credentials over ESP-IDF BLE provisioning protocol
  -> printer joins WiFi
  -> UI shows next step: open http://<printer-ip>/almanach or print a test page
```

### Engineering problem

The frontend must bridge three different worlds:

- React UI state and existing Almanach Studio interactions.
- Web Bluetooth's permissioned browser API.
- ESP-IDF's BLE provisioning protocol, which is not a simple JSON-over-GATT protocol; it is protocomm plus protobuf plus a security handshake.

The design must avoid mixing all three directly into the current `AlmanachStudio()` component. That component already owns layout editing, paper preview rendering, JSON import/export, PNG export, direct print POSTing, localStorage settings, and the headless render API. Adding provisioning inline would make the file much harder for a new engineer to reason about.

### In scope

- Add a provisioning entry point to `./almanach/web`.
- Implement a React provisioning UI flow for BLE device selection, PoP, WiFi credentials, progress, and final status.
- Add a frontend adapter layer around Web Bluetooth and ESP-IDF provisioning operations.
- Add explicit browser capability checks and secure-context errors.
- Preserve existing print/export/headless render behavior.
- Document where firmware endpoints/status fit and where they do not.
- Provide pseudocode, diagrams, API references, and file-level implementation guidance.

### Out of scope for this guide

- Implementing firmware BLE provisioning itself. That is covered by the existing `01-ble-wifi-provisioning-port-analysis-design-and-implementation-guide.md` and ongoing firmware tasks.
- Adding HTTPS to the ESP32 firmware HTTP server.
- Replacing the ESP-IDF provisioning protocol with a custom BLE service.
- Building iOS/Safari support. Web Bluetooth is not broadly available there.
- Full cloud account/device registry behavior.

## Current-State Analysis

### Current web app shape

The frontend is currently a small build wrapper around one large React file:

```text
web/
├── package.json
├── esbuild.mjs
├── index.html
└── src/
    ├── index.jsx
    └── almanach-studio.jsx
```

`web/package.json` has only React, React DOM, Lucide icons, Vite, and esbuild dependencies. The package is private, uses `pnpm@10.15.0`, and builds with `node esbuild.mjs` (`web/package.json:1-22`). This is good for an embedded firmware UI because the dependency surface is small. Adding BLE provisioning should be done deliberately because protobuf and crypto dependencies can increase bundle size.

The esbuild script bundles `src/index.jsx` into a single IIFE at `dist/almanach-bundle.js`, sets the browser global name to `AlmanachStudio`, targets ES2020, and writes a generated `dist/index.html` shell that loads `/almanach/bundle.js` (`web/esbuild.mjs:6-18`, `web/esbuild.mjs:21-40`). This matters because any provisioning code added to the main app will be included in the firmware-embedded bundle unless the build is split.

The app component is currently monolithic. `web/src/almanach-studio.jsx` imports React hooks and all icons at the top (`web/src/almanach-studio.jsx:1-9`), defines all themes, block renderers, editors, export helpers, the headless render API, and the top-level UI in one file. The main app starts at `AlmanachStudio()` (`web/src/almanach-studio.jsx:1481`). It stores user settings in localStorage (`web/src/almanach-studio.jsx:1484-1512`), exposes headless functions for the Go render service (`web/src/almanach-studio.jsx:1514-1617`), and renders the top bar/workspace/rails (`web/src/almanach-studio.jsx:1860-2594`).

### Current print and firmware HTTP integration

The browser-side direct print path renders the paper DOM into an off-screen canvas, converts pixels into a 1-bit bitmap, pads width to a byte boundary, and sends the bitmap to the firmware endpoint:

```javascript
const r = await fetch("/api/print/bitmap", {
  method: "POST",
  headers: {
    "Content-Type": "application/octet-stream",
    "X-Width": String(Wpadded),
    "X-Height": String(H),
    "X-Feed": String(feedLines),
  },
  body: bitmap,
});
```

That observed path appears in `handlePrint()` (`web/src/almanach-studio.jsx:1666-1779`), with the actual POST at `web/src/almanach-studio.jsx:1755-1765`. The firmware implements the target HTTP endpoint and status APIs in `web_server.c`. `/api/status` reports WiFi connectivity and IP (`firmware/atoms3r/main/web_server.c:99-119`). `/almanach` and `/almanach/bundle.js` serve the embedded React app (`firmware/atoms3r/main/web_server.c:336-355`), and the HTTP server registers those routes plus `/api/print/bitmap` (`firmware/atoms3r/main/web_server.c:360-424`).

This existing HTTP integration only works after the device has WiFi and the HTTP server is running. `app_main.c` starts a background task that waits up to 30 seconds for WiFi and then starts the web server (`firmware/atoms3r/main/app_main.c:91-105`). If WiFi is absent, the task logs that the web server was not started. So a first-boot web UI cannot be served by the firmware over LAN, and a first-boot Web Bluetooth provisioning UI must be served from a separate origin such as localhost or HTTPS.

### Current firmware BLE provisioning state

The firmware now has a provisioning manager source file. It builds a service identity from the station MAC address (`firmware/atoms3r/main/provisioning_mgr.c:26-41`), logs an Espressif-compatible provisioning payload (`firmware/atoms3r/main/provisioning_mgr.c:43-52`), tracks provisioning, BLE transport, and security events (`firmware/atoms3r/main/provisioning_mgr.c:54-134`), initializes `wifi_prov_scheme_ble` with BTDM cleanup (`firmware/atoms3r/main/provisioning_mgr.c:158-186`), starts Security 1 provisioning with the derived PoP (`firmware/atoms3r/main/provisioning_mgr.c:220-241`), resets provisioning state (`firmware/atoms3r/main/provisioning_mgr.c:257-265`), and exposes a `provisioning_status_t` structure in the header (`firmware/atoms3r/main/provisioning_mgr.h:10-26`).

The firmware build already includes provisioning components. `main/CMakeLists.txt` includes `provisioning_mgr.c` in `SRCS` and `protocomm`, `wifi_provisioning`, and `bt` in `PRIV_REQUIRES` (`firmware/atoms3r/main/CMakeLists.txt:1-22`). The default config enables BLE and NimBLE plus protocomm security 1 (`firmware/atoms3r/sdkconfig.defaults:26-34`).

`app_main.c` now includes `provisioning_cmd.h` and `provisioning_mgr.h`, delegates WiFi onboarding to `start_network_onboarding()`, checks ESP-IDF provisioning state, falls back to console-saved NVS credentials, and starts BLE provisioning when no saved credentials exist (`firmware/atoms3r/main/app_main.c:18-23`, `firmware/atoms3r/main/app_main.c:91-124`). It also registers `provisioning_cmd_register()` beside the printer and WiFi console commands (`firmware/atoms3r/main/app_main.c:184-189`). The remaining firmware gap for this web UI design is not BLE advertising in general; it is a browser-consumable status/diagnostic surface and hardware validation that the Web Bluetooth client can complete the same ESP-IDF provisioning flow.

### Current topbar placement for a setup entrypoint

The top bar already has grouped actions for block panels, open/save, PNG export, and print (`web/src/almanach-studio.jsx:2361-2426`). A provisioning entry point belongs in this topbar because it is a device-level action, not a block-level editor action. It should sit near the Print button or behind a small device/status button, not in the right rail with theme/font/feed controls.

## Critical Constraint: Web Bluetooth Secure Contexts

A new intern must understand this before writing code: Web Bluetooth is only exposed in secure browser contexts. In Chrome-family browsers this usually means `https://...` or `http://localhost`. A firmware page served from `http://192.168.0.126/almanach` is not a secure context, so `navigator.bluetooth` will generally be missing there.

This has direct product implications:

- **Good development path:** `pnpm dev` or Go render service on `localhost` can host the provisioning UI.
- **Good production path:** a future `https://almanach.local` or cloud-hosted Almanach setup page can host the provisioning UI.
- **Bad first-boot assumption:** a user cannot first connect to a printer web page if the printer has no WiFi.
- **Bad embedded-page assumption:** even after WiFi, the firmware HTTP page cannot reliably start Web Bluetooth unless the browser grants secure-context treatment, which should not be assumed.

Therefore the initial implementation should add provisioning UI to `./almanach/web` but design it as a host-served setup tool. The same React bundle can still be embedded into firmware, but the UI must show a clear unsupported message when `window.isSecureContext` or `navigator.bluetooth` is unavailable.

Capability check pseudocode:

```javascript
export function getBluetoothSupport() {
  if (!window.isSecureContext) {
    return {
      ok: false,
      reason: "Web Bluetooth requires HTTPS or localhost.",
      hint: "Open Almanach from localhost during development or an HTTPS setup page in production.",
    };
  }
  if (!navigator.bluetooth) {
    return {
      ok: false,
      reason: "This browser does not expose navigator.bluetooth.",
      hint: "Use desktop Chrome, Edge, or Android Chrome. Safari/iOS are not supported.",
    };
  }
  return { ok: true };
}
```

## Proposed Architecture

### Architectural principle

Do not put Web Bluetooth protocol code directly inside `AlmanachStudio()`. The current file is already the whole editor. Add a thin UI entrypoint there, but move provisioning logic into modules that can be tested and replaced.

Recommended new files:

```text
web/src/provisioning/
├── types.js                  Shared state enums and small validation helpers
├── bluetooth-support.js      Secure-context and navigator.bluetooth capability checks
├── provisioning-client.js    Stable adapter interface used by React
├── esp-idf-ble-client.js     Real ESP-IDF BLE provisioning implementation
├── mock-client.js            Development/test fake for UI without hardware
├── ProvisioningWizard.jsx    Modal/panel React component
└── provisioning.css.js       Optional style string or exported style fragments
```

Minimal integration in `almanach-studio.jsx`:

```javascript
import { ProvisioningWizard } from "./provisioning/ProvisioningWizard";
import { Bluetooth } from "lucide-react";

// inside AlmanachStudio()
const [showProvisioning, setShowProvisioning] = useState(false);

<button className="btn" onClick={() => setShowProvisioning(true)}>
  <Bluetooth size={13} /> Set up printer
</button>

{showProvisioning && (
  <ProvisioningWizard
    onClose={() => setShowProvisioning(false)}
    onProvisioned={(result) => {
      flashToast("ok", `Printer joined WiFi${result.ip ? ` at ${result.ip}` : ""}`);
    }}
  />
)}
```

### Runtime flow diagram

```text
User browser, served from localhost or HTTPS
  |
  | click Set up printer (required user gesture)
  v
ProvisioningWizard
  |
  | getBluetoothSupport()
  | navigator.bluetooth.requestDevice({ filters/prefixes, optionalServices })
  v
Bluetooth device ALM_xxxxxx
  |
  | GATT connect
  | discover ESP provisioning service UUID
  | open protocomm security session with PoP
  | optionally scan networks
  | send SSID/password
  | poll apply/status
  v
AtomS3R firmware wifi_prov_mgr
  |
  | receives credentials
  | connects STA
  | stores WiFi through ESP-IDF provisioning/WiFi stack
  | emits WIFI_PROV_CRED_SUCCESS / WIFI_PROV_END
  v
ProvisioningWizard final screen
  |
  | show success and next actions
  | optionally ask user for printer IP or poll known /api/status URL
  v
Open /almanach or print test page
```

### State machine

The UI should be a state machine because BLE provisioning has many asynchronous phases and user-visible failure modes.

Recommended states:

```text
idle
  -> unsupported
  -> ready
ready
  -> choosing-device
choosing-device
  -> connecting
  -> error
connecting
  -> establishing-session
  -> error
establishing-session
  -> scanning-networks
  -> entering-wifi
  -> error
scanning-networks
  -> entering-wifi
  -> error
entering-wifi
  -> sending-credentials
sending-credentials
  -> waiting-for-device
  -> error
waiting-for-device
  -> provisioned
  -> error
provisioned
  -> closed
error
  -> ready
```

State shape:

```javascript
const initialState = {
  step: "idle",
  support: null,
  deviceName: "",
  serviceName: "",
  pop: "",
  ssid: "",
  password: "",
  networks: [],
  progress: [],
  error: null,
  result: null,
};
```

Actions:

```javascript
{ type: "SUPPORT_CHECKED", support }
{ type: "DEVICE_SELECTED", deviceName }
{ type: "SESSION_READY" }
{ type: "NETWORKS_SCANNED", networks }
{ type: "CREDENTIALS_SENT" }
{ type: "PROVISIONED", result }
{ type: "ERROR", error }
{ type: "RESET" }
```

### Adapter interface

React should talk to a small client interface, not directly to a library. This interface can be backed by a real ESP-IDF BLE implementation or by a mock during UI development.

```javascript
export class AlmanachProvisioningClient {
  async checkSupport() {}
  async chooseDevice({ namePrefix, serviceUuid }) {}
  async connect(device) {}
  async establishSession({ pop }) {}
  async scanNetworks() {}
  async sendCredentials({ ssid, password }) {}
  async waitForResult({ timeoutMs }) {}
  async disconnect() {}
}
```

Concrete return values:

```javascript
// chooseDevice
{
  device,
  name: "ALM_A1B2C3",
  id: "browser-private-device-id"
}

// scanNetworks
[
  { ssid: "Office", rssi: -48, auth: "WPA2" },
  { ssid: "Guest", rssi: -63, auth: "WPA2" }
]

// waitForResult
{
  ok: true,
  deviceName: "ALM_A1B2C3",
  message: "Credentials accepted. Printer is joining WiFi.",
  ip: null
}
```

The first implementation may skip network scanning if the chosen library makes scanning difficult. Manual SSID entry is acceptable for Phase 1; scanning can be Phase 2.

## BLE and ESP-IDF Provisioning Protocol Notes

### Firmware service identity

The current firmware derives the advertised service/device name from the last three bytes of the WiFi station MAC address:

```c
snprintf(s_service_name, sizeof(s_service_name),
         "ALM_%02X%02X%02X", mac[3], mac[4], mac[5]);
snprintf(s_pop, sizeof(s_pop),
         "alm-%02x%02x%02x", mac[3], mac[4], mac[5]);
```

This is in `provisioning_mgr.c:37-40`. The browser UI should default to scanning for devices whose name starts with `ALM_`, but it should also allow a manual advanced mode where the user can choose any compatible device. The UI should not hard-code a single service name.

### Service UUID

The firmware sets a custom BLE provisioning service UUID in `provisioning_mgr_init()` (`firmware/atoms3r/main/provisioning_mgr.c:174-178`):

```c
uint8_t custom_service_uuid[] = {
    0xb4, 0xdf, 0x5a, 0x1c, 0x3f, 0x6b, 0xf4, 0xbf,
    0xea, 0x4a, 0x82, 0x03, 0x04, 0x90, 0x1a, 0x02,
};
wifi_prov_scheme_ble_set_service_uuid(custom_service_uuid);
```

Browser code must use the same UUID in the `optionalServices` list and when discovering GATT services. Be careful about byte order. ESP-IDF's array is written in the byte order passed to the BLE stack; browser APIs usually use canonical UUID strings. The intern must validate the displayed service UUID with a BLE scanner or with Espressif's app before finalizing the string constant.

Recommended constant placeholder, to be verified on hardware:

```javascript
export const ALMANACH_PROV_SERVICE_UUID_BYTES = [
  0xb4, 0xdf, 0x5a, 0x1c, 0x3f, 0x6b, 0xf4, 0xbf,
  0xea, 0x4a, 0x82, 0x03, 0x04, 0x90, 0x1a, 0x02,
];

// Verify exact canonical string with nRF Connect / Chrome device logs.
export const ALMANACH_PROV_SERVICE_UUID = "021a9004-0382-4aea-bff4-6b3f1c5adfb4";
```

### Security

The firmware uses `WIFI_PROV_SECURITY_1` with PoP (`firmware/atoms3r/main/provisioning_mgr.c:229-236`). ESP-IDF protocomm security 1 uses a Curve25519 key exchange and AES-CTR encryption/decryption, with proof-of-possession supported by security 1. The browser client must perform the same session setup before sending WiFi credentials.

Do not send raw SSID/password bytes to arbitrary BLE characteristics. Use ESP-IDF's provisioning protocol.

### Endpoint names and protobuf messages

ESP-IDF provisioning normally uses named endpoints such as:

- `proto-ver` for version/capability information.
- `prov-session` for security session setup.
- `prov-scan` for WiFi scan requests/responses.
- `prov-config` for sending and applying credentials.

The exact BLE characteristic UUID mapping is handled by ESP-IDF's BLE protocomm scheme and by compatible provisioning clients. If using a package such as `esp-idf-provisioning-web`, prefer its abstractions first. If implementing manually, inspect ESP-IDF's generated protobuf definitions and BLE transport mapping in the ESP-IDF version used by the firmware (`~/esp/esp-idf-5.4.2`).

## Library Strategy

### Recommended Phase 1: adapter plus third-party package spike

The npm package `esp-idf-provisioning-web` advertises itself as an ESP-IDF WiFi Provisioning SDK for web browsers and depends on `protobufjs`. It is a good candidate for a first spike because it may already solve protobuf encoding, security handshake, and BLE endpoint mapping.

Recommended dependency spike:

```bash
cd almanach/web
pnpm add esp-idf-provisioning-web protobufjs
pnpm run build
```

Then create a tiny isolated module:

```javascript
// web/src/provisioning/espIdfProvisioningPackageSpike.js
import { /* package exports */ } from "esp-idf-provisioning-web";

export async function packageSmokeTest() {
  // 1. request device with ALM_ prefix
  // 2. connect
  // 3. read version
  // 4. establish security session
}
```

If the package API is clean and build size is acceptable, wrap it in `esp-idf-ble-client.js`. If the package is incomplete or too opaque, keep the same adapter interface and implement the Web Bluetooth transport manually.

### Manual fallback design

Manual implementation layers:

```text
ProvisioningWizard.jsx
  -> provisioning-client.js       UI-friendly state-machine API
  -> esp-idf-ble-client.js        protocol orchestration
  -> web-bluetooth-transport.js   GATT request/response primitives
  -> protocomm-security1.js       session handshake and encrypt/decrypt
  -> wifi-provisioning-proto.js   protobuf message encode/decode
```

Manual Web Bluetooth pseudocode:

```javascript
async function chooseDevice() {
  return navigator.bluetooth.requestDevice({
    filters: [{ namePrefix: "ALM_" }],
    optionalServices: [ALMANACH_PROV_SERVICE_UUID],
  });
}

async function connect(device) {
  const server = await device.gatt.connect();
  const service = await server.getPrimaryService(ALMANACH_PROV_SERVICE_UUID);
  return { device, server, service };
}

async function requestResponse(service, endpointName, requestBytes) {
  const characteristic = await resolveEndpointCharacteristic(service, endpointName);
  await characteristic.startNotifications();
  const responsePromise = onceNotification(characteristic);
  await characteristic.writeValue(requestBytes);
  return new Uint8Array(await responsePromise);
}
```

The hard part is `resolveEndpointCharacteristic()`. ESP-IDF's BLE provisioning scheme maps endpoint names to BLE characteristics through protocomm. A compatible library is preferred because it already knows these details.

## UI Design

### Placement

Add a topbar button near the existing Print button:

```text
[Blocks] [Inspector] | [Open] [Save] [PNG] [Set up printer] [Print]
```

Rationale:

- It is a device action, not a layout action.
- Users look near Print when the printer is not reachable.
- It keeps setup discoverable without occupying permanent rail space.

### Modal layout

The provisioning UI should be a modal or drawer, not a new page. Almanach Studio is a single-page editor and the user may be in the middle of editing a layout. A modal lets setup happen without losing the layout state.

Suggested structure:

```text
+------------------------------------------------------------+
| Set up Almanach Printer                               [x]  |
+------------------------------------------------------------+
| Step 1. Browser check                                      |
|   Web Bluetooth available: yes/no                          |
|   Secure context: yes/no                                   |
|                                                            |
| Step 2. Select printer                                     |
|   [Choose BLE printer]                                     |
|   Device: ALM_A1B2C3                                       |
|                                                            |
| Step 3. Proof of possession                                |
|   PoP: [alm-a1b2c3____________]                            |
|   Hint: printed label, serial console, or provisioning log  |
|                                                            |
| Step 4. WiFi credentials                                   |
|   SSID: [____________________] [Scan networks]              |
|   Password: [________________]                              |
|                                                            |
| Step 5. Provision                                          |
|   [Send credentials]                                       |
|   Progress log...                                          |
+------------------------------------------------------------+
```

### User-visible states

Use plain language. Avoid exposing protocomm details unless the user opens an advanced section.

Examples:

- Unsupported insecure origin:
  - "Web Bluetooth requires HTTPS or localhost. Open this setup page from `http://localhost:8199/almanach` during development or from the HTTPS Almanach setup site."
- No browser support:
  - "This browser does not support Web Bluetooth. Use desktop Chrome, Edge, or Android Chrome."
- Device selection:
  - "Choose the printer named `ALM_XXXXXX`. Hold the printer near this computer."
- PoP mismatch:
  - "The proof-of-possession did not match this printer. Check the code and try again."
- Credential failure:
  - "The printer could not join that network. Check SSID, password, signal strength, and 2.4 GHz compatibility."
- Success:
  - "Credentials sent. The printer is joining WiFi. If you know the printer IP, open `/almanach`; otherwise check your router or serial console."

### Styling

The current UI uses a large inline `<style>` block inside `AlmanachStudio()` (`web/src/almanach-studio.jsx:1860-2358`). For Phase 1, the intern may add modal CSS to that same block to minimize build changes. For maintainability, the better direction is to move styles for provisioning into `ProvisioningWizard.jsx` as a small `<style>` fragment or into a `provisioning.css.js` export.

Suggested classes:

```css
.prov-backdrop
.prov-modal
.prov-header
.prov-step
.prov-step.active
.prov-status
.prov-log
.prov-error
.prov-actions
```

Keep button styles compatible with existing `.btn`, `.btn.primary`, `.btn.ghost`, and `.spinner` classes (`web/src/almanach-studio.jsx:1949-2014`).

## Data and API Contracts

### Frontend provisioning status model

```javascript
export const ProvisioningStep = Object.freeze({
  IDLE: "idle",
  UNSUPPORTED: "unsupported",
  READY: "ready",
  CHOOSING_DEVICE: "choosing-device",
  CONNECTING: "connecting",
  ESTABLISHING_SESSION: "establishing-session",
  SCANNING_NETWORKS: "scanning-networks",
  ENTERING_WIFI: "entering-wifi",
  SENDING_CREDENTIALS: "sending-credentials",
  WAITING_FOR_DEVICE: "waiting-for-device",
  PROVISIONED: "provisioned",
  ERROR: "error",
});
```

### Provisioning result

```javascript
export type ProvisioningResult = {
  ok: boolean;
  deviceName: string;
  ssid: string;
  ip?: string | null;
  message: string;
  raw?: unknown;
};
```

### Firmware HTTP status extension

The current `/api/status` response only reports WiFi and printer (`firmware/atoms3r/main/web_server.c:99-119`). A later firmware phase should extend it with provisioning status once `provisioning_mgr_get_status()` is integrated:

```json
{
  "ok": true,
  "wifi": { "connected": true, "ip": "192.168.0.126" },
  "printer": { "baud": 460800, "swapped": true },
  "provisioning": {
    "initialized": true,
    "provisioned": true,
    "running": false,
    "client_connected": false,
    "security_ok": false,
    "service_name": "ALM_A1B2C3"
  }
}
```

Do not expose `pop` over HTTP in normal status. The firmware status struct contains `pop` (`firmware/atoms3r/main/provisioning_mgr.h:10-18`), but an HTTP status endpoint should not leak proof-of-possession to anyone on the LAN. If a debug endpoint exposes it during development, it should be guarded or removed before product use.

### Optional frontend printer discovery

After BLE provisioning succeeds, the browser may not know the printer IP. Options:

1. Ask the user to read the IP from serial console or router.
2. Have firmware print a small receipt containing the IP after provisioning.
3. Add mDNS later, e.g. `almanach-a1b2c3.local`.
4. Add a custom provisioning endpoint that returns the assigned IP after connection.

For Phase 1, show success and next-step instructions. Do not overpromise automatic IP discovery.

## Implementation Plan

### Phase 0: Confirm firmware baseline

Before frontend implementation, validate that firmware provisioning advertises and works with Espressif's app.

Commands:

```bash
cd almanach/firmware/atoms3r
./build.sh /dev/ttyACM0 build
./build.sh /dev/ttyACM0 flash-monitor
```

Expected monitor evidence:

```text
Provision with Espressif's ESP BLE Provisioning app or compatible client:
  Transport : BLE
  Device    : ALM_xxxxxx
  Security  : Security 1
  PoP       : alm-xxxxxx
```

If the firmware is not advertising, stop. Do not debug the browser first.

### Phase 1: Create frontend module skeleton and mock UI

Files to add:

```text
web/src/provisioning/types.js
web/src/provisioning/bluetooth-support.js
web/src/provisioning/mock-client.js
web/src/provisioning/ProvisioningWizard.jsx
```

`bluetooth-support.js`:

```javascript
export function getBluetoothSupport() {
  if (!window.isSecureContext) return { ok: false, code: "insecure-context" };
  if (!navigator.bluetooth) return { ok: false, code: "unsupported-browser" };
  return { ok: true };
}
```

`mock-client.js`:

```javascript
export function createMockProvisioningClient(log) {
  return {
    async chooseDevice() {
      log("Selected mock printer ALM_MOCK01");
      return { name: "ALM_MOCK01" };
    },
    async establishSession({ pop }) {
      if (!pop) throw new Error("PoP required");
      log("Security session established");
    },
    async scanNetworks() {
      return [{ ssid: "Office", rssi: -42 }, { ssid: "Guest", rssi: -60 }];
    },
    async sendCredentials({ ssid, password }) {
      if (!ssid || !password) throw new Error("SSID and password required");
      log(`Credentials sent for ${ssid}`);
    },
    async waitForResult() {
      return { ok: true, deviceName: "ALM_MOCK01", ssid: "Office", ip: null };
    },
  };
}
```

Acceptance criteria:

- `pnpm --prefix web run build` passes.
- The modal opens/closes from the topbar.
- Mock provisioning can run through success and failure states.
- Existing PNG export and Print buttons still work.

### Phase 2: Add topbar integration

Modify `web/src/almanach-studio.jsx` minimally:

- Import Bluetooth icon if available from Lucide.
- Import `ProvisioningWizard`.
- Add `showProvisioning` state.
- Add topbar button near Print.
- Render modal near the toast/workspace root.

Pseudocode:

```javascript
const [showProvisioning, setShowProvisioning] = useState(false);

<button className="btn" onClick={() => setShowProvisioning(true)} title="Set up printer WiFi over BLE">
  <Bluetooth size={13} /> Setup
</button>

{showProvisioning && (
  <ProvisioningWizard
    onClose={() => setShowProvisioning(false)}
    onProvisioned={(result) => flashToast("ok", "Printer setup complete")}
  />
)}
```

Acceptance criteria:

- No headless API regression. `window.almanachReady`, `window.almanachLoadLayout`, and `window.almanachExportBitmap` still exist after page load (`web/src/almanach-studio.jsx:1518-1617`).
- Existing `handlePrint()` still POSTs to `/api/print/bitmap` (`web/src/almanach-studio.jsx:1755-1765`).

### Phase 3: Spike the package client

Add dependency:

```bash
cd almanach/web
pnpm add esp-idf-provisioning-web protobufjs
```

Create:

```text
web/src/provisioning/esp-idf-ble-client.js
```

Because package APIs may change, keep all imports and assumptions inside this one file. The UI should import only `createProvisioningClient()` from `provisioning-client.js`.

Package-backed adapter shape:

```javascript
export function createProvisioningClient({ log }) {
  return new EspIdfBleProvisioningClient({
    serviceUuid: ALMANACH_PROV_SERVICE_UUID,
    namePrefix: "ALM_",
    log,
  });
}
```

Spike tasks:

1. Confirm `navigator.bluetooth.requestDevice()` opens and lists `ALM_` devices.
2. Confirm GATT connection works.
3. Confirm the package can read protocol/version information.
4. Confirm Security 1 session works with `alm-xxxxxx` PoP.
5. Confirm credentials are accepted by firmware.

If any of steps 3-5 fail, keep the mock UI and document the package limitation before writing manual protocol code.

### Phase 4: Real client workflow

Real workflow pseudocode:

```javascript
async function provision({ pop, ssid, password }) {
  dispatch({ type: "STEP", step: "choosing-device" });
  const device = await client.chooseDevice({ namePrefix: "ALM_", serviceUuid });

  dispatch({ type: "STEP", step: "connecting" });
  await client.connect(device);

  dispatch({ type: "STEP", step: "establishing-session" });
  await client.establishSession({ pop });

  dispatch({ type: "STEP", step: "sending-credentials" });
  await client.sendCredentials({ ssid, password });

  dispatch({ type: "STEP", step: "waiting-for-device" });
  const result = await client.waitForResult({ timeoutMs: 45000 });

  dispatch({ type: "PROVISIONED", result });
}
```

Error mapping:

```javascript
function userMessageForError(error) {
  if (error.name === "NotFoundError") return "No printer selected.";
  if (error.name === "NotAllowedError") return "Bluetooth permission was denied.";
  if (/pop|proof|mismatch/i.test(error.message)) return "Proof-of-possession did not match.";
  if (/auth/i.test(error.message)) return "WiFi password was rejected.";
  if (/not found|ap/i.test(error.message)) return "The printer could not find that WiFi network.";
  return error.message || "Provisioning failed.";
}
```

### Phase 5: Firmware status and post-provision UX

After firmware exposes provisioning status in `/api/status`, add a small device status widget to the web app. Keep it separate from Web Bluetooth; use HTTP only after the device is reachable.

Possible file:

```text
web/src/device/device-status.js
```

Pseudocode:

```javascript
export async function fetchDeviceStatus(baseUrl = "") {
  const r = await fetch(`${baseUrl}/api/status`, { cache: "no-store" });
  if (!r.ok) throw new Error(`status failed: ${r.status}`);
  return r.json();
}
```

In the firmware-served page, `baseUrl` is empty. In the hosted setup page, `baseUrl` must be user-provided or discovered later.

### Phase 6: Build, sync, and firmware asset validation

Once the web UI works from localhost, sync it into the firmware assets and make sure it still loads there for normal post-WiFi use.

Commands:

```bash
cd almanach
devctl build-web
devctl sync-firmware-web
cd firmware/atoms3r
./build.sh /dev/ttyACM0 build
```

Remember: the firmware-embedded copy may show Web Bluetooth as unsupported if served over plain HTTP. That is expected. The embedded page should still edit and print layouts.

## Testing Strategy

### Unit-level frontend testing

The repo does not currently have a frontend test runner. Start with mock-driven manual testing and then consider adding Vitest only if the UI grows.

Testable pure functions:

- `getBluetoothSupport()` with mocked `window.isSecureContext` and `navigator.bluetooth`.
- `userMessageForError()`.
- reducer/state-machine transitions.
- SSID/password validation helpers.

### Manual browser test matrix

| Scenario | Expected result |
|---|---|
| Open from `http://localhost:5173` in Chrome with Bluetooth | Setup button opens, support check passes. |
| Open from `http://localhost:8199/almanach` via render service | Support check passes if Chrome treats localhost as secure. |
| Open from `http://192.168.x.y/almanach` on firmware | UI shows secure-context unsupported message; layout editing still works. |
| Open in Firefox/Safari | UI shows unsupported browser message. |
| User cancels device chooser | UI returns to ready state with "No printer selected". |
| Wrong PoP | UI reports PoP mismatch. |
| Wrong WiFi password | UI reports WiFi authentication failure. |
| Successful provisioning | UI reaches provisioned state and gives next-step instructions. |

### Hardware validation

1. Reset provisioning state on firmware.
2. Confirm serial monitor logs service name and PoP.
3. Open hosted Almanach setup page from Chrome.
4. Choose `ALM_xxxxxx` BLE device.
5. Enter PoP and WiFi credentials.
6. Confirm firmware logs `WIFI_PROV_CRED_RECV`, `WIFI_PROV_CRED_SUCCESS`, and `WIFI_PROV_END`.
7. Confirm firmware gets IP.
8. Confirm `GET http://<ip>/api/status` returns `wifi.connected: true`.
9. Confirm `/almanach` loads and existing Print path still works.

### Regression tests for existing app behavior

- `devctl build` should still pass.
- `devctl render examples/layouts/01-minimal.yaml /tmp/almanach-ble-ui-regression.png` should still produce a PNG.
- Existing `handlePrint()` should still send `X-Width`, `X-Height`, and `X-Feed` headers.
- `window.almanachLoadLayout()` should still load layouts for the Go render service.

## Risks and Tradeoffs

### Risk: Web Bluetooth unavailable on many user devices

Web Bluetooth is not universal. It works best on Chrome/Edge desktop and Android Chrome. Safari/iOS support is not a safe assumption. The UI must detect this and point users to alternatives: Espressif's mobile app, USB console, or a supported browser.

### Risk: secure-context mismatch

The firmware serves HTTP, not HTTPS. The embedded page should not promise BLE provisioning. The product story should be "open the Almanach setup page from localhost/HTTPS to provision" and "use the firmware page after provisioning."

### Risk: protocol complexity

ESP-IDF provisioning is protocomm/protobuf/security, not simple BLE writes. A package spike is recommended to avoid reimplementing security incorrectly. If manual implementation becomes necessary, budget time for protobuf generation, crypto review, and packet tracing.

### Risk: bundle size growth

Adding protobuf and crypto may increase `almanach-bundle.js`. The current esbuild output was about 224 KB during devctl validation. Track bundle size after adding dependencies. If it grows too much for firmware embedding, split the provisioning UI into a separate hosted-only bundle or lazy-load it only in host builds.

### Risk: PoP disclosure

The firmware status struct includes `pop`, but the web API should not expose it over LAN. The user should get the PoP from a label, serial console, printed receipt, QR code, or setup docs. Avoid adding `/api/provisioning/status` that returns the PoP by default.

### Risk: IP discovery after provisioning

BLE provisioning may not automatically tell the browser the final LAN IP in a convenient way. Design the success screen to be useful even without IP discovery. Later add mDNS or a custom endpoint if needed.

## Alternatives Considered

### Alternative 1: Use Espressif's mobile app only

This is the lowest engineering effort and should remain a fallback. It is not enough for an Almanach-branded onboarding flow because users leave the product UI and must understand Espressif terminology.

### Alternative 2: Custom BLE service with JSON credentials

This would be easier in browser JavaScript but worse overall. It bypasses ESP-IDF's standard provisioning manager, loses compatibility with Espressif tools, and risks insecure credential handling. Do not choose this unless ESP-IDF provisioning proves impossible in browsers.

### Alternative 3: SoftAP/captive portal provisioning

SoftAP is more browser-compatible than BLE because it can use normal HTTP, but it requires the user to switch WiFi networks and may interfere with the printer's target network onboarding. It can be a future alternative, not the first implementation for this BLE ticket.

### Alternative 4: Put all provisioning code into `almanach-studio.jsx`

This is fast but not maintainable. The file already owns too many concerns. The design recommends a small topbar integration plus separate provisioning modules.

### Alternative 5: Firmware-served Web Bluetooth setup page

This is attractive because the device could serve its own UI, but it conflicts with both first-boot reachability and Web Bluetooth secure-context requirements. Keep the firmware-served page for post-WiFi control and printing.

## Intern Implementation Checklist

Use this checklist when implementing.

1. Confirm firmware advertises with Espressif app before writing browser code.
2. Add `web/src/provisioning/` skeleton and mock wizard.
3. Add topbar Setup button and modal integration.
4. Build and smoke-test existing editor behavior.
5. Add capability checks for secure context and browser support.
6. Spike `esp-idf-provisioning-web` or another compatible ESP-IDF browser library.
7. Wrap the real client behind the adapter interface.
8. Test wrong PoP, cancelled chooser, and wrong password paths.
9. Validate successful provisioning on AtomS3R hardware.
10. Add post-provision instructions and optional HTTP status check.
11. Run `devctl build`, `devctl render`, `devctl sync-firmware-web`, and firmware build.
12. Update the ticket diary with commands, failures, and screenshots/log snippets.

## File Reference Map

### Frontend files to read first

- `web/src/almanach-studio.jsx`
  - Main React app and topbar: `1481-2594`.
  - LocalStorage settings: `1484-1512`.
  - Headless render API: `1514-1617`.
  - Direct print path: `1666-1779`.
  - Topbar buttons: `2361-2426`.
- `web/esbuild.mjs`
  - esbuild entry/output and generated HTML shell: `6-40`.
- `web/package.json`
  - current dependencies and build scripts: `1-22`.

### Firmware files to understand

- `firmware/atoms3r/main/provisioning_mgr.c`
  - service name and PoP: `26-41`.
  - provisioning payload logging: `43-52`.
  - provisioning/BLE/security event tracking: `54-134`.
  - BLE provisioning initialization and service UUID: `158-186`.
  - Security 1 start: `220-241`.
  - reset/status APIs: `257-290`.
- `firmware/atoms3r/main/provisioning_mgr.h`
  - status struct and public API: `10-26`.
- `firmware/atoms3r/main/web_server.c`
  - `/api/status`: `99-119`.
  - embedded Almanach routes: `336-355`.
  - route registration: `360-424`.
- `firmware/atoms3r/main/app_main.c`
  - provisioning headers included: `18-23`.
  - start network onboarding decision: `91-124`.
  - web server starts only after WiFi: `127-141`.
  - current boot flow and console registration: `144-190`.
- `firmware/atoms3r/main/CMakeLists.txt`
  - embedded web assets and provisioning dependencies: `1-29`.
- `firmware/atoms3r/sdkconfig.defaults`
  - BLE/NimBLE/protocomm settings: `26-34`.

## API References

### Browser APIs

- `window.isSecureContext`
  - Use before showing Web Bluetooth controls.
- `navigator.bluetooth.requestDevice(options)`
  - Must be called from a user gesture.
  - Use `filters: [{ namePrefix: "ALM_" }]` and `optionalServices: [ALMANACH_PROV_SERVICE_UUID]`.
- `BluetoothDevice.gatt.connect()`
  - Returns a `BluetoothRemoteGATTServer`.
- `BluetoothRemoteGATTServer.getPrimaryService(uuid)`
  - Finds the provisioning GATT service.
- `BluetoothRemoteGATTCharacteristic.writeValue()` and notifications
  - Used by the underlying ESP-IDF provisioning transport or package.

### ESP-IDF APIs represented by firmware

- `wifi_prov_mgr_init()`
- `wifi_prov_scheme_ble_set_service_uuid()`
- `wifi_prov_mgr_is_provisioned()`
- `wifi_prov_mgr_start_provisioning()`
- `wifi_prov_mgr_stop_provisioning()`
- `wifi_prov_mgr_reset_provisioning()`
- `WIFI_PROV_EVENT`
- `PROTOCOMM_TRANSPORT_BLE_EVENT`
- `PROTOCOMM_SECURITY_SESSION_EVENT`

The browser does not call these C APIs directly. It talks to the BLE/protocomm protocol surface created by these APIs.

## Recommended First Pull Request Shape

Keep the first PR small and reviewable:

```text
web/src/provisioning/types.js
web/src/provisioning/bluetooth-support.js
web/src/provisioning/mock-client.js
web/src/provisioning/ProvisioningWizard.jsx
web/src/almanach-studio.jsx          small import/button/modal changes only
```

Acceptance criteria for the first PR:

- `pnpm --prefix web run build` passes.
- `devctl build` passes.
- `devctl render examples/layouts/01-minimal.yaml /tmp/almanach-provisioning-ui-smoke.png` passes.
- Browser from localhost shows the provisioning modal.
- Browser from insecure non-localhost origin shows clear unsupported copy.
- Existing Save/PNG/Print buttons remain present and functional.

The real BLE client can be a second PR after the UI state machine is reviewed.
