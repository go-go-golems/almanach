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

## Step 5: Web Bluetooth provisioning UI design for `almanach/web`

This step added a second design document focused specifically on the browser UI work. The earlier design document focused on firmware BLE provisioning; this new guide explains how a Web Bluetooth provisioning experience should fit into the existing Almanach React app, which browser constraints matter, and how an intern should phase the implementation without destabilizing the existing editor, render-service, and print paths.

The most important finding is that the firmware-embedded `/almanach` page is not the right first-boot Web Bluetooth entrypoint because Web Bluetooth requires a secure context and first-boot devices are not yet reachable over WiFi. The UI can still live in `./almanach/web`, but it should be served from localhost during development or from a future HTTPS setup origin for production provisioning.

### Prompt Context

**User prompt (verbatim):** "Look at docmgr ticket ALMANACH-BLE-PROVISION and create a design implementation guide to add BLE provisioning to the web UI of ./almanach/web .

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Add a new ticket document for Web Bluetooth provisioning UI design, grounded in the existing BLE provisioning ticket and the current web/firmware code, then upload the deliverable to reMarkable.

**Inferred user intent:** Prepare an intern-facing handoff for the next phase: implementing a browser provisioning flow in the Almanach web app.

**Commit (code):** N/A — documentation-only work.

### What I did

- Reviewed the existing `ALMANACH-BLE-PROVISION` ticket documents, tasks, and diary.
- Inspected current frontend files:
  - `web/src/almanach-studio.jsx`
  - `web/esbuild.mjs`
  - `web/package.json`
- Inspected current firmware provisioning and HTTP files:
  - `firmware/atoms3r/main/provisioning_mgr.c`
  - `firmware/atoms3r/main/provisioning_mgr.h`
  - `firmware/atoms3r/main/app_main.c`
  - `firmware/atoms3r/main/web_server.c`
  - `firmware/atoms3r/main/CMakeLists.txt`
  - `firmware/atoms3r/sdkconfig.defaults`
- Created `design-doc/02-web-bluetooth-provisioning-ui-design-and-implementation-guide.md`.
- Related the new doc to the key frontend and firmware files with `docmgr doc relate`.
- Added and checked the task `Design Web Bluetooth provisioning UI for top-level Almanach web app`.

### Why

- Adding BLE provisioning to the web UI crosses frontend architecture, browser API limitations, ESP-IDF BLE provisioning semantics, firmware status endpoints, and build/firmware asset sync. An intern needs a cohesive map before writing code.
- The current React app is a monolithic 2,594-line file. The guide recommends a small topbar integration plus separate `web/src/provisioning/` modules to avoid making the monolith harder to maintain.
- The secure-context constraint changes the product shape: browser provisioning should be hosted from localhost/HTTPS, not assumed to work from the firmware's plain HTTP page.

### What worked

- The existing ticket already had a strong firmware provisioning design, which provided the firmware-side context for the web UI guide.
- Source inspection found clear anchors for the design:
  - The web app topbar and print path in `almanach-studio.jsx`.
  - The esbuild IIFE output and generated HTML shell in `esbuild.mjs`.
  - The firmware service name, PoP, service UUID, events, and status struct in `provisioning_mgr.c/.h`.
  - The current `/api/status` and embedded `/almanach` routes in `web_server.c`.
- The new guide includes architecture, diagrams, pseudocode, API references, phases, test strategy, risks, alternatives, and file references.

### What didn't work

- No code implementation was attempted in this step, so there were no build/test failures.
- I initially drafted one paragraph as if `app_main.c` had not yet integrated provisioning. A later source check showed the ticket implementation has progressed: `app_main.c` now includes `provisioning_mgr.h`, calls `start_network_onboarding()`, and registers provisioning console commands. I corrected the document before upload.
- The prompt said `nupload`; I interpreted that as upload to reMarkable.

### What I learned

- `almanach/web` is both the hosted render-service UI source and the source that can be synced into firmware assets. A provisioning feature added there must explicitly handle both runtime contexts.
- Web Bluetooth support is origin-dependent, so UX copy and capability checks are not optional polish; they are core correctness.
- The npm package `esp-idf-provisioning-web` exists and is worth a spike, but the design should hide any third-party package behind an adapter interface so the UI does not depend on a single library's API shape.

### What was tricky to build

