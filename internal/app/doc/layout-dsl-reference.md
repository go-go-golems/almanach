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
        text: WRAPPED REQUEST
        subtitle: Layout plus render options
render:
  selector: .paper-body
  threshold: 128
  viewportWidth: 800
  viewportHeight: 3000
```

Use this form when a file should carry both content and render preferences.

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
title, date, divider, plan, news, weather, note, habits, mood,
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

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| A block disappears | Unknown `type`. | Use one of the supported type strings exactly. |
| A field renders blank | Wrong data key. | Compare the block against this reference. |
| YAML parser fails | Unquoted colon or invalid indentation. | Quote the value and use two-space indentation. |
| Page renders but is too tall | Content is too verbose. | Lower `bodyScale` or split into multiple layouts. |
| Shell preview has extra height | `.paper-shell` includes zigzag edges. | Use `.paper-body` for print output. |

## See Also

- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help layouts-user-guide`
- `almanach-render-service help tutorial-daily-briefing`
- `almanach-render-service help tutorial-knowledge-strip`
