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
# CGO_ENABLED=1 required for go-sqlite3 (indirect dep via glazed).
RUN CGO_ENABLED=1 go build -tags=embed -o /almanach-render-service ./cmd/almanach-render-service

# ---- Stage 2: Install Google Fonts ----
# The headless-shell base only has DejaVu fonts. Almanach Studio uses DM Sans,
# Cormorant Garamond, EB Garamond, JetBrains Mono, Caveat, and Kalam.
# Without these, Chrome falls back to DejaVu which has larger line-height metrics,
# causing layouts to render ~40% taller than on desktop Chrome.
FROM debian:trixie-slim AS font-installer
RUN apt-get update && \
    apt-get install -y --no-install-recommends fontconfig fonts-ebgaramond && \
    rm -rf /var/lib/apt/lists/*
# Google Fonts are downloaded at build time from github.com/google/fonts.
# If the download fails for any font, the build continues (|| true) since
# the remaining fonts still improve rendering significantly.
RUN mkdir -p /usr/share/fonts/truetype/google && \
    for F in \
      "https://github.com/google/fonts/raw/main/ofl/dmsans/DMSans%5Bopsz%2Cwght%5D.ttf" \
      "https://github.com/google/fonts/raw/main/ofl/cormorantgaramond/CormorantGaramond%5Bwght%5D.ttf" \
      "https://github.com/google/fonts/raw/main/ofl/cormorantgaramond/CormorantGaramond-Italic%5Bwght%5D.ttf" \
      "https://github.com/google/fonts/raw/main/ofl/jetbrainsmono/JetBrainsMono%5Bwght%5D.ttf" \
      "https://github.com/google/fonts/raw/main/ofl/caveat/Caveat%5Bwght%5D.ttf" \
      "https://github.com/google/fonts/raw/main/ofl/kalam/Kalam-Regular.ttf" \
      "https://github.com/google/fonts/raw/main/ofl/kalam/Kalam-Bold.ttf" \
      "https://github.com/google/fonts/raw/main/ofl/kalam/Kalam-Light.ttf" \
    ; do \
      BASE=$(basename "$F" | sed 's/%5B.*//;s/%5D//') && \
      curl -fsSL "$F" -o "/usr/share/fonts/truetype/google/$BASE.ttf" || true ; \
    done && \
    fc-cache -f || true

# ---- Stage 3: Runtime with Chrome headless-shell ----
FROM chromedp/headless-shell:latest

# Install fonts from the font-installer stage
COPY --from=font-installer /usr/share/fonts /usr/share/fonts
COPY --from=font-installer /usr/share/fontconfig /usr/share/fontconfig
COPY --from=font-installer /etc/fonts /etc/fonts

# Copy the Go binary
COPY --from=builder /almanach-render-service /usr/local/bin/almanach-render-service

# Copy the SPA static files
COPY --from=builder /build/web/ /opt/almanach/web/

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
