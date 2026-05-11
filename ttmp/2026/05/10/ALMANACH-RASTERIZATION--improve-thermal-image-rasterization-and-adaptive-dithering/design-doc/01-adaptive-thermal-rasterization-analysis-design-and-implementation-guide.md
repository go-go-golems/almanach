---
title: Adaptive Thermal Rasterization Analysis Design and Implementation Guide
docType: design-doc
topics: [almanach, thermal-printer, backend, firmware, go]
status: active
intent: long-term
---

# Adaptive Thermal Rasterization Analysis Design and Implementation Guide

## Executive summary

Almanach currently renders a React/HTML layout with headless Chrome, captures a PNG screenshot, converts that screenshot to a packed 1-bit bitmap with a single fixed grayscale threshold, and posts the packed bytes to the AtomS3R/K118 printer firmware. This pipeline is simple and reliable for text, but it is not a good final image pipeline for photographs or dense illustrations. A fixed threshold loses midtones, fills dark regions, and can erase thin edges such as cat whiskers, fur boundaries, and fine line-art detail.

This document proposes a staged rasterization improvement plan. The first phase keeps the existing server and firmware protocol stable, but replaces `PngToBitmap(..., threshold)` with a pluggable Go rasterization package that supports threshold, ordered/Bayer dither, blue-noise ordered dither, Floyd-Steinberg, Atkinson, Stucki/Burkes/Sierra variants, and an edge-aware hybrid mode. The second phase adds a comparison-sheet workflow so real thermal output can be reviewed quickly. The third phase makes raster choices available through layout/render options and image-block metadata. The fourth phase aligns with the future segmented/chunked print endpoint so text can remain crisp while images receive photo-specific dithering and printer density/speed settings.

The recommended first hardware test set is:

1. fixed threshold at 128 and 160,
2. Atkinson with light thermal tone,
3. Floyd-Steinberg with light thermal tone,
4. Stucki or Sierra-2 with light thermal tone,
5. Bayer 8x8,
6. blue-noise ordered dither,
7. edge-hybrid: light Atkinson tone layer plus restrained Sobel/DoG edge mask.

For the cat portraits printed during `ALMANACH-PRINTER-UART`, the likely best default is **light tone + Atkinson** or **light tone + edge-hybrid Atkinson**. Atkinson tends to print lighter than Floyd-Steinberg and often looks better on thermal paper; the edge layer can recover whiskers and face outlines without turning dark fur into a solid black blob.

## Audience and prerequisites

This guide is written for a new intern implementing rasterization improvements in Almanach.

You should understand:

- basic Go image processing (`image.Image`, `color.Color`, slices),
- basic browser rendering and screenshots,
- 1-bit packed raster formats,
- the difference between host-side rendering and firmware-side printing,
- thermal printer constraints: monochrome dots, heat, paper darkness, UART/network transfer limits.

You do **not** need to be a halftoning expert before starting. The algorithm sections below define the terms and include pseudocode.

## Problem statement

The current renderer has only one monochrome conversion mode:

```text
RGB screenshot pixel -> grayscale luminance -> fixed threshold -> packed bit
```

This works when the source is already close to black-and-white, especially text and icons. It fails for natural images because a photograph contains continuous tones. A 1-bit printer cannot print gray directly; gray must be simulated by dot patterns or by controlling thermal energy over time. The K118 path currently receives one 1-bit bitmap body, so the host must decide which pixels are black.

The goal is to improve image output without destabilizing the working print pipeline:

- preserve edges and fine details,
- keep text crisp,
- avoid over-dark thermal output,
- provide deterministic comparison workflows,
- keep the firmware API compatible in the first phase,
- prepare for segmented printing later.

## Current-state architecture

### High-level rendering path

```text
layout JSON / ZIP bundle
        |
        v
internal/app layout loader
        |
        v
headless Chrome renders /almanach
        |
        v
chromedp screenshot of .paper-body or .paper-shell
        |
        v
internal/app/bitmap.go PngToBitmap fixed threshold
        |
        v
packed MSB-first 1-bit Bitmap
        |
        v
internal/app/printer.go POST /api/print/bitmap
        |
        v
firmware web_server.c validates width/height/body size
        |
        v
printer_drv_print_bitmap sends one GS v 0 raster command
```

