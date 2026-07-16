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
    - Path: internal/app/renderer.go
    - Path: repo://internal/app/fontconfig.go
      Note: AA-off fontconfig + renderFontEnv (commit 93abdd3)
    - Path: repo://internal/app/supersample.go
      Note: Supersample downscale + pngToBitmapSupersampled (commit f61ec55)
    - Path: web/src/almanach-studio.jsx
ExternalSources: []
Summary: 'Chronological diary of the ALMANACH-PIXELFONT work: making small text render crisp in the thermal pipeline via pixel/bitmap fonts.'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Record what was tried, what worked, what failed, and what to do next.
WhenToUse: Read before resuming ALMANACH-PIXELFONT work.
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

## Step 3: Phase 1 — disable AA in the render browser (shipped)

Wired the AA-off approach into the production render path. Added
`internal/app/fontconfig.go` with a fontconfig (`antialias=false`, full hinting)
and `renderFontEnv()`, which writes it to a temp file and returns
`FONTCONFIG_FILE=...`; `newChromeAllocatorWithViewport` passes it to the local
Chrome exec allocator via `chromedp.Env`. An env override
`ALMANACH_FONT_ANTIALIAS=1` keeps AA on.

Verified end-to-end with the real `render` command (pointing chrome at
`/usr/bin/google-chrome-stable`, since the default snap chromium is blocked in
this sandbox): the render screenshot went from **2.09% gray (AA on) to 0.00%
gray (AA off)**. Only the headless render browser is affected; the studio UI a
human edits is untouched. `go build`, `golangci-lint`, `go test` all pass.

**Commit (code):** 93abdd3.

### What worked
- `chromedp.Env("FONTCONFIG_FILE=...")` cleanly scopes AA-off to the render
  browser process. 0% gray confirmed in the real pipeline.

### What I learned
- The default chromedp browser resolved to snap chromium, which fails under the
  sandbox (`snap-confine ... cap_dac_override`); `--chrome-path
  /usr/bin/google-chrome-stable` works. Noted for local verification runs.
- A serif *display* font (theme scaffold) renders a little rough AA-off at
  medium size, but it is still 1-bit-legible and the threshold would have
  thinned it anyway; small body text (the goal) is the win. Confirm on the real
  page in Phase 3.

### What warrants a second pair of eyes
- Remote-allocator path (`ChromeWSURL`) cannot receive the env var, so AA-off
  does not apply to a remote browser; documented as a limitation.

### Code review instructions
- `internal/app/fontconfig.go` (`renderFontEnv`, the conf) and the
  `chromedp.Env` call in `newChromeAllocatorWithViewport`.
- Reproduce: `render --layout <yaml> --out o.png --debug-dir d --web-dir web/dist
  --chrome-path /usr/bin/google-chrome-stable`, then check `screenshot.png` has
  ~0% gray; set `ALMANACH_FONT_ANTIALIAS=1` to see it rise.

## Step 4: Phase 3 verification on a real page, Phase 4 paper, and the Phase 2 decision

Rendered the realistic `examples/layouts/03-knowledge-strip.yaml` (word,
phonetics, history, did-you-know, quote — genuine small text) through the real
pipeline both ways. AA-off produced 0.01% gray versus 4.58% with AA on. Cropping
the small-text region and comparing: **AA-off renders every stroke complete and
legible** — the dropped-stroke problem the ticket set out to fix is gone. Then I
printed the AA-off render to the physical K118 (`192.168.0.126`); it rendered and
printed cleanly in 2 segments through the production `print` command.

This end-to-end result also settles Phase 2. The `minimal` theme uses a **serif**
body font (EB Garamond). AA-off, the serif is complete but looks a little rough,
because hairline serifs are inherently hard at 1-bit low resolution — and AA-on
would have printed *worse*, since the threshold drops those light-gray hairlines.
A hinted **sans** or pixel font would render small text more cleanly, but swapping
a theme's font is an **aesthetic/product decision**, not a bug fix, so I did not
change theme fonts unilaterally. The mechanism to get crisper small text already
exists: choose a sans theme, or (future) add a pixel-vector small-text face.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Verify the fix on a real page and on paper; treat
the serif-vs-hinted-sans small-text choice as a user aesthetic decision.

