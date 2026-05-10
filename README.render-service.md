# Almanach Render Service

A Go CLI and HTTP service that renders Almanach Studio pages using Chrome headless and sends 1-bit bitmaps to the stoms3r ESP32 thermal-printer firmware.

The same binary supports two workflows:

- **One-shot CLI mode** for local iteration, previews, selector debugging, and direct print jobs.
- **HTTP server mode** for automation, Docker/devctl supervision, and future scheduling.

## Quick Start

### One-shot CLI preview

```bash
# Requires Chrome/Chromium installed on the host.
make build

./almanach-render-service render \
  --layout ./daily.yaml \
  --out /tmp/almanach.png \
  --output yaml

./almanach-render-service inspect \
  --layout ./daily.yaml \
  --output yaml
```

### One-shot CLI print

```bash
./almanach-render-service print \
  --layout ./daily.yaml \
  --printer-ip 192.168.0.126 \
  --output yaml
```

Dry-run print renders and converts the bitmap without contacting the printer:

```bash
./almanach-render-service print \
  --layout ./daily.yaml \
  --dry-run \
  --output yaml
```

### HTTP server mode

```bash
# Explicit server mode
ALMANACH_PRINTER_IP=192.168.0.126 ./almanach-render-service serve

# Backwards-compatible: no subcommand also starts server mode
ALMANACH_PRINTER_IP=192.168.0.126 ./almanach-render-service
```

Open the SPA served by the server:

```text
http://localhost:8199/almanach
```

### Docker Compose

```bash
ALMANACH_PRINTER_IP=192.168.0.126 docker compose up
```

This starts two containers:

- **chrome** — `chromedp/headless-shell`.
- **render** — the Go server.

## CLI Commands

| Command | Description |
|---------|-------------|
| `serve` | Start the long-running HTTP API server. |
| `render` | Render a JSON/YAML layout once to PNG or raw packed bitmap. |
| `inspect` | Render once and emit DOM metrics for cutoff/selector debugging. |
| `print` | Render once and send the bitmap to the ESP32 printer endpoint. |

### `render`

```bash
# YAML or JSON layout to PNG
./almanach-render-service render \
  --layout daily.yaml \
  --out /tmp/almanach.png

# Render to raw 1-bit bitmap
./almanach-render-service render \
  --layout daily.yaml \
  --format bitmap \
  --out /tmp/almanach.bin

# Capture decorative paper shell instead of print body
./almanach-render-service render \
  --layout daily.yaml \
  --selector .paper-shell \
  --out /tmp/almanach-shell.png

# Save debug artifacts
./almanach-render-service render \
  --layout daily.yaml \
  --out /tmp/almanach.png \
  --debug-dir /tmp/almanach-debug
```

Debug directory contents:

- `screenshot.png` — Chrome screenshot of the selected paper element.
- `bitmap.bin` — packed 1-bit bitmap.
- `layout.json` — normalized layout JSON sent to the SPA.
- `metrics.json` — DOM measurements after capture CSS was applied.

### `inspect`

Use `inspect` when a render or print appears cut off.

```bash
./almanach-render-service inspect \
  --layout daily.yaml \
  --output yaml
```

It emits metrics for:

- `.paper-shell`
- `.paper-body`
- `.canvas`
- `.workspace`
- `.almanach-app`

Check that overflow is `visible`, width is `384`, and the selected paper element has the expected height.

### `print`

```bash
./almanach-render-service print \
  --layout daily.yaml \
  --printer-ip 192.168.0.126 \
  --feed-lines 3
```

You can also pass a full endpoint:

```bash
./almanach-render-service print \
  --layout daily.yaml \
  --printer-url http://192.168.0.126/api/print/bitmap
```

## Layout Files

`--layout` uses Glazed `objectFromFile`, so both YAML and JSON are accepted.

Minimal YAML example:

```yaml
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.6
feedLines: 3
blocks:
  - id: t1
    type: title
    data:
      text: THE ALMANACH
      subtitle: Friday, May 8, 2026
  - id: d1
    type: date
    data:
      date: May 8, 2026
      day: Friday
```

Wrapped render request example:

```yaml
layout:
  theme: minimal
  paperWidth: 384
  bodyScale: 1.6
  feedLines: 3
  blocks:
    - id: t1
      type: title
      data:
        text: THE ALMANACH
        subtitle: Wrapped request
render:
  selector: .paper-body
  threshold: 128
```

Valid block types are defined by `web/src/almanach-studio.jsx`: `title`, `date`, `divider`, `plan`, `news`, `weather`, `note`, `habits`, `mood`, `reading`, `reflection`, `quote`, `word`, `history`, and `did`.

## HTTP API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /health` | GET | Health check |
| `GET /almanach` | GET | Almanach Studio SPA |
| `GET /almanach/bundle.js` | GET | React bundle |
| `POST /api/render` | POST | Render an almanac page |
| `POST /api/render-and-print` | POST | Render + send to ESP32 printer |

### Render over HTTP

```bash
# Auto-generate from live data
curl -X POST http://localhost:8199/api/render

# Custom layout to PNG
curl -X POST http://localhost:8199/api/render \
  -H "Content-Type: application/json" \
  -H "Accept: image/png" \
  --data @daily.json \
  -o page.png

# Custom layout to raw bitmap
curl -X POST http://localhost:8199/api/render \
  -H "Content-Type: application/json" \
  -H "Accept: application/octet-stream" \
  --data @daily.json \
  -o page.bin
```

### Render and print over HTTP

```bash
curl -X POST http://localhost:8199/api/render-and-print \
  -H "Content-Type: application/json" \
  --data @daily.json
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `ALMANACH_PORT` | `8199` | HTTP listen port |
| `ALMANACH_WEB_DIR` | `./web/dist` | SPA static files directory |
| `ALMANACH_PRINTER_IP` | *(empty)* | ESP32 stoms3r device IP |
| `ALMANACH_CHROME_PATH` | *(auto)* | Chrome binary path (local mode) |
| `CHROME_WS_URL` | *(empty)* | WebSocket URL for remote Chrome (Docker mode) |
| `ALMANACH_PAPER_WIDTH` | `384` | Paper width in pixels |
| `ALMANACH_FONT_SCALE` | `1.6` | Body font scale multiplier |
| `ALMANACH_DEFAULT_FEED` | `3` | Feed lines after printing |
| `ALMANACH_DEFAULT_THEME` | `minimal` | Default theme key |
| `ALMANACH_LOG_LEVEL` | `info` | Log verbosity |

## Cross-compile for Raspberry Pi

```bash
make build-pi
# produces almanach-render-service-arm64
```
