---
Title: 'Thermal rasterization system: analysis, design, and implementation guide'
Ticket: ALMANACH-RASTER-LAB
Status: active
Topics:
    - almanach
    - thermal-printer
    - printing
    - go
    - firmware
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: firmware/atoms3r/main/printer_drv.c
      Note: 'GS v 0 raster + ESC ## density/speed/graphics commands (lever 3)'
    - Path: firmware/atoms3r/main/web_server.c
      Note: /api/print/bitmap and density/speed/status endpoints
    - Path: internal/app/bitmap.go
      Note: Current fixed-threshold rasterizer (lever 2 seam)
    - Path: internal/app/cmd_print.go
    - Path: internal/app/printer.go
      Note: Segmented bitmap POST + feed baking + transport safety
    - Path: internal/app/renderer.go
ExternalSources: []
Summary: End-to-end intern guide to how Almanach turns an HTML layout into thermal-printer dots, why photos and mixed pages currently print hard to read, the three tuning levers (host tone curve, dithering algorithm, printer heat/density), the browser dithering lab, the firmware ESC/POS API, and the lab-and-paper co-evolution workflow used to tune all of it.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Onboard a new contributor to the thermal rasterization pipeline and the experimental tuning workflow.
WhenToUse: Read before touching rasterization, the dithering lab, or printer density/speed tuning.
---


# Thermal rasterization system: analysis, design, and implementation guide

## 0. How to read this document

This guide is written for a **new intern** who has never seen the Almanach
codebase and is not (yet) a halftoning expert. It has three jobs:

1. **Explain the whole system** — from an HTML layout in a browser to physical
   dots burned onto thermal paper — so you understand every stage a pixel passes
   through and where it can go wrong.
2. **Explain the problem** — why photographs and mixed text+image pages
   currently print "hard to read," in precise terms, and what the *levers* are
   for fixing it.
3. **Explain the working method** — the **lab-and-paper co-evolution loop**: how
   we tune rasterization in a browser lab, print candidates on real paper,
   photograph the results, feed them back, and only then merge the winning
   settings into the production Go service.

You do **not** need to read it all at once. Sections 1–3 are the mental model.
Section 4 is the algorithm theory. Sections 5–6 are the tools. Section 7 is the
workflow you'll actually run day to day. Sections 8–10 are the roadmap, API
reference, and glossary.

A companion **research/design doc** already exists from an earlier pass and
covers the algorithm menu in even more depth:
`ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/design-doc/01-adaptive-thermal-rasterization-analysis-design-and-implementation-guide.md`.
This guide supersedes it as the *working* document because it adds the
lab/paper workflow and the firmware **heat/density** lever, which the earlier
doc did not treat as a first-class tuning axis.

---

## 1. Executive summary

Almanach renders a React/HTML "page" with headless Chrome, screenshots it to a
PNG, converts that PNG to a **1-bit monochrome bitmap using a single fixed
grayscale threshold**, and POSTs the packed bytes to an ESP32 (AtomS3R) that
drives a K118 thermal print head over UART using ESC/POS commands.

That pipeline is correct and reliable, but the **image-quality stage is the
weakest link**: a fixed threshold at gray value 128 destroys every midtone. A
1-bit device cannot print gray, so continuous-tone content (photos, shaded
illustrations, anti-aliased text edges) must be *simulated* with patterns of
black and white dots — this is **dithering / halftoning** — and the result must
be **calibrated to the physical medium**, because thermal paper darkens
(each dot spreads: "dot gain") and the print head has an adjustable **heat /
density** setting that shifts the whole image darker or lighter.

Today none of that exists in the Go path. `internal/app/bitmap.go` is the entire
rasterizer and it is just `if gray < threshold { black }`. There is no gamma
correction, no dithering, no edge preservation, and the printer's density knob
is left at a fixed default and never coordinated with the image.

There are exactly **three independent levers** we can pull, and "hard to read"
touches all three:

| # | Lever | Where it lives | What it controls |
|---|-------|----------------|------------------|
| 1 | **Tone curve** (gamma / brightness / contrast / sharpen) | Host, before dithering | Pre-compensates dot gain; sets perceived lightness and local contrast |
| 2 | **1-bit conversion** (threshold vs. ordered vs. error-diffusion vs. edge-hybrid) | Host, the rasterizer itself | How gray is simulated with dots; sharpness vs. smoothness vs. pattern |
| 3 | **Printer heat / density** (and speed) | Firmware, ESC/POS | Global darkness of every dot; interacts strongly with lever 2 |

The **central untested interaction** is lever 2 × lever 3 (dither algorithm ×
density). That interaction is what this ticket exists to explore, using the
lab-and-paper loop, before we commit code.

---

## 1b. Paper-verified calibration results (living summary)

These are the recipes confirmed on the real K118 during this ticket (see the
diary for the run-by-run evidence). Update as tuning continues.

