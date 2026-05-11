---
Title: BLE WiFi Provisioning Port Analysis Design and Implementation Guide
Ticket: ALMANACH-BLE-PROVISION
Status: active
Topics:
    - almanach
    - firmware
    - esp-idf
    - ble
    - wifi-provisioning
    - thermal-printer
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/main.c
      Note: |-
        Donor BLE WiFi provisioning firmware using ESP-IDF wifi_prov_mgr and BLE scheme.
        Donor BLE WiFi provisioning implementation using wifi_prov_mgr
    - Path: ../../../../../../../esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/sdkconfig.defaults
      Note: |-
        Donor provisioning Kconfig defaults for BLE, NimBLE, and protocomm security.
        Donor BLE/NimBLE/protocomm Kconfig defaults
    - Path: firmware/atoms3r/main/app_main.c
      Note: |-
        Current AtomS3R firmware boot sequence, saved WiFi autoconnect, console startup, and web-server start task.
        Current firmware boot sequence
    - Path: firmware/atoms3r/main/nvs_store.c
      Note: |-
        Current NVS WiFi credential persistence API.
        Explicit flash persistence helpers for WiFi credentials
    - Path: firmware/atoms3r/main/web_server.c
      Note: HTTP status endpoint and post-WiFi web server behavior
    - Path: firmware/atoms3r/main/wifi_cmd.c
      Note: |-
        Existing esp_console WiFi commands for scan/connect/status/disconnect/forget.
        Existing esp_console WiFi commands and NVS-backed save/forget behavior
    - Path: firmware/atoms3r/main/wifi_mgr.c
      Note: |-
        Current WiFi station manager and event handling to reuse or refactor around provisioning.
        Current station-mode WiFi manager and event handling
ExternalSources:
    - 'ESP-IDF WiFi Provisioning Manager: https://docs.espressif.com/projects/esp-idf/en/latest/esp32/api-reference/provisioning/wifi_provisioning.html'
    - 'ESP-IDF Protocomm: https://docs.espressif.com/projects/esp-idf/en/latest/esp32/api-reference/provisioning/protocomm.html'
Summary: Design and implementation guide for porting the existing ESP-IDF BLE WiFi provisioning prototype into the Almanach AtomS3R firmware while preserving USB esp_console WiFi commands and flash-backed settings.
LastUpdated: 2026-05-10T13:20:00-04:00
WhatFor: Use this as the intern-facing technical plan for implementing BLE WiFi provisioning, console WiFi configuration, NVS persistence, and post-connect web/printer behavior in almanach/firmware/atoms3r.
WhenToUse: Read before changing the AtomS3R firmware boot flow, WiFi manager, provisioning manager integration, sdkconfig defaults, console commands, or browser/mobile onboarding path.
---


# BLE WiFi Provisioning Port Analysis Design and Implementation Guide

## Executive Summary

The Almanach firmware already knows how to join WiFi, save credentials in NVS, expose a USB Serial/JTAG console, start an HTTP printer/web UI after WiFi connects, and report its IP address over `/api/status`. What it does not yet have is first-boot BLE WiFi provisioning. A separate older prototype under `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov` demonstrates the missing provisioning path with ESP-IDF's standard `wifi_prov_mgr`, BLE transport, protocomm security 1, and a provisioning service name derived from the device MAC address.

This document proposes porting that provisioning flow into `almanach/firmware/atoms3r` as a small, explicit subsystem rather than copying the whole prototype wholesale. The new subsystem should make the AtomS3R firmware behave like a consumer device:

1. On first boot, if WiFi credentials are absent, start BLE provisioning.
2. A mobile app or future Web Bluetooth client sends SSID/password through the ESP-IDF provisioning protocol.
3. The firmware stores credentials through ESP-IDF WiFi provisioning/NVS behavior and connects as a station.
4. Once connected, the firmware starts the HTTP server and optionally prints/logs the assigned IP address.
5. The USB console remains available as a recovery and factory-floor path for scan/connect/status/forget.

The implementation should not block on browser provisioning. The firmware should first support Espressif-compatible BLE provisioning. Browser provisioning is a client-side follow-up: it requires a Web Bluetooth implementation of Espressif's GATT/protocomm/protobuf/security flow. The firmware side should use the standard provisioning manager so both Espressif's mobile app and a future browser client can target the same protocol.

## Problem Statement and Scope

A new Almanach printer device needs WiFi before the web UI and render-service print flow become useful. Today the firmware requires a USB serial console command:

```text
wifi_connect --ssid <ssid> --pass <password>
```

That is acceptable for development but not for normal onboarding. Users expect to configure WiFi from a phone or browser without installing an ESP-IDF toolchain or opening a serial terminal. The requested work is therefore to port BLE WiFi provisioning into the Almanach AtomS3R firmware and to ensure console-based WiFi setting/save support remains present.

### In scope

