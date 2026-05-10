---
Title: Investigation Diary
Ticket: ALMANACH-BLE-PROVISION
Status: active
Topics:
    - almanach
    - firmware
    - esp-idf
    - ble
    - wifi-provisioning
    - thermal-printer
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/app_main.c
      Note: Current boot and console integration studied for provisioning design.
    - Path: /home/manuel/workspaces/2026-05-08/extract-almanach/almanach/firmware/atoms3r/main/wifi_cmd.c
      Note: Existing esp_console WiFi save/status/forget behavior studied.
    - Path: /home/manuel/workspaces/2026-05-08/extract-almanach/esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/main.c
      Note: Donor BLE provisioning implementation studied.
ExternalSources: []
Summary: Chronological investigation diary for the ALMANACH-BLE-PROVISION ticket.
LastUpdated: 2026-05-10T13:20:00-04:00
WhatFor: Use this to understand what was inspected and why the BLE provisioning design recommends a staged port.
WhenToUse: Read before resuming implementation work on BLE provisioning or console WiFi behavior.
---

# Diary

## Goal

This diary records the investigation and documentation work for porting BLE WiFi provisioning into the Almanach AtomS3R firmware while preserving USB esp_console WiFi configuration and flash-backed credential storage.

## Step 1: Ticket setup, evidence gathering, and intern-facing design document

I created a new docmgr ticket for the BLE provisioning port and inspected both the current `almanach/firmware/atoms3r` firmware and the older ESP-IDF BLE provisioning prototype. The key finding is that the current Almanach firmware already has console WiFi scan/connect/status/forget commands with NVS persistence, while the donor prototype has the missing ESP-IDF `wifi_prov_mgr` BLE flow.

The design document therefore recommends a port, not a rewrite. The first implementation should add a small `provisioning_mgr` subsystem around ESP-IDF's standard provisioning manager, keep the existing `wifi_mgr` and `wifi_cmd` behavior, and treat browser/Web Bluetooth provisioning as a later client-side feature against the same firmware protocol.

### Prompt Context

**User prompt (verbatim):** "Let's create a ticket to port it to almanach firmware.

Also add esp_console support to set wifi settings and save them to flash as well, as we  have in plenty of other firmwares.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new ticket in the `almanach` repo, analyze current firmware and the existing BLE provisioning prototype, produce a detailed intern-facing design and implementation guide, keep a diary, and upload the final docs to reMarkable.

**Inferred user intent:** Prepare a clear technical handoff so a new engineer can implement BLE WiFi provisioning and preserve console-based WiFi settings without needing to rediscover the firmware architecture.

**Commit (code):** N/A — documentation-only work in progress.

### What I did

- Created ticket `ALMANACH-BLE-PROVISION` in `almanach/ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware`.
- Created the primary design document `design-doc/01-ble-wifi-provisioning-port-analysis-design-and-implementation-guide.md`.
- Created this investigation diary at `reference/01-investigation-diary.md`.
- Inspected current Almanach firmware files:
  - `firmware/atoms3r/main/app_main.c`
  - `firmware/atoms3r/main/wifi_mgr.c`
  - `firmware/atoms3r/main/wifi_cmd.c`
  - `firmware/atoms3r/main/nvs_store.c`
  - `firmware/atoms3r/main/web_server.c`
  - `firmware/atoms3r/main/CMakeLists.txt`
  - `firmware/atoms3r/sdkconfig.defaults`
- Inspected donor BLE provisioning files:
  - `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/main.c`
  - `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/sdkconfig.defaults`
  - `esp32-s3-m5/0092-m5-printer-esp-idf-provision/source/atomlite-printer-prov/main/CMakeLists.txt`
- Wrote the design guide with architecture mapping, file references, pseudocode, diagrams, API references, implementation phases, validation plan, risks, alternatives, and intern checklist.

### Why