- **Photo / continuous-tone:** **Atkinson dither, gamma 0.8, density 20,
  speed 80.** Beats threshold (which collapses midtones), Floyd (slightly dark),
  and Bayer8 (visible grid). Gamma 0.8 (`v'=v^γ`, γ<1 lightens) opens shadow
  detail; the ramp crush confirmed the lighten direction on paper.
- **Small text:** **use a bitmap font (6x9 or 6x10 sweet spot).** Anti-aliased
  vector text + threshold drops sub-pixel strokes at 8–9 px; bitmap glyphs are
  pure 1-bit with no AA, so nothing drops — 6x9 bitmap out-reads 11 px AA vector.
  Host-side algorithm/threshold tweaks help less than switching the font.
- **Text heat:** **density ~28–32** (hotter than the photo); above ~36 the
  smallest glyphs bleed/close counters. **Slower speed** (e.g. 37) reduces the
  within-line grey variation caused by **print-head power droop** (many
  simultaneous dots sag the rail).
- **Mixed page:** text and photo want *different* heat, so **set density/speed
  per segment** — text hot+slower, photo cool (density 20) + speed 80. The
  existing segmented print path (`printer.go`) can set density between segment
  POSTs, so this needs **no firmware change**.
- **Printer facts:** "heat" = the density register (`ESC ## STDP`, 0..39); there
  is no separate contrast register. Speed (`ESC ## STSP`) is the second heat
  lever. The firmware prints a "Print depth: N level" banner on each density
  change.

## 2. System overview

### 2.1 The end-to-end pipeline

```
  ┌──────────────────────────────────────────────────────────────────────┐
  │ HOST (Go: almanach-render-service)                                     │
  │                                                                        │
  │  layout.yaml / .json / .zip bundle                                     │
  │        │                                                               │
  │        ▼                                                               │
  │  render_oneshot.go   ── parse layout + `render:` options + templates   │
  │        │                                                               │
  │        ▼                                                               │
  │  renderer.go ─ headless Chrome renders React SPA at /almanach          │
  │        │        (EmulateViewport → load layout → wait fonts/frames)    │
  │        ▼                                                               │
  │  chromedp.Screenshot(selector)  ── one flat PNG of `.paper-body`       │
  │        │                                                               │
  │        ▼                                                               │
  │  bitmap.go  PngToBitmap(png, threshold)   ◄── THE RASTERIZER (lever 2) │
  │        │        luminance → threshold → MSB-first packed 1-bit          │
  │        ▼                                                               │
  │  printer.go sendBitmapToPrinter()                                      │
  │        │        segment if > ~36 KiB, bake/append feed rows,           │
  │        │        POST octet-stream + X-Width/X-Height/X-Feed             │
  └────────┼───────────────────────────────────────────────────────────────┘
           │  HTTP  POST /api/print/bitmap
           ▼
  ┌──────────────────────────────────────────────────────────────────────┐
  │ FIRMWARE (ESP32 AtomS3R: web_server.c + printer_drv.c)                  │
  │                                                                        │
  │  web_server.c  validate width%8==0, body size, buffer whole body       │
  │        │                                                               │
  │        ▼                                                               │
  │  printer_drv_print_bitmap(w,h,body)                                    │
  │        │   GS v 0 (1D 76 30) raster header + packed bytes over UART    │
  │        ▼                                                               │
  │  K118 thermal head  ── density (lever 3) + speed set separately        │
  │        │   ESC ## STDP n (density 0..39), ESC ## STSP n (speed)        │
  │        ▼                                                               │
  │   ▓▓ physical dots on paper ▓▓                                         │
  └──────────────────────────────────────────────────────────────────────┘
```

### 2.2 Why the architecture matters for image quality

Two structural facts dominate everything downstream:

- **The page is flattened to a single PNG before rasterization**
  (`renderer.go` captures `chromedp.Screenshot(selector)` and hands the buffer
  to `PngToBitmap`). After that point, **there is no way to tell which pixels
  came from text and which came from a photo.** This is why a single global
  rasterization mode cannot be simultaneously ideal for crisp text and for
  toned photos. (See §4.7 and §8 for the segmented-printing escape hatch.)
- **The firmware contract is a packed 1-bit bitmap.** Width must be a multiple
  of 8, bits are MSB-first, and the host may split tall bitmaps into vertical
  segments. Any change to the rasterizer must keep this byte format identical
  (see §6.2 and §9.1). We change *which pixels are black*, never the wire format.

---

## 3. The problem, stated precisely

### 3.1 What "hard to read" actually is

A grayscale pixel has 256 levels. The K118 has **2** (dot / no dot). The current
code maps all 256 → 2 with one cut at 128:

```
out(x,y) = luminance(x,y) < threshold   // threshold defaults to 128
```

Consequences, each of which someone has described as "hard to read":