- Integrate ESP-IDF `wifi_prov_mgr` BLE provisioning into `almanach/firmware/atoms3r`.
- Preserve and, if needed, polish existing `esp_console` WiFi commands.
- Use flash-backed credential storage so provisioned or console-entered WiFi survives reboot.
- Keep the HTTP server startup behavior tied to successful WiFi connection.
- Provide clear service naming, proof-of-possession strategy, reset/re-provision path, validation commands, and file-level implementation steps.
- Document browser/Web Bluetooth as a later client-side project, not as a prerequisite for this firmware port.

### Out of scope for the first implementation

- Implementing a full Web Bluetooth provisioning client in `almanach/web`.
- Replacing the current firmware web UI.
- Redesigning the printer protocol.
- Renaming the firmware project from `stoms3r` to an Almanach-specific binary name.
- Adding a full captive portal SoftAP provisioning flow.

## Current-State Architecture

### Repository layout

The relevant standalone repository paths are:

```text
almanach/
├── firmware/atoms3r/            ESP-IDF firmware for AtomS3R Lite + K118 printer
│   ├── CMakeLists.txt
│   ├── sdkconfig.defaults
│   ├── partitions.csv
│   └── main/
│       ├── app_main.c           boot sequence and console startup
│       ├── wifi_mgr.c/.h        station-mode WiFi manager
│       ├── wifi_cmd.c/.h        esp_console WiFi commands
│       ├── nvs_store.c/.h       NVS helpers for WiFi and printer settings
│       ├── web_server.c/.h      embedded HTTP server and print APIs
│       └── printer_*.c/.h       K118 thermal printer driver and commands
└── web/                         top-level Almanach Studio source, not yet auto-synced into firmware
```

The donor provisioning prototype remains outside the new repo:

```text
esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/
├── main/main.c                  donor provisioning boot flow
├── main/app_button.c/.h         factory reset button helper
├── main/app_printer.c/.h        minimal printer status receipt helper
├── sdkconfig.defaults           BLE/NimBLE/protocomm settings
└── partitions.csv               larger factory partition for BLE stack
```

### Current AtomS3R boot flow

`firmware/atoms3r/main/app_main.c` initializes NVS, network stack, WiFi manager, printer UART, autoconnects saved WiFi credentials, starts a background task that launches the web server once WiFi is connected, and finally starts the USB Serial/JTAG REPL.

Evidence:

- NVS init is first (`app_main.c:116-118`).
- The network stack and default event loop are created before WiFi (`app_main.c:119-124`).
- Saved credentials are loaded with `nvs_store_load_wifi()` and passed to `wifi_mgr_connect()` (`app_main.c:130-139`).
- `web_server_task` polls `wifi_mgr_is_connected()` and calls `web_server_start()` after connection (`app_main.c:91-107`).
- The REPL prompt is `stoms3r>`, and command registration includes `printer_cmd_register()` and `wifi_cmd_register()` (`app_main.c:145-163`).

Current boot sequence diagram:

```text
app_main
  |
  +--> nvs_store_init()
  +--> esp_netif_init()
  +--> esp_event_loop_create_default()
  +--> wifi_mgr_init()
  +--> printer_drv_init()
  +--> nvs_store_load_wifi()
  |      |
  |      +-- found     --> wifi_mgr_connect(ssid, password)
  |      +-- not found --> no WiFi, no provisioning today
  |
  +--> xTaskCreate(web_server_task)
  |      |
  |      +-- wait up to 30s for wifi_mgr_is_connected()
  |      +-- web_server_start()
  |
  +--> start esp_console REPL
```

### Current WiFi manager

`firmware/atoms3r/main/wifi_mgr.c` wraps station-mode operations around ESP-IDF WiFi APIs. It creates an event group, creates the default station netif, initializes WiFi, registers event handlers, scans, connects, disconnects, and returns the current IP address.

Important details:

- `wifi_mgr_init()` creates `s_eg`, creates default station netif, calls `esp_wifi_init()`, and registers `WIFI_EVENT` and `IP_EVENT_STA_GOT_IP` handlers (`wifi_mgr.c:65-90`).
- The event handler auto-connects on `WIFI_EVENT_STA_START` (`wifi_mgr.c:35-40`).
- The event handler auto-reconnects on disconnect (`wifi_mgr.c:44-51`).
- `wifi_mgr_connect()` sets station mode, starts WiFi, configures SSID/password, and waits up to 15 seconds for `CONNECTED_BIT` (`wifi_mgr.c:159-188`).
- `wifi_mgr_get_ip()` reads IP info from the stored station netif (`wifi_mgr.c:204-214`).

The current manager is simple and development-friendly, but it will need careful coordination with `wifi_prov_mgr`, because the provisioning manager also owns parts of WiFi initialization, provisioning state, and event flow.

### Current console WiFi commands

`firmware/atoms3r/main/wifi_cmd.c` already implements the requested esp_console support for WiFi settings and flash persistence:

