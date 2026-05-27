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
| `theme` | string | no | Theme key. Use `minimal` for thermal output. |
| `paperWidth` | integer | no | Paper width in pixels. Use `384`. |
| `bodyScale` | number | no | Font scale from `1.0` to `2.0`. |
| `feedLines` | integer | no | Blank trailing feed rows expressed as line units. |
| `blocks` | array | yes | Ordered block list. |

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

These fields are valid under the wrapped `render` object. Command-line flags can also supply these values.

| Field | Type | Default | Description |
|---|---|---:|---|
| `selector` | string | `.paper-body` in CLI | CSS selector to screenshot. |
| `threshold` | integer | `128` | Grayscale threshold for bitmap conversion. |
| `viewportWidth` | integer | `800` | Browser viewport width. |
| `viewportHeight` | integer | `3000` | Browser viewport height. |

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
| `type` | string | yes | One of the supported block types. |
| `data` | object | yes | Type-specific fields. |

Supported block types:

```text
title, date, divider, plan, news, weather, note, image, habits, mood,
reading, reflection, quote, word, history, did
```

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
| `{{$ENV_VAR}}` | Look up environment variable `ENV_VAR`. Error if unset. |
| `{{$ENV_VAR:fallback}}` | Look up environment variable. Use fallback if unset. |

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
--define flags  >  --data file  >  environment variable fallbacks in expressions
```

The `--define` flag accepts a comma-separated list of `key=value` pairs (e.g. `--define "title=HELLO,day=Monday"`). Values from `--define` always override the same keys from the `--data` file. Environment variable fallbacks (`{{$ENV_VAR:fallback}}`) are used only when neither `--data` nor `--define` provides a value for that key.

### Rules

- Template resolution only runs when a data context is provided (`--data` or `--define`).
- Layouts without `{{...}}` expressions are completely unaffected.
- Only string values are processed. Numbers, booleans, and null pass through unchanged.
- An unclosed `{{` without a matching `}}` is an error.
- A missing variable without a fallback is an error with a helpful message.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| A block disappears | Unknown `type`. | Use one of the supported type strings exactly. |
| A field renders blank | Wrong data key. | Compare the block against this reference. |
| Template variable not provided | Missing key in data context. | Use `{{key:fallback}}` or provide the key via `--data` or `--define`. |
| `unclosed {{` error | Missing `}}` in a template expression. | Close all expressions or use literal text. |
| YAML parser fails | Unquoted colon or invalid indentation. | Quote the value and use two-space indentation. |
| Page renders but is too tall | Content is too verbose. | Lower `bodyScale` or split into multiple layouts. |
| Shell preview has extra height | `.paper-shell` includes zigzag edges. | Use `.paper-body` for print output. |

## See Also

- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help layouts-user-guide`
- `almanach-render-service help tutorial-daily-briefing`
- `almanach-render-service help tutorial-knowledge-strip`
