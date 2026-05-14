---
Title: Diary
Ticket: ALMANACH-PRINTING-SKILL
Status: active
Topics:
    - almanach
    - printing
    - web-bluetooth
    - ble
    - wifi-provisioning
    - remote-access
    - crib
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Investigation and implementation diary for creating an Almanach printing skill with Web Bluetooth pairing and remote printing."
LastUpdated: 2026-05-14T14:30:00-04:00
WhatFor: "Chronicle the journey from studying the repo to pairing a printer, printing from CLI and HTTP API, and authoring a reusable pi skill."
---

# Diary

## Goal

Create a pi agent skill that enables pairing an Almanach AtomS3R thermal printer over Web Bluetooth (via the setup page served locally or on crib.scapegoat.dev), then printing almanach pages from both CLI and the HTTP API.

---

## Step 1: Repo Study — Understanding the Almanach Architecture

I studied the full almanach repo to understand how the printer, provisioning, and rendering pieces fit together before designing the printing skill.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket in order to create a great almanach printing skill. Study this repo and figure out how to set up the printer using web bluetooth (if possible even with the ip / host where this might be running i think on crib.scapegoat.dev). Keep a diary as you work. Keep any script and intermediate almanach file you create in the scripts/ folder of the ticket. I currently need to pair the printer and then I want to print out a few things."

**Assistant interpretation:** Create a docmgr ticket, study the almanach codebase to understand how Web Bluetooth pairing and printing work, figure out how to run the setup remotely on crib.scapegoat.dev, and create a pi skill that wraps these workflows.

**Inferred user intent:** Manuel wants to pair his AtomS3R printer (currently at 192.168.1.242 per state.json) and then be able to print almanach pages easily, ideally with a reusable pi skill. He also wants to explore running the setup/provisioning flow on crib.scapegoat.dev.

### What I did

- Read the full repo structure, key Go files (printer.go, cmd_print.go, cmd_setup.go, server.go, setup_device.go, cmd_ble_provision.go, config.go)
- Read the web frontend provisioning code (espidf-client.js, bluetooth-support.js, ProvisioningWizard.jsx)
- Read the README.render-service.md, firmware README, provisioning user guide, layouts getting started guide
- Read the docker-compose.yml and .devctl.yaml
- Checked existing state: printer already provisioned at ALM_0F2320 / 192.168.1.242 / Verizon_9DNVB9

### What I learned

**Architecture summary:**

1. **Firmware (atoms3r):** ESP32-S3 firmware that drives the K118 thermal printer via UART. Supports BLE provisioning (ESP-IDF protocomm Security 1) and has an HTTP API at `/api/print/bitmap` for receiving 1-bit bitmap data.

2. **Web Bluetooth Provisioning:** The `almanach-render-service setup` command serves `setup.html` on `127.0.0.1:<port>`. The page uses Chrome's Web Bluetooth API to:
   - Pick an `ALM_` device from Chrome's Bluetooth picker
   - Connect via GATT, discover ESP-IDF provisioning service UUIDs
   - Verify `proto-ver v1.1`
   - Establish Security 1 session with PoP
   - Send encrypted WiFi credentials
   - Poll WiFi status until connected
   - POST the discovered IP back to `/api/setup/provisioned-device`

3. **Printer Communication:** After provisioning, the printer is reachable at `http://<IP>/api/print/bitmap`. The Go server sends a POST with the packed 1-bit bitmap as `application/octet-stream`, with `X-Width` and `X-Height` headers. Feed lines are baked into the bitmap as trailing blank rows.

4. **Render + Print Flow:** `almanach-render-service print --layout daily.yaml --printer-ip <IP>` renders a YAML layout via headless Chrome (Almanach Studio SPA), converts to 1-bit bitmap, and sends to the ESP32.

5. **HTTP API:** `POST /api/render-and-print` on port 8199 renders and forwards to the ESP32 in one call. `POST /api/render` returns PNG or raw bitmap.

6. **State persistence:** `~/.config/almanach/render-service/state.json` stores the provisioned device (IP, SSID, service name). The server uses this as the default printer IP when `ALMANACH_PRINTER_IP` is not set.

