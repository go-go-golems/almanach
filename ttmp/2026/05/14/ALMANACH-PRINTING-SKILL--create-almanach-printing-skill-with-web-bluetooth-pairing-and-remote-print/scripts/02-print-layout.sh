#!/usr/bin/env bash
# 02-print-layout.sh — Print an almanach layout via CLI or remote HTTP API
#
# Usage:
#   ./02-print-layout.sh [layout.yaml] [--local|--remote] [--dry-run]
#
# Defaults:
#   layout: 01-first-test.yaml (in same directory)
#   mode:   --local (CLI print), use --remote for crib.scapegoat.dev

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BINARY="$REPO_ROOT/dist/almanach-render-service"
PRINTER_IP="192.168.0.126"
REMOTE_URL="https://almanach.crib.scapegoat.dev"

LAYOUT="${1:-$SCRIPT_DIR/01-first-test.yaml}"
MODE="${2:---local}"
DRY_RUN="${3:-}"

if [ ! -f "$LAYOUT" ]; then
  echo "ERROR: Layout file not found: $LAYOUT"
  exit 1
fi

case "$MODE" in
  --local)
    echo "Printing $LAYOUT locally to printer $PRINTER_IP"
    if [ "$DRY_RUN" = "--dry-run" ]; then
      "$BINARY" print --layout "$LAYOUT" --printer-ip "$PRINTER_IP" --dry-run --output yaml
    else
      "$BINARY" print --layout "$LAYOUT" --printer-ip "$PRINTER_IP" --feed-lines 3 --output yaml
    fi
    ;;
  --remote)
    echo "Printing $LAYOUT via $REMOTE_URL"
    LAYOUT_JSON=$(python3 -c "import yaml,json,sys; print(json.dumps(yaml.safe_load(open('$LAYOUT'))))")
    if [ "$DRY_RUN" = "--dry-run" ]; then
      echo "Would POST to $REMOTE_URL/api/render-and-print"
      echo "Layout JSON: $LAYOUT_JSON" | head -c 200
      echo "..."
    else
      echo "$LAYOUT_JSON" | curl -sk -X POST "$REMOTE_URL/api/render-and-print" \
        -H "Content-Type: application/json" \
        -d @- | python3 -m json.tool
    fi
    ;;
  *)
    echo "Usage: $0 [layout.yaml] [--local|--remote] [--dry-run]"
    exit 1
    ;;
esac