### Evidence: current bitmap conversion

`internal/app/bitmap.go` is the central conversion point. It decodes a PNG and calls `imageToBitmap`:

- `internal/app/bitmap.go:10-17` documents `PngToBitmap` as fixed threshold conversion.
- `internal/app/bitmap.go:20-21` defines `imageToBitmap`.
- `internal/app/bitmap.go:34-37` converts each pixel to BT.601 grayscale and sets a black bit when `gray < threshold`.

The current conversion is therefore global thresholding only. There is no error diffusion, ordered dither, local threshold, edge preservation, gamma correction, or black-density control.

### Evidence: Chrome screenshot handoff

`internal/app/renderer.go` renders the layout and screenshots it:

- `internal/app/renderer.go:151` starts `renderWithChrome`.
- `internal/app/renderer.go:194` captures the screenshot with `chromedp.Screenshot`.
- `internal/app/renderer.go:202` logs the PNG screenshot size.
- `internal/app/renderer.go:204` converts the screenshot to 1-bit with `PngToBitmap(screenshotBuf, opts.Threshold)`.

This means rasterization currently happens after the whole page has been flattened into one PNG. At that point text and images are not semantically separate anymore.

### Evidence: render options and threshold setting

`internal/app/renderer.go:19` defines `defaultRenderThreshold = 128`. `RenderOptions` carries the threshold field at `internal/app/renderer.go:25-34`.

The CLI exposes threshold:

- `internal/app/cmd_print.go:29` has `Threshold int`.
- `internal/app/cmd_print.go:66` creates `--threshold` with default `128`.
- `internal/app/cmd_print.go:94` passes it into `RenderOptions`.

Layout files can wrap render options:

- `internal/app/render_oneshot.go:89-94` extracts a top-level `render` object when the file has `{ layout: ..., render: ... }`.
- `internal/app/render_oneshot.go:115-132` parses integer render options.

This is a useful extension point for `rasterMode`, `gamma`, `ditherStrength`, and similar options.

### Evidence: print body size and firmware contract

Host-side print transfer is in `internal/app/printer.go`:

- `internal/app/printer.go:14` defines `maxPrinterBitmapBodyBytes = 90 * 1024`.
- `internal/app/printer.go:18-21` rejects a bitmap larger than that limit.
- `internal/app/printer.go:31-33` sends `X-Width`, `X-Height`, and the packed bytes.

Firmware validates the same contract:

- `firmware/atoms3r/main/web_server.c:39` defines `MAX_BITMAP_BODY_BYTES (90 * 1024)`.
- `firmware/atoms3r/main/web_server.c:283-288` requires width/height and computes `expected = (width / 8) * height`.
- `firmware/atoms3r/main/web_server.c:298-304` rejects bodies over the maximum.
- `firmware/atoms3r/main/web_server.c:311-314` explains why the full body is buffered before printer output: network gaps must not occur inside one raster command.
- `firmware/atoms3r/main/web_server.c:323-324` calls `printer_drv_print_bitmap(width, height, body)`.

Any first-phase rasterization improvement should keep this contract unchanged: same width, height, MSB-first packed data.

### Evidence: current image-block preprocessing in the React UI

The React studio already has image-specific display controls:

- `web/src/almanach-studio.jsx:202-207` defines image defaults including `height`, `border`, `grayscale`, and `thermalTone`.
- `web/src/almanach-studio.jsx:581-589` maps `thermalTone` to CSS filters.
- `web/src/almanach-studio.jsx:591-615` renders image blocks and applies the filter to `<img>`.
- `web/src/almanach-studio.jsx:979-986` exposes the thermal image tone selector in the editor.

This is useful, but it is still CSS preprocessing before screenshot capture. It does not provide real dithering or per-block monochrome conversion.

## Constraints and invariants

### Printer and transport constraints

- The printer endpoint currently accepts one complete packed bitmap body.
- Width must be divisible by 8.
- Bits are packed MSB-first.
- Current safe body size is 90 KiB.
- Long jobs should eventually move to segmented or chunked printing; this is tracked separately by the GitHub issue created during `ALMANACH-PRINTER-UART`.
- The K118 settings currently validated on hardware are baud `460800`, density `20`, speed `80`, graphics mode `31`.

### Visual constraints

