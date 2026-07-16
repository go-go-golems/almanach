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
