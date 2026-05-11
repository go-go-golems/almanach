# Almanach Render Service

Almanach is a Go CLI and HTTP service for rendering Almanach Studio thermal-printer pages with Chrome headless and sending packed 1-bit bitmaps to the ESP32 `stoms3r` printer firmware.

This repository currently contains the extracted Almanach render service and AtomS3R firmware from `esp32-s3-m5/stoms3r`:

- `cmd/almanach-render-service/` — render-service binary entrypoint.
- `internal/app/` — Glazed/Cobra CLI verbs, HTTP API, rendering, layout loading, bitmap packing, and printer client.
- `internal/app/doc/` — embedded Glazed help pages.
- `web/` — Almanach Studio web app source.
- `cmd/build-web/` — Dagger-first pnpm web build that copies `web/dist/` into `internal/web/embed/public/`.
- `internal/web/` — `go:embed` target and local fallback filesystem for bundled web assets.
- `firmware/atoms3r/` — ESP-IDF firmware for the M5Stack AtomS3R Lite printer endpoint.
- `examples/` — layout and ZIP-bundle examples.

## Build

```bash
# Build the web bundle locally and copy it into internal/web/embed/public.
BUILD_WEB_LOCAL=1 go run ./cmd/build-web

# Run tests.
go test ./...

# Build a single binary with bundled web assets.
go build -tags embed ./cmd/almanach-render-service
```

Without `BUILD_WEB_LOCAL=1`, `cmd/build-web` tries the Dagger container build first and falls back to local pnpm only when Dagger is unavailable.

## Firmware

```bash
cd firmware/atoms3r
./build.sh /dev/ttyACM0 build
./build.sh /dev/ttyACM0 flash-monitor
```

The firmware embeds its own checked-in web assets under `firmware/atoms3r/main/assets/almanach/`. Use `devctl sync-firmware-web` after `devctl build-web` to copy the top-level `web/dist/` output into those embedded firmware assets.

## devctl

```bash
direnv allow              # optional, loads .envrc defaults and ESP-IDF 5.4.x when available
devctl plugins list
devctl plan
devctl build              # build web assets and var/devctl/almanach-render-service
devctl up --force
devctl status
devctl render examples/layouts/01-minimal.yaml /tmp/almanach-render.png
devctl down
```

Useful helper commands:

```bash
devctl build-web
devctl sync-firmware-web
devctl firmware-build
```

## CLI

```bash
./almanach-render-service --help
./almanach-render-service render --layout examples/layouts/01-minimal.yaml --out /tmp/almanach.png
./almanach-render-service inspect --layout examples/layouts/01-minimal.yaml --output yaml
./almanach-render-service print --layout examples/layouts/01-minimal.yaml --printer-ip 192.168.0.126 --dry-run
./almanach-render-service serve --port 8199
```

The root command uses the Glazed command framework for help, logging, and command parsing. The verbs emit simple rows for status/metadata; they are not designed as rich structured data APIs.