- `wifi_scan` calls `wifi_mgr_scan()` (`wifi_cmd.c:29-37`).
- `wifi_connect --ssid <ssid> --pass <password>` calls `wifi_mgr_connect()`, prints the IP, and saves credentials with `nvs_store_save_wifi()` after successful connection (`wifi_cmd.c:49-80`).
- `wifi_status` prints connected/disconnected state and saved SSID (`wifi_cmd.c:87-106`).
- `wifi_disconnect` calls `wifi_mgr_disconnect()` (`wifi_cmd.c:112-117`).
- `wifi_forget` disconnects and erases saved credentials with `nvs_store_erase_wifi()` (`wifi_cmd.c:123-132`).
- Registration uses `esp_console_cmd_register()` with argtable definitions (`wifi_cmd.c:139-175`).

This means the console feature is not missing in the current moved firmware. The implementation task should preserve it, add any missing ergonomics, and ensure it coexists with BLE provisioning.

### Current NVS persistence

`firmware/atoms3r/main/nvs_store.c` provides explicit flash persistence helpers:

- `nvs_store_init()` initializes NVS and erases/retries if pages are exhausted or version mismatched (`nvs_store.c:14-28`).
- `nvs_store_save_wifi()` opens namespace `wifi`, saves string keys `ssid` and `password`, and commits (`nvs_store.c:30-58`).
- `nvs_store_load_wifi()` reads the same keys (`nvs_store.c:60-78`).
- `nvs_store_erase_wifi()` erases both keys (`nvs_store.c:80-96`).

ESP-IDF's provisioning manager also persists WiFi credentials using the WiFi driver/NVS path. The implementation must decide whether to keep these explicit keys as the source of truth, migrate to `wifi_prov_mgr_is_provisioned()`, or synchronize both. The recommended approach below keeps explicit console keys for compatibility and uses the provisioning manager's provisioned state for BLE onboarding.

### Current HTTP server dependency on WiFi

`firmware/atoms3r/main/web_server.c` serves the embedded web UI and APIs after WiFi connects. `/api/status` reports `wifi.connected` and the current IP (`web_server.c:99-119`). This is important because a provisioning client can eventually use the device IP to open the web UI after provisioning.

```json
{
  "ok": true,
  "wifi": { "connected": true, "ip": "192.168.0.126" },
  "printer": { "baud": 460800, "swapped": true }
}
```

The server is not useful before WiFi because the device is not reachable on the LAN. BLE provisioning must happen before or alongside the web server startup flow.

## Donor BLE Provisioning Prototype

The donor firmware in `0092-m5-printer-esp-idf-provision` is a minimal ESP-IDF application for ATOM Lite. It is not the same hardware target as AtomS3R Lite, but its provisioning logic is the right conceptual source.

### Donor behavior

The donor boot flow:

1. Initializes NVS, netif, event loop, event group, button, and printer.
2. Registers provisioning, BLE transport, security, WiFi, and IP event handlers.
3. Initializes WiFi.
4. Calls `start_ble_provisioning()`.
5. Prints a WiFi status receipt after it gets an IP.

Evidence:

- Provisioning headers include `protocomm_security.h`, `protocomm_security1.h`, `wifi_provisioning/manager.h`, and `wifi_provisioning/scheme_ble.h` (`main.c:27-30`).
- It derives a service name from the WiFi MAC address (`main.c:58-64`).
- It logs Espressif app provisioning payload including service name, PoP, and transport (`main.c:66-75`).
- It handles `WIFI_PROV_EVENT`, `PROTOCOMM_TRANSPORT_BLE_EVENT`, `PROTOCOMM_SECURITY_SESSION_EVENT`, `WIFI_EVENT`, and `IP_EVENT_STA_GOT_IP` (`main.c:83-158`).
- `start_ble_provisioning()` configures `wifi_prov_scheme_ble`, frees BTDM after provisioning, checks `wifi_prov_mgr_is_provisioned()`, sets a custom BLE service UUID, uses `WIFI_PROV_SECURITY_1`, and starts provisioning (`main.c:160-198`).
- The app initializes NVS/netif/event loop/registers events before WiFi init and provisioning (`main.c:200-235`).

### Donor Kconfig requirements

The donor `sdkconfig.defaults` enables BLE provisioning via NimBLE:

```text
CONFIG_BT_ENABLED=y
CONFIG_BTDM_CTRL_MODE_BLE_ONLY=y
CONFIG_BTDM_CTRL_MODE_BR_EDR_ONLY=n
CONFIG_BTDM_CTRL_MODE_BTDM=n
CONFIG_BT_NIMBLE_ENABLED=y
CONFIG_ESP_PROTOCOMM_SUPPORT_SECURITY_VERSION_1=y
```

Evidence: `0092.../sdkconfig.defaults:10-18`.

For AtomS3R, these settings should be added to `firmware/atoms3r/sdkconfig.defaults`, but the memory constraints differ. AtomS3R Lite has ESP32-S3 with 8 MB flash and 8 MB PSRAM, so it has more headroom than the ATOM Lite ESP32-PICO-D4 donor.

### Donor limitations

The donor is a proof-of-concept, not a drop-in module:

- It targets ATOM Lite/ESP32, not AtomS3R Lite/ESP32-S3.
- It has its own printer and button helpers that do not match the current `printer_drv` architecture.
- It does not start the Almanach HTTP server.
- It does not preserve the current esp_console WiFi commands.
- It uses a fixed PoP string `12345678`, which is acceptable for development but weak for production.

## Gap Analysis

### Current firmware gaps

The current AtomS3R firmware lacks these capabilities:

- First-boot BLE provisioning when no WiFi credentials are present.
- A provisioning service name / onboarding identity for the device.
- Provisioning event handling for success/failure/client connect/security mismatch.
- BLE/NimBLE/protocomm sdkconfig defaults.
- A reset/re-provision command that resets provisioning manager state in addition to erasing explicit NVS keys.
- A structured way to print or expose the assigned IP immediately after provisioning.

### Current firmware strengths to preserve

Do not discard these existing features:

- USB Serial/JTAG console works and is valuable for factory/debug recovery.
- Console WiFi commands already save settings to flash.
- NVS error handling already handles no-free-pages/new-version conditions.
- Web server startup is cleanly delayed until WiFi is connected.
- `/api/status` already exposes connected/IP state after the HTTP server starts.

## Proposed Architecture

### High-level design

Add a new provisioning subsystem around ESP-IDF `wifi_prov_mgr` and integrate it with the current boot flow. Keep `wifi_mgr` as the station-mode helper for console/manual flows, but avoid double-initializing WiFi or fighting the provisioning manager during provisioning.

Recommended new files:

```text
firmware/atoms3r/main/provisioning_mgr.c
firmware/atoms3r/main/provisioning_mgr.h
```

Recommended responsibilities:

- Own BLE provisioning manager setup/teardown.
- Generate service name and proof-of-possession.
- Register provisioning-related event handlers or expose handlers for `app_main` to register.
- Provide `provisioning_mgr_start_if_needed()`.
- Provide `provisioning_mgr_reset()` for console/factory reset.
- Provide status fields for console and logging.

### New boot flow

The new boot flow should be:

```text
app_main
  |
  +--> nvs_store_init()
  +--> esp_netif_init()
  +--> esp_event_loop_create_default()
  +--> wifi_mgr_init_base() or wifi_mgr_init()
  +--> printer_drv_init()
  +--> provisioning_mgr_init()
  |
  +--> if provisioning manager says device is provisioned:
  |       start station mode / autoconnect
  |    else if explicit console credentials exist:
  |       connect using nvs_store_load_wifi + wifi_mgr_connect
  |    else:
  |       start BLE provisioning
  |
  +--> xTaskCreate(web_server_task)
  +--> start esp_console REPL
```

There are two viable source-of-truth strategies. The recommended one is **hybrid but explicit**:

1. Prefer ESP-IDF provisioning state for BLE-provisioned credentials.
2. Keep existing `nvs_store_*` keys for console-entered credentials.
3. Make `wifi_forget` erase both explicit console keys and provisioning manager state.
4. On boot, if either provisioning state or explicit console credentials exist, start station mode.

The reason is practical: the current console path works and writes its own namespace. The provisioning manager writes through ESP-IDF WiFi storage. Both are valid sources unless we deliberately migrate one into the other.

### Component dependency changes

Update `firmware/atoms3r/main/CMakeLists.txt` `PRIV_REQUIRES`:

```cmake
PRIV_REQUIRES
    console
    esp_driver_gpio
    driver
    nvs_flash
    esp_wifi
    esp_netif
    esp_event
    esp_http_server
    protocomm
    wifi_provisioning
    bt
```

Depending on ESP-IDF 5.4 component boundaries, `bt` may be required directly or pulled by `wifi_provisioning`/scheme BLE. Start with explicit dependencies for clarity.

Update `firmware/atoms3r/sdkconfig.defaults`:

```text
# BLE WiFi provisioning transport.
CONFIG_BT_ENABLED=y
CONFIG_BTDM_CTRL_MODE_BLE_ONLY=y
CONFIG_BTDM_CTRL_MODE_BR_EDR_ONLY=n
CONFIG_BTDM_CTRL_MODE_BTDM=n
CONFIG_BT_NIMBLE_ENABLED=y

# Protocomm security 1 for PoP-based provisioning.
CONFIG_ESP_PROTOCOMM_SUPPORT_SECURITY_VERSION_1=y
```

Keep existing AtomS3R settings for USB Serial/JTAG console, PSRAM, 8 MB flash, partitions, CPU, and console history.

### Provisioning manager API sketch

`provisioning_mgr.h` should be small and boring:

```c
#pragma once

#include <stdbool.h>
#include "esp_err.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    bool initialized;
    bool provisioned;
    bool running;
    bool client_connected;
    bool security_ok;
    char service_name[32];
    char pop[32];
} provisioning_status_t;

esp_err_t provisioning_mgr_init(void);
esp_err_t provisioning_mgr_start_if_needed(void);
esp_err_t provisioning_mgr_start_force(void);
esp_err_t provisioning_mgr_stop(void);
esp_err_t provisioning_mgr_reset(void);
esp_err_t provisioning_mgr_get_status(provisioning_status_t *out);

#ifdef __cplusplus
}
#endif
```

