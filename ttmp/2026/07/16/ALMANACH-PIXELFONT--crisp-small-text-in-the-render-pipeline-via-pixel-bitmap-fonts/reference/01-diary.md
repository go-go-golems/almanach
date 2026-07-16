---
Title: Diary
Ticket: ALMANACH-PIXELFONT
Status: active
Topics:
    - almanach
    - thermal-printer
    - printing
    - frontend
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - internal/app/renderer.go
    - web/src/almanach-studio.jsx
ExternalSources: []
Summary: "Chronological diary of the ALMANACH-PIXELFONT work: making small text render crisp in the thermal pipeline via pixel/bitmap fonts."
WhatFor: "Record what was tried, what worked, what failed, and what to do next."
WhenToUse: "Read before resuming ALMANACH-PIXELFONT work."
---

# Diary

## Goal

Make small text (~8–11 px) render crisp in the Almanach thermal render pipeline
by replacing anti-aliased vector fonts with a pixel/bitmap font, so sub-pixel
strokes are not dropped by the 1-bit conversion. This is the production
follow-through on the paper-verified small-text finding from `ALMANACH-RASTER-LAB`.

## Step 1: Stand up the ticket, design doc, and task plan

The user asked to open a new docmgr ticket, write a design + implementation doc
and detailed tasks, then work task by task with commits and a detailed diary.
I interpreted "it" as the small-text pixel/bitmap web-font follow-up (the piece I
had offered to start next), not the sibling per-segment-heat follow-up, which is
listed as a non-goal.

Before writing the doc I grounded it in the real pipeline: the render browser is
pinned to device-pixel-ratio 1 (`force-device-scale-factor 1.0` in
`newChromeAllocatorWithViewport`), the paper element is a fixed CSS-pixel width
(384), and the screenshot of `.paper-body` is thresholded by `PngToBitmap`. That
means a bitmap font used at its native pixel size lands 1:1 on the dot grid — the
key enabling fact for the recommended approach.

### Prompt Context

**User prompt (verbatim):** "Open a new docmgr ticket, create a design + implementation doc, as well as detailed tasks, and then work on it, task by task, committing at appropriate intervals, keeping a detailed diary"

**Follow-up (verbatim):** "fontforge is installed, i guess it's because you are in a sandbox? Would it be better for you?"

**Assistant interpretation:** Create the ticket + design/impl doc + tasks for the
small-text pixel-font work, then implement phase by phase with commits and a diary.

**Inferred user intent:** Turn the validated bitmap-font finding into a real,
reviewable production change, tracked and documented like the previous ticket.

**Commit (code):** see changelog (planning only in this step).

