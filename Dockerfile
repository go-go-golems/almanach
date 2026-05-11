# =========================================================================
# Almanach Render Service — self-contained Docker image
#
# Uses chromedp/headless-shell as the runtime base, which bundles a minimal
# Chrome headless shell (~137 MB). The Go binary is copied in on top.
#
# Two modes:
#   1. Single container (default): Chrome and the Go server share one container.
#      The Go server launches Chrome as a subprocess.
#
#   2. Docker Compose: Chrome runs in its own headless-shell container.
#      The Go server connects via CHROME_WS_URL=ws://chrome:9222.
#
# Build:
#   docker build -t almanach-render-service .
#
# Run (single container):
#   docker run -p 8199:8199 \
#     -e ALMANACH_PRINTER_IP=192.168.0.126 \
#     almanach-render-service
#
# Run (docker compose):
#   docker compose up
# =========================================================================

# ---- Stage 1: Build the Go binary ----
FROM golang:1.26-bookworm AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /almanach-render-service ./cmd/almanach-render-service

# ---- Stage 2: Runtime with Chrome headless-shell ----
FROM chromedp/headless-shell:latest

# Copy the Go binary
COPY --from=builder /almanach-render-service /usr/local/bin/almanach-render-service

# Copy the SPA static files
COPY --from=builder /build/web/ /opt/almanach/web/

# Create a non-root user for the Go server (Chrome already runs as non-root)
RUN useradd -m -s /bin/bash almanach

# Environment defaults
ENV ALMANACH_PORT=8199 \
    ALMANACH_WEB_DIR=/opt/almanach/web/dist \
    ALMANACH_PRINTER_IP= \
    ALMANACH_CHROME_PATH=/headless-shell/headless-shell \
    ALMANACH_DEFAULT_THEME=minimal \
    ALMANACH_DEFAULT_FEED=3 \
    ALMANACH_FONT_SCALE=1.6 \
    ALMANACH_PAPER_WIDTH=384

EXPOSE 8199

ENTRYPOINT ["almanach-render-service"]