- The tricky part was drawing the boundary between firmware-served UI and hosted setup UI. The same bundle can be embedded into firmware, but the same browser APIs will not necessarily be available when served over firmware HTTP. The guide therefore recommends clear capability detection and user-facing unsupported messages.
- Another subtle point is the ESP-IDF service UUID byte order. The firmware sets a byte array in C; browser code usually needs a canonical UUID string. The guide calls this out as a hardware validation item rather than pretending a copied string is guaranteed correct.

### What warrants a second pair of eyes

- Review the recommendation to use `esp-idf-provisioning-web` as a package spike before manual protocomm implementation.
- Review the proposed product stance that first-boot browser provisioning should be hosted from localhost/HTTPS rather than the firmware page.
- Review whether the guide should also propose splitting the current monolithic `almanach-studio.jsx` beyond provisioning modules.

### What should be done in the future

- Implement the mock `ProvisioningWizard` and topbar entrypoint first.
- Validate firmware BLE provisioning with Espressif's app before debugging custom browser provisioning.
- Spike `esp-idf-provisioning-web` against real AtomS3R hardware.
- Add a firmware `/api/status` provisioning block that does not expose PoP.

### Code review instructions

- Start with the new design doc's Executive Summary and Critical Constraint sections.
- Then compare `web/src/almanach-studio.jsx` topbar/print paths with the proposed minimal integration.
- Review `firmware/atoms3r/main/provisioning_mgr.c` for the firmware protocol facts used by the UI plan.
- Validate documentation with:
  - `cd almanach && docmgr doctor --ticket ALMANACH-BLE-PROVISION --stale-after 30`

### Technical details

New document:

```text
ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/design-doc/02-web-bluetooth-provisioning-ui-design-and-implementation-guide.md
```

Key design output:

- Add `web/src/provisioning/` modules.
- Add a topbar `Set up printer` modal entrypoint.
- Use `window.isSecureContext` and `navigator.bluetooth` capability checks.
- Keep the BLE/protocomm implementation behind an adapter interface.
- Treat firmware-served `/almanach` as post-WiFi UI, not the primary first-boot provisioning origin.

## Step 5: Flash and first serial-console validation on AtomS3R

This step flashed the provisioning firmware to the physical AtomS3R and validated the first-boot BLE advertising path plus the new provisioning console commands from `idf.py monitor`.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Validate the implemented phases on hardware where possible before moving on.

**Inferred user intent:** Catch integration problems that only appear on the ESP32-S3 runtime, not just during build.

**Commit (code):** pending — this step also includes a small follow-up fix to `provisioning_mgr_start_if_needed()` discovered during monitor testing.

### What I did

- Erased flash and flashed the current firmware to `/dev/ttyACM0`:
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 erase-flash`
  - `cd almanach/firmware/atoms3r && ./build.sh /dev/ttyACM0 flash`
- Started `idf.py monitor` in a tmux session.
- Verified first-boot logs after erased flash:
  - No saved WiFi credentials were found.
  - BLE provisioning started automatically.
  - Advertised provisioning identity was `ALM_0F2320`.
  - PoP was `alm-0f2320`.
  - QR payload was printed as JSON.
- Ran `prov_status` from the monitor console and confirmed:
  - manager initialized: yes
  - provisioned: no
  - BLE running: yes
  - WiFi: disconnected
- Ran `wifi_status` and confirmed the legacy console command still responds while BLE provisioning is active.
- Ran `prov_start` while BLE provisioning was already running and found the message was misleading: it said provisioning started even though the manager logged that it was already running.
- Fixed `provisioning_mgr_start_if_needed()` so it returns success without setting `out_started=true` when BLE provisioning is already running.
- Rebuilt and reflashed the firmware.
- Re-ran monitor validation and confirmed `prov_start` now prints:
  - `BLE provisioning not started: device is already provisioned or already running.`

### Why

- The build can only prove that the APIs link. It cannot prove BLE controller initialization, NimBLE startup, console registration, or real boot ordering on the AtomS3R.
- The misleading `prov_start` output was exactly the kind of runtime UX issue this phase was meant to catch.

### What worked

- Erase/flash completed successfully.
- Boot completed successfully.
- BLE/NimBLE initialized successfully.
- `wifi_prov_mgr` started BLE provisioning.
- Console remained usable while BLE provisioning was advertising.
- `prov_status`, `prov_start`, and `wifi_status` executed from the serial monitor.

### What didn't work

- I did not complete mobile-app provisioning in this step. The firmware is advertising the standard ESP-IDF BLE provisioning service, but the next validation pass still needs a phone/client to send credentials and confirm connected/reboot behavior.

### What I learned

- On the tested AtomS3R, the provisioning values are:
  - Service name: `ALM_0F2320`
  - PoP: `alm-0f2320`
- `wifi_prov_scheme_ble: BT memory released` appears during init before BLE provisioning starts; BLE still initializes and advertises correctly afterward.

### What was tricky to build

- `prov_start` needed a clear idempotent path. Calling it while BLE is already active should not look like a fresh start.

### What warrants a second pair of eyes

- Review the `s_running` guard in `provisioning_mgr_start_if_needed()` and confirm it is the desired public API behavior.
- During mobile-app provisioning, verify whether provisioning manager state transitions produce the expected status output for client-connected/security/session events.

### What should be done in the future

- Provision with Espressif's BLE provisioning app using:
  - Device: `ALM_0F2320`
  - PoP: `alm-0f2320`
- Confirm successful WiFi connection, web server startup, `/api/status`, reboot autoconnect, and reset/re-provision flow.

### Code review instructions

- Review the monitor transcript evidence in this diary step.
- Re-run on hardware with:
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 erase-flash && ./build.sh /dev/ttyACM0 flash`
  - `cd firmware/atoms3r && ./build.sh /dev/ttyACM0 monitor`