- Text should not be dithered like a photo. Dithered glyphs look fuzzy.
- QR codes and barcodes should use strict thresholding and exact sizing.
- Photos need tone simulation, usually through dithering.
- Thermal printers tend to make dense images too dark.
- Edges matter: whiskers, eyes, face outlines, text strokes, icon contours.

### Architectural constraints

- The current screenshot-to-bitmap path has no block mask. Once the whole page is flattened, we cannot easily tell which pixels came from text versus images.
- First-phase work should be host-side only. Firmware should continue to receive the same packed bitmap body.
- Later segmented printing can make block-aware rasterization much better because text and image segments can be printed separately.

## Rasterization and halftoning concepts

### Term: grayscale conversion

Before 1-bit conversion, RGB pixels are converted to luminance. The current code uses BT.601-like weights:

```text
gray = 0.299 R + 0.587 G + 0.114 B
```

This is reasonable. The improvement should keep this as a baseline but make it explicit in a raster package.

### Term: thresholding

Thresholding converts gray to black/white:

```text
if gray < threshold:
    black
else:
    white
```

Thresholding is good for text and high-contrast graphics. It is bad for photographs because midtones collapse.

### Term: dithering / halftoning

Dithering simulates gray by spatially distributing black and white dots. For a thermal printer, that means deciding which pixels become heated dots.

The two most relevant families are:

- ordered dithering: compare each pixel to a repeated threshold matrix or mask,
- error diffusion: threshold one pixel, compute the error, and push the error to neighboring future pixels.

### Term: edge-aware hybrid

An edge-aware hybrid algorithm creates two layers:

1. a tone layer, usually dithered,
2. an edge layer, usually detected by Sobel or Difference-of-Gaussians.

The final bitmap combines them carefully:

```text
final = tone_layer OR restrained_edge_layer
```

The edge layer should be restrained so it protects outlines without turning noise into black speckles.

## Algorithm comparison

### Fixed threshold

```text
for y, x:
    gray = luminance(pixel[x,y])
    out[x,y] = gray < threshold
```

Use for:

- text,
- QR codes,
- icons,
- already-high-contrast line art.

Do not use as the only photo mode.

### Adaptive threshold

Adaptive threshold computes a local threshold from a window around each pixel.

Common variants:

- local mean,
- Gaussian weighted mean,
- Niblack,
- Sauvola,
- Wolf-Jolion.

Simplified pseudocode:

```text
for each pixel p:
    window = grayscale pixels around p
    local_mean = mean(window)
    local_stddev = stddev(window)
    threshold = local_mean * (1 + k * (local_stddev / R - 1))  # Sauvola-like
    out[p] = gray[p] < threshold
```

Use for:

- scans,
- ink drawings,
- pencil drawings,
- uneven lighting.

Risk:

- can amplify paper texture and image noise,
- not a universal photo solution.

### Ordered Bayer dithering

Ordered dithering uses a repeated threshold matrix. Example 4x4 Bayer order:

```text
 0  8  2 10
12  4 14  6
 3 11  1  9
15  7 13  5
```

Pseudocode:

```text
matrix = bayer8x8
for y, x:
    t = matrix[y % 8][x % 8]
    adjusted = threshold_from_matrix_value(t)
    out[x,y] = gray[x,y] < adjusted
```

Use for:

- fast deterministic output,
- retro aesthetic,
- comparison baseline.

Risk:

- visible grid,
- can interfere with fine line art.

### Blue-noise ordered dithering

Blue-noise dithering is also mask-based, but the mask is designed to push noise into high spatial frequencies. It usually looks less patterned than Bayer.

Pseudocode:

```text
mask = blue_noise_64x64_values_0_to_255
for y, x:
    t = mask[y % 64][x % 64]
    out[x,y] = gray[x,y] < t
```

Use for:

- photographic images,
- stable deterministic texture,
- avoiding Floyd-Steinberg worm artifacts.

Implementation notes:

- store a small embedded 64x64 or 128x128 mask in Go,
- allow seed/mask version to keep output reproducible,
- test against actual thermal paper.

### Floyd-Steinberg error diffusion

Kernel:

```text
       X   7/16
3/16 5/16 1/16
```

Pseudocode:

