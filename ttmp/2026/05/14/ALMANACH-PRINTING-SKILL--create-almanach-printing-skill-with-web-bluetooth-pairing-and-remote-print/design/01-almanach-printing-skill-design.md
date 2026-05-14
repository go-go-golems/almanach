---
Title: Almanach Printing Skill Design
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
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for a pi agent skill that enables BLE pairing and thermal printing of almanach pages."
LastUpdated: 2026-05-14T14:35:00-04:00
WhatFor: "Reference design for building the almanach printing skill."
---

# Almanach Printing Skill Design

## Overview

The almanach printing skill enables the pi agent to:
1. **Pair** an AtomS3R thermal printer over Web Bluetooth (BLE provisioning)
2. **Print** almanach pages from YAML layouts via CLI or HTTP API
3. **Manage** the printer connection (check status, update IP, reset)

## Architecture

### Two-Phase Workflow

```
Phase 1: Pairing (BLE Provisioning)
  Browser (Chrome/localhost) ──Web Bluetooth──> AtomS3R
  AtomS3R joins WiFi ──> reports IP ──> POST back to setup server
  
Phase 2: Printing (HTTP)
  pi agent ──> almanach-render-service ──> ESP32 /api/print/bitmap
```

Pairing is a one-time (or infrequent) operation. Printing happens whenever you want output.

### Printer Communication

The ESP32 firmware exposes a simple HTTP endpoint:
- `POST http://<printer-ip>/api/print/bitmap`
- Content-Type: `application/octet-stream`
- X-Width: bitmap width (384 for K118)
- X-Height: bitmap height
- Body: packed 1-bit bitmap (MSB first, 48 bytes/row for 384px width)

### Provisioning Details

The printer uses ESP-IDF's protocomm BLE provisioning with Security 1:
- BLE service UUID: `021a9004-0382-4aea-bff4-6b3f1c5adfb4`
- Device name prefix: `ALM_`
- Proof of possession (PoP): `alm-<last 6 hex digits>` (from firmware log/screen)
- Endpoints: proto-ver, prov-session, prov-config, prov-ctrl, prov-scan
- After WiFi connect: printer reports IP via BLE status poll, browser POSTs to setup server

## Skill Capabilities

### 1. Pair Printer (`almanach pair`)

**When:** Printer is in BLE provisioning mode (fresh or after physical reset)

**Steps:**
1. Start `almanach-render-service setup` on localhost
2. Open `http://localhost:8199/setup` in Chrome
3. User enters PoP, WiFi SSID, password
4. Chrome Bluetooth picker → select `ALM_` device
5. Follow provisioning wizard
6. Printer IP gets saved to state file

**CLI alternative:**
```bash
almanach-render-service ble-provision --implementation native \
  --action provision \
  --service-name ALM_0F2320 \
  --pop alm-0f2320 \
  --ssid yolobolo
```

**Limitation:** Web Bluetooth requires secure context (localhost or HTTPS). Cannot run pairing from a non-HTTPS remote host unless Chrome flags are used.

### 2. Print Layout (`almanach print`)

**When:** Printer is already on WiFi and reachable

**CLI:**
```bash
almanach-render-service print \
  --layout daily.yaml \
  --printer-ip <IP> \
  --feed-lines 3
```

**HTTP API (local):**
```bash
curl -X POST http://localhost:8199/api/render-and-print \
  -H "Content-Type: application/json" \
  --data @daily.json
```

**HTTP API (remote/crib):**
```bash
curl -X POST http://crib.scapegoat.dev:8199/api/render-and-print \
  -H "Content-Type: application/json" \
  --data @daily.json
```

### 3. Check Printer Status (`almanach status`)

```bash
# Check if printer is reachable
curl -s http://<printer-ip>/api/print/bitmap -X POST -d "" -o /dev/null -w "%{http_code}"

# Check setup server state
curl -s http://localhost:8199/api/setup/provisioned-device

# Check health (includes printer IP)
curl -s http://localhost:8199/health
```

### 4. Update Printer IP (`almanach set-ip`)

```bash
# If the printer got a new IP (e.g. after router reboot)
curl -X POST http://localhost:8199/api/setup/provisioned-device \
  -H "Content-Type: application/json" \
  -d '{"serviceName":"ALM_0F2320","ip":"192.168.0.XXX","ssid":"yolobolo","source":"manual"}'

# Or set env var
export ALMANACH_PRINTER_IP=192.168.0.XXX
```

### 5. Reset Printer WiFi (`almanach reset`)

**Physical:** Long-press the AtomS3R button until reset threshold

**BLE (if still connected):**
```bash
almanach-render-service ble-provision --implementation native --action reset \
  --service-name ALM_0F2320 --pop alm-0f2320
```

**Via setup page:** Click "Reset printer WiFi" button

## Remote Access on crib.scapegoat.dev

### Current Situation

- crib.scapegoat.dev DNS does not resolve (NXDOMAIN)
- Even if it did, Web Bluetooth requires secure context (HTTPS or localhost)
- The **printing** API (POST /api/render-and-print) can work remotely once the printer is on WiFi

### Making It Work

**Option A: Tailscale + Reverse Proxy**
1. Run almanach-render-service on a Tailscale-connected host
2. Use Caddy/nginx as reverse proxy with HTTPS on crib.scapegoat.dev
3. Printer must be on same L2 network as the render service

**Option B: Direct Docker Compose on a server**
1. `ALMANACH_PRINTER_IP=<IP> docker compose up` on crib
2. Chrome + headless shell in Docker
3. Printer reachable from crib's network

**Option C: Local-only setup**
1. Run the render service locally on the laptop
2. Pair via localhost setup page
3. Print from anywhere that can reach the printer IP

**Recommended for now:** Option C. Pair once locally, then print from CLI.

## Skill File Structure

```
~/.pi/agent/skills/almanach-printing/
├── SKILL.md           # Main skill instructions
├── scripts/
│   ├── 01-pair-printer.sh     # Start setup server, open browser
│   ├── 02-print-layout.sh     # Print a YAML layout
│   ├── 03-check-printer.sh    # Verify printer reachability
│   └── 04-update-ip.sh         # Update printer IP in state file
└── examples/
    ├── daily-briefing.yaml     # Sample daily layout
    ├── knowledge-strip.yaml   # Sample knowledge strip
    └── minimal.yaml            # Minimal test layout
```
