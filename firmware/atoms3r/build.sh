#!/usr/bin/env bash
set -euo pipefail

# Source ESP-IDF if idf.py is not already on PATH. Prefer an ESP-IDF 5.4.x
# install because this firmware uses components such as esp_driver_gpio.
if ! command -v idf.py >/dev/null 2>&1; then
    if [[ -n "${IDF_PATH:-}" && -f "${IDF_PATH}/export.sh" ]]; then
        # shellcheck source=/dev/null
        source "${IDF_PATH}/export.sh"
    elif [[ -f "${HOME}/esp/esp-idf-5.4.2/export.sh" ]]; then
        # shellcheck source=/dev/null
        source "${HOME}/esp/esp-idf-5.4.2/export.sh"
    elif [[ -f "${HOME}/esp/esp-idf-5.4.1/export.sh" ]]; then
        # shellcheck source=/dev/null
        source "${HOME}/esp/esp-idf-5.4.1/export.sh"
    elif [[ -f "${HOME}/esp/esp-idf/export.sh" ]]; then
        # shellcheck source=/dev/null
        source "${HOME}/esp/esp-idf/export.sh"
    else
        echo "ERROR: idf.py not found. Source ESP-IDF 5.4.x export.sh or set IDF_PATH first." >&2
        exit 1
    fi
fi

PORT="${1:-/dev/ttyACM0}"
ACTION="${2:-build}"
TARGET="${IDF_TARGET:-esp32s3}"

case "$ACTION" in
    build)
        idf.py -D "IDF_TARGET=${TARGET}" build
        ;;
    flash)
        idf.py -D "IDF_TARGET=${TARGET}" -p "$PORT" flash
        ;;
    monitor)
        idf.py -p "$PORT" monitor
        ;;
    flash-monitor|tmux-flash-monitor)
        idf.py -D "IDF_TARGET=${TARGET}" -p "$PORT" flash monitor
        ;;
    erase-flash)
        idf.py -p "$PORT" erase-flash
        ;;
    *)
        echo "Usage: $0 [PORT] {build|flash|monitor|flash-monitor|erase-flash}" >&2
        exit 1
        ;;
esac