Potential implementation outline:

```c
static EventGroupHandle_t s_prov_events;
static bool s_initialized;
static bool s_running;
static char s_service_name[32];
static char s_pop[32];

esp_err_t provisioning_mgr_init(void) {
    if (s_initialized) return ESP_OK;

    wifi_prov_mgr_config_t cfg = {
        .scheme = wifi_prov_scheme_ble,
        .scheme_event_handler = WIFI_PROV_SCHEME_BLE_EVENT_HANDLER_FREE_BTDM,
        .app_event_handler = WIFI_PROV_EVENT_HANDLER_NONE,
    };

    ESP_RETURN_ON_ERROR(wifi_prov_mgr_init(cfg), TAG, "prov mgr init");
    make_service_name(s_service_name, sizeof(s_service_name));
    make_pop(s_pop, sizeof(s_pop));
    set_service_uuid();
    s_initialized = true;
    return ESP_OK;
}

esp_err_t provisioning_mgr_start_if_needed(void) {
    bool provisioned = false;
    ESP_RETURN_ON_ERROR(wifi_prov_mgr_is_provisioned(&provisioned), TAG, "is provisioned");
    if (provisioned) {
        wifi_prov_mgr_deinit();
        s_initialized = false;
        return ESP_ERR_INVALID_STATE; // means: already provisioned; caller starts station
    }

    const wifi_prov_security_t sec = WIFI_PROV_SECURITY_1;
    const wifi_prov_security1_params_t *params = (const wifi_prov_security1_params_t *)s_pop;
    ESP_RETURN_ON_ERROR(wifi_prov_mgr_start_provisioning(sec, params, s_service_name, NULL), TAG, "start provisioning");
    s_running = true;
    log_qr_payload(s_service_name, s_pop);
    return ESP_OK;
}
```

Use `ESP_ERR_INVALID_STATE` only if the caller treats it explicitly. Otherwise return a custom boolean through an out parameter:

```c
esp_err_t provisioning_mgr_start_if_needed(bool *started);
```

That is clearer for interns and avoids overloading error codes.

### Service name and proof-of-possession

The donor uses:

```c
PROV_PREFIX = "M5PRN_"
service_name = M5PRN_ + last 3 bytes of WiFi STA MAC
PROV_POP = "12345678"
```

For Almanach AtomS3R, use a clearer prefix:

```text
ALM_<last-three-mac-bytes>
```

For development, a fixed PoP is acceptable if it is visible in logs:

```text
ALMANACH_POP = "almanach"
```

For production, derive PoP from device identity or print it on a setup card/label. Examples:

- `pop = last 6 MAC hex chars`
- `pop = "alm-" + last 6 MAC hex chars`
- `pop = random value stored in NVS at first boot and printed once`

Recommended first implementation:

```text
service_name = ALM_<MAC[3]><MAC[4]><MAC[5]>
pop          = alm-<MAC[3]><MAC[4]><MAC[5]>
```

This is not high security, but it avoids a universal hard-coded PoP and is easy to explain in logs and docs.

Provisioning QR payload format:

```json
{"ver":"v1","name":"ALM_A1B2C3","pop":"alm-a1b2c3","transport":"ble"}
```

### Event flow

The firmware should register handlers for these event bases:

- `WIFI_PROV_EVENT`
- `PROTOCOMM_TRANSPORT_BLE_EVENT`
- `PROTOCOMM_SECURITY_SESSION_EVENT`
- `WIFI_EVENT`
- `IP_EVENT`

The current `wifi_mgr` already handles `WIFI_EVENT` and `IP_EVENT`. To avoid duplicate reconnect loops, choose one of these designs:

#### Option A: Centralize WiFi/IP events in `wifi_mgr`

`wifi_mgr` continues to own `WIFI_EVENT` and `IP_EVENT`. `provisioning_mgr` owns only provisioning/protocomm events. This is preferred if it works cleanly with `wifi_prov_mgr`.

```text
WIFI_EVENT / IP_EVENT ------------------> wifi_mgr
WIFI_PROV_EVENT / PROTOCOMM_* ----------> provisioning_mgr
```

Pros:

- Fewer duplicate event handlers.
- Keeps `wifi_mgr_is_connected()` and `wifi_mgr_get_ip()` authoritative.
- Web server task remains unchanged.

Risk:

- `wifi_mgr` auto-connects on every `WIFI_EVENT_STA_START`, while provisioning manager may also control station start. This needs testing.

#### Option B: Centralize all provisioning-time events in `provisioning_mgr`

`provisioning_mgr` owns provisioning and WiFi/IP events until provisioning ends. Then normal `wifi_mgr` starts.

Pros:

- Follows donor flow more closely.
- Avoids `wifi_mgr` reconnect behavior interfering with provisioning.

Risk:

- More state transitions and more code.
- Harder to keep console `wifi_status` accurate during provisioning.

