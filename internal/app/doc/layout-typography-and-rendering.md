---
Title: "Typography, Themes, and Rendering (Layout DSL v2)"
Slug: "layout-typography-and-rendering"
Short: "Control fonts, presets, themes, margins, and per-block rasterization/heat for legible thermal pages."
Topics:
- layouts
- typography
- themes
- rendering
- thermal-printer
Commands:
- render
- print
- inspect
Flags:
- layout
- threshold
- supersample
- printer-ip
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

This guide covers the typography and rendering controls added in Layout DSL v2:
typography presets with sensible defaults, data-driven themes with hinted print
fonts, page margins, typed render options, and per-block rasterization and heat.
It is task-oriented: each section states what the control does, shows a runnable
layout, and explains why the control matters for a 1-bit thermal printer. The
field-level reference lives in `layout-dsl-reference`; this page shows how to use
the fields together.

The controlling constraint throughout is that the printer emits one bit per dot.
Anti-aliased sub-pixel strokes do not exist on paper — a stroke is either burned
or not. That single fact explains why bigger sizes, heavier weights, hinted
fonts, and dithering all matter, and why the defaults are tuned the way they are.

## Getting good typography for free

Typography is expressed as **named presets**, and the built-in defaults already
encode the print-legibility recipe: larger sizes, an absolute minimum-size floor,
heavier body text, and bold italic for quotes. A layout that sets no typography
at all still prints legibly. The recipe is a data default, not something you must
re-derive per page.

Render the knowledge-strip example and you get the tuned defaults with no
typography block:

```bash
almanach-render-service render \
  --layout examples/layouts/03-knowledge-strip.yaml \
  --out /tmp/out.png --format png \
  --web-dir web/dist --chrome-path /usr/bin/google-chrome-stable
```

The presets are named by role: `title`, `sectionLabel`, `overline`, `word`,
`metric`, `body`, `bodyStrong`, `emphasis` (quotes/notes), `caption`, `small`,
and `meta`. Each block draws from the presets its content needs; you refer to
them only when you want to change them.

## Overriding presets per page and per block

Override a preset for the whole page under `typography.presets`. Overrides are a
per-field merge over the defaults, so you change only what you name:

```yaml
theme: crisp
paperWidth: 384
typography:
  presets:
    body:
      size: 15          # bigger running text on this page
      weight: 700       # and heavier, to survive the threshold
blocks:
  - id: h
    type: history
    data:
      label: Today in History
      items:
        - { year: "1969", event: "Apollo 11 lands on the Moon." }
```

Override a single block's primary text with `style`. This wins over the page and
default layers:

```yaml
  - id: q
    type: quote
    style:
      italic: false     # this one quote is upright and bold
      weight: 700
    data:
      text: Weight survives the threshold.
```

The full resolution order is `built-in default <- theme presetOverrides <- layout
typography <- block style`. `bodyScale` multiplies every resolved size, and each
preset's `minSize` is an absolute floor applied after scaling, so text never
drops below what the head can render.

## Choosing a font: themes and the hinted palette

A theme sets fonts and colors. For print, the font choice dominates: well-hinted
fonts stay crisp at small sizes, while delicate display fonts lose strokes. Two
built-in themes use the embedded hinted DejaVu families and are the recommended
starting point for paper:

```yaml
theme: crisp        # DejaVu Serif — crisp small serif
# or
theme: crispsans    # DejaVu Sans — crisp small sans
```

You can also define a theme inline — the "data-driven" form — patching a base
theme with colors, a font palette, and preset overrides, without any code
change:

```yaml
theme:
  base: minimal
  fontPalette:
    - "'DejaVu Sans', sans-serif"     # display
    - "'DejaVu Sans', sans-serif"     # body
  presetOverrides:
    body:
      weight: 700
```

## Controlling the margin

`margin` sets the padding of the printed body. A tight margin fits more content
per strip; the theme default is roughly `20×22px`.

```yaml
margin: { x: 6, y: 10 }     # or a number, or { top, right, bottom, left }
```