- Console smoke commands:
  - `prov_status`
  - `prov_start`
  - `wifi_status`

### Technical details

Observed first-boot provisioning log excerpt:

```text
No saved WiFi credentials — starting BLE provisioning
wifi_prov_mgr: Provisioning started with service name : ALM_0F2320
provisioning: BLE WiFi provisioning started
provisioning:   Device    : ALM_0F2320
provisioning:   Security  : Security 1
provisioning:   PoP       : alm-0f2320
```

## Step 6: Linux Go/Glazed BLE provisioning command

This step added a developer feedback-loop command to the Almanach binary: `ble-provision`. The command is implemented as a Glazed verb in Go, but it delegates the low-level ESP-IDF BLE/protocomm protocol to Espressif's maintained `esp_prov.py` client.

### Prompt Context

**User prompt (verbatim):** "can you build a linux go cli tool (actually make it a verb with glazed commands in the current almanach binary to do ble provisioning, we are on linux?) Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable. That way we can test the provisioning in a feedback loop locally"

**Assistant interpretation:** Add a practical Linux command to the existing Go CLI now, document how it works for a new intern, validate it against the currently advertising AtomS3R, and publish the guide to reMarkable.

**Inferred user intent:** Avoid relying only on phone/Web Bluetooth provisioning while firmware is still being iterated; keep all local testing in the Almanach CLI/terminal workflow.

**Commit (code):** pending after documentation and upload.

### What I did

- Added `internal/app/cmd_ble_provision.go`.
- Registered `ble-provision` in `internal/app/cmd_root.go`.
- Implemented Glazed flags for:
  - `--action provision|reset|reprov|version`
  - `--service-name`
  - `--ssid`
  - `--passphrase`
  - `--pop`
  - `--sec-ver`
  - `--proto-ver`
  - `--idf-path`
  - `--python`
  - `--esp-prov`
  - `--timeout`
  - `--dry-run`
  - `--install-hints`
- Used Espressif's `esp_prov.py` for the actual provisioning/reset/reprovision operations.
- Added a special `version` action that imports Espressif's Python helper functions and exits after the `proto-ver` check, because upstream `esp_prov.py` otherwise continues into WiFi scan/config after checking the version.
- Installed missing ESP-IDF Python provisioning dependencies into the local ESP-IDF Python env:
  - `protobuf`
  - `bleak`
  - `dbus-fast`
  - `cryptography` was already installed
- Wrote the intern-facing design guide:
  - `design-doc/03-linux-go-cli-ble-provisioning-feedback-loop-design-and-implementation-guide.md`

### Validation

Commands run:

```bash
go test ./...
go build ./cmd/almanach-render-service
go run ./cmd/almanach-render-service ble-provision --action version --service-name ALM_0F2320 --pop alm-0f2320 --timeout 30 --output yaml
```

