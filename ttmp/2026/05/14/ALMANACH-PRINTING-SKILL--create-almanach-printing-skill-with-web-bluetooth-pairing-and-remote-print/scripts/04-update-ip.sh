#!/usr/bin/env bash
# 04-update-ip.sh — Update the printer IP in the local state file and/or the remote service
#
# Usage:
#   ./04-update-ip.sh <NEW_IP> [--local|--remote]

set -euo pipefail

NEW_IP="${1:?Usage: $0 <NEW_IP> [--local|--remote]}"
MODE="${2:---local}"
LOCAL_URL="http://localhost:8199"
REMOTE_URL="https://almanach.crib.scapegoat.dev"

case "$MODE" in
  --local)
    echo "Updating printer IP to $NEW_IP on local service..."
    curl -s -X POST "$LOCAL_URL/api/setup/provisioned-device" \
      -H "Content-Type: application/json" \
      -d "{\"serviceName\":\"ALM_0F2320\",\"ip\":\"$NEW_IP\",\"ssid\":\"yolobolo\",\"source\":\"manual-update\"}" | python3 -m json.tool
    ;;
  --remote)
    echo "Note: Remote service uses ALMANACH_PRINTER_IP env var in k8s deployment."
    echo "To update, edit the deployment in crib-k3s:"
    echo "  kubectl edit deployment almanach-render -n almanach"
    echo "  Change ALMANACH_PRINTER_IP from current to $NEW_IP"
    ;;
esac