```text
work = grayscale float array
for y from 0 to h-1:
    for x from 0 to w-1:
        old = work[y][x]
        new = 0 if old < threshold else 255
        out[y][x] = new == 0
        err = old - new
        work[y][x+1]   += err * 7/16
        work[y+1][x-1] += err * 3/16
        work[y+1][x]   += err * 5/16
        work[y+1][x+1] += err * 1/16
```

Use for:

- general photographic dither baseline,
- comparison against Atkinson and Stucki/Sierra.

Risk:

- worm-like artifacts,
- directional texture,
- dark regions may become muddy on thermal paper.

### Atkinson error diffusion

Atkinson diffuses less total error than Floyd-Steinberg. It was used by early Macintosh graphics and often produces lighter output.

Approximate kernel:

```text
      X  1  1
1  1  1
   1
/8
```

Pseudocode:

```text
neighbors = [(1,0), (2,0), (-1,1), (0,1), (1,1), (0,2)]
for each pixel:
    old = work[y][x]
    new = 0 if old < threshold else 255
    err = (old - new) / 8
    for neighbor in neighbors:
        work[neighbor] += err
```

Use for:

- thermal photos,
- illustrations,
- cat portraits,
- a lighter image mode.

Risk:

- less tonal accuracy,
- can posterize near extremes,
- may lose very dark shadow detail unless combined with edge preservation.

### Stucki, Burkes, Sierra

These are larger or alternate error-diffusion kernels.

Stucki:

```text
          X   8  4
 2  4  8  4  2
 1  2  4  2  1
/42
```

Burkes:

```text
          X   8  4
 2  4  8  4  2
/32
```

Sierra 3-row:

```text
          X   5  3
 2  4  5  4  2
    2  3  2
/32
```

Use for:

- smoother photographs,
- comparison sheets,
- possible high-quality image mode.

Risk:

- larger kernels soften edges,
- more compute than Atkinson/Floyd-Steinberg,
- may print too dark without tone control.

### Edge-aware hybrid mode

Recommended experimental hybrid:

```text
gray = luminance(image)
gray = applyToneCurve(gray, brightness=+20%, contrast=0.82, gamma=0.85)
gray = mildUnsharpMask(gray, amount=0.25)

tone = atkinsonDither(gray, threshold=128)
edge = sobelMagnitude(gray)
edgeMask = edge > edgeThreshold AND gray < edgeMaxGray
edgeMask = thinOrSuppressIsolatedDots(edgeMask)

final = tone OR edgeMask
final = capLocalBlackDensity(final, window=16x16, maxDensity=0.45)
```

This is the candidate most directly aligned with the user request: “make sure we don't lose edges.”

The edge mask should not be a raw Sobel threshold. Raw edges can add too much noise in fur. A practical implementation should:

- restrict edges to meaningful contrast,
- suppress isolated single-pixel noise,
- optionally only add edges where the tone layer is mostly white,
- expose edge strength as a parameter.

## Proposed architecture

### Phase 1: internal raster package

Create a new package:

```text
internal/app/raster/
  options.go
  luminance.go
  pack.go
  threshold.go
  ordered.go
  error_diffusion.go
  edge.go
  density.go
  compare.go
  raster_test.go
```

API sketch:

```go
package raster

import "image"

type Mode string

const (
    ModeThreshold       Mode = "threshold"
    ModeAdaptive        Mode = "adaptive"
    ModeBayer4          Mode = "bayer4"
    ModeBayer8          Mode = "bayer8"
    ModeBlueNoise64     Mode = "blue-noise64"
    ModeFloydSteinberg  Mode = "floyd-steinberg"
    ModeAtkinson        Mode = "atkinson"
    ModeStucki          Mode = "stucki"
    ModeBurkes          Mode = "burkes"
    ModeSierra2         Mode = "sierra2"
    ModeSierraLite      Mode = "sierra-lite"
    ModeEdgeHybrid      Mode = "edge-hybrid"
)

type Options struct {
    Mode            Mode
    Threshold       uint8
    Gamma           float64
    Brightness      float64
    Contrast        float64
    SharpenAmount   float64
    EdgePreserve    bool
    EdgeThreshold   float64
    EdgeStrength    float64
    MaxBlackDensity float64
    DensityWindow   int
}

func DefaultOptions() Options
func FromRenderOptions(map[string]interface{}, fallback Options) Options
func ImageToBitmap(img image.Image, opts Options) (*app.Bitmap, error)
func PNGToBitmap(pngData []byte, opts Options) (*app.Bitmap, error)
```