The protocol check succeeded against the physical AtomS3R BLE advertisement:

```text
Discovering...
Connecting...
Getting Services...
proto-ver response :  {
        "prov": {
                "ver": "v1.1",
                "sec_ver": 1,
                "sec_patch_ver": 0,
                "cap": ["wifi_scan"]
        }
}
==== Verified protocol version successfully ====
Disconnecting...
```

### What worked

- Go/Glazed integration compiled and tests passed.
- Linux BLE discovery and GATT service lookup worked through `bleak`/BlueZ.
- The AtomS3R returned ESP-IDF provisioning protocol `v1.1` with Security 1 and `wifi_scan` capability.
- The command returns a structured Glazed row and redacts the WiFi passphrase from displayed command output.

### What didn't work

- Initial direct `esp_prov.py` runs failed because the selected ESP-IDF Python environment lacked `protobuf`.
- The first `version` action design used `esp_prov.py --proto_ver v1.1`, but upstream `esp_prov.py` continued into WiFi scanning after successful version verification and then failed at an interactive AP selection prompt. I replaced that path with a short Python snippet that imports Espressif functions and exits immediately after version verification.

### What I learned

- ESP-IDF 5.4 `wifi_prov_mgr` reports provisioning protocol version `v1.1`.
- A host-side version check is enough to prove the Linux BLE stack, service discovery, and `proto-ver` endpoint without sending WiFi credentials.

### What was tricky to build

- Keeping the command inside Go/Glazed while avoiding a risky pure-Go reimplementation of ESP protocomm. The wrapper approach gives a working feedback loop immediately while documenting the trade-off.
- The upstream Python client is interactive by default for scans and passphrases, so the Glazed command must require SSID for provisioning and provide a custom non-mutating version check.

### What warrants a second pair of eyes

- Review whether passing WiFi passphrase to `esp_prov.py` as a process argument is acceptable for local development. The command redacts outputs, but argv may still briefly contain the secret.
- Review whether the next phase should patch Espressif tooling for stdin secrets or jump directly to a pure-Go protocomm implementation.

### What should be done in the future

- Run `ble-provision --action provision` with real WiFi credentials and validate full WiFi join, web server startup, `/api/status`, reboot autoconnect, and reset/reprovision.
- Consider adding a native `scan` action returning Glazed rows.
- Consider adding `wait-api` to poll the printer after provisioning succeeds.

### Code review instructions

- Review `internal/app/cmd_ble_provision.go` for flag behavior, path resolution, timeout handling, and secret redaction.
- Review `internal/app/cmd_root.go` for command registration.
- Validate with:
  - `go test ./...`
  - `go run ./cmd/almanach-render-service ble-provision --action version --service-name ALM_0F2320 --pop alm-0f2320 --timeout 30 --output yaml`

## Step 8: Storybook and css-visual-diff coverage for the setup UI

This step focused only on the React web setup page. I added Storybook so the provisioning flow can be reviewed as deterministic UI states, then used `css-visual-diff compare` to capture screenshots for each state and a side-by-side comparison between the main Almanach editor and the new setup page.

### Prompt Context

**User prompt (verbatim):** "Use Storybook to create stories, and use `css-visual-diff help --all` to learn how to capture screenshots of the different components, so the setup page can be visually verified against the main page."

**Assistant interpretation:** Add Storybook infrastructure and stories for the provisioning UI states, learn the available css-visual-diff commands, and generate visual artifacts that make it easy to compare the setup page with the existing main app style.

**Inferred user intent:** Make the new standalone localhost setup page reviewable without hardware and without needing to run through the full BLE flow manually.

**Commit (code):** pending — these React/Storybook changes are not committed yet.

### What I did

- Added Storybook dependencies and scripts in `web/package.json`.
- Added Storybook config:
  - `web/.storybook/main.js`
  - `web/.storybook/preview.js`
- Added setup page stories in `web/src/provisioning/ProvisioningWizard.stories.jsx` for:
  - ready/default browser support
  - insecure/unsupported origin
  - WiFi details entered
  - provisioning in progress
  - success
  - error
- Added a main editor story in `web/src/AlmanachStudio.stories.jsx` so visual comparison uses the existing Almanach UI as a reference.
- Adjusted `ProvisioningWizard` to accept story-friendly props (`initialState`, `supportOverride`, `clientFactory`, and `storyMode`) while preserving the standalone page behavior.
- Added `storybook-static/` to `web/.gitignore` so build output is not accidentally committed.
- Used `css-visual-diff compare` to capture screenshots under:
  - `ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/artifacts/storybook-visuals/`

