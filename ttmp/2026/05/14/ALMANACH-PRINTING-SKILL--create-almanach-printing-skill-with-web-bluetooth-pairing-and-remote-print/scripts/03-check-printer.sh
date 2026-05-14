#!/usr/bin/env bash
# 03-check-printer.sh — Check if the printer is reachable and the render service is healthy
#
# Usage:
#   ./03-check-printer.sh [--local|--remote]

set -euo pipefail

PRINTER_IP="192.168.0.126"
REMOTE_URL="https://almanach.crib.scapegoat.dev"
LOCAL_URL="http://localhost:8199"
MODE="${1:---local}"

echo "=== Printer Status ==="
PRINTER_STATUS=$(curl -s --connect-timeout 3 "http://$PRINTER_IP/api/status" 2>&1)
if echo "$PRINTER_STATUS" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d.get('ok') else 1)" 2>/dev/null; then
  echo "Printer: ONLINE at $PRINTER_IP"
  echo "$PRINTER_STATUS" | python3 -m json.tool
else
  echo "Printer: OFFLINE or unreachable at $PRINTER_IP"
fi

echo ""
echo "=== Render Service Status ==="
case "$MODE" in
  --local)
    HEALTH=$(curl -s --connect-timeout 3 "$LOCAL_URL/health" 2>&1)
    if echo "$HEALTH" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d.get('ok') else 1)" 2>/dev/null; then
      echo "Local service: ONLINE"
      echo "$HEALTH" | python3 -m json.tool
    else
      echo "Local service: OFFLINE (start with: almanach-render-service serve)"
    fi
    ;;
  --remote)
    HEALTH=$(curl -sk --connect-timeout 5 "$REMOTE_URL/health" 2>&1)
    if echo "$HEALTH" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d.get('ok') else 1)" 2>/dev/null; then
      echo "Remote service: ONLINE at $REMOTE_URL"
      echo "$HEALTH" | python3 -m json.tool
    else
      echo "Remote service: OFFLINE at $REMOTE_URL"
    fi
    ;;
esac