- BLE provisioning touches boot order, WiFi state, NVS, event handling, Bluetooth stack configuration, and post-connect web-server startup. A new intern needs the full system map before writing C code.
- The existing firmware already satisfies part of the request: console WiFi settings are present and saved to flash. The implementation plan should preserve and verify this rather than duplicate it blindly.
- The donor provisioning prototype is useful but targets different hardware and has a different application shape, so the correct plan is to port concepts and APIs into the AtomS3R firmware.

### What worked

- `docmgr ticket create-ticket` created the ticket workspace successfully.
- `docmgr doc add` created both the design document and the investigation diary.
- Source inspection found concrete evidence that `wifi_connect` saves credentials using `nvs_store_save_wifi()` after a successful connection.
- Source inspection found the donor ESP-IDF provisioning manager code path using `wifi_prov_scheme_ble`, `WIFI_PROV_SECURITY_1`, service-name generation, and provisioning event handlers.

### What didn't work

- No implementation was attempted in this step, so there were no compile/test failures.
- The prompt said "nupload"; I interpreted that as "upload" to reMarkable.

### What I learned

- The moved Almanach firmware already has stronger console WiFi support than expected: `wifi_scan`, `wifi_connect`, `wifi_status`, `wifi_disconnect`, and `wifi_forget` are implemented.
- The missing feature is first-boot BLE provisioning and reset/re-provision lifecycle management.
- Browser/Web Bluetooth provisioning is not currently present; the firmware should first expose the standard ESP-IDF BLE provisioning protocol so both Espressif's mobile app and a future browser client can work against it.

### What was tricky to build

- The main design tension is ownership of WiFi events. `wifi_mgr` already handles `WIFI_EVENT` and `IP_EVENT`, while the donor provisioning prototype also handles those events directly. The guide recommends keeping `wifi_mgr` authoritative if possible and only adding mode flags if provisioning conflicts with auto-connect behavior.
- Another subtle point is credential source of truth. Console credentials are stored explicitly in namespace `wifi`, while ESP-IDF provisioning stores credentials through the WiFi/provisioning stack. The design recommends keeping both paths initially and making reset commands erase both.

### What warrants a second pair of eyes

- Review the proposed hybrid source-of-truth strategy for provisioned credentials plus console-saved credentials.
- Review the proposed MAC-derived proof-of-possession before implementation; it is better than a fixed universal PoP but not a high-security production scheme.
- Review whether `wifi_forget` should reset provisioning state or whether only `prov_reset` should do that.

### What should be done in the future

- Implement the phased plan in the design doc.
- Create a separate ticket for browser/Web Bluetooth provisioning client if desired.
- Validate on physical AtomS3R hardware with Espressif's provisioning app before writing a custom browser client.

### Code review instructions

- Start with the design doc's Current-State Architecture and Proposed Architecture sections.
- Then inspect `app_main.c`, `wifi_mgr.c`, `wifi_cmd.c`, and donor `main.c` side by side.
- Validate the docs with `docmgr doctor --ticket ALMANACH-BLE-PROVISION --stale-after 30`.
- Upload bundle should include the design doc and this diary.

### Technical details

Commands run during setup and inspection included:

```bash
cd almanach
docmgr status --summary-only
docmgr ticket create-ticket --ticket ALMANACH-BLE-PROVISION --title "Port BLE WiFi Provisioning to Almanach AtomS3R Firmware" --topics almanach,firmware,esp-idf,ble,wifi-provisioning,thermal-printer
docmgr doc add --ticket ALMANACH-BLE-PROVISION --doc-type design-doc --title "BLE WiFi Provisioning Port Analysis Design and Implementation Guide"
docmgr doc add --ticket ALMANACH-BLE-PROVISION --doc-type reference --title "Investigation Diary"
```

Key evidence commands used `nl -ba` on current firmware and donor prototype files to capture line-referenced behavior for the design guide.

## Step 2: Phase 1 BLE provisioning build dependencies

This step added the firmware build-time dependencies required before writing any provisioning code. I enabled BLE-only NimBLE and protocomm security 1 in `sdkconfig.defaults`, then declared `protocomm`, `wifi_provisioning`, and `bt` as private component requirements for the firmware main component.