Because `app.Bitmap` currently lives in package `app`, avoid an import cycle. Either:

1. move `Bitmap` into `internal/app/raster` and type-alias it from `app`, or
2. keep the `Bitmap` struct in `app` and place raster code in `internal/app` initially, or
3. create `internal/bitmap` for the shared packed bitmap type.

Recommended: create `internal/app/raster` with its own `Bitmap` type only if the package boundaries are cleaned at the same time. For a minimal first implementation, keep code in `internal/app` as `raster_*.go` files, then extract once stable.

### Phase 2: replace `PngToBitmap` with options-based rasterization

Current call:

```go
bitmap, err := PngToBitmap(screenshotBuf, opts.Threshold)
```

Proposed call:

```go
bitmap, err := PngToBitmapWithRasterOptions(screenshotBuf, opts.Raster)
```

Extend `RenderOptions`:

```go
type RenderOptions struct {
    BaseURL        string
    Selector       string
    Threshold      uint8
    RasterMode     string
    RasterOptions  RasterOptions
    ViewportWidth  int
    ViewportHeight int
    WaitAfterLoad  time.Duration
    DebugDir       string
    CollectMetrics bool
}
```

CLI flags:

```text
--raster-mode threshold|atkinson|floyd-steinberg|stucki|burkes|sierra2|bayer8|blue-noise64|edge-hybrid
--gamma 1.0
--brightness 0.0
--contrast 1.0
--edge-preserve
--edge-threshold 24
--max-black-density 0.45
```

Layout wrapper example:

```yaml
render:
  selector: .paper-body
  threshold: 128
  rasterMode: atkinson
  gamma: 0.9
  brightness: 0.12
  contrast: 0.82
layout:
  theme: minimal
  paperWidth: 384
  blocks:
    - type: image
      data:
        src: images/cat.png
        height: 360
        thermalTone: light
```

### Phase 3: comparison sheet generator

Add a script or command that renders one input image with multiple algorithms and creates a labeled Almanach bundle.

Script location:

```text
ttmp/.../ALMANACH-RASTERIZATION.../scripts/01-create-raster-comparison-sheet.py
```

Future CLI command:

```text
almanach-render-service raster compare \
  --image assets/cat-portraits/portraits/cat-portrait-r02-c02.png \
  --modes threshold,atkinson,floyd-steinberg,stucki,bayer8,blue-noise64,edge-hybrid \
  --printer-ip 192.168.1.242
```

Comparison output should include:

- algorithm label,
- input tone settings,
- output bitmap byte size,
- density estimate,
- optional black pixel percentage,
- physical print order.

### Phase 4: block-aware rasterization

The current screenshot path flattens text and images. Block-aware rasterization has two possible designs.

#### Option A: browser produces per-block masks

The browser export API could capture:

- full screenshot,
- block bounding boxes,
- image block bounding boxes and metadata,
- maybe per-image source pixels.

Then Go applies different raster modes to different rectangles.

Pros:

- keeps Chrome layout fidelity,
- avoids rewriting the renderer.

Cons:

- text and image pixels may overlap due to captions/borders,
- masks are approximate,
- still one final bitmap.

#### Option B: host composites blocks itself

Render text/vector blocks and image blocks into an offscreen Go canvas-like representation.

Pros:

- full control,
- text can remain thresholded,
- images can be dithered independently.

Cons:

- much more work,
- duplicate browser layout engine behavior.

#### Option C: segmented printer endpoint

Use browser/Go to render each block or segment, then send segments separately:

```text
segment 1: title/text, threshold, high density
segment 2: image, Atkinson, lower density
segment 3: note text, threshold
```

Pros:

- best match for printer controls,
- different density/speed per image/text,
- avoids huge single bitmap bodies.

Cons:

- requires firmware API work,
- needs careful vertical spacing and ordering.

Recommended path: Phase 1 and Phase 2 now; Phase 4C after segmented endpoint exists.

## Implementation details and pseudocode

### Packed bitmap invariant

Rows are padded to a multiple of 8 pixels. Black pixels set bits MSB-first.