**Commit (code):** none (verification + docs); render/print code shipped in Step 3.

### What worked
- Real page AA-off: 0.01% gray, all strokes present, legible; printed to paper via
  the production pipeline (2 segments, printer_ok).

### What I learned
- The core complaint (small strokes missing) is fixed by Phase 1 for real content;
  the residual is serif roughness at low res, which is a font-choice matter.

### Decision (Phase 2)
- Do **not** unilaterally change theme fonts. AA-off delivers complete strokes;
  crisper small serif text is an aesthetic call. Recommend to the user: for
  small-text-heavy pages, use a sans theme; optionally add a bundled pixel-vector
  small-text face later if they want a deliberate pixel look.

### What should be done in the future
- If the user wants it: bundle a hinted pixel-vector face and a `.pixel-text`
  class / theme option for the smallest captions.

### Technical details
- Compared crops: `/tmp/ks/screenshot-off-crop2x.png` vs `-on-crop2x.png`.
- Printed: `print --layout examples/layouts/03-knowledge-strip.yaml
  --printer-ip 192.168.0.126 --chrome-path /usr/bin/google-chrome-stable`.

## Step 5: Supersampling — the shipped fix for small serif italics

On the printed AA-off strip the user reported the smaller text, "especially the
italic," was still missing pixels and asked whether it was "rasterized as an
image." It is not: all text is real font glyphs (EB Garamond has 14 real italic
faces embedded — the italic is genuine, not synthetic) screenshotted and
thresholded. The residual is that a delicate serif *italic* at ~12 px is the
worst case for 1-bit, and italic hinting is weak.

I compared two fixes on rendered paper samples: a hinted sans for small text
(crisp but changes the serif look) versus **supersampling**. The user said "both,
you decide," so I chose supersampling because it fixes the problem *without*
changing any theme font: render the page at 3× with AA on, box-average down to
the target, then threshold. On the real page the small italic quote went from
broken to complete and legible while staying serif; printed to the K118.

### Prompt Context

**User prompt (verbatim):** "it's better but i think the smaller text is still maybe missing out on some pixels, especially the italic one (which i suspect is still rasterized as an image?"

**User prompt (verbatim):** "I guess both? you deide what's best"

**Assistant interpretation:** Fix the residual small-italic pixel loss; decide the
best approach, keeping the serif aesthetic if possible.

**Commit (code):** f61ec55.

### What I did
- `internal/app/supersample.go`: fast integer box-average downscale (indexes a
  packed RGBA buffer via `draw.Draw` then Pix, ~2–3× faster than `image.At`) +
  `pngToBitmapSupersampled`; tests in `supersample_test.go`.
- `renderer.go`: `RenderOptions.SupersampleScale` (default 3),
  `chromedp.EmulateScale` + device-scale-factor set to the factor, AA kept on when
  supersampling (AA-off retained for scale 1).
- `--supersample` flag on render/print + layout `render.supersample`.

### What worked
- 3× render: screenshot 1152 px → box-average → 384 px bitmap; small serif italic
  complete and legible; printed to paper. Speeds: 3× ≈ 2.5 s, 2× ≈ 1.9 s.

### What didn't work (then fixed)
- First naive downscale used `image.At().RGBA()` per sample → 17 s render.
  Rewrote to draw into a packed RGBA buffer and index `Pix` with fixed-point
  luminance → ~2.5 s at 3×.

### What was tricky to build
- Making the 3× actually take: `chromedp.EmulateViewport` defaults device scale to
  1 and would override the launch flag, so the scale must be passed via
  `chromedp.EmulateScale`; verified by the 1152-px screenshot.

### What warrants a second pair of eyes
- The box-average uses an integer scale and drops a partial last row/column if the
  screenshot isn't an exact multiple of the scale (paper widths are, so fine).