The goal was deliberately narrow: prove the AtomS3R firmware still clean-builds with the Bluetooth/provisioning stack linked in before adding new application code. This gives the next implementation phases a known-good base and catches ESP-IDF component/Kconfig mistakes early.

### Prompt Context

**User prompt (verbatim):** "add detailed tasks to the ticket, then work on them one by one, commit at aprpopriate intervals, keep a detailed diary. flash when needed, you can also run idf.py monitor in a tmux and use that."

**Assistant interpretation:** Add concrete ticket tasks, implement the BLE provisioning plan incrementally, commit logical steps, and keep this diary current with commands and validation.

**Inferred user intent:** Move from design into implementation while preserving reviewability and hardware validation discipline.

**Commit (code):** pending — this step will be committed after the diary/changelog update.

### What I did

- Added detailed implementation tasks to `tasks.md` for phases 1 through 6.
- Added BLE provisioning Kconfig defaults to `firmware/atoms3r/sdkconfig.defaults`:
  - `CONFIG_BT_ENABLED=y`
  - `CONFIG_BTDM_CTRL_MODE_BLE_ONLY=y`
  - `CONFIG_BT_NIMBLE_ENABLED=y`
  - `CONFIG_ESP_PROTOCOMM_SUPPORT_SECURITY_VERSION_1=y`
- Added `protocomm`, `wifi_provisioning`, and `bt` to `firmware/atoms3r/main/CMakeLists.txt` `PRIV_REQUIRES`.
- Ran a clean firmware build:
  - `cd almanach/firmware/atoms3r && rm -rf build sdkconfig && ./build.sh /dev/ttyACM0 build`

### Why

- The provisioning manager code will require ESP-IDF BLE provisioning components, so the build graph must include them first.
- A clean build after only dependency changes proves the toolchain and Kconfig baseline are valid before application behavior changes.

### What worked

- The clean firmware build completed successfully.
- ESP-IDF built the Bluetooth, protocomm, and wifi_provisioning components.
- Output image was generated:
  - `build/stoms3r.bin`
  - Size: `0x116a30`
  - Smallest app partition: `0x400000`
  - Free space: `0x2e95d0 bytes (73%)`

### What didn't work

- No failures in this phase.

### What I learned

- Enabling BLE/NimBLE and provisioning components only increased the binary modestly relative to the previous moved-firmware build.
- The existing partition table has ample room for the provisioning stack.

### What was tricky to build

- The important invariant is using ESP-IDF 5.4.x and target `esp32s3`. That was already fixed in `build.sh` during the earlier firmware migration, so this phase could focus on Kconfig/component dependencies.

### What warrants a second pair of eyes

- Review whether `bt` should remain an explicit `PRIV_REQUIRES` or can be transitively pulled through `wifi_provisioning`. I left it explicit for clarity.
- Review whether additional NimBLE memory tuning is needed after runtime testing.

### What should be done in the future

- Implement `provisioning_mgr.c/.h` and validate another build.
- Later flash and monitor runtime logs after provisioning code exists.

### Code review instructions

- Review `firmware/atoms3r/sdkconfig.defaults` for the new BLE/protocomm settings.
- Review `firmware/atoms3r/main/CMakeLists.txt` for the new component dependencies.
- Validate with:
  - `cd firmware/atoms3r && rm -rf build sdkconfig && ./build.sh /dev/ttyACM0 build`

### Technical details

The successful build reported:

```text
Successfully created esp32s3 image.
Generated .../firmware/atoms3r/build/stoms3r.bin
stoms3r.bin binary size 0x116a30 bytes. Smallest app partition is 0x400000 bytes. 0x2e95d0 bytes (73%) free.
```

## Step 3: Phase 2 and 3 provisioning manager and boot-flow integration

This step added the first real firmware implementation for BLE WiFi provisioning. I created a dedicated `provisioning_mgr` module, then changed the firmware boot flow so first boot can start BLE provisioning while existing provisioned or console-saved credentials still take the station-mode path.

The implementation keeps WiFi/IP state in `wifi_mgr` and makes provisioning own only ESP-IDF provisioning/protocomm events. This follows the design guide's preferred first approach and avoids rewriting the working console WiFi path before hardware validation proves a conflict exists.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue implementing the ticket phases and commit at logical milestones.