- **Midtone collapse.** A face, a sky, or gray fur is mostly values 90–180.
  With a hard cut those regions become either a solid black blob or blank white,
  so all internal detail vanishes.
- **Dot gain darkening.** Even where the threshold is "right" on screen, each
  black dot physically spreads on thermal paper (~1.2×), so printed output is
  darker than the preview. Dense regions fill in and become muddy.
- **Edge loss.** Thin high-value features — cat whiskers, hairline strokes,
  1px rules — sit above the threshold and disappear entirely.
- **Text vs. photo conflict.** Text wants a hard threshold at high density
  (crisp, dark glyphs). Photos want dithering at lower density (tonal, not
  muddy). One global setting cannot serve both, so mixed pages compromise both.

### 3.2 The three levers, restated as knobs you will actually turn

**Lever 1 — Tone curve (host, pre-dither).** Reshape the 256-level grayscale
*before* it hits the 1-bit conversion:

- **Gamma** bends midtones. Defined here as an exponent applied to normalized
  gray `v ∈ [0,1]`: `v' = v^γ`. `γ < 1` **lightens** midtones; `γ > 1`
  **darkens** them. ⚠️ *Convention warning:* the older parc note says "gamma
  1.8 makes midtones lighter," but `v^1.8` mathematically *darkens* `v<1`.
  The two statements disagree because "display gamma" and "encoding gamma" are
  reciprocals. **This guide fixes the convention as `v' = v^γ` and lets the
  paper decide the direction** — do not trust the prose, trust the printed ramp
  (see §7.4). The lab's current default is `γ = 0.90` (slight lighten).
- **Brightness** adds/subtracts a constant. Lab default `+24` (of 255).
- **Contrast** scales around 0.5. Lab default `0.82×` (slightly *reduced*
  contrast, which protects both shadow and highlight detail on paper).
- **Unsharp / sharpen** boosts local edge contrast before dithering so fine
  detail survives; not yet in the lab, candidate to add.

**Lever 2 — The 1-bit conversion (host).** The algorithm menu (full theory in
§4): fixed threshold, adaptive threshold, ordered (Bayer / blue-noise),
error diffusion (Floyd–Steinberg, Atkinson, Stucki, Burkes, Sierra family),
and edge-hybrid (tone layer OR restrained edge layer).

**Lever 3 — Printer heat / density (firmware).** The K118 has an adjustable
strobe energy exposed as **density 0..39** (ESC/POS `ESC ## STDP n`) and a
**speed** setting. Higher density = darker every dot = more dot gain. This is a
*global* darkness control completely orthogonal to the dither pattern, and it is
currently pinned at a fixed default and never tuned against the image. It is the
most underused lever and the prime suspect for "hard to read."

### 3.3 The one experiment that matters most

Because lever 2 and lever 3 both affect darkness but in different ways (pattern
vs. global energy), the key unknown is their **interaction**. Atkinson dithering
(prints light, 75% error diffused) at a *higher* density can beat Floyd–Steinberg
(prints accurate/dark) at a lower density — or not. You cannot predict this from
a screen preview; dot gain only exists on paper. So the highest-value experiment
is a **density × algorithm grid** printed on real paper (see §7.3).

---

## 4. Rasterization and halftoning theory

This section is the reference for the algorithms. Each has a one-line "use when."

### 4.1 Grayscale conversion (luminance)

Before any 1-bit decision, RGB collapses to a single luminance value. Both the
Go code and the lab use ITU-R BT.601 weights:

```
gray = 0.299·R + 0.587·G + 0.114·B      // per channel in [0,255]
```

Keep this as the baseline. (A future option is BT.709 weights
`0.2126/0.7152/0.0722`, but the difference is minor for our content.)

### 4.2 Fixed threshold

```
out(x,y) = gray(x,y) < T          // T in [0,255]
```

**Use when:** text, QR codes, icons, already-high-contrast line art.
**Avoid for:** photographs (midtones collapse).

### 4.3 Adaptive (local) threshold

Compute the threshold from a neighborhood window around each pixel instead of a
global constant. Variants: local mean, Gaussian mean, Niblack, Sauvola,
Wolf–Jolion. Sauvola-like form:

```
for each pixel p:
    m = mean(window around p)
    s = stddev(window around p)
    T = m · (1 + k·(s/R − 1))      // k≈0.34, R≈128 typical
    out(p) = gray(p) < T
```

**Use when:** scans, ink/pencil drawings, uneven lighting.
**Risk:** amplifies paper texture and noise; not a general photo solution.

### 4.4 Ordered dithering (Bayer, blue-noise)

Tile a fixed threshold matrix across the image; compare each pixel to the matrix
value at its position. No error propagation → fully parallel, deterministic.

```
Bayer 4×4 (values 0..15, scale to 0..255):     Bayer 8×8 exists too (0..63)
   0  8  2 10
  12  4 14  6
   3 11  1  9
  15  7 13  5

out(x,y) = gray(x,y) < scale(matrix[y%N][x%N])
```