```go
func packBits(black [][]bool, width, height int) *Bitmap {
    paddedWidth := ((width + 7) / 8) * 8
    bytesPerRow := paddedWidth / 8
    data := make([]byte, bytesPerRow*height)

    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            if black[y][x] {
                data[y*bytesPerRow+x/8] |= byte(0x80) >> (x % 8)
            }
        }
    }

    return &Bitmap{Width: paddedWidth, Height: height, BytesPerRow: bytesPerRow, Data: data}
}
```

This must remain byte-compatible with `firmware/atoms3r/main/web_server.c` and `printer_drv_print_bitmap`.

### Luminance and tone preprocessing

```go
func luminance(img image.Image) [][]float64 {
    bounds := img.Bounds()
    gray := make2D(bounds.Dy(), bounds.Dx())
    for y := 0; y < bounds.Dy(); y++ {
        for x := 0; x < bounds.Dx(); x++ {
            r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
            rf := float64(r) / 65535.0
            gf := float64(g) / 65535.0
            bf := float64(b) / 65535.0
            gray[y][x] = 255.0 * (0.299*rf + 0.587*gf + 0.114*bf)
        }
    }
    return gray
}

func applyTone(gray [][]float64, gamma, brightness, contrast float64) {
    for each pixel:
        v := gray[y][x] / 255.0
        if gamma > 0 { v = pow(v, gamma) }
        v = (v - 0.5) * contrast + 0.5 + brightness
        gray[y][x] = clamp(v*255, 0, 255)
}
```

For thermal photos, start with:

```text
gamma:      0.85 to 1.00
brightness: +0.08 to +0.18
contrast:   0.75 to 0.90
```

### Generic error diffusion kernel

```go
type DiffusionTap struct {
    DX, DY int
    Weight float64
}

type Kernel struct {
    Name string
    Divisor float64
    Taps []DiffusionTap
}

func errorDiffuse(gray [][]float64, threshold float64, kernel Kernel) [][]bool {
    h, w := len(gray), len(gray[0])
    work := clone(gray)
    black := makeBool2D(h, w)

    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            old := clamp(work[y][x], 0, 255)
            newVal := 255.0
            if old < threshold {
                newVal = 0
                black[y][x] = true
            }
            err := old - newVal
            for _, tap := range kernel.Taps {
                nx, ny := x+tap.DX, y+tap.DY
                if 0 <= nx && nx < w && 0 <= ny && ny < h {
                    work[ny][nx] += err * tap.Weight / kernel.Divisor
                }
            }
        }
    }
    return black
}
```

### Serpentine scanning

Classic left-to-right diffusion can create directional artifacts. A later option is serpentine scanning:

```text
row 0: left -> right
row 1: right -> left, mirror kernel horizontally
row 2: left -> right
...
```

Do not add it until baseline algorithms are testable, because it complicates comparison with known kernels.

### Local black-density cap

Thermal output can become too dark. A post-pass can reduce dense regions.

Simple sketch:

```go
func capLocalBlackDensity(black [][]bool, window int, maxDensity float64, gray [][]float64) [][]bool {
    for each window tile:
        density := blackCount / (window*window)
        if density <= maxDensity { continue }
        // Remove least important black pixels first: pixels with higher gray values
        candidates := black pixels in tile sorted by gray descending
        remove until density <= maxDensity
    return black
}
```

This should be optional. It can damage text, so use it only for image/photo modes or whole-page photo tests.

### Edge-hybrid pseudocode

```go
func edgeHybrid(gray [][]float64, opts Options) [][]bool {
    toneGray := clone(gray)
    applyTone(toneGray, opts.Gamma, opts.Brightness, opts.Contrast)
    toneGray = unsharpMask(toneGray, opts.SharpenAmount)

    tone := errorDiffuse(toneGray, float64(opts.Threshold), AtkinsonKernel)

    edges := sobelMagnitude(gray)
    edgeMask := makeBool2D(h, w)
    for y, x:
        if edges[y][x] > opts.EdgeThreshold && gray[y][x] < 230 {
            edgeMask[y][x] = true
        }

    edgeMask = suppressIsolated(edgeMask)

    final := clone(tone)
    for y, x:
        if edgeMask[y][x] {
            final[y][x] = true
        }

    if opts.MaxBlackDensity > 0 {
        final = capLocalBlackDensity(final, opts.DensityWindow, opts.MaxBlackDensity, toneGray)
    }
    return final
}
```