**Inferred user intent:** Add the actual BLE provisioning firmware plumbing while keeping changes reviewable.

**Commit (code):** pending — this step will be committed after diary/changelog update.

### What I did

- Added `firmware/atoms3r/main/provisioning_mgr.h`.
- Added `firmware/atoms3r/main/provisioning_mgr.c`.
- Added `provisioning_mgr.c` to `firmware/atoms3r/main/CMakeLists.txt`.
- Implemented MAC-derived BLE service identity:
  - service name: `ALM_<last-three-mac-bytes>`
  - PoP: `alm-<last-three-mac-bytes>`
- Implemented provisioning/protocomm event logging for:
  - `WIFI_PROV_EVENT`
  - `PROTOCOMM_TRANSPORT_BLE_EVENT`
  - `PROTOCOMM_SECURITY_SESSION_EVENT`
- Implemented provisioning manager APIs:
  - `provisioning_mgr_init`
  - `provisioning_mgr_is_provisioned`
  - `provisioning_mgr_start_if_needed`
  - `provisioning_mgr_start_force`
  - `provisioning_mgr_stop`
  - `provisioning_mgr_reset`
  - `provisioning_mgr_get_status`
- Added `wifi_mgr_start_station()` for ESP-IDF stored credentials.
- Updated `app_main.c` boot flow to:
  - initialize provisioning manager
  - start station mode if ESP-IDF says the device is provisioned
  - fall back to explicit `nvs_store_load_wifi()` console credentials
  - start BLE provisioning when no credentials exist
- Built firmware after integration:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Why

- The firmware needs first-boot onboarding without USB serial, but the current console and saved-WiFi paths should remain valid.
- ESP-IDF provisioning manager can store credentials in the WiFi stack's NVS storage; `wifi_mgr_start_station()` lets the firmware connect using those stored credentials without knowing the SSID/password itself.

### What worked

- Firmware build passed after adding the manager and boot-flow integration.
- Output image was generated successfully:
  - `stoms3r.bin binary size 0x158140 bytes`
  - Free app partition space: `0x2a7ec0 bytes (66%)`

### What didn't work

- The first version of the manager used `ESP_RETURN_ON_ERROR(wifi_prov_mgr_stop_provisioning(), ...)`, but ESP-IDF 5.4's `wifi_prov_mgr_stop_provisioning()` returns `void`, not `esp_err_t`.
- Fix: call `wifi_prov_mgr_stop_provisioning()` directly and then update local state.
- The first Kconfig defaults copied older ESP32 symbols (`CONFIG_BTDM_CTRL_MODE_*`) and the old console history symbol. ESP-IDF 5.4 for ESP32-S3 reported them as unknown.
- Fix: remove those unknown symbols and keep only `CONFIG_BT_ENABLED`, `CONFIG_BT_NIMBLE_ENABLED`, and protocomm security 1.

### What I learned

- ESP-IDF 5.4's WiFi provisioning manager API differs slightly from older examples around return types and target-specific Kconfig symbols.
- Linking the provisioning manager into application code pulls in significantly more binary than dependency-only linking, but the app partition still has substantial headroom.

### What was tricky to build

- The subtle part is avoiding duplicate event handler registration. Provisioning can deinitialize itself on `WIFI_PROV_END`, but the event handlers should not be registered repeatedly on later manager init calls. I added a local `s_handlers_registered` guard.
- Another subtle point is source-of-truth. The boot flow now checks ESP-IDF provisioning state first, then falls back to explicit console NVS credentials. This should support both BLE-provisioned and console-configured devices.

### What warrants a second pair of eyes

- Review `provisioning_mgr_reset()` behavior with `wifi_prov_mgr_reset_provisioning()` before relying on it for `prov_reset`.
- Review whether deinitializing the provisioning manager on `WIFI_PROV_END` is correct alongside later status queries.
- Review whether `wifi_mgr_start_station()` should tolerate a different already-started WiFi error code than `ESP_ERR_WIFI_CONN`.

