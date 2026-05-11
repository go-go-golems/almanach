---
Title: Storybook and css-visual-diff Playbook
Ticket: ALMANACH-BLE-PROVISION
Status: active
Topics:
  - almanach
  - storybook
  - css-visual-diff
  - visual-regression
  - provisioning-ui
DocType: playbook
Intent: operational
Summary: How to run the Almanach Storybook setup UI stories and capture visual comparison artifacts with css-visual-diff.
LastUpdated: 2026-05-10T13:55:00-04:00
---

# Storybook and css-visual-diff Playbook

Use this playbook when reviewing the localhost BLE provisioning setup page visually against the existing Almanach editor UI.

The workflow is intentionally local and disposable: Storybook gives deterministic component states, while `css-visual-diff compare` captures screenshots and reports under ignored artifact directories.

## 1. Start the Almanach Storybook

From the repository root:

```bash
cd almanach
pnpm --prefix web exec storybook dev -p 6010 --host 127.0.0.1
```

Use port `6010` instead of the default `6006` when another repository already has a Storybook running.

Open the Storybook shell:

```text
http://127.0.0.1:6010
```

Direct story URLs that are useful for screenshots:

```text
http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--ready&viewMode=story
http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--unsupported-insecure-origin&viewMode=story
http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--wifi-details-entered&viewMode=story
http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--provisioning-progress&viewMode=story
http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--success&viewMode=story
http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--error-state&viewMode=story
http://127.0.0.1:6010/iframe.html?id=almanach-studio-page--default&viewMode=story
```

## 2. Validate Storybook builds

Before relying on visual artifacts, make sure the static Storybook build still works:

```bash
cd almanach
pnpm --prefix web run build-storybook
```

The generated `web/storybook-static/` directory is ignored and should not be committed.

## 3. Capture one setup story screenshot/report

`css-visual-diff compare` always compares two URLs. To use it as a screenshot/report generator for a single story, compare the story to itself:

```bash
cd almanach
OUT='ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/artifacts/storybook-visuals/ready'

css-visual-diff compare \
  --url1 'http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--ready&viewMode=story' \
  --url2 'http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--ready&viewMode=story' \
  --selector1 body \
  --selector2 body \
  --out "$OUT" \
  --viewport-w 1280 \
  --viewport-h 820 \
  --wait-ms1 700 \
  --wait-ms2 700
```

Expected result for a self-comparison:

```text
Changed percent: 0.0000%
```

Important outputs:

- `url1_screenshot.png` — the story screenshot to inspect.
- `url1_full.png` — full-page screenshot.
- `compare.md` — human-readable report.
- `compare.json` — machine-readable report.
- `diff_comparison.png` and `diff_only.png` — should be empty/no-op for self-comparisons.

## 4. Capture all setup states

```bash
cd almanach
BASE='ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/artifacts/storybook-visuals'

for id in \
  ready \
  unsupported-insecure-origin \
  wifi-details-entered \
  provisioning-progress \
  success \
  error-state
 do
  css-visual-diff compare \
    --url1 "http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--$id&viewMode=story" \
    --url2 "http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--$id&viewMode=story" \
    --selector1 body \
    --selector2 body \
    --out "$BASE/$id" \
    --viewport-w 1280 \
    --viewport-h 820 \
    --wait-ms1 700 \
    --wait-ms2 700
 done
```

Review the `url1_screenshot.png` files in each state directory.

## 5. Compare the main editor against the setup page

This comparison is a design reference, not a regression assertion. The two pages are intentionally different layouts, so a large pixel diff is expected.

```bash
cd almanach
OUT='ttmp/2026/05/10/ALMANACH-BLE-PROVISION--port-ble-wifi-provisioning-to-almanach-atoms3r-firmware/artifacts/storybook-visuals/main-vs-setup-ready'

css-visual-diff compare \
  --url1 'http://127.0.0.1:6010/iframe.html?id=almanach-studio-page--default&viewMode=story' \
  --url2 'http://127.0.0.1:6010/iframe.html?id=provisioning-setup-page--ready&viewMode=story' \
  --selector1 body \
  --selector2 body \
  --out "$OUT" \
  --viewport-w 1280 \
  --viewport-h 820 \
  --wait-ms1 700 \
  --wait-ms2 700
```

Review these files first:

- `url1_screenshot.png` — existing Almanach editor.
- `url2_screenshot.png` — setup page.
- `diff_comparison.png` — side-by-side diff visualization.
- `compare.md` — changed-pixel percentage and CSS style details.

## 6. Keep generated artifacts out of Git

The generated visual artifacts are ignored by `.gitignore`:

```gitignore
css-visual-diff-compare-*/
ttmp/**/artifacts/storybook-visuals/
```

This keeps local screenshot review output from polluting commits. If a specific screenshot should be preserved in a design document, copy it intentionally to a non-ignored documentation location and mention why it is permanent evidence.

## 7. Troubleshooting

### Port already in use

Check which process owns the port:

```bash
lsof -i :6010 -P -n
```

Use another port if needed:

```bash
pnpm --prefix web exec storybook dev -p 6011 --host 127.0.0.1
```

Then update URLs from `6010` to `6011`.

### Story iframe returns 404

Make sure Storybook is running from `almanach/web`, which `pnpm --prefix web` handles automatically. Also confirm the story ID has not changed in Storybook's sidebar.

### Screenshots look half-rendered

Increase waits:

```bash
--wait-ms1 1500 --wait-ms2 1500
```

### Need command help

```bash
css-visual-diff help --all
css-visual-diff compare --help
css-visual-diff help compare
```