Recommended first implementation: **Option A with guardrails**. If `wifi_mgr` auto-connect behavior conflicts with provisioning, add a `wifi_mgr_set_autoreconnect(bool)` or `wifi_mgr_init_without_autostart()` rather than rewriting everything.

### Console command additions

Existing commands are good. Add provisioning-specific commands:

```text
prov_status      Show BLE provisioning status, provisioned state, service name, and PoP hint
prov_start       Force-start BLE provisioning if not connected/provisioned
prov_reset       Reset provisioning and saved console WiFi credentials, then reboot or require manual reboot
```

Potential command behavior:

```c
static int do_prov_status(int argc, char **argv) {
    provisioning_status_t st;
    provisioning_mgr_get_status(&st);
    printf("Provisioned: %s\n", st.provisioned ? "yes" : "no");
    printf("BLE running: %s\n", st.running ? "yes" : "no");
    printf("Service: %s\n", st.service_name);
    printf("PoP: %s\n", st.pop);
    if (wifi_mgr_is_connected()) { print IP; }
}

static int do_prov_reset(int argc, char **argv) {
    wifi_mgr_disconnect();
    nvs_store_erase_wifi();
    provisioning_mgr_reset();
    printf("Provisioning reset. Rebooting...\n");
    esp_restart();
}
```

Question: should these live in `wifi_cmd.c` or a new `provisioning_cmd.c`?

Recommendation: create `provisioning_cmd.c/.h` so WiFi station commands and provisioning lifecycle commands remain separate.

### Browser/Web Bluetooth path

A browser client is a future feature. The firmware should expose the standard ESP-IDF provisioning BLE service; then a future `web/provisioning` page can implement a client.

A Web Bluetooth client would need to implement:

1. `navigator.bluetooth.requestDevice()` with a filter for the provisioning service UUID or service-name prefix.
2. GATT connection.
3. Discovery of Espressif provisioning characteristics/endpoints.
4. Protocomm version/security session establishment.
5. Protobuf encoding/decoding for scan/config/apply/status commands.
6. Security 1 PoP handshake or Security 0 for development only.
7. WiFi scan UI and SSID/password form.
8. Polling for provisioning status and eventual IP.

Browser flow diagram:

```text
Browser page
  |
  +--> requestDevice({ filters: [{ services: [prov_uuid] }] })
  +--> connect GATT
  +--> open protocomm security endpoint
  +--> send WiFi scan request
  +--> show SSIDs
  +--> send WiFi config { ssid, passphrase }
  +--> send apply config
  +--> poll status until connected
  +--> show IP / open http://<ip>/
```

This is not in the firmware ticket's first implementation because it is a separate JS/protobuf/client compatibility project. The firmware design should not prevent it.

## Implementation Plan

### Phase 0: Baseline validation

Before editing, validate the moved firmware still builds:

```bash
cd almanach/firmware/atoms3r
./build.sh /dev/ttyACM0 build
```

Expected: ESP-IDF 5.4.x, target `esp32s3`, successful `stoms3r.bin`.

### Phase 1: Add Kconfig and CMake provisioning dependencies

Files:

- `firmware/atoms3r/sdkconfig.defaults`
- `firmware/atoms3r/main/CMakeLists.txt`

Add BLE/NimBLE/protocomm settings from donor, adapted for ESP32-S3. Add `protocomm`, `wifi_provisioning`, and Bluetooth dependencies to the component manifest.

Validation:

```bash
rm -rf firmware/atoms3r/build firmware/atoms3r/sdkconfig
cd firmware/atoms3r
./build.sh /dev/ttyACM0 build
```

This phase should compile even before provisioning code exists if dependencies are valid.

### Phase 2: Extract provisioning manager skeleton

Files:

- Add `firmware/atoms3r/main/provisioning_mgr.h`
- Add `firmware/atoms3r/main/provisioning_mgr.c`
- Update `firmware/atoms3r/main/CMakeLists.txt` `SRCS`

Implement:

- service-name generation from MAC
- PoP generation
- status struct
- event handler with logs only
- init/deinit scaffolding

Do not start provisioning yet. Build after skeleton.

### Phase 3: Wire provisioning into boot flow

File:

- `firmware/atoms3r/main/app_main.c`

Replace the current saved-credentials-only block with a provisioning decision function.

Pseudocode:

```c
static void start_network_onboarding(void) {
    bool prov_started = false;
    bool provisioned = false;

    ESP_ERROR_CHECK(provisioning_mgr_init());
    ESP_ERROR_CHECK(provisioning_mgr_is_provisioned(&provisioned));

    if (provisioned) {
        ESP_LOGI(TAG, "Provisioned WiFi found; starting station");
        wifi_mgr_start_station();
        return;
    }

    char ssid[64], password[64];
    if (nvs_store_load_wifi(ssid, sizeof(ssid), password, sizeof(password)) == ESP_OK) {
        ESP_LOGI(TAG, "Console-saved WiFi found; connecting");
        wifi_mgr_connect(ssid, password);
        return;
    }

    ESP_LOGI(TAG, "No WiFi credentials; starting BLE provisioning");
    ESP_ERROR_CHECK(provisioning_mgr_start(&prov_started));
}
```