### Why

- The setup page has multiple important visual states that are hard to inspect by just running the mock flow once.
- Storybook keeps those states stable and URL-addressable for future screenshot capture.
- `css-visual-diff compare` writes screenshots, diffs, and Markdown reports; comparing each story to itself gives deterministic screenshot artifacts, while comparing the editor story to the setup story gives a quick visual style reference.

### What worked

- `pnpm --prefix web run build-storybook` completed successfully.
- `css-visual-diff compare` captured screenshots for all setup stories.
- Self-comparisons for setup stories reported `Changed percent: 0.0000%`, which is expected because each story was compared to itself only to produce stable screenshot artifacts.
- The main-editor-vs-setup comparison generated a report and screenshots at `artifacts/storybook-visuals/main-vs-setup-ready/`.
- The normal web build and Go tests still pass:
  - `BUILD_WEB_LOCAL=1 go run ./cmd/build-web`
  - `go test ./...`

### What didn't work

- The first static HTTP server attempt served `web/` instead of `web/storybook-static/`, so `/iframe.html` returned 404. I killed the server on port 6007 and restarted it from the correct directory.
- Storybook initially warned that `.storybook/main.js` was reparsed as an ES module. I added `"type": "module"` to `web/package.json`, after which the warning disappeared.

### What I learned

- `css-visual-diff compare` is the useful verb for this workflow because it writes `url1_screenshot.png`, `url2_screenshot.png`, `diff_comparison.png`, `diff_only.png`, `compare.json`, and `compare.md`.
- The setup page can be reviewed without hardware via Storybook state fixtures and the existing mock provisioning client.

### What was tricky to build

- The wizard originally owned all state internally, which made story variants awkward. The small prop seam keeps runtime behavior unchanged while allowing static story states.
- The visual comparison against the main editor should be interpreted as a style/design reference, not as a low-diff regression target; the pages are intentionally different layouts.

### What warrants a second pair of eyes

- Review the captured screenshots in `artifacts/storybook-visuals/`, especially `ready/url1_screenshot.png`, `success/url1_screenshot.png`, and `main-vs-setup-ready/diff_comparison.png`.
- Decide whether the setup page should move closer to the editor's dense toolbar/side-panel language or remain a simpler wizard card.

### Technical details

Commands run:

```bash
cd almanach
css-visual-diff help --all
css-visual-diff compare --help
pnpm --prefix web add -D storybook @storybook/react-vite
pnpm --prefix web run build-storybook
cd web/storybook-static && python3 -m http.server 6007 --bind 127.0.0.1
css-visual-diff compare --url1 http://127.0.0.1:6007/iframe.html?id=provisioning-setup-page--ready\&viewMode=story --url2 http://127.0.0.1:6007/iframe.html?id=provisioning-setup-page--ready\&viewMode=story --selector1 body --selector2 body --out .../artifacts/storybook-visuals/ready --viewport-w 1280 --viewport-h 820 --wait-ms1 700 --wait-ms2 700
BUILD_WEB_LOCAL=1 go run ./cmd/build-web
go test ./...
```

## Step 9: Localhost setup route and setup command

This step wired the committed React setup page into the Go binary. The service can now serve the standalone provisioning UI at `/setup` with its JavaScript bundle at `/setup/bundle.js`, using either `web/dist` in local development or the bundled `internal/web/embed/public` assets.

I also added a dedicated `almanach-render-service setup` command that binds to `127.0.0.1` and prints a `http://localhost:<port>/setup` URL. This keeps the setup page local-only while still satisfying Web Bluetooth's secure-context requirement for localhost.

### Prompt Context

**User prompt (verbatim):** "add tasks to the ticket and then work on them , commit at appropriate intervals, keep a diary (read skills if necessary)"

**Assistant interpretation:** Expand the ticket task list for the remaining setup-page serving work, implement the next tasks, validate them, commit at a clean boundary, and record the implementation in the ticket diary.

**Inferred user intent:** Continue from the reviewed React/Storybook setup UI into a usable localhost-served setup page from the Go binary, without mixing unrelated colleague firmware work.