## Render options are typed and validated

The page-level `render:` block controls how the screenshot becomes a bitmap and
how the printer burns it. Values are validated up front — an out-of-range value
is an error, not a silent clamp — and the block works either flat (top level) or
under a `layout:` wrapper.

```yaml
theme: crisp
paperWidth: 384
render:
  threshold: 128           # 1-bit cut point, 0-255
  printerDensity: 38       # head heat; text reads best hot
blocks:
  - id: t
    type: title
    data: { text: HELLO }
```

`printerDensity` is applied to the printer before the page is sent, so the recipe
"text at density 38" is a layout field rather than a manual step.

## Mixed pages: block-aware rasterization and per-segment heat

Text and photographs want opposite treatment. Text wants a hard threshold;
photographs want Atkinson dithering with a gamma tone curve. On the head, text
reads best hot and photographs best cool. A per-block `render` override turns a
block's bounding box into a raster region and a heat band, so one page does both.

```yaml
theme: crisp
paperWidth: 384
render:
  printerDensity: 38                   # page default: hot text
blocks:
  - id: t
    type: title
    data: { text: FIELD NOTES }
  - id: photo
    type: image
    render:
      rasterMode: RASTER_MODE_ATKINSON  # dither this region
      gamma: 0.8                         # lift shadows before dithering
      printerDensity: 20                 # print this band cooler
    data:
      src: data:image/png;base64,...
      grayscale: false                   # keep midtones for the dither
  - id: q
    type: quote
    data: { text: One page, two treatments. }
```

Print it and the page is sent in density bands — the text bands at 38, the photo
band at 20:

```bash
almanach-render-service print \
  --layout mixed.yaml --printer-ip 192.168.0.126 \
  --web-dir web/dist --chrome-path /usr/bin/google-chrome-stable
```

To inspect the dithering without a printer, render with `--debug-dir` and unpack
the 1-bit `bitmap.bin`: a dithered region shows scattered dots (many black/white
transitions per row), a thresholded region shows a single hard edge.

## Preview-first workflow

Iterate on screen before spending paper. Render to PNG to see the screenshot,
and to `--debug-dir` to see the exact bitmap the printer receives:

```bash
# screenshot PNG (grayscale, what the browser drew)
almanach-render-service render --layout page.yaml --out /tmp/page.png --format png \
  --web-dir web/dist --chrome-path /usr/bin/google-chrome-stable

# add the 1-bit bitmap + metrics for inspection
almanach-render-service render --layout page.yaml --out /tmp/page.png --format png \
  --debug-dir /tmp/dbg --web-dir web/dist --chrome-path /usr/bin/google-chrome-stable
```

Studio changes (JSX) require rebuilding the SPA before a headless render sees
them: `pnpm --dir web build`. Layout JSON/YAML changes need no rebuild.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Small text loses strokes | Delicate theme font, or size below the floor. | Use `crisp`/`crispsans`, raise `weight`, or bump `size`/`minSize`. |
| Italic is unreadable | Normal italic thins strokes. | Use bold italic (`emphasis` already does; set `weight: 700` + `italic: true`). |
| A photo prints as a black blob | Threshold on a gradient + hot head. | Add `rasterMode: RASTER_MODE_ATKINSON`, `gamma: 0.8`, `grayscale: false`, lower `printerDensity`. |
| Dithering has no effect | Image pre-binarized by the thermal filter. | Set the image `data.grayscale: false` so midtones reach the rasterizer. |
| `render.threshold must be 0-255` | Out-of-range render value. | Use an in-range value; render options are validated. |
| A studio change is not visible when rendering | Stale SPA bundle. | Run `pnpm --dir web build` before rendering. |
| A block shows a dashed placeholder | Unknown block `type`. | Fix the type string; unknown types placeholder instead of vanishing. |

## See Also

- `almanach-render-service help layout-dsl-reference`
- `almanach-render-service help layouts-user-guide`
- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help tutorial-knowledge-strip`