Important: if ESP-IDF provisioning stores credentials in the WiFi driver NVS, the "provisioned" path should start station mode rather than call `wifi_mgr_connect()` with explicit SSID/password.

### Phase 4: Add provisioning console commands

Files:

- Add `firmware/atoms3r/main/provisioning_cmd.h`
- Add `firmware/atoms3r/main/provisioning_cmd.c`
- Update `firmware/atoms3r/main/CMakeLists.txt`
- Update `firmware/atoms3r/main/app_main.c` to call `provisioning_cmd_register()`

Commands:

```text
prov_status
prov_start
prov_reset
```

Also review `wifi_forget`: it should probably call `provisioning_mgr_reset()` or clearly state that it erases console credentials only. Prefer a single reset command that erases both.

### Phase 5: Preserve and test console WiFi settings

The current console behavior already saves WiFi settings to flash. Do not remove it.

Polish options:

- Add `wifi_save --ssid <s> --pass <p>` for saving without connecting, if needed.
- Add `wifi_reconnect` to connect saved credentials.
- Add clearer `wifi_status` output showing both connected IP and whether console credentials exist.

Only add these if useful; the required behavior already exists through `wifi_connect` and `wifi_forget`.

### Phase 6: Print/log post-provisioning IP

After `IP_EVENT_STA_GOT_IP`, ensure the device tells the user where to go.

Minimum:

- Log IP with `ESP_LOGI`.
- `wifi_status` prints IP.

Nice-to-have:

- Print a small thermal receipt once after provisioning:

```text
Almanach ready
IP: 192.168.0.126
Open: http://192.168.0.126/
```

Avoid printing on every reconnect. Store a one-shot in RAM or trigger only on first post-provision connection.

### Phase 7: Validation with Espressif app

Use Espressif's mobile app first:

```text
Transport: BLE
Device: ALM_<last-mac-bytes>
PoP: alm-<last-mac-bytes>
```

Validate:

1. Fresh flash / erased NVS starts BLE provisioning.
2. App sees the device.
3. App sends WiFi credentials successfully.
4. Firmware logs `WIFI_PROV_CRED_SUCCESS`.
5. Firmware gets IP.
6. HTTP server starts.
7. Browser can open `http://<ip>/`.
8. `/api/status` reports connected and the same IP.
9. Reboot autoconnects without BLE provisioning.
10. `prov_reset` or `wifi_forget` reset behavior works as documented.

## Testing Strategy

### Build tests

```bash
cd almanach/firmware/atoms3r
rm -rf build sdkconfig
./build.sh /dev/ttyACM0 build
```

Expected:

- target `esp32s3`
- no missing component errors
- binary fits partition

### Serial console smoke tests

After flashing:

```text
help
wifi_status
wifi_scan
wifi_connect --ssid "..." --pass "..."
wifi_status
wifi_forget
wifi_status
prov_status
```

Expected:

- WiFi commands still register.
- Successful `wifi_connect` saves credentials.
- Reboot autoconnects with console-saved credentials.
- `wifi_forget` or `prov_reset` removes credentials as documented.

### BLE provisioning tests

Fresh erased NVS:

```bash
cd firmware/atoms3r
./build.sh /dev/ttyACM0 erase-flash
./build.sh /dev/ttyACM0 flash-monitor
```

Expected monitor output:

```text
Device not provisioned; starting BLE provisioning service
Transport : BLE
Device    : ALM_XXXXXX
Security  : Security 1
PoP       : alm-xxxxxx
QR data   : {...}
```

Provision with Espressif app. Then verify:

```text
WiFi connected, IP=...
WiFi connected — starting web server
```

### HTTP tests after provisioning

From host on same LAN:

```bash
curl http://<device-ip>/api/status
curl -X POST http://<device-ip>/api/print/text \
  -H 'Content-Type: application/json' \
  -d '{"text":"Almanach provisioning test"}'
```

### Regression tests

- Wrong PoP should fail and log `PROTOCOMM_SECURITY_SESSION_CREDENTIALS_MISMATCH`.
- Wrong WiFi password should log provisioning credential failure.
- Power cycle after successful provisioning should not start BLE provisioning again.
- Reset command should make BLE provisioning available again.

## Risks and Mitigations

### Risk: WiFi manager conflicts with provisioning manager

`wifi_mgr` currently auto-connects on `WIFI_EVENT_STA_START`. During provisioning, the provisioning manager may start WiFi and expect control.

Mitigation:

- Start with minimal integration and test.
- If conflicts appear, add an explicit mode flag:

```c
typedef enum {
    WIFI_MGR_MODE_MANUAL,
    WIFI_MGR_MODE_PROVISIONING,
    WIFI_MGR_MODE_STATION,
} wifi_mgr_mode_t;
```

Then skip auto-connect during provisioning.

### Risk: BLE stack increases firmware size

BLE/NimBLE/protocomm adds code and RAM usage. The moved firmware build currently has a large partition headroom, but this must be checked after enabling BLE.