### What should be done in the future
- Optional: a hinted/pixel small-text face for the very smallest captions.

### Technical details
- Default scale `defaultSupersampleScale=3`; tunable `--supersample` (1 disables,
  using the AA-off path). Verified: `render ... --supersample 2|3 --format bitmap`.

## Step 6: Font/size/technique matrix — hinted font at 1x AA-off wins

The user said the result "is not great yet" and asked for a matrix over fonts,
sizes, and techniques. I built `scripts/02-font-matrix.py`: it renders the same
small text across 6 fonts (EB Garamond, DejaVu Serif, DM Sans, DejaVu Sans, Noto
Sans, JetBrains Mono) at sizes 8–13 (plus an 11px italic), through 5 techniques
(1x AA-off, 2x/3x/4x supersample, 3x threshold 160), each downsampled to the
384px 1-bit target exactly like the pipeline. Rendered all five and printed the
1x-AA-off and 3x sheets to the K118.

The matrix reframed the whole problem: **the font's hinting quality dominates,
and for a well-hinted font 1x AA-off is *sharper* than supersampling at small
sizes**, because bytecode hinting is designed for exact-pixel rendering and
supersampling bypasses it. Concretely, at 1x AA-off:

- **JetBrains Mono** (already in the SPA): crisp and legible at **8 px**.
- **DejaVu Sans** (hinted): crisp to ~9 px.
- **DejaVu Serif** (hinted): legible to ~9–10 px and keeps a serif look.
- **EB Garamond, DM Sans, Noto Sans**: rough/broken at 8–9 px in every technique
  (light hinting) — supersampling helps them a little at medium sizes only.

So the shipped supersampling default helps a *specific delicate font* (EB
Garamond) but is not the best general answer: the best small text comes from
choosing a **hinted font** and rendering **1x AA-off**.

### Prompt Context

**User prompt (verbatim):** "ok, now do a matrix over different fonts and sizes and techniques and such. It's not great yet."

**Assistant interpretation:** Systematically compare fonts x sizes x techniques
to find the actually-legible small-text recipe.

**Commit (code):** see changelog (harness only; no production change yet — the
font direction is a user aesthetic decision).

### What I did
- `scripts/02-font-matrix.py` (grid HTML + supersample/threshold render +
  print); result sheets `scripts/matrix/{1x-aaoff,2x,3x,4x,3x-T160}.png`.
- Fixed a geometry bug (window-size is CSS px; must be the paper width, not
  width×scale) and PIL's decompression-bomb guard for our own screenshots.

### Findings / recommendation
- Best small text = hinted font @ 1x AA-off. Options: DejaVu Serif (serif look),
  DejaVu Sans, or JetBrains Mono (crispest, to 8px).
- Supersampling is a fallback for keeping a delicate font (EB Garamond) as-is.
- Decision is the user's: which font direction for body/small text.

### What should be done next
- On the user's font choice: switch theme body/small-text font to a hinted face
  (and likely default supersample back to 1 for those, since 1x is sharper).

## Step 7: Labeled sheets, heat/speed sweep, x-height-matched fonts