### What I did
- Created ticket `ALMANACH-PIXELFONT` on branch `almanach-pixelfont` (based on
  `main`, so it is independent of the open raster-lab PR #4).
- Investigated the frontend: fonts are embedded as base64 woff2 in
  `web/src/fonts-embedded.css`; the studio renders blocks in
  `web/src/almanach-studio.jsx`; paper width defaults to 384.
- Confirmed render-process facts in `internal/app/renderer.go`: DPR pinned to 1,
  Chrome exec allocator flags, screenshot→`PngToBitmap` path.
- Checked tooling: fonttools 4.61 (EBDT/EBLC support), node 22, chromium +
  google-chrome present; fontforge runs headless via `APPIMAGE_EXTRACT_AND_RUN=1`.
- Wrote the design + implementation doc (approaches A/B/C, device-pixel analysis,
  toolchain, integration points, verification, risks, phased plan).
- Added 5 phase tasks to the ticket and the tracker.

### Why
- The recommended approach (embedded-bitmap web font) is self-contained in the
  SPA and needs no render-process env changes, and the DPR-1 capture makes native
  strike sizes align to dots.

### What worked
- fontforge runs with `APPIMAGE_EXTRACT_AND_RUN=1` (the earlier failure was FUSE,
  a sandbox limitation), which unlocks PCF→embedded-bitmap OpenType conversion.

### What didn't work
- Direct `fontforge` failed first with a FUSE/AppImage error; the env-var
  workaround fixed it. Noted so the conversion script sets it.

### What I learned
- On Linux/Chrome, `-webkit-font-smoothing` is a no-op; disabling anti-aliasing
  is a fontconfig concern, which is why Approach B routes through `FONTCONFIG_FILE`.
- The riskiest unknown is whether Blink renders an embedded bitmap strike
  verbatim or blurs it with subpixel positioning; the plan resolves this first in
  a spike before any SPA wiring.

### What warrants a second pair of eyes
- The choice of small-text hook in the SPA (a CSS class bound to an exact px size
  vs. a block option vs. a theme mapping) — decided in Phase 2.

### What should be done in the future
- Execute Phase 0 spike next; gate Approach A on its result.

### Code review instructions
- Start with the design doc §3 (device-pixel constraint) and §4 (approaches).
- Key files: `internal/app/renderer.go` (DPR, allocator), `web/src/fonts-embedded.css`.

### Technical details
- Ticket: `ttmp/2026/07/16/ALMANACH-PIXELFONT--.../`.
- Bundled PCF sources: `ttmp/2026/07/16/ALMANACH-RASTER-LAB--.../scripts/fonts/*.pcf`.

## Step 2: Phase 0 spike — Approach A rejected, Approach B (AA off) adopted

The spike reversed the design's initial recommendation, which is exactly why it
ran first. I converted a bundled PCF (6x10, 6x9) to an OpenType font with an
embedded bitmap strike (fontforge via `APPIMAGE_EXTRACT_AND_RUN=1`), embedded it
in an HTML page via `@font-face`, rendered at 384 px in headless Chrome
(`force-device-scale-factor=1`), and thresholded the screenshot to 1-bit with the
`PngToBitmap` rule.

Chrome did **not** use the embedded bitmap strike. Blink renders the font's
vector outlines and ignores `EBDT`/`EBLC` for normal text; fontforge's
auto-traced outlines were crude, so the custom font rendered *worse* than the
stock font. Approach A is dead.

Then I disabled anti-aliasing via a fontconfig file (`antialias=false`) supplied
through `FONTCONFIG_FILE` and re-rendered. The render contained **0.00% gray
pixels** — pure 1-bit — and a stock hinted font (DejaVu Sans) was crisp and fully
legible down to 8 px. Rendering the SPA's real body font (DM Sans, extracted from
`web/src/fonts-embedded.css`) AA-off showed it crisp to ~10–11 px but degraded at
8–9 px, because DM Sans is lightly hinted compared to DejaVu.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Run the spike to pick the approach before wiring.

**Commit (code):** see changelog.

### What I did
- `scripts/convert_font.py` (fontforge PCF→OTB), produced AlmanachPixel9/10.
- `scripts/fonts-noaa.conf` (fontconfig AA-off), `scripts/spike*.html`, render +
  1-bit checks (`scripts/spike-*.png`).
- Updated the design doc decision and phased plan to Approach B.

### What worked
- fontconfig `antialias=false` via `FONTCONFIG_FILE` makes headless Chrome
  rasterize text monochrome (0% gray), which is the production lever.
- Hinted vector fonts (DejaVu) render crisp 1-bit to 8 px AA-off.

### What didn't work
- Embedded-bitmap web font: Chrome ignores the strike, renders outlines. Rejected.
- `woff2` export blocked locally (no brotli); will ship `woff` (zlib) if a
  bundled small-text font is needed — Chrome decodes both.

### What I learned
- The real fix is not a custom font; it is moving the 1-bit decision from the
  luminance threshold to FreeType's hint-aware monochrome rasterizer by turning
  AA off. Small-text quality then follows the font's hinting, so ≤9 px wants a
  strongly-hinted (or pixel-vector) face.

### What warrants a second pair of eyes
- Whether AA-off regresses large serif display type (Cormorant/EB Garamond) —
  checked end-to-end in Phase 3 on the real page.

### What should be done in the future
- Phase 1: wire AA-off into `newChromeAllocatorWithViewport` via `FONTCONFIG_FILE`.
- Phase 2: small-text CSS class using a strongly-hinted font.

### Technical details
- AA-off conf: `scripts/fonts-noaa.conf`. Spike renders: `spike-1bit.png`
  (AA on, dropped strokes), `spike-noaa-1bit.png` (AA off, crisp),
  `spike-dmsans-noaa-1bit.png` (real DM Sans AA-off).