**Bayer** → recognizable cross-hatch grid; cheap; retro. **Blue-noise** uses a
precomputed mask whose energy is pushed to high frequencies, so it looks like
pleasant film grain with no grid and no clumping. The lab currently fakes
blue-noise with a hash function (`hashNoise`) — a **true void-and-cluster
64×64 mask is a candidate improvement**.

**Use when:** deterministic texture, avoiding error-diffusion "worms," photos at
higher resolution.

### 4.5 Error diffusion (the workhorse family)

Quantize one pixel, then push its quantization error to not-yet-processed
neighbors. Different kernels spread the error differently. Generic form:

```
work = float copy of gray
for y in rows:
  for x in cols:
    old = work[y][x]
    new = (old < T) ? 0 : 255
    out[y][x] = (new == 0)          // black where we rounded down
    err = old − new
    for (dx,dy,wgt) in kernel.taps:
        work[y+dy][x+dx] += err · wgt / kernel.div
```

Kernels (taps as `[dx, dy, weight]`, divisor after):

```
Floyd–Steinberg  /16 :  (1,0,7) (-1,1,3) (0,1,5) (1,1,1)
Atkinson         /8  :  (1,0,1) (2,0,1) (-1,1,1) (0,1,1) (1,1,1) (0,2,1)   ← only 6/8 of error diffused
Stucki           /42 :  (1,0,8)(2,0,4)(-2,1,2)(-1,1,4)(0,1,8)(1,1,4)(2,1,2)(-2,2,1)(-1,2,2)(0,2,4)(1,2,2)(2,2,1)
Burkes           /32 :  (1,0,8)(2,0,4)(-2,1,2)(-1,1,4)(0,1,8)(1,1,4)(2,1,2)
Sierra-2         /16 :  (1,0,4)(2,0,3)(-2,1,1)(-1,1,2)(0,1,3)(1,1,2)(2,1,1)
Sierra-lite      /4  :  (1,0,2)(-1,1,1)(0,1,1)
```

- **Floyd–Steinberg**: accurate tone, but "worm" artifacts and can print muddy.
- **Atkinson**: diffuses only **6/8** of the error, so it deliberately loses a
  little density → **prints lighter**, which suits thermal paper. Historically
  the Mac dithering look. **Best default candidate for cat portraits.**
- **Stucki/Burkes/Sierra**: larger kernels, smoother gradients, slightly softer
  edges, more compute (all still trivial at 384px width).
- **Serpentine scanning** (alternate row direction, mirror kernel) reduces
  directional artifacts; add *after* baselines are comparable, not before.

### 4.6 Edge-aware hybrid

Two layers combined so outlines survive even when the tone layer would drop them:

```
gray  = luminance(image)
toneG = sharpen(applyTone(gray, γ, brightness, contrast))
tone  = atkinson(toneG, T)                    // tonal simulation
edges = sobelMagnitude(gray)
edgeMask = (edges > edgeT) AND (gray < edgeMaxGray)   // meaningful, dark-ish edges only
edgeMask = suppressIsolatedDots(edgeMask)     // kill single-pixel speckle
final = tone OR edgeMask
final = capLocalBlackDensity(final, window=16, maxDensity≈0.45)   // optional, photos only
```

**Use when:** the user asks "don't lose the whiskers/eyes/outlines." The edge
layer must be **restrained** (contrast-gated, speckle-suppressed) or it turns
fur noise into black confetti. The lab implements a simple version
(`edgeHybrid` = Atkinson OR raw Sobel); the density-cap and speckle-suppression
are candidates to add.

### 4.7 The block-awareness ceiling

All of the above operate on the **already-flattened page PNG**. That means
whole-page error diffusion will slightly fuzz text edges, and a density that
suits the photo may make text too light. The clean fixes are structural
(§8): browser-emitted block masks, or **segmented printing** where text
segments print thresholded at high density and image segments print dithered at
lower density. Until then, for mixed pages we look for the *least-bad global
compromise* and gather evidence that justifies the structural work.

---

## 5. The browser dithering lab

### 5.1 What it is and why it exists

A **self-contained, offline, single-purpose web app** for tuning rasterization
interactively — no Go rebuild, no firmware flash, instant visual feedback, and a
one-click "print this exact result to the real printer" button. It is the
fast half of the co-evolution loop. It lives in the *older* ticket and we will
evolve it from this one:

```
ttmp/2026/05/10/ALMANACH-RASTERIZATION--.../various/raster-dither-lab.html   (UI shell + macOS-1 retro styling)
ttmp/2026/05/10/ALMANACH-RASTERIZATION--.../various/raster-dither-lab.js     (all algorithms + controls + print path)
ttmp/2026/05/10/ALMANACH-RASTERIZATION--.../scripts/01-serve-raster-dither-lab.py  (localhost server + printer proxy)
```

### 5.2 Lab architecture

