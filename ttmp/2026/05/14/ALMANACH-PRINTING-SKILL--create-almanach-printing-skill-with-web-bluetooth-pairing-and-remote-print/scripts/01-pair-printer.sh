#!/usr/bin/env bash
# 01-pair-printer.sh — Pair an Almanach printer over native BLE provisioning
#
# Prerequisites:
#   - Printer is in BLE provisioning mode (long-press button or factory reset)
#   - BlueZ is running and user has BLE access
#   - almanach-render-service binary is built (make build)
#
# Usage:
#   ./01-pair-printer.sh [SSID] [PASSWORD] [SERVICE_NAME] [POP]
#
# Defaults:
#   SSID:         current WiFi network
#   PASSWORD:     from NetworkManager
#   SERVICE_NAME: ALM_0F2320
#   POP:          alm-0f2320

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
BINARY="$REPO_ROOT/dist/almanach-render-service"

SSID="${1:-$(nmcli -t -f active,ssid dev wifi | grep '^yes' | cut -d: -f2)}"
PASSWORD="${2:-$(nmcli -t -s -f 802-11-wireless-security.psk connection show "$SSID" 2>/dev/null | cut -d: -f2)}"
SERVICE_NAME="${3:-ALM_0F2320}"
POP="${4:-alm-0f2320}"

if [ -z "$SSID" ]; then
  echo "ERROR: Could not determine current WiFi SSID. Pass it as first argument."
  exit 1
fi

if [ -z "$PASSWORD" ]; then
  echo "ERROR: Could not determine WiFi password. Pass it as second argument."
  exit 1
fi

echo "Pairing printer $SERVICE_NAME with PoP $POP"
echo "WiFi SSID: $SSID"
echo ""

# Step 1: Verify BLE connection with version check
echo "=== Step 1: Verifying BLE connection ==="
"$BINARY" ble-provision --implementation native \
  --action version \
  --service-name "$SERVICE_NAME" \
  --pop "$POP" \
  --output yaml

# Step 2: Provision WiFi credentials
echo ""
echo "=== Step 2: Provisioning WiFi credentials ==="
"$BINARY" ble-provision --implementation native \
  --action provision \
  --service-name "$SERVICE_NAME" \
  --pop "$POP" \
  --ssid "$SSID" \
  --passphrase "$PASSWORD" \
  --output yaml

echo ""
echo "=== Pairing complete ==="
echo "The printer should now be on WiFi. Scan for its IP:"
echo "  arp -a | grep 192.168.0"
echo ""
echo "Or check the printer's status page:"
echo "  curl -s http://<IP>/api/status"
