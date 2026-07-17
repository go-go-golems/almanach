---
Title: "Almanach Layout DSL Reference"
Slug: "layout-dsl-reference"
Short: "Complete field reference for Almanach YAML/JSON layout files and supported block types."
Topics:
- layouts
- reference
- yaml
- blocks
Commands:
- render
- inspect
- print
Flags:
- layout
- selector
- threshold
- viewport-width
- viewport-height
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The Almanach layout DSL is a JSON/YAML object consumed by Almanach Studio. It is intentionally small: one page object plus an ordered list of blocks. Each block has a known `type`, and its `data` object follows the fields documented here.

Use this reference when generating layouts from scripts, LLMs, cron jobs, or external data sources. The examples use YAML, but the same structure works as JSON.

## Top-Level Raw Layout

A raw layout has this shape:

```yaml
almanach_studio_version: 1
exported_at: "2026-05-08T12:00:00Z"
theme: minimal
paperWidth: 384
bodyScale: 1.45
feedLines: 3
blocks:
  - id: title-1
    type: title
    data:
      text: THE ALMANACH
      subtitle: Your daily digest
```

| Field | Type | Required | Description |
|---|---|---:|---|
| `almanach_studio_version` | integer | no | Layout file version. Use `1`. |
| `exported_at` | string | no | ISO timestamp for provenance. |
| `theme` | string \| object | no | A built-in theme key (e.g. `minimal`, `crisp`) **or** an inline theme object. See [Themes](#themes). |
| `paperWidth` | integer | no | Paper width in pixels. Use `384`. |
| `bodyScale` | number | no | Global font-size multiplier from `1.0` to `2.0`. Scales every typography preset. |
| `feedLines` | integer | no | Blank trailing feed rows expressed as line units. |
| `margin` | number \| object | no | Paper margin in px. See [Page Margin](#page-margin). |
| `typography` | object | no | Named typography preset overrides. See [Typography Presets](#typography-presets). |
| `render` | object | no | Page-level render/raster options. Works at the top level (flat) or under a `layout:` wrapper. See [Render Options](#render-options). |
| `blocks` | array | yes | Ordered block list. |

> **Fonts and thermal output.** For crisp small text on the printer, prefer the
> hinted `crisp` (DejaVu Serif) or `crispsans` (DejaVu Sans) themes over the
> delicate `minimal`/`classic` Garamond themes — thin strokes get thresholded
> away at 1-bit. The typography defaults already bake in the paper-verified
> recipe (bigger sizes, a minimum-size floor, bold body and bold-italic quotes).

## Typography Presets

Typography is modeled as **named presets**, not per-element sizes. Each preset
maps to a concrete text style (font, size, weight, line height, letter spacing,
case, italic). Blocks reference presets implicitly by their role — a `quote`'s
text uses the `body`/`emphasis` presets, its attribution uses `caption`, and so
on. You never have to set sizes to get good typography: the built-in defaults
already encode the print-legibility recipe.

You override typography at two levels, resolved as a shallow, per-field merge:

```
built-in default  <-  layout typography override  <-  per-block style
```

Override presets for the whole page under `typography.presets`:

```yaml
typography:
  presets:
    body:                 # bump body text everywhere on this page
      size: 15
      weight: 700
    caption:
      font: "'DejaVu Sans', sans-serif"
```

A `TextStyle` accepts these fields (all optional; unset fields inherit):

| Field | Type | Description |
|---|---|---|
| `font` | string | CSS font-family stack. Unset inherits the theme font for the preset's role. |
| `size` | number | Base font size in px (before `bodyScale`). |
| `weight` | integer | Font weight, `100`–`900`. Heavier survives the 1-bit threshold. |
| `lineHeight` | number | Unitless line-height multiplier. |
| `letterSpacing` | number | Letter spacing in em. |
| `textCase` | string | `TEXT_CASE_UPPER`, `TEXT_CASE_LOWER`, `TEXT_CASE_CAPITALIZE`, `TEXT_CASE_NONE`, or the plain CSS values `uppercase`/`lowercase`/etc. |
| `italic` | boolean | Italic. Bold italic is the legible italic on thermal. |
| `minSize` | number | Absolute floor (px) applied after `bodyScale` — the legibility guard. |

Built-in preset names: `title`, `sectionLabel`, `overline`, `word`, `metric`,
`body`, `bodyStrong`, `emphasis`, `caption`, `small`, `meta`.

A single block can override its primary text with `style` (see
[Block Object](#block-object)):

```yaml
- id: q1
  type: quote
  style:              # bold, non-italic just for this quote
    italic: false
    weight: 700
  data:
    text: Weight survives the threshold.
```

## Themes

A theme controls fonts and colors. `theme` may be a **built-in name** or an
**inline object** that patches a base theme with no code change ("data-driven
themes").

Built-in theme keys:

| Key | Fonts | Notes |
|---|---|---|
| `classic`, `minimal`, `botanical`, `notebook`, `ledger`, `space` | Garamond / display fonts | Elegant on screen; thin strokes at small sizes. |
| `crisp` | DejaVu Serif (hinted) | **Recommended for print.** Crisp small text. |
| `crispsans` | DejaVu Sans (hinted) | Sans alternative, also crisp small. |

Select a built-in theme by name:

```yaml
theme: crisp
```

Or supply an inline theme that patches a base:

```yaml
theme:
  base: minimal                       # start from a built-in (default: classic)
  colors:                             # any of paper/ink/muted/accent/rule
    ink: "#000000"
    paper: "#ffffff"
  fontPalette:                        # [display, body, mono] in preference order
    - "'DejaVu Sans', sans-serif"
    - "'DejaVu Sans', sans-serif"
  presetOverrides:                    # theme-level typography (below layout typography)
    body:
      weight: 700
```

Inline theme fields: `base`, `colors` (`paper`/`ink`/`muted`/`accent`/`rule`),
`fontPalette` (array) or explicit `fontDisplay`/`fontBody`/`fontMono`,
`titleSize`/`titleWeight`/`titleSpacing`/`titleCase`, and `presetOverrides`
(same shape as `typography.presets`). Theme preset overrides sit **below** the
layout's `typography` in the resolution chain.

## Page Margin

`margin` controls the padding of the printed page body. Omit it to keep the
theme default (~20×22px). Three forms are accepted:

```yaml
margin: 8                                     # uniform px on all sides
margin: { x: 6, y: 10 }                        # horizontal / vertical
margin: { top: 10, right: 6, bottom: 10, left: 6 }
```

Use a tight margin to fit more content per strip, or a wider one for a calmer page.

## Wrapped Render Request

The CLI also accepts a wrapper with `layout` and `render` sections:

```yaml
layout:
  theme: minimal
  paperWidth: 384
  bodyScale: 1.5
  blocks:
    - id: t1
      type: title
      data:
        text: "{{title}}"
        subtitle: Layout plus render options
render:
  selector: .paper-body
  threshold: 128
  viewportWidth: 800
  viewportHeight: 3000
data:
  title: HELLO WORLD
```

Use this form when a file should carry content, render preferences, and template data together.

## Render Options

Render options control how the screenshot becomes a 1-bit bitmap and how the
printer burns it. They are validated up front: an out-of-range value (e.g.
`threshold: 300`) is a hard error rather than a silent clamp. Command-line flags
supply the same values as defaults.

`render` works in **two positions**:

- **Flat** — a top-level `render:` alongside `blocks:` (recommended):

  ```yaml
  theme: crisp
  paperWidth: 384
  render:
    threshold: 128
    printerDensity: 38
  blocks:
    - id: t1
      type: title
      data: { text: HELLO }
  ```

- **Wrapped** — under a `layout:` wrapper (see [Wrapped Render Request](#wrapped-render-request)).

| Field | Type | Default | Description |
|---|---|---:|---|
| `selector` | string | `.paper-body` | CSS selector to screenshot (page-level only). |
| `threshold` | integer | `128` | 1-bit threshold, `0`–`255`. |
| `supersampleScale` | integer | `1` | Render oversampling factor `1`–`8`; downscaled before 1-bit. |
| `viewportWidth` | integer | `800` | Browser viewport width. |
| `viewportHeight` | integer | `3000` | Browser viewport height. |
| `rasterMode` | string | `RASTER_MODE_THRESHOLD` | `RASTER_MODE_THRESHOLD`, `RASTER_MODE_ATKINSON`, `RASTER_MODE_FLOYD_STEINBERG`, `RASTER_MODE_BAYER`. |
| `gamma` | number | unset | Tone-curve gamma applied before dithering. Photos want `~0.8`. |
| `printerDensity` | integer | unset | Printer head heat, `0`–`255`. Text reads best `~38`, photos `~20`. |
| `printerSpeed` | integer | unset | Printer feed speed. |

### Per-block render overrides

A block may carry its own `render` object with the same fields. This drives
**block-aware rasterization** and **per-segment heat**: the block's bounding box
becomes a raster region and/or a heat band. See
[Block-Aware Rasterization](#block-aware-rasterization-and-per-segment-heat).

```yaml
- id: photo
  type: image
  render:
    rasterMode: RASTER_MODE_ATKINSON   # dither this region
    gamma: 0.8                          # lift shadows first
    printerDensity: 20                  # print it cooler than the text
  data:
    src: data:image/png;base64,...
```

## Block Object

Every block uses this envelope:

```yaml
- id: unique-block-id
  type: quote
  data:
    label: Quote of the Day
    text: Stay curious.
    author: Unknown
```

| Field | Type | Required | Description |
|---|---|---:|---|
| `id` | string | recommended | Unique ID. Use stable IDs for generated layouts. |
| `type` | string | yes | A block type. Unknown types render a visible placeholder rather than being dropped. |
| `data` | object | yes | Type-specific fields. |
| `style` | object | no | Per-block typography override (a `TextStyle`) applied to the block's primary text. See [Typography Presets](#typography-presets). |
| `render` | object | no | Per-block render/raster override. See [Per-block render overrides](#per-block-render-overrides). |

Supported block types:

```text
title, date, divider, plan, news, weather, note, image, habits, mood,
reading, reflection, quote, word, history, did
```

An unrecognized `type` is not an error: the renderer draws a dashed
"Unknown block type" placeholder in its place, so a typo or a newer block type
degrades visibly instead of silently vanishing.

## `title`

Use `title` at the top of a page.

```yaml
- id: title-1
  type: title
  data:
    text: DAILY BRIEFING
    subtitle: Friday desk edition
```

| Data field | Type | Description |
|---|---|---|
| `text` | string | Main title. Keep it short. |
| `subtitle` | string | Smaller line under the title. |

## `date`

Use `date` directly under the title.

```yaml
- id: date-1
  type: date
  data:
    date: May 8, 2026
    day: Friday
```

| Data field | Type | Description |
|---|---|---|
| `date` | string | Human-readable date. |
| `day` | string | Day name. |

## Scaffold Layout

When no layout file is provided (no `--layout` flag or empty POST body), the system generates a minimal scaffold layout instead of fetching or inventing content. The scaffold contains only a title block ("ALMANACH") and a date block with today's date. No APIs are called, no random content is selected.

This behavior replaces the former default layout which called fetcher functions for weather, news, quotes, words, and history. If you want a full daily page, provide a layout file or a template with a data context.

## `divider`

Use `divider` to create a visual pause.

```yaml
- id: div-1
  type: divider
  data:
    style: dots
```

| Data field | Type | Values |
|---|---|---|
| `style` | string | `line`, `dots`, `wave`, `leaves` |

## `weather`

Use `weather` for compact current conditions.

```yaml
- id: weather-1
  type: weather
  data:
    temp: 18°C
    condition: Clear morning
    high: 23°C
    low: 12°C
    sunrise: "05:47"
    sunset: "20:32"
```

| Data field | Type | Description |
|---|---|---|
| `temp` | string | Current temperature. |
| `condition` | string | Short condition text. |
| `high` | string | Daily high. |
| `low` | string | Daily low. |
| `sunrise` | string | Sunrise time. Quote it in YAML. |
| `sunset` | string | Sunset time. Quote it in YAML. |

## `plan`

Use `plan` for time-ordered tasks.

```yaml
- id: plan-1
  type: plan
  data:
    label: Today's Plan
    items:
      - time: "08:30"
        text: Morning review
        done: true
      - time: "10:00"
        text: Deep work
        done: false
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `items` | array | List of plan items. |

Plan item fields:

| Field | Type | Description |
|---|---|---|
| `time` | string | Time label. Quote it in YAML. |
| `text` | string | Task text. |
| `done` | boolean | Renders a checked box and strikethrough. |

## `news`

Use `news` for short headlines.

```yaml
- id: news-1
  type: news
  data:
    label: Top News
    items:
      - headline: One-shot renderer accepts YAML layouts.
        source: Almanach Lab
        time: now
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `items` | array | List of headlines. |

News item fields:

| Field | Type | Description |
|---|---|---|
| `headline` | string | Short headline. |
| `source` | string | Source label. |
| `time` | string | Relative time or timestamp. |

## `image`

Use `image` to embed a photograph or illustration on the page. The image is rendered at the full paper width with configurable height, fit mode, and a thermal-specific grayscale filter.

```yaml
- id: image-1
  type: image
  data:
    label: Photo Plate
    src: data:image/jpeg;base64,/9j/4AAQ...
    alt: Morning desk photo
    caption: The desk at 7 AM
    height: 160
    fit: cover
    border: true
    grayscale: true
    thermalTone: normal
```

| Data field | Type | Default | Description |
|---|---|---|---|
| `label` | string | `"Image Plate"` | Section heading. |
| `src` | string | `""` | Image URL (`https://…`) or data URL (`data:image/…;base64,…`). Required for the image to render. |
| `alt` | string | `""` | Alt text for accessibility. Falls back to `caption`. |
| `caption` | string | `""` | Italic caption below the image. |
| `height` | integer | `160` | Image height in pixels, clamped to 48–420. |
| `fit` | string | `"cover"` | CSS object-fit: `cover` (fill, crop) or `contain` (fit, letterbox). |
| `border` | boolean | `true` | Draw a thin border around the image. |
| `grayscale` | boolean | `true` | Apply thermal grayscale filter for print preview. |
| `thermalTone` | string | `"normal"` | Grayscale tone preset: `normal` (high contrast) or `light` (brighter, lower contrast — better for faint source images). |

When `src` is empty, the block renders a dashed placeholder box with "Add an image URL or upload a file" text.

### Image sources

- **Data URLs**: The Studio editor embeds uploaded images as `data:image/…;base64,…` strings. These work in both the browser and headless CLI without fetching external files.
- **HTTP URLs**: Remote images work in the browser but may fail in headless CLI if the server is unreachable. The CLI loads images with `crossOrigin: "anonymous"`, so the server must allow CORS.
- **ZIP bundles**: Place image files next to `layout.yaml` in a ZIP archive. The CLI inlines relative `src` paths as data URLs before rendering.

## `note`

Use `note` for a short italic callout.

```yaml
- id: note-1
  type: note
  data:
    label: Daily Note
    text: Preview first, then print.
    author: Almanach Studio
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Optional heading. |
| `text` | string | Note body. |
| `author` | string | Optional attribution. |

## `quote`

Use `quote` for centered quotation text.

```yaml
- id: quote-1
  type: quote
  data:
    label: Quote of the Day
    text: Simplicity is prerequisite for reliability.
    author: Edsger W. Dijkstra
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `text` | string | Quote text without surrounding quotes. |
| `author` | string | Attribution. |

## `word`

Use `word` for vocabulary pages.

```yaml
- id: word-1
  type: word
  data:
    label: Word of the Day
    word: apricity
    phonetic: a-pri-ci-ty
    part: noun
    definition: The warmth of the sun in winter.
    example: We enjoyed the brief apricity.
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `word` | string | Main word. |
| `phonetic` | string | Pronunciation hint. |
| `part` | string | Part of speech. |
| `definition` | string | Definition. |
| `example` | string | Optional example sentence. |

## `history`

Use `history` for dated facts.

```yaml
- id: history-1
  type: history
  data:
    label: Today in History
    items:
      - year: "1945"
        event: Victory in Europe Day is celebrated.
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `items` | array | History facts. |

History item fields:

| Field | Type | Description |
|---|---|---|
| `year` | string | Year label. Quote years to keep them strings. |
| `event` | string | Event description. |

## `did`

Use `did` for fun facts.

```yaml
- id: did-1
  type: did
  data:
    label: Did You Know?
    items:
      - Honey never spoils when stored properly.
      - A day on Venus is longer than a Venusian year.
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `items` | array of strings | Fact list. |

## `habits`

Use `habits` for a compact weekly tracker.

```yaml
- id: habits-1
  type: habits
  data:
    label: Habit Tracker
    range: May 4 — May 10
    columns: [M, T, W, T, F, S, S]
    items:
      - name: Meditate
        days: [1, 1, 1, 1, 1, 0, 0]
    reflection: Good consistency.
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `range` | string | Date range label. |
| `columns` | array | Column labels. |
| `items` | array | Habit rows. |
| `reflection` | string | Optional summary. |

Habit item fields:

| Field | Type | Description |
|---|---|---|
| `name` | string | Habit name. |
| `days` | array of integers | Seven values, `1` for filled and `0` for empty. |

## `mood`

Use `mood` for daily personal state.

```yaml
- id: mood-1
  type: mood
  data:
    label: Mood & Energy
    mood: 4
    energy: 3
    sleep: 7h 05m
    notes: Focus was strong before lunch.
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `mood` | integer | 1 to 5. |
| `energy` | integer | 1 to 5. |
| `sleep` | string | Sleep summary. |
| `notes` | string | Optional note. |

## `reading`

Use `reading` for one current book plus a short queue.

```yaml
- id: reading-1
  type: reading
  data:
    label: Reading List
    current:
      title: The Design of Everyday Things
      author: Don Norman
      progress: 68
    next:
      - Deep Work — Cal Newport
      - Thinking in Systems — Donella Meadows
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `current` | object | Current book. |
| `next` | array of strings | Want-to-read queue. |

Current book fields:

| Field | Type | Description |
|---|---|---|
| `title` | string | Book title. |
| `author` | string | Author. |
| `progress` | integer | Percent complete, 0 to 100. |

## `reflection`

Use `reflection` for a daily journal footer.

```yaml
- id: reflection-1
  type: reflection
  data:
    label: Daily Reflection
    well: Built the preview loop before touching paper.
    better: Keep test layouts shorter.
    learned: Screenshots need layout metrics.
    quote: Measure twice, print once.
```

| Data field | Type | Description |
|---|---|---|
| `label` | string | Section heading. |
| `well` | string | What went well. |
| `better` | string | What could be better. |
| `learned` | string | What was learned. |
| `quote` | string | Optional closing quote. |

## Block-Aware Rasterization and Per-Segment Heat

The printer emits pure 1-bit output, and text and photographs want opposite
treatment. Text wants a hard threshold so strokes stay crisp; photographs want
error-diffusion dithering plus a gamma tone curve so gradients survive. On the
printer head, text reads best hot and photographs best cool. A per-block `render`
override lets one page do both.

When a block carries a `render` override, its on-page bounding box becomes a
region:

- **Rasterization.** A `rasterMode` of `RASTER_MODE_ATKINSON` (or another dither
  mode) converts that region with Atkinson error diffusion; a `gamma` below `1`
  lifts shadows first so a hot head does not turn the photo into a solid mass.
  Rows outside any region use the page `threshold`.
- **Heat.** A `printerDensity` on the block prints that region's rows at that
  density; the rest of the page uses the page-level `printerDensity`. The page is
  sent to the printer in density bands.

A mixed page — crisp hot text around a soft cool photo — looks like this:

```yaml
theme: crisp
paperWidth: 384
render:
  printerDensity: 38            # page default: hot text
blocks:
  - id: t
    type: title
    data: { text: FIELD NOTES }
  - id: photo
    type: image
    render:
      rasterMode: RASTER_MODE_ATKINSON
      gamma: 0.8
      printerDensity: 20        # this band prints cooler
    data:
      src: data:image/png;base64,...
      grayscale: false          # keep midtones so dithering has tones to work with
  - id: q
    type: quote
    data: { text: One page, two treatments. }
```

> **Tip.** The `image` block's default thermal filter boosts contrast, which
> pre-binarizes the source before dithering. Set `grayscale: false` on an image
> you want dithered so its midtones reach the rasterizer.

## YAML Safety Rules

Use these rules when generating YAML programmatically:

- Quote times: `"08:30"`.
- Quote years when they are labels: `"1945"`.
- Quote strings that contain `: `.
- Keep block IDs unique.
- Prefer plain ASCII punctuation for thermal clarity unless the glyph has been tested.

## Template Syntax

Layout files can contain template expressions that are resolved at render time using a data context. This allows automation scripts, cron jobs, and LLMs to produce layouts without generating full YAML each time.

### Expression Format

A template expression uses double curly braces:

```yaml
text: "{{title}}"
```

When a data context is provided (via `--data` file or `--define` flag), all `{{expr}}` expressions in string values are replaced. When no data context is provided, template resolution is skipped entirely — existing layouts without expressions are unaffected.

### Expression Types

| Expression | Meaning |
|---|---|
| `{{key}}` | Look up `key` in the data context. Error if missing. |
| `{{key:fallback value}}` | Look up `key`. Use fallback string if missing. First colon separates key from fallback. |

Environment variables are deliberately **not** resolvable from a layout. A
layout file is passive data that may come from another person or system;
letting it name process environment variables would leak them into rendered
PNGs, prints, and debug artifacts. Every value must arrive explicitly via
`--data` or `--define`.

### Example Template

```yaml
# template.yaml
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.45
feedLines: 3
blocks:
  - id: title-1
    type: title
    data:
      text: "{{title}}"
      subtitle: "{{subtitle}}"
  - id: date-1
    type: date
    data:
      date: "{{date}}"
      day: "{{day}}"
  - id: quote-1
    type: quote
    data:
      label: Quote of the Day
      text: "{{quote_text}}"
      author: "{{quote_author}}"
```

### Example Data Context

```yaml
# data.yaml
title: "MORNING SIGNAL"
subtitle: "Coffee, weather, tasks"
date: "May 26, 2026"
day: "Monday"
quote_text: "The only way to do great work is to love what you do."
quote_author: "Steve Jobs"
```

### CLI Usage

```bash
# From file
almanach-render-service render --layout template.yaml --data data.yaml --out /tmp/out.png

# Inline override (comma-separated key=value pairs)
almanach-render-service render --layout template.yaml --define "title=HELLO WORLD" --out /tmp/out.png

# Multiple inline overrides
almanach-render-service render --layout template.yaml --define "title=HELLO WORLD,day=Monday" --out /tmp/out.png

# Mixed: file + override (override wins)
almanach-render-service render --layout template.yaml --data data.yaml --define "title=OVERRIDE" --out /tmp/out.png
```

### Data Context Priority

```
--define flags  >  --data file  >  in-expression fallbacks ({{key:fallback}})
```

The `--define` flag accepts a comma-separated list of `key=value` pairs (e.g. `--define "title=HELLO,day=Monday"`). Values from `--define` always override the same keys from the `--data` file. An in-expression fallback is used only when neither `--data` nor `--define` provides the key.

### Rules

- Template resolution only runs when a data context is provided (`--data` or `--define`).
- Layouts without `{{...}}` expressions are completely unaffected.
- Only string values are processed. Numbers, booleans, and null pass through unchanged.
- An unclosed `{{` without a matching `}}` is an error.
- A missing variable without a fallback is an error with a helpful message.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| A block shows a dashed "Unknown block type" box | Unknown `type`. | Use one of the supported type strings exactly (the block no longer vanishes — it placeholders). |
| `render.threshold must be 0-255` (or similar) | Out-of-range render option. | Render options are validated; use a value in range. |
| Small text loses strokes on paper | Delicate theme font. | Use the `crisp`/`crispsans` themes, raise `weight`, or bump `size`/`minSize`. |
| A photo prints as a black blob | Threshold on a gradient + hot head. | Add `render.rasterMode: RASTER_MODE_ATKINSON`, `gamma: 0.8`, `grayscale: false`, and a lower `printerDensity`. |
| A field renders blank | Wrong data key. | Compare the block against this reference. |
| Template variable not provided | Missing key in data context. | Use `{{key:fallback}}` or provide the key via `--data` or `--define`. |
| `unclosed {{` error | Missing `}}` in a template expression. | Close all expressions or use literal text. |
| YAML parser fails | Unquoted colon or invalid indentation. | Quote the value and use two-space indentation. |
| Page renders but is too tall | Content is too verbose. | Lower `bodyScale` or split into multiple layouts. |
| Shell preview has extra height | `.paper-shell` includes zigzag edges. | Use `.paper-body` for print output. |

## See Also

- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help layouts-user-guide`
- `almanach-render-service help layout-typography-and-rendering`
- `almanach-render-service help tutorial-daily-briefing`
- `almanach-render-service help tutorial-knowledge-strip`
