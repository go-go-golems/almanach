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