## API and data model recommendations

### Render options

Add render options first, because they already exist for threshold and CLI rendering.

YAML example:

```yaml
render:
  selector: .paper-body
  rasterMode: edge-hybrid
  threshold: 128
  gamma: 0.9
  brightness: 0.12
  contrast: 0.82
  sharpenAmount: 0.2
  edgeThreshold: 26
  edgeStrength: 0.5
  maxBlackDensity: 0.45
layout:
  theme: minimal
  paperWidth: 384
  bodyScale: 1.4
  blocks:
    - id: cat
      type: image
      data:
        src: images/cat.png
        height: 360
        fit: contain
        border: false
        thermalTone: light
```

### CLI flags

Add to `render`, `print`, and possibly `inspect`:

```text
--raster-mode string
--gamma float
--brightness float
--contrast float
--sharpen float
--edge-threshold float
--max-black-density float
--density-window int
```

Keep `--threshold` for compatibility.

### Debug artifacts

When `--debug-dir` is set, write:

```text
screenshot.png
bitmap.bin
bitmap.png
raster.json
raster-gray.png
raster-tone.png
raster-edge.png       # only for edge modes
raster-density.json   # black pixel stats
```

This will make algorithm comparison far easier.

### Metrics

Add these metrics to render output rows:

```text
raster_mode
threshold
gamma
brightness
contrast
black_pixels
black_density
max_tile_density
```

## Testing strategy

### Unit tests

Add tests for:

- packing MSB-first bits,
- threshold mode matches current `imageToBitmap`,
- width padding remains correct,
- each diffusion kernel conserves expected behavior on tiny known arrays,
- options parsing from layout `render` map,
- invalid modes fall back or return clear errors.

Example table test:

```go
func TestThresholdMatchesLegacy(t *testing.T) {
    img := syntheticGradient(16, 4)
    old, _ := imageToBitmap(img, 128)
    got, _ := Rasterize(img, Options{Mode: ModeThreshold, Threshold: 128})
    require.Equal(t, old.Data, got.Data)
}
```

### Golden images

Create small synthetic fixtures:

- horizontal gradient,
- vertical gradient,
- checkerboard,
- text-like strokes,
- thin diagonal lines,
- stored cat portrait.

For each mode, write a PNG preview and compare with a golden file. Do not make physical print quality depend only on golden tests; golden tests catch regressions, not thermal appearance.

### Hardware tests

Print a comparison sheet:

```text
ALMANACH RASTER TEST
threshold 128
threshold 160
bayer8
blue-noise64
floyd-steinberg
atkinson
stucki
sierra2
edge-hybrid
```

Record:

- which mode preserves eyes and whiskers,
- which mode over-darkens black fur,
- whether text labels remain readable,
- print byte size,
- print time,
- printer status/temperature if available.

### Performance tests

The target page width is usually 384 pixels; height may be 500-2000 pixels. Error diffusion is cheap at this scale, but benchmark anyway:

```go
func BenchmarkRasterAtkinson384x2000(b *testing.B)
func BenchmarkRasterEdgeHybrid384x2000(b *testing.B)
```

## Implementation plan

### Phase 0: preserve current behavior

- Add tests around current threshold behavior.
- Ensure `threshold` output remains byte-identical.
- Do not change firmware.

### Phase 1: pluggable raster modes

- Add raster option types.
- Implement threshold and Atkinson first.
- Wire `--raster-mode` into `render` and `print`.
- Keep default mode as `threshold`.
- Add debug artifact previews.

### Phase 2: comparison sheet

- Add script or command to generate comparison bundles from a stored portrait.
- Use ticket-local cat portraits from `ALMANACH-PRINTER-UART` as initial source material.
- Print physical comparison sheet and record observations in the ticket diary.

### Phase 3: more algorithms

- Add Floyd-Steinberg.
- Add Stucki or Sierra-2.
- Add Bayer 8x8.
- Add blue-noise 64x64 mask.
- Add black-density statistics.

### Phase 4: edge hybrid

- Add Sobel edge map.
- Add mild unsharp mask.
- Add edge-hybrid mode.
- Add density cap optional post-pass.
- Tune on cat portraits and one text-heavy Almanach.