Mitigation:

- Keep NimBLE BLE-only.
- Keep Classic BT disabled.
- Validate binary size after Phase 1 and Phase 3.

### Risk: universal PoP is insecure

A fixed PoP such as `12345678` is simple but weak.

Mitigation:

- Use MAC-derived PoP for first implementation.
- Later generate/store a random PoP and print it or include it on packaging.

### Risk: browser provisioning expectations

Users may hear "BLE provisioning" and expect browser setup immediately.

Mitigation:

- Document that firmware supports the standard ESP-IDF BLE provisioning protocol first.
- Treat Web Bluetooth as a separate client implementation against the same firmware protocol.

## Alternatives Considered

### SoftAP captive portal provisioning

The device could create its own WiFi AP and serve a setup page. This avoids BLE/Web Bluetooth complexity but requires users to switch networks and adds AP-mode state management.

Rejected for first implementation because there is already a working BLE provisioning prototype and ESP-IDF's BLE provisioning app is a known path.

### Browser-only custom BLE protocol

The firmware could expose custom GATT characteristics for SSID/password/IP without using Espressif provisioning manager.

Rejected because it would create a custom security and compatibility problem. ESP-IDF provisioning already solves scan/config/apply/status, security, and mobile-app compatibility.

### Remove console WiFi commands

The firmware could rely only on BLE provisioning.

Rejected because the console path is already implemented, useful for debugging, and explicitly requested.

## API and File Reference

### ESP-IDF APIs

- `wifi_prov_mgr_init()` — initialize provisioning manager.
- `wifi_prov_mgr_is_provisioned()` — determine whether WiFi credentials already exist.
- `wifi_prov_mgr_start_provisioning()` — start provisioning service.
- `wifi_prov_mgr_deinit()` — release provisioning manager resources.
- `wifi_prov_mgr_reset_provisioning()` — reset provisioning state if available in the target ESP-IDF version.
- `wifi_prov_scheme_ble` — BLE transport scheme.
- `wifi_prov_scheme_ble_set_service_uuid()` — set custom BLE service UUID.
- `protocomm_security1` / `WIFI_PROV_SECURITY_1` — PoP-based provisioning security.
- `esp_wifi_set_mode(WIFI_MODE_STA)` and `esp_wifi_start()` — start station mode after provisioning.
- `esp_event_handler_register()` / `esp_event_handler_instance_register()` — event handling.
- `nvs_flash_init()` / `nvs_flash_erase()` — NVS storage lifecycle.

### Current Almanach firmware files

- `firmware/atoms3r/main/app_main.c` — boot sequence, autoconnect, web server task, console startup.
- `firmware/atoms3r/main/wifi_mgr.c` — station mode manager and IP tracking.
- `firmware/atoms3r/main/wifi_cmd.c` — esp_console WiFi commands.
- `firmware/atoms3r/main/nvs_store.c` — explicit credential persistence.
- `firmware/atoms3r/main/web_server.c` — HTTP status and print APIs after WiFi connects.
- `firmware/atoms3r/main/CMakeLists.txt` — source list, component dependencies, embedded assets.
- `firmware/atoms3r/sdkconfig.defaults` — target, flash, console, PSRAM, and future BLE provisioning Kconfig defaults.

### Donor prototype files

- `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/main.c` — provisioning flow to port conceptually.
- `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/sdkconfig.defaults` — BLE/NimBLE/protocomm Kconfig reference.
- `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/README.md` — donor usage notes.

## Intern Checklist

Before implementation:

- [ ] Build current firmware from `almanach/firmware/atoms3r`.
- [ ] Read `app_main.c`, `wifi_mgr.c`, `wifi_cmd.c`, and `nvs_store.c`.
- [ ] Read donor `main.c` provisioning flow.
- [ ] Confirm ESP-IDF version is 5.4.x.

During implementation:

- [ ] Add Kconfig defaults and CMake dependencies.
- [ ] Add `provisioning_mgr.c/.h`.
- [ ] Add `provisioning_cmd.c/.h` if adding console provisioning commands.
- [ ] Wire boot flow carefully.
- [ ] Keep `wifi_connect`/`wifi_status`/`wifi_forget` working.
- [ ] Build after each phase.

Before handoff:

- [ ] Build passes from clean `build/` and no committed `sdkconfig`.
- [ ] Console WiFi path works.
- [ ] BLE provisioning works with Espressif app.
- [ ] Reboot autoconnect works.
- [ ] Reset/re-provision works.
- [ ] `/api/status` reports connected IP after provisioning.

## Open Questions

1. Should PoP be MAC-derived for the first release, or should we generate a random PoP and print it?
2. Should `wifi_forget` reset both console credentials and ESP-IDF provisioning state, or should that be reserved for `prov_reset`?
3. Should the firmware print a status receipt after every new provisioning success?
4. Should browser/Web Bluetooth provisioning be a separate ticket after firmware support lands?
5. Should the firmware project eventually rename `stoms3r` identifiers to Almanach-specific names?