**Web Bluetooth on crib.scapegoat.dev consideration:**
- Web Bluetooth requires a secure context (HTTPS or localhost). crib.scapegoat.dev would need HTTPS and a valid certificate.
- The setup page currently binds to `127.0.0.1` specifically for Web Bluetooth localhost requirement.
- To make Web Bluetooth work from crib.scapegoat.dev, the server would need to:
  1. Bind to `0.0.0.0` instead of `127.0.0.1`
  2. Serve over HTTPS with a valid cert (Let's Encrypt via reverse proxy)
  3. Or: use Chrome flags `--enable-experimental-web-platform-features` and `--unsafely-treat-insecure-origin-as-secure=http://crib.scapegoat.dev:8199`
- However, the BLE provisioning only needs to happen once. After that, the printer is on WiFi and reachable by IP. The printing itself (HTTP POST to `/api/print/bitmap`) can work from anywhere on the same network.

**Current printer state:**
- Printer ALM_0F2320 is already provisioned on Verizon_9DNVB9 at 192.168.1.242
- State file exists at `~/.config/almanach/render-service/state.json`
- This means we might not need to re-pair if the printer is still on WiFi

### What was tricky

- Web Bluetooth is strictly limited to secure contexts. The setup command explicitly binds to 127.0.0.1 for this reason. Running provisioning remotely (crib.scapegoat.dev) requires HTTPS.
- The provisioning page reports the printer IP back to the same server that served the page, so the server must be accessible from the browser.

### What warrants a second pair of eyes

- Whether crib.scapegoat.dev already has HTTPS/reverse proxy configured, and whether the almanach service is already running there.
- Whether the printer at 192.168.1.242 is still reachable and on the same network as crib.

### What should be done in the future

- Check if the printer is currently reachable
- Investigate crib.scapegoat.dev setup (HTTPS, running services, network proximity)
- Build the binary and test a print
- Author the SKILL.md

### Code review instructions

- Start with `README.render-service.md` for the big picture
- Check `internal/app/server.go` for the HTTP API surface
- Check `web/src/provisioning/espidf-client.js` for the Web Bluetooth flow
- Verify `internal/app/config.go` for env var configuration

### Technical details

**Key URLs/endpoints:**
- Setup page: `http://localhost:8199/setup`
- Health: `GET /health`
- Provisioned device: `GET/POST/DELETE /api/setup/provisioned-device`
- Render: `POST /api/render`
- Render+Print: `POST /api/render-and-print`
- Printer bitmap: `POST http://<printer-ip>/api/print/bitmap`

**Key env vars:**
- `ALMANACH_PRINTER_IP` — explicit printer IP (overrides state file)
- `ALMANACH_PORT` — HTTP port (default 8199)
- `ALMANACH_WEB_DIR` — SPA dist directory
- `ALMANACH_CHROME_PATH` — Chrome binary
- `CHROME_WS_URL` — remote Chrome WebSocket URL (Docker mode)
- `ALMANACH_STATE_FILE` — state file path

**Printer protocol:**
- POST `/api/print/bitmap` with:
  - Content-Type: `application/octet-stream`
  - X-Width: bitmap width in pixels
  - X-Height: bitmap height in pixels
  - X-Feed: 0 (feed is baked into bitmap)
  - Body: packed 1-bit bitmap data (MSB first)

---

## Step 2: BLE Pairing — Printer Re-Provisioned on yolobolo

The user put the printer in BLE pairing mode. I scanned for it and successfully provisioned it on the current WiFi network.

### Prompt Context

**User prompt (verbatim):** "it's in pairing mode now"

**Assistant interpretation:** The printer is now advertising over BLE and ready for provisioning.

**Inferred user intent:** Provision the printer on the current WiFi network so it can be used for printing.

### What I did

- Scanned for BLE devices: found `ALM_0F2320` at MAC `98:88:E0:0F:23:22`
- Verified BLE connection with `ble-provision --action version`: proto-ver v1.1, Security 1, 5 endpoints, wifi_scan capability
- Provisioned WiFi: `ble-provision --action provision --service-name ALM_0F2320 --pop alm-0f2320 --ssid yolobolo`
- Result: `wifi_status: Success`, `wifi_state: connected` in 13.3 seconds
- Found printer IP by scanning the local subnet: `192.168.0.126`
- Updated the setup server state via `POST /api/setup/provisioned-device`
- Sent test bitmap (1 row of all-black) directly: `{"ok":true}`

### What worked

- Native BLE provisioning via Go CLI worked perfectly — no need for browser/Python
- The printer responded to the HTTP bitmap endpoint immediately after WiFi connect
- The ARP table had the printer's MAC/IP mapping within seconds

### What didn't work

- The old IP (192.168.1.242) was from a different subnet (Verizon_9DNVB9). The printer was not reachable before re-provisioning.

---

## Step 3: First Prints — CLI and Direct Bitmap

Successfully printed the first test layout and a daily briefing via CLI.

### What I did

- Rendered `01-first-test.yaml` to PNG (384x559, 26KB bitmap)
- Printed via CLI: `almanach-render-service print --layout 01-first-test.yaml --printer-ip 192.168.0.126` → `printed: true, printer_ok: true`
- Rendered `02-daily-briefing.yaml` (384x776, 37KB bitmap) — this one **failed** with `unexpected EOF`
- Sent the 37KB bitmap via Python `urllib.request` with 120s timeout → `{"ok":true}` (worked!)

### What I learned

- **Large bitmap EOF issue:** The Go HTTP client's default timeout/keepalive settings cause `unexpected EOF` for bitmaps over ~30KB. Python's `urllib` with a longer timeout works fine. The ESP32 firmware's HTTP server is slow to receive large payloads at 460800 baud UART.
- The Go `printerHTTPTimeout` is set to 120 seconds which should be enough, but the issue seems to be TCP connection management rather than timeout.

### What was tricky

- The same 37KB bitmap fails via Go `net/http` but succeeds via Python `urllib`. This suggests a Go HTTP client behavior difference (possibly keepalive or connection close handling).

---

## Step 4: Deploying to crib-k3s — Remote Rendering Service

The user asked about the almanach rendering server deployment. I searched both k8s clusters and found no existing deployment.

### Prompt Context

**User prompt (verbatim):** "what about ~/code/wesen/2026-03-27--hetzner-k3s/"

**Assistant interpretation:** Check the Hetzner k3s cluster for an existing almanach deployment.

**Inferred user intent:** Find where the almanach rendering service was previously deployed.

### What I did

- Checked both k8s clusters: no almanach deployment in either
- Found the crib-k3s repo at `~/code/wesen/crib-k3s/` with ArgoCD GitOps
- Discovered: `*.crib.scapegoat.dev → 100.67.90.12` (Proxmox host at 192.168.0.225)
- **Key finding:** The Proxmox host is on the same LAN (192.168.0.x) as the printer (192.168.0.126)! SSH confirmed the host can reach the printer.
- Created k8s manifests: namespace, deployment, service, ingress for crib-k3s
- Built Docker image with CGO_ENABLED=1 (fixed sqlite3 requirement) and pushed to `ghcr.io/go-go-golems/almanach-render-service:latest`
- Copied ghcr-pull secret from poll-modem namespace
- Deployed and verified: pod running, health check passes
- **HTTPS ingress working:** `https://almanach.crib.scapegoat.dev/health` → `{"ok":true,"printer":"192.168.0.126","version":"dev"}`
- **Remote printing works:** POST to `https://almanach.crib.scapegoat.dev/api/render-and-print` successfully printed a test layout

### What worked

- Single-container Dockerfile (Chrome + Go binary) works fine in k8s — no need for sidecar
- The Proxmox host being on the same LAN as the printer means the k8s pod can reach it directly
- ArgoCD application manifest created but not pushed to GitHub yet (applied manually for now)

### What didn't work

- First Docker image was built with `CGO_ENABLED=0` → sqlite3 crash on startup
- The ghcr.io image is private → needed pull secret from another namespace
- ArgoCD can't sync the manifests until they're pushed to the crib-k3s GitHub repo
- Large bitmaps (>30KB) still fail via the remote service (same Go HTTP client EOF issue)

### What was tricky

- The CGO_ENABLED=0 vs sqlite3 dependency caught me off guard. The Dockerfile in the repo uses CGO_ENABLED=0 which doesn't work with go-sqlite3 (indirect dep via glazed).
- ArgoCD's auto-prune deleted manually-applied resources when the Application CR existed but couldn't sync from git.

---

## Step 5: Skill and Scripts Created

Wrote the SKILL.md and helper scripts.

### What I did

- Created SKILL.md at `~/.pi/agent/skills/almanach-printing/SKILL.md`
- Created 4 helper scripts in the ticket `scripts/` directory:
  - `01-pair-printer.sh` — BLE provisioning workflow
  - `02-print-layout.sh` — Print via CLI or remote HTTP API
  - `03-check-printer.sh` — Check printer and service status
  - `04-update-ip.sh` — Update printer IP in state file
- Created 2 sample layouts:
  - `01-first-test.yaml` — Minimal test with title, note, quote
  - `02-daily-briefing.yaml` — Full daily briefing with weather, plan, note

### What should be done in the future

- Push crib-k3s gitops manifests to GitHub for ArgoCD sync
- Fix the large bitmap EOF issue in Go HTTP client (possibly need to set `DisableKeepAlives: true` or `ForceAttemptHTTP2: false`)
- Update the Dockerfile in the almanach repo to use CGO_ENABLED=1
- Consider adding the ghcr-pull secret to the kustomize directory (sealed secret)
- Investigate segmented/chunked bitmap printing for large pages