### Phase 5: layout and UI integration

- Add render-level options to documentation.
- Add image-block `rasterMode` only once block-aware or segmented printing exists.
- In the React UI, expose simple presets first:
  - Text/default,
  - Photo light,
  - Photo detailed,
  - Line art,
  - Edge preserve.

### Phase 6: segmented endpoint integration

- Coordinate with the chunked/segmented endpoint issue.
- Print image blocks as separate segments.
- Allow per-segment printer density/speed.
- Keep text thresholded and images dithered.

## Risks and tradeoffs

### Whole-page dithering can hurt text

If the entire screenshot is error-diffused, text edges may get noisy. Mitigation:

- keep default as threshold until tested,
- use comparison sheets with text labels,
- move toward block-aware/segmented printing.

### Thermal darkness differs from screen preview

A mode that looks good as a PNG may print too dark. Mitigation:

- print physical comparison sheets,
- add black-density metrics,
- tune with real paper.

### External library adoption risk

A Go dithering library can speed implementation, but dependencies need review:

- license,
- maintenance,
- allocation/performance,
- exact bit packing and grayscale behavior,
- whether it handles monochrome the way we need.

A small internal implementation may be better because the algorithms are short and our output contract is specific.

### Embedded browser path hides semantic blocks

The screenshot path makes it difficult to rasterize text and images differently. Mitigation:

- first phase is whole-page modes,
- later use block metadata or segmented endpoint.

## External references collected in this ticket

Stored source files:

- `sources/00-source-index.md`
- `sources/02-imagemagick-quantize-dithering.md`
- `sources/03-tanner-helland-dithering-eleven-algorithms.md`
- `sources/04-sweetcorn-dithering-algorithms.md`
- `sources/05-kagi-assistant-rasterization-research.md`

Notable findings:

- ImageMagick documents quantization and ordered/error-diffusion dither workflows; it is a good reference for CLI experiments and vocabulary.
- Tanner Helland’s article provides practical descriptions and source-oriented explanations for many classic error diffusion algorithms.
- Sweetcorn’s algorithm notes explicitly call out blue-noise as a good photographic option and Bayer as recognizably patterned.
- Kagi search surfaced `esimov/dithergo`, a Go implementation of Floyd-Steinberg, Atkinson, Burkes, Stucki, Sierra-2, Sierra-3, and Sierra-Lite; evaluate license/output before adoption.

## File references

Primary code paths:

- `internal/app/bitmap.go` — current fixed-threshold conversion and MSB-first packing.
- `internal/app/renderer.go` — Chrome screenshot capture and conversion call.
- `internal/app/cmd_print.go` — print CLI threshold flag and printer forwarding.
- `internal/app/cmd_render.go` — render CLI threshold/debug path.
- `internal/app/render_oneshot.go` — layout wrapper render options extraction.
- `internal/app/layout_bundle.go` — ZIP layout bundle loader and image inlining.
- `internal/app/printer.go` — host-side bitmap size guard and POST contract.
- `internal/app/server.go` — `/api/render-and-print` path.
- `web/src/almanach-studio.jsx` — image block metadata, thermal CSS filters, headless browser API.
- `firmware/atoms3r/main/web_server.c` — firmware `/api/print/bitmap` validation and buffered body handling.
- `firmware/atoms3r/main/printer_drv.c` — low-level printer bitmap command output.

Related ticket assets:

- `../ALMANACH-PRINTER-UART.../assets/cat-portraits/contact-sheet.png` — verified cat crop source for comparison sheets.
- `../ALMANACH-PRINTER-UART.../scripts/03-create-single-large-cat-bundle.py` — single portrait print bundle generator.

## Recommended first intern task

Implement a minimal, reversible Phase 1 branch:

1. Add `RasterMode` options but default to `threshold`.
2. Move existing threshold logic behind `Rasterize(img, Options{Mode: Threshold})`.
3. Add Atkinson mode.
4. Add `--raster-mode` to `render` and `print`.
5. Add debug PNG output for the raster bitmap.
6. Generate and print a two-mode comparison: `threshold` vs `atkinson` on `cat-portrait-r02-c02.png`.
7. Record physical observations in the diary before adding more algorithms.

This keeps the first change small, measurable, and safe.