**Commit (code):** dc36e39b28d1e47dd1c63541799c47c32f597a1f — "Serve setup page from localhost command"

### What I did

- Expanded the setup-page tasks with explicit subtasks for static handlers, route tests, setup subcommand, URL logging, and smoke validation.
- Added `/setup` and `/setup/bundle.js` handlers in `internal/app/static.go`.
- Kept the existing `/almanach` and `/almanach/bundle.js` editor routes unchanged.
- Refactored `RunServe` in `internal/app/cmd_serve.go` through a shared `runHTTPServer` helper that accepts an explicit listen address.
- Added `internal/app/cmd_setup.go` with `almanach-render-service setup`.
- Registered the new setup command from `internal/app/cmd_root.go`.
- Added `internal/app/static_test.go` to verify `/setup`, `/setup/bundle.js`, `/almanach`, and `/almanach/bundle.js` serve expected content types and body markers.

### Why

- The printer cannot serve a setup page before WiFi exists, so the setup UI must be served by the local Go binary.
- Web Bluetooth works from localhost, so the command prints a `localhost` URL while binding only to `127.0.0.1`.
- Route tests protect the embedded/local static asset mapping, especially the intentionally different bundle filenames: `setup-bundle.js` on disk and `/setup/bundle.js` in the browser.

### What worked

- `go test ./...` passed.
- `BUILD_WEB_LOCAL=1 go run ./cmd/build-web` passed and regenerated `setup.html`/`setup-bundle.js` into the embed directory.
- A local smoke test passed with:
  - `go run ./cmd/almanach-render-service setup --port 18299`
  - `curl http://127.0.0.1:18299/setup`
  - `curl http://127.0.0.1:18299/setup/bundle.js`
- The setup HTML contained `/setup/bundle.js` and the setup bundle contained `ALMANACH SETUP`.

### What didn't work

- During the smoke loop, the first two `curl` attempts failed because the `go run` server was still compiling/starting:
  - `curl: (7) Failed to connect to 127.0.0.1 port 18299 after 0 ms: Couldn't connect to server`
- Retrying in the loop succeeded once the server printed:
  - `Open setup page: http://localhost:18299/setup`
  - `Listening:   http://127.0.0.1:18299`

### What I learned

- The static asset serving layer already had the right disk-vs-bundled abstraction; setup serving only needed two additional routes.
- Sharing a lower-level HTTP server helper made it easy for the normal `serve` command to retain all behavior while the setup command uses a localhost-only address.

### What was tricky to build

- The browser-facing setup bundle URL is `/setup/bundle.js`, but the generated asset file is `setup-bundle.js`. The route handler deliberately maps between those names.
- `RunServe` previously used `ListenAndServe` directly, which only supported the implicit all-interfaces `:<port>` address. The setup command needed localhost-only binding, so I changed the server startup to create a `net.Listener` explicitly and serve on that listener.

### What warrants a second pair of eyes

- Review whether `almanach-render-service setup` should expose only static setup routes or whether it is acceptable that it currently registers the same API routes as the normal service.
- Review whether the default setup port should remain `ALMANACH_PORT`/8199 or move to a dedicated setup default.

### What should be done in the future

- Add an optional browser-open flag if desired.
- Spike the real Web Bluetooth provisioning adapter behind the current mock-client boundary.
- Consider route redirects such as `/setup/` to `/setup` if users type the trailing slash often.

### Code review instructions

- Start with `internal/app/static.go` and verify the setup route/file mapping.
- Then inspect `internal/app/cmd_setup.go` and `internal/app/cmd_serve.go` for the localhost bind behavior.
- Validate with:
  - `go test ./...`
  - `BUILD_WEB_LOCAL=1 go run ./cmd/build-web`
  - `go run ./cmd/almanach-render-service setup --port 18299`
  - open `http://localhost:18299/setup`

### Technical details

Commands run:

```bash
cd almanach
gofmt -w internal/app/cmd_serve.go internal/app/cmd_setup.go internal/app/cmd_root.go internal/app/static.go internal/app/static_test.go
go test ./...
BUILD_WEB_LOCAL=1 go run ./cmd/build-web
go run ./cmd/almanach-render-service setup --port 18299
curl -fsS http://127.0.0.1:18299/setup
curl -fsS http://127.0.0.1:18299/setup/bundle.js
```
