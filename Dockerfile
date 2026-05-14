FROM golang:1.26-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -tags=embed -o /almanach-render-service ./cmd/almanach-render-service

FROM chromedp/headless-shell:latest

COPY --from=builder /almanach-render-service /usr/local/bin/almanach-render-service
COPY --from=builder /build/web/ /opt/almanach/web/

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