Two user directives arrived: label the sheets (couldn't tell technique apart) and
experiment with printer heat/speed; plus paper feedback that the italic is only
legible in DejaVu Sans and the first (1x AA-off) sheet was better. I baked a
technique+density+speed header into every sheet, added richer italic rows, and
added `set_heat` + `--densities/--speeds` to sweep printer heat. I printed the
1x-AA-off matrix at d22/s80, d30/s62, d38/s37.

The user then reported **d38/s37 (dark, slow) was the best** heat, and made a
typographic point: **EB Garamond renders ~2–3 px smaller than DejaVu at the same
nominal size** (Garamond 13 ≈ DejaVu 10–11), so it must be set larger. I rebuilt
the sheet with x-height-matched sizes — EB Garamond 13–16, DejaVu Serif/Sans
10–13 — and printed it at d38/s37, d34/s37, and d39/s25 (max heat) to confirm the
heat around the winner.

### Prompt Context

**User prompt (verbatim):** "how do I know which technique is which, but really, t 11px the italic is not legible, except maybe dejavu sans. The first group you printed was better"

**User prompt (verbatim):** "in general the default should be bigger text, especially for serif"

**User prompt (verbatim):** "also experiment with printer heat and speed"

**User prompt (verbatim):** "print depth 38 was the best so far (the last one). do more experiments.  use deja serif, or use bigger sizes for EB garamond, because it is much smaller at the same given size. (garmong 13 seems to be about dejavu 10 or 11"

**Assistant interpretation:** Label sheets; sweep heat/speed; compare
x-height-matched EB Garamond (bigger) vs DejaVu Serif/Sans; converge on
density ~38 + slow speed.

**Commit (code):** 07c931a + this step.

### Findings (paper)
- **Heat:** density ~38 + slow speed (~37) is the best text heat so far.
- **X-height:** EB Garamond ~15–16 px ≈ DejaVu Serif ~11–12 px. To keep EB
  Garamond, set it ~+3 px larger; or use DejaVu Serif at the smaller size.
- **Italic:** DejaVu Sans best; DejaVu Serif good at ≥12; EB Garamond needs ≥16.

### What should be done next
- Confirm the exact heat from the 3 x-height sheets (d38/s37 vs d34/s37 vs
  d39/s25), then implement production defaults: bigger serif sizes (or DejaVu
  Serif), hinted font for small/italic, default 1x AA-off, chosen density/speed.

### Technical details
- `02-font-matrix.py` now uses per-font `(sizes, italic_sizes)`; sheet header
  baked; `--densities/--speeds` sweep with `set_heat`. Sheets in `scripts/mtmp/`
  and `scripts/matrix/`.

## Step 11: Wrap-up — flip default to 1x AA-off; recipe locked

Density × speed grid (d{24,32,38} × s{37,80}, compact heat card) confirmed on
paper: **density 38 is best and speed is not distinguishable** (37 vs 80), so the
faster default speed stays. The weight sweep confirmed **bold is the biggest
remaining lever, especially for italics**. With the matrix showing 1x AA-off is
as crisp as / crisper than supersampling for hinted fonts (and 3x faster) and the
user preferring the 1x sheets, I flipped the shipped default:
`defaultSupersampleScale` 3 → 1. Verified end-to-end (render is 384px, 0.01% gray,
~0.95s vs ~2.5s) and printed the real knowledge-strip at density 38.

### Prompt Context

**User prompt (verbatim):** "can you play with speed and heat and stuff? or did you already? or print more experiments ?"

**User prompt (verbatim):** "ok i think the last one is the best. i can' really tell a diff between 37 and 80. Let's wrap it up here, except you have better ideas"

**Assistant interpretation:** Do a clean density×speed grid, then wrap up by
baking the proven finding into the default.

**Commit (code):** see changelog.

### What I did
- `internal/app/supersample.go`: `defaultSupersampleScale` 3 → 1 (1x AA-off is
  the default; supersample opt-in via `--supersample`). Updated design doc.
- Verified default render (1x AA-off, ~1s) and printed at density 38.

### Locked recipe
- Render: **1x AA-off** (default); supersample opt-in for delicate fonts.
- Heat: **density ~38**, speed either (37/80 indistinguishable).
- Recommended theme tuning (aesthetic, maintainer's call): bigger text esp
  serif; hinted font (DejaVu Serif/Sans) or EB Garamond ~16–17; bold for small/
  italic text.

### What warrants a second pair of eyes
- Flipping the default to 1x is best *with* hinted/bigger fonts; with the current
  small EB Garamond, supersampling helped its italics — hence the theme tuning is
  recommended alongside. Documented; not forced.

### What should be done in the future (optional theme change)
- Bump default text sizes / bodyScale; switch small/italic text to a hinted or
  bold face; set a print density default of ~38 (via the raster-lab `--density`).
