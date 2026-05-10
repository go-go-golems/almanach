---
Title: "Almanach Layout User Guide"
Slug: "layouts-user-guide"
Short: "Design readable thermal pages with blocks, scale, selectors, themes, and preview-first workflows."
Topics:
- layouts
- design
- thermal-printer
- yaml
Commands:
- render
- inspect
- print
Flags:
- layout
- selector
- feed-lines
- debug-dir
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

An Almanach layout is a tiny document design language for 384px thermal paper. It is not a general HTML template system. It is a structured list of known block types that Almanach Studio can render consistently in a browser and that the CLI can convert into printer-ready monochrome bitmaps.

This user guide explains how to think like a layout author: choose the right blocks, keep text short, control scale, preview output, and design pages that look deliberate rather than cramped.

## The Mental Model

A layout has two layers:

1. **Page settings** decide the global paper shape and style.
2. **Blocks** decide what content appears from top to bottom.

```text
layout
  theme: minimal
  paperWidth: 384
  bodyScale: 1.45
  feedLines: 3
  blocks:
    - title
    - date
    - weather
    - plan
    - quote
```

The renderer loads the layout into the same React components used by Almanach Studio. That means a layout file should describe content, not pixel-perfect placement. The block renderers handle typography, spacing, icons, boxes, and rules.

## Page Settings

Use page settings to control the overall feel before tuning individual blocks.

| Field | Recommended value | What it controls |
|---|---:|---|
| `theme` | `minimal` | The visual theme. Current thermal workflow expects black on white. |
| `paperWidth` | `384` | Pixel width for a 58mm thermal printer. |
| `bodyScale` | `1.2` to `1.6` | Text size multiplier. Larger is more readable but taller. |
| `feedLines` | `3` to `6` | Blank paper after printing for tear-off. |

Start with `bodyScale: 1.45` for daily pages. Use `1.6` for short cards and `1.25` for dense tracker pages.

## Block Order

A good thermal page has a clear rhythm. Readers scan vertically, so put the most identifying information first and the most optional information last.

A reliable order is:

```yaml
blocks:
  - type: title
  - type: date
  - type: weather
  - type: plan
  - type: news
  - type: quote
```

For a knowledge page, use:

```yaml
blocks:
  - type: title
  - type: date
  - type: word
  - type: history
  - type: did
  - type: quote
```

For a personal tracker, use:

```yaml
blocks:
  - type: title
  - type: date
  - type: habits
  - type: mood
  - type: reading
  - type: reflection
```

## Writing for 384px Paper

Thermal layouts reward short writing. Long prose wraps aggressively and makes pages tall. Prefer crisp, scannable content.

Good headlines:

```yaml
headline: Inspect command reports paper-body and canvas metrics.
```

Too long:

```yaml
headline: The newly implemented inspect command in the Almanach Render Service reports paper-body and canvas metrics so developers can determine whether the capture selector is responsible for clipping problems.
```

For most blocks:

- Keep titles under 24 characters.
- Keep news headlines under 80 characters.
- Keep plan items under one line when possible.
- Keep notes to one or two sentences.
- Use fewer blocks at larger `bodyScale` values.

## Choosing a Selector

The render command captures a CSS selector from the browser page.

| Selector | Use when | Includes |
|---|---|---|
| `.paper-body` | Printing or bitmap generation. | Content area only. |
| `.paper-shell` | Decorative preview PNGs. | Zigzag top/bottom shell edges plus body. |

Most layouts should render with the default `.paper-body`:

```bash
almanach-render-service render --layout daily.yaml --out daily.png
```

Use `.paper-shell` when you want a visual preview that includes paper-edge decoration:

```bash
almanach-render-service render \
  --layout daily.yaml \
  --selector .paper-shell \
  --out daily-shell.png
```

## Wrapped Render Requests

A raw layout file describes only the page. A wrapped request can carry layout and render preferences together:

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

Use wrapped requests when sharing examples that should always render with a particular selector. Use raw layouts when you want command-line flags to be the main source of render behavior.

## Preview-First Workflow

The safest workflow is:

```bash
# 1. Render a PNG.
almanach-render-service render --layout daily.yaml --out /tmp/daily.png

# 2. Inspect metrics.
almanach-render-service inspect --layout daily.yaml --output yaml

# 3. Optional dry-run print.
almanach-render-service print --layout daily.yaml --dry-run --output yaml

# 4. Physical print.
almanach-render-service print --layout daily.yaml --printer-ip 192.168.0.126
```

This workflow prevents two common mistakes: printing editor UI by accident and printing a clipped layout.

## Feed Lines and Tear-Off Space

`--feed-lines` adds blank raster rows to the bottom of the bitmap before the host sends it to the printer. This is more reliable than sending a separate feed command after a bitmap.

Practical values:

| Value | Use for |
|---:|---|
| `0` | No added blank space. |
| `3` | Normal tear-off spacing. |
| `6` | Extra spacing for short cards. |

The command still reports `feed_lines` in its structured output, and the printed bitmap height includes the appended blank rows.

## Style Recipes

### Short Card

Use a high body scale and a few blocks:

```yaml
bodyScale: 1.6
blocks:
  - type: title
  - type: date
  - type: quote
```

### Daily Briefing

Use medium scale and practical blocks:

```yaml
bodyScale: 1.35
blocks:
  - type: title
  - type: date
  - type: weather
  - type: plan
  - type: news
```

### Dense Tracker

Use lower scale and compact content:

```yaml
bodyScale: 1.25
blocks:
  - type: title
  - type: date
  - type: habits
  - type: mood
  - type: reading
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Page is readable in PNG but too faint on paper | Antialiasing in PNG does not represent final 1-bit bitmap exactly. | Print a short test or render bitmap output for printer path validation. |
| Layout is too tall | Too many blocks or `bodyScale` too high. | Lower `bodyScale`, remove blocks, or shorten text. |
| A block is missing | Unknown `type` or malformed `data`. | Check `layout-dsl-reference` for exact block type names and fields. |
| YAML parsing changes a value | Unquoted date/time-like strings. | Quote dates, times, and strings with colons. |
| Feed does not seem to work | Old post-bitmap ESC feed path was unreliable. | Use current CLI `print`; it bakes feed into blank raster rows. |

## See Also

- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help layout-dsl-reference`
- `almanach-render-service help tutorial-daily-briefing`
- `almanach-render-service help tutorial-knowledge-strip`