```
  ┌───────────────────────────── raster-dither-lab.html ─────────────────────────────┐
  │  <select #algorithm>  + sliders: maxWidth threshold brightness contrast gamma      │
  │                                  strength edgeThreshold  + printerUrl + feedLines   │
  │  <canvas source>   <canvas output(1-bit preview)>   <stats panel>                   │
  └───────────────┬───────────────────────────────────────────────────────────────────┘
                  │ every slider 'input' → rasterize()
                  ▼
  ┌───────────────────────────── raster-dither-lab.js ───────────────────────────────┐
  │  controls()   → read UI into {algorithm, threshold, brightness, contrast,          │
  │                  gamma, strength, edgeThreshold, feedLines}                         │
  │  prepareGray()→ downscale to maxWidth, luminance(), tone() per pixel                │
  │  rasterize()  → switch(algorithm): threshold | adaptiveMean | ordered(bayer4/8)    │
  │                  | blueNoise | diffuse(floyd/atkinson/stucki/burkes/sierra2/lite)   │
  │                  | edgeHybrid                                                       │
  │  stats        → out dims, packed printer bytes, black density %, 90 KiB margin      │
  │  PRINT PATH   → makePrintCanvas() draws a settings header + output,                 │
  │                  packCanvasWithFeed() thresholds+packs MSB-first, POST proxy        │
  └───────────────────────────────────────────────────────────────────────────────────┘
```

Key function map (in `raster-dither-lab.js`):

| Function | Role |
|---|---|
| `controls()` | Read all sliders/select into a params object |
| `luminance(r,g,b)` | BT.601 gray |
| `tone(v,c)` | Apply gamma → contrast → brightness to normalized `v` |
| `prepareGray()` | Downscale to `maxWidth`, produce toned gray array |
| `threshold / ordered / blueNoise / adaptiveMean` | Non-diffusion modes |
| `DIFFUSION` table + `diffuse()` | All 6 error-diffusion kernels (directly portable to Go) |
| `sobelEdges()` / `edgeHybrid()` | Edge layer + Atkinson OR edge |
| `settingsLines()` / `makePrintCanvas()` | Print a labeled header above the image |
| `packCanvasWithFeed()` | MSB-first pack + trailing feed rows (mirrors `printer.go`) |

### 5.3 Current control defaults (as shipped in the lab today)

```
algorithm    = threshold        maxWidth = 384 px       feedLines = 3
threshold    = 128              brightness = +24        printerUrl = /api/print/bitmap
contrast     = 0.82×            gamma      = 0.90        edgeThreshold = 32
strength     = 1.00×
```

These are a starting point, **not** calibrated to paper. Re-deriving them
against real prints is literally the point of this ticket.

### 5.4 Running the lab and printing from it

```bash
# From the older ticket's scripts/ dir:
python 01-serve-raster-dither-lab.py --port 18301 --printer http://192.168.1.242
# Open:
#   http://localhost:18301/raster-dither-lab.html
# Keep the Printer URL field as /api/print/bitmap so the browser posts
# same-origin to the Python proxy, which forwards to the printer (avoids CORS).
```

The proxy exists because a `file://` page cannot POST custom `X-Width`/`X-Height`
headers cross-origin to the printer IP (browser CORS + private-network rules).
Serving from localhost + proxying to the printer is the reliable pattern. A
future improvement is to move this proxy into the Go `setup` server so all local
tooling lives in one binary (see §8).

### 5.5 Known lab gaps (candidate improvements this ticket will make)

- **True blue-noise mask** instead of the `hashNoise` approximation.
- **Thermal-calibrated tone defaults** (γ/brightness/contrast derived from a
  printed gray ramp, not guessed).
- **Density/speed controls** in the lab UI that hit `/api/printer/density` and
  `/api/printer/speed` so the lab can drive lever 3, not just levers 1–2.
- **Comparison-sheet export**: render one source image through N (mode, γ,
  density) combinations, each labeled, as one printable bundle (see §7.3).
- **Preset save/load** as JSON so a winning combination is reproducible.
- **Local black-density cap** and **speckle suppression** for edge-hybrid.

---

## 6. Host (Go) code walkthrough

### 6.1 The rasterizer — `internal/app/bitmap.go`

This *is* the production rasterizer, in full, today:

```go
// PngToBitmap: decode PNG → imageToBitmap(threshold)
// imageToBitmap:
//   gray = 0.299R + 0.587G + 0.114B      (BT.601, RGBA()/256)
//   if gray < threshold: set MSB-first bit
//   width padded up to a multiple of 8
```