### What should be done in the future

- Add console commands for provisioning status/start/reset.
- Flash and monitor runtime behavior after commands are available.

### Code review instructions

- Start with `provisioning_mgr.h` for the API contract.
- Review `provisioning_mgr.c` for event ownership, service identity, and start/reset behavior.
- Review `app_main.c` `start_network_onboarding()` for boot decision order.
- Review `wifi_mgr_start_station()` in `wifi_mgr.c`.
- Validate with:
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Technical details

Successful build summary:

```text
Successfully created esp32s3 image.
stoms3r.bin binary size 0x158140 bytes. Smallest app partition is 0x400000 bytes. 0x2a7ec0 bytes (66%) free.
```

## Step 4: Phase 4 and 5 provisioning console commands and reset semantics

This step added user-visible console commands for the BLE provisioning lifecycle. The firmware now exposes `prov_status`, `prov_start`, and `prov_reset` alongside the existing WiFi commands, and `wifi_forget` now clears both the explicit console WiFi namespace and the ESP-IDF provisioning state.

The point of this phase is operability. Once the firmware is flashed, a developer can use USB Serial/JTAG to inspect the provisioning service name and PoP, start provisioning if needed, or reset the device back into first-boot provisioning mode without manually erasing flash from the host.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the phased implementation and keep console-based recovery paths available.

**Inferred user intent:** Make the provisioning work testable and recoverable from the serial console before hardware BLE validation.

**Commit (code):** pending — this step will be committed after diary/changelog update.

### What I did

- Added `firmware/atoms3r/main/provisioning_cmd.h`.
- Added `firmware/atoms3r/main/provisioning_cmd.c`.
- Registered new commands in `app_main.c`:
  - `prov_status`
  - `prov_start`
  - `prov_reset`
- Added `provisioning_cmd.c` to `main/CMakeLists.txt`.
- Updated `wifi_cmd.c` so `wifi_forget` now calls `provisioning_mgr_reset()` after erasing explicit WiFi credentials.
- Built firmware successfully:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Why

- BLE provisioning needs a console recovery surface because wrong WiFi credentials, wrong PoP, or stale provisioning state are common during development.
- A reset command should clear both storage paths: explicit `nvs_store` WiFi keys and ESP-IDF provisioning state.

### What worked

- Firmware built successfully after adding the command module and reset integration.
- Output image remained well inside the app partition:
  - `stoms3r.bin binary size 0x158a30 bytes`
  - Free space: `0x2a75d0 bytes (66%)`

### What didn't work

- No compile failures in this phase.

### What I learned

- The provisioning manager API created in Step 3 was sufficient for console status/start/reset without exposing ESP-IDF provisioning internals to command handlers.
- Separating `provisioning_cmd.c` from `wifi_cmd.c` keeps station-mode commands and provisioning lifecycle commands easier to review.

### What was tricky to build

- Reset semantics are easy to get wrong because there are two credential paths. `prov_reset` and `wifi_forget` now both erase explicit WiFi credentials and reset provisioning state; `prov_reset` additionally reboots immediately so the device re-enters the normal boot decision tree.

### What warrants a second pair of eyes

- Review whether `wifi_forget` should reboot like `prov_reset` or whether non-rebooting behavior is preferable for a console command named `wifi_forget`.
- Review whether `prov_start` should force provisioning after reset, or whether refusing when already provisioned is safer.

### What should be done in the future

- Flash to hardware and use monitor/console to verify the commands are registered and behave as intended.
- Test `prov_reset` after successful provisioning.

### Code review instructions

- Review `provisioning_cmd.c` command behavior and messages.
- Review `app_main.c` command registration.
- Review `wifi_cmd.c` `do_wifi_forget()` reset behavior.
- Validate with:
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 build`

### Technical details

New command summary:

```text
prov_status  Show provisioning manager state, BLE service name, PoP, and current IP
prov_start   Start BLE provisioning if the device is not already provisioned
prov_reset   Erase explicit/provisioned WiFi state and reboot
```
