---
Title: "Getting Started with Almanach Layouts"
Slug: "layouts-getting-started"
Short: "Create your first YAML layout, render it to PNG, inspect it for clipping, and print it safely."
Topics:
- layouts
- yaml
- rendering
- printing
Commands:
- render
- inspect
- print
Flags:
- layout
- out
- selector
- debug-dir
- printer-ip
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Almanach layouts are small YAML or JSON documents that describe a thermal paper page. You write the content as blocks, pass the file to `almanach-render-service render`, and the CLI opens Almanach Studio in headless Chrome to produce the same visual output you see in the browser editor.

This guide gets you from an empty directory to a rendered PNG. It also shows the safe review loop: render first, inspect metrics second, print only after the preview looks correct.

## What You Will Build

You will build a 384px-wide thermal page with a title, date strip, weather block, plan, and closing note. The result is suitable for the M5Stack K118 thermal printer used by the stoms3r firmware.

The workflow is:

```text
layout.yaml -> render command -> PNG preview -> inspect metrics -> optional print
```

This loop matters because thermal paper is unforgiving. A bad selector, clipped canvas, or oversized layout wastes paper. The CLI gives you a fast way to catch those problems before printing.

## Prerequisites

Before starting, make sure the binary can run locally:

```bash
cd almanach
go build -o almanach-render-service .
./almanach-render-service --help
```

You also need Chrome or Chromium available on the host. If Chrome is not auto-detected, pass `--chrome-path` or set `ALMANACH_CHROME_PATH`.

## Step 1 — Create a Layout File

Create `daily.yaml`:

```yaml
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.45
feedLines: 3
blocks:
  - id: title-1
    type: title
    data:
      text: DAILY BRIEFING
      subtitle: A tiny newspaper for your desk

  - id: date-1
    type: date
    data:
      date: May 8, 2026
      day: Friday

  - id: weather-1
    type: weather
    data:
      temp: 18°C
      condition: Clear morning, light breeze
      high: 23°C
      low: 12°C
      sunrise: "05:47"
      sunset: "20:32"

  - id: plan-1
    type: plan
    data:
      label: Today's Plan
      items:
        - time: "08:30"
          text: Morning review and coffee
          done: true
        - time: "10:00"
          text: Deep work on the almanac
          done: false
        - time: "17:00"
          text: Notes, commit, and handoff
          done: false

  - id: note-1
    type: note
    data:
      label: Daily Note
      text: Preview first, then print. The paper remembers every mistake.
      author: Almanach Studio
```

A layout has page-level settings and a `blocks` list. Each block has an `id`, a `type`, and a `data` object. The block `type` selects the React renderer, and `data` supplies the fields that renderer expects.

Quote times such as `"08:30"` and `"05:47"`. YAML otherwise may treat some colon-containing values as special scalars.

## Step 2 — Render a PNG Preview

Run:

```bash
./almanach-render-service render \
  --layout daily.yaml \
  --out /tmp/daily.png \
  --output yaml
```

Expected output looks like this:

```yaml
artifact: /tmp/daily.png
format: png
selector: .paper-body
width: 384
height: 700
threshold: 128
```

The exact height depends on your content. The width should be `384` for the K118 printer. If the width is not `384`, check `paperWidth` in the layout.

## Step 3 — Inspect for Clipping

Run:

```bash
./almanach-render-service inspect \
  --layout daily.yaml \
  --output yaml
```

Look for these selectors:

- `.paper-body`
- `.paper-shell`
- `.canvas`
- `.workspace`
- `.almanach-app`

For normal CLI rendering, they should report `overflow: visible`. If `.canvas` or `.workspace` reports `overflow: hidden`, the render capture CSS did not apply correctly and the output may be cut off.

## Step 4 — Save Debug Artifacts

When something looks wrong, render with a debug directory:

```bash
./almanach-render-service render \
  --layout daily.yaml \
  --out /tmp/daily.png \
  --debug-dir /tmp/almanach-debug \
  --output yaml
```

The debug directory contains:

| File | What it tells you |
|---|---|
| `screenshot.png` | The exact PNG captured from Chrome. |
| `bitmap.bin` | The packed 1-bit bitmap used by bitmap/print paths. |
| `layout.json` | The normalized layout sent into the SPA. |
| `metrics.json` | DOM measurements after render-mode CSS was applied. |

Debug artifacts are the fastest way to determine whether a problem is in the YAML, the selector, the browser render, or the printer.

## Step 5 — Print After Preview Looks Correct

When the PNG looks right, print:

```bash
./almanach-render-service print \
  --layout daily.yaml \
  --printer-ip 192.168.0.126 \
  --feed-lines 3 \
  --output yaml
```

The print command renders again, converts the PNG to a 1-bit bitmap, appends trailing blank rows according to `--feed-lines`, and posts the bitmap to the ESP32 firmware.

Use dry-run when you want to test the render path without paper:

```bash
./almanach-render-service print \
  --layout daily.yaml \
  --dry-run \
  --output yaml
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| PNG width is not 384px | `paperWidth` is missing or changed. | Set `paperWidth: 384`. |
| Render fails with `layout.blocks must be an array` | The YAML file is not a raw layout or wrapped request. | Add a top-level `blocks:` list or use `layout: { blocks: ... }`. |
| Printed output has no tear-off space | Feed is too small. | Increase `--feed-lines`; the host appends blank raster rows. |
| Text is too large and page is very tall | `bodyScale` is high or content is too verbose. | Try `bodyScale: 1.25` to `1.45` and shorten blocks. |
| Output looks clipped | Wrong selector or capture CSS failure. | Run `inspect` and check overflow/height metrics. |

## Using Templates

If you want to generate the same layout structure every day with different content, use a template. A template is a layout file where some field values contain `{{variable}}` expressions instead of literal text.

Create a template (`morning.yaml`):

```yaml
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
  - id: note-1
    type: note
    data:
      label: Daily Note
      text: "{{note_text}}"
```

Create a data context (`data.yaml`):

```yaml
title: "MORNING SIGNAL"
subtitle: "Coffee and tasks"
date: "May 26, 2026"
day: "Monday"
note_text: "Preview first, then print."
```

Render with both:

```bash
./almanach-render-service render \
  --layout morning.yaml \
  --data data.yaml \
  --out /tmp/morning.png
```

You can also override individual values inline:

```bash
./almanach-render-service render \
  --layout morning.yaml \
  --data data.yaml \
  --define "title=OVERRIDE" \
  --out /tmp/morning.png
```

When no `--data` or `--define` is provided, template expressions are skipped. Existing layouts without `{{...}}` work unchanged.

See `almanach-render-service help layout-dsl-reference` for the full template expression reference.

## See Also

- `almanach-render-service help layouts-user-guide`
- `almanach-render-service help layout-dsl-reference`
- `almanach-render-service help tutorial-daily-briefing`
- `almanach-render-service help tutorial-knowledge-strip`