Everything in §4 beyond "fixed threshold" is missing here. The clean way to add
it (per the older design doc's Phase 1) is a small internal `raster` package with
a `Mode` enum and an `Options` struct, keeping `threshold` mode **byte-identical**
to today so the change is reversible:

```go
type Mode string
const ( ModeThreshold Mode = "threshold"; ModeAtkinson Mode = "atkinson"; /* … */ )

type Options struct {
    Mode                                   Mode
    Threshold                              uint8
    Gamma, Brightness, Contrast, Sharpen   float64
    EdgePreserve                           bool
    EdgeThreshold, EdgeStrength            float64
    MaxBlackDensity                        float64
    DensityWindow                          int
    // Lever 3, new: coordinate printer heat with the raster choice
    Density                                *int    // 0..39, nil = leave firmware default
    Speed                                  *int
}
func Rasterize(img image.Image, opts Options) (*Bitmap, error)
```

The `DIFFUSION` table in the lab JS ports directly to a Go `[]DiffusionTap`
table — the kernels are identical.

### 6.2 The packing invariant — must never change

```go
paddedWidth := ((width + 7) / 8) * 8
bytesPerRow := paddedWidth / 8
data := make([]byte, bytesPerRow*height)
if black[y][x] {
    data[y*bytesPerRow + x/8] |= byte(0x80) >> (x % 8)   // MSB-first
}
```

This is byte-compatible with the firmware `GS v 0` reader. A rasterization change
alters *which bits are set*, never this layout.

### 6.3 Render options seam — `renderer.go`, `render_oneshot.go`, `cmd_print.go`

- `RenderOptions` (renderer.go) carries `Threshold uint8` and friends. Add
  `RasterMode`, `Gamma`, `Brightness`, `Contrast`, `Density`, `Speed` here.
- Layouts already support a top-level `render:` block that
  `render_oneshot.go` extracts (`intFromRenderOptions`/`stringFromRenderOptions`).
  This is the natural home for per-layout raster settings:

```yaml
render:
  selector: .paper-body
  rasterMode: atkinson
  gamma: 1.4
  brightness: 0.10
  contrast: 0.85
  density: 24            # lever 3, per-job
layout:
  blocks:
    - type: image
      data: { src: images/cat.png, thermalTone: light }
```

- `cmd_print.go` exposes CLI flags (`--threshold`, `--selector`, …); add
  `--raster-mode`, `--gamma`, `--brightness`, `--contrast`, `--density`,
  `--speed`. Keep `--threshold` for compatibility; default mode stays
  `threshold` until paper says otherwise.

### 6.4 Transport — `internal/app/printer.go`

- `sendBitmapToPrinter` posts `application/octet-stream` with headers
  `X-Width`, `X-Height`, `X-Feed`.
- Bitmaps larger than `maxSafePrinterBitmapBodyBytes` (**~36 KiB**, the ESP32
  httpd reliability limit) are split by `splitBitmap` into vertical segments;
  only the **last** segment carries feed. Hard cap `maxPrinterBitmapBodyBytes`
  is 1 MiB.
- Feed is **baked as trailing blank raster rows** (`bitmapWithTrailingBlankRows`,
  24 px per feed line) because `ESC d n` alone doesn't reliably advance this
  mechanism; `X-Feed` is sent as a backup signal.
- Per-request fresh connection (`DisableKeepAlives`) with a 120 s timeout,
  because the UART bitmap transfer is slow and stale keep-alive connections
  cause EOF.

To drive **lever 3** we need a sibling call that POSTs
`/api/printer/density` and `/api/printer/speed` *before* the bitmap (see §9.2).

---

## 7. The lab-and-paper co-evolution workflow

This is the working method the user asked for: **the agent optimizes in the lab
autonomously; the user prints candidates and returns photos of the paper; both
sides evolve together; only calibrated winners merge into the Go service.**

### 7.1 The loop

```
        ┌──────────────────────────────────────────────────────────────┐
        │  (A) Agent tunes in the lab  ── change algorithm/params,       │
        │      add code to the lab, generate candidate bitmaps           │
        └───────────────┬──────────────────────────────────────────────┘
                        │ produces N labeled candidates (mode,γ,contrast,density)
                        ▼
        ┌──────────────────────────────────────────────────────────────┐
        │  (B) Human prints them on the real K118 and PHOTOGRAPHS paper  │
        └───────────────┬──────────────────────────────────────────────┘
                        │ photos returned to the agent
                        ▼
        ┌──────────────────────────────────────────────────────────────┐
        │  (C) Agent reads the photos, records observations in the diary,│
        │      updates lab defaults, narrows the search, repeats         │
        └───────────────┬──────────────────────────────────────────────┘
                        │ once a combo wins on paper for the target content
                        ▼
        ┌──────────────────────────────────────────────────────────────┐
        │  (D) Merge the winning settings into the Go raster package     │
        │      (phased, threshold-preserving), then re-verify on paper   │
        └──────────────────────────────────────────────────────────────┘
```

### 7.2 Division of labor (who does what)

- **Agent (autonomous):** edits the lab, adds experimental software, generates
  labeled comparison bundles, proposes the next parameter sweep, reads returned
  photos, keeps the diary, and merges winners into Go.
- **Human (in the loop):** loads paper, triggers the physical prints (or clicks
  the lab's print button), and sends back **photographs of the printed strips**.
  The human is the sensor for the one thing the agent cannot observe: physical
  dot gain and legibility on real paper.
- **Handoff artifact:** every printed candidate carries a printed **settings
  header** (already implemented via `settingsLines`/`makePrintCanvas`) so a photo
  is self-describing — the agent can read the algorithm and parameters straight
  off the strip.

### 7.3 The first concrete experiment: the density × algorithm grid

The single most informative first print. Fix one representative **mixed sample**
(a heading + a paragraph of body text + one photo), then print the matrix:

```
             density 12     density 20     density 28     density 35
 threshold      □              □              □              □
 atkinson       □              □              □              □
 floyd          □              □              □              □
 bayer8         □              □              □              □
```

Each cell prints with its settings header. Photograph the sheet. We are looking
for the cell where **text is solid and legible AND the photo shows tone without
muddiness**. If no single cell satisfies both — the likely outcome for mixed
pages — that is the hard evidence that justifies **segmented printing** (§8).

### 7.4 The tone-calibration experiment (resolve the gamma convention on paper)

Print a **grayscale ramp** (0→255 in ~16 steps) at a fixed density, dithered
with Atkinson, at γ ∈ {0.7, 1.0, 1.4, 1.8}. Measure/eyeball which γ makes the
printed ramp look perceptually linear (even steps, no early black-crush, no
washed highlights). That γ is *the* correct pre-correction for this paper +
density, and it settles the §3.2 convention ambiguity empirically instead of by
argument.

### 7.5 What to record for every round (diary + metrics)

For each printed candidate, capture: `source image`, `mode`, `threshold`, `γ`,
`brightness`, `contrast`, `edgeThreshold`, `density`, `speed`, `black density %`,
`packed bytes`, and the **paper observation** (text legible? photo muddy?
whiskers present? over/under-dark?). These rows become the dataset that picks the
production default.

---

## 8. Roadmap: from lab to production

Phased so each step is small, verifiable on paper, and reversible.

1. **Phase 0 — Freeze current behavior.** Golden test that `threshold` mode is
   byte-identical to today's `imageToBitmap`. No firmware change.
2. **Phase 1 — Lab upgrades (this ticket, now).** Thermal-calibrated tone
   defaults; density/speed controls in the lab; comparison-sheet export; true
   blue-noise mask; preset save/load. Run §7.3 and §7.4 experiments.
3. **Phase 2 — Go raster package.** Port the lab's kernels into
   `internal/app/raster` behind `Options{Mode,…}`; wire `--raster-mode`, `--gamma`,
   `--brightness`, `--contrast` into `render`/`print`; debug PNG artifacts.
   Default stays `threshold`.
4. **Phase 3 — Density as a first-class per-job lever.** Add a Go client for
   `/api/printer/density` + `/api/printer/speed`; set density before the bitmap;
   expose `--density`/`--speed` and `render.density`. Optionally move the lab
   proxy into the Go `setup` server.
5. **Phase 4 — Edge-hybrid + density cap.** Sobel edge map, unsharp, speckle
   suppression, local black-density cap; tune on cat portraits and one
   text-heavy page.
6. **Phase 5 — Block-aware / segmented printing.** The real fix for mixed pages:
   browser emits per-block bounding boxes/masks (or the host composites blocks),
   text segments print thresholded at high density, image segments print
   dithered at lower density, printed as ordered segments. Depends on a
   segmented firmware endpoint (tracked separately).
7. **Phase 6 — UI presets.** Expose simple presets in the studio
   (Text/default, Photo light, Photo detailed, Line art, Edge preserve) once
   block-aware printing exists.

---

## 9. API and format reference

### 9.1 Host → firmware: print a bitmap

```
POST http://<printer>/api/print/bitmap
Content-Type: application/octet-stream
X-Width:  <padded width, multiple of 8>
X-Height: <rows>
X-Feed:   <feed lines, backup signal; primary feed is baked blank rows>
body:     packed 1-bit MSB-first, (X-Width/8)*X-Height bytes

Firmware (web_server.c): validates width%8==0, expected body size,
  buffers the WHOLE body before emitting so no UART gap splits GS v 0.
Driver (printer_drv.c): printer_drv_print_bitmap()
  → GS v 0 header:  1D 76 30 00  wL wH hL hH   then packed bytes.
Host limits (printer.go): safe segment ≤ ~36 KiB; hard cap 1 MiB;
  tall bitmaps auto-split by splitBitmap(); 120 s HTTP timeout.
```

### 9.2 Host → firmware: heat / density and speed (lever 3)

```
POST /api/printer/density   {"density": 0..39}
   → printer_drv_set_density(n)  → ESC ## STDP n
     bytes: 1B 23 23 53 54 44 50 <n>          (default in use: 20)

POST /api/printer/speed     {"speed": <one of 25,30,37,50,56,62,70,80,90,100,120,150,180,200,220>}
   → printer_drv_set_speed(n)    → ESC ## STSP n
     bytes: 1B 23 23 53 54 53 50 <n>          (default in use: 80)

POST /api/printer/graphics-mode  {mode: 30|31|32}
   → printer_drv_set_graphics_mode(n) → ESC ## SPSM n
     bytes: 1B 23 23 53 50 53 4D <n>          (default in use: 31)

GET  /api/printer/status  → JSON incl. "overheated", "paper_near_end", "paper_out"
```

These are **separate endpoints** from the bitmap POST. To coordinate heat with
the image, POST density/speed first, then POST the bitmap. Watch `overheated` on
long or dense jobs.

### 9.3 CLI (today)

```
almanach-render-service print --layout daily.yaml --printer-ip 192.168.0.126 \
    [--threshold 128] [--selector .paper-body] [--feed-lines N] [--dry-run] \
    [--debug-dir DIR] [--viewport-width 800] [--viewport-height 3000]
```

Proposed additions (Phase 2/3): `--raster-mode`, `--gamma`, `--brightness`,
`--contrast`, `--density`, `--speed`.

### 9.4 Firmware knobs currently validated on hardware

`baud 460800, density 20, speed 80, graphics mode 31`. Width must be divisible
by 8. Density range 0..39; speed from the fixed table above.

---

## 10. Glossary

- **1-bit / monochrome:** each pixel is dot or no-dot; no gray.
- **Dithering / halftoning:** simulating gray by spatial black/white dot
  patterns that the eye averages.
- **Error diffusion:** dithering that pushes quantization error to neighbors
  (Floyd–Steinberg, Atkinson, Stucki, Burkes, Sierra).
- **Ordered dithering:** dithering by comparing to a tiled threshold matrix
  (Bayer, blue-noise).
- **Dot gain:** physical spreading of a printed dot; makes thermal output darker
  than the on-screen preview.
- **Gamma (here):** exponent γ on normalized gray, `v' = v^γ`; γ<1 lightens
  midtones, γ>1 darkens. Direction is settled empirically on paper (§7.4).
- **Density (heat):** K118 strobe-energy setting 0..39; global darkness knob.
- **Serpentine scan:** alternating row direction in error diffusion to reduce
  directional artifacts.
- **Edge-hybrid:** tone (dithered) layer OR a restrained detected-edge layer to
  preserve outlines.
- **Segmented printing:** sending a page as multiple print commands so text and
  images can use different modes/densities.
- **GS v 0:** ESC/POS raster bitmap command (`1D 76 30`).

---

## 11. File reference index

Host (Go):

- `internal/app/bitmap.go` — current fixed-threshold rasterizer + MSB packing.
- `internal/app/renderer.go` — Chrome render + screenshot handoff; `RenderOptions`.
- `internal/app/render_oneshot.go` — layout parse + `render:` options extraction.
- `internal/app/printer.go` — segmented POST, feed baking, transport safety.
- `internal/app/cmd_print.go` / `cmd_render.go` — CLI flags and wiring.
- `web/src/almanach-studio.jsx` — image block metadata + CSS thermal-tone preview.

Firmware (ESP32):

- `firmware/atoms3r/main/web_server.c` — `/api/print/bitmap`, density/speed/
  graphics/status endpoints, whole-body buffering.
- `firmware/atoms3r/main/printer_drv.c` — `GS v 0` raster, `ESC ## ST*`
  density/speed/graphics commands, status parsing.

Lab + prior research (older ticket):

- `.../ALMANACH-RASTERIZATION--.../various/raster-dither-lab.{html,js}` — the lab.
- `.../ALMANACH-RASTERIZATION--.../scripts/01-serve-raster-dither-lab.py` — server+proxy.
- `.../ALMANACH-RASTERIZATION--.../design-doc/01-...guide.md` — algorithm-depth design doc.
- `go-go-parc/Research/KB/On-Ramp/dithering-and-rasterization.md` — the standing
  recommendation (Atkinson + gamma pre-correction).
- `go-go-parc/Research/KB/On-Ramp/esc-pos-thermal-printer.md` — `GS v 0` framing.

---

## 12. First tasks for the intern

1. Read §1–3, then open the lab and reproduce today's `threshold` look.
2. Add density/speed controls to the lab UI (they POST to the firmware
   endpoints in §9.2). This unlocks lever 3 in the fast loop.
3. Build the §7.3 density × algorithm comparison sheet for one mixed sample.
4. Ask the human to print it and return photos.
5. Record observations in the diary (`reference/01-diary.md`), narrow the sweep,
   repeat.
6. Only once a combo wins on paper, port it into `internal/app/raster` (Phase 2),
   keeping `threshold` byte-identical, and re-verify on paper.
