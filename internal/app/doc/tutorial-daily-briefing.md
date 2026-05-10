---
Title: "Tutorial: Build a Tiny Daily Briefing"
Slug: "tutorial-daily-briefing"
Short: "Build a friendly morning newspaper with weather, plan, news, and a note."
Topics:
- layouts
- tutorial
- daily-briefing
- yaml
Commands:
- render
- inspect
- print
Flags:
- layout
- out
- debug-dir
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial builds a tiny morning newspaper for thermal paper. The layout is intentionally practical: a title, date, weather summary, plan, news headlines, and a note. It is the kind of page you can generate every morning from live data.

The goal is not to fill the page with everything you know. The goal is to make a page that your future self can read while holding a cup of coffee.

## What You Will Build

You will create this block stack:

```text
title -> date -> weather -> plan -> news -> note
```

The design uses `bodyScale: 1.45`, which is a good middle ground for daily output. It is readable on paper but still leaves room for a few sections.

## Step 1 — Write the Layout

Create `morning.yaml`:

```yaml
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.45
feedLines: 3
blocks:
  - id: title-morning
    type: title
    data:
      text: MORNING SIGNAL
      subtitle: Coffee, weather, tasks, and tiny headlines

  - id: date-morning
    type: date
    data:
      date: May 8, 2026
      day: Friday

  - id: weather-morning
    type: weather
    data:
      temp: 18°C
      condition: Crisp and bright
      high: 23°C
      low: 12°C
      sunrise: "05:47"
      sunset: "20:32"

  - id: plan-morning
    type: plan
    data:
      label: Today's Plan
      items:
        - time: "08:30"
          text: Coffee and inbox sweep
          done: true
        - time: "10:00"
          text: Build the layout generator
          done: false
        - time: "14:00"
          text: Print and review the daily card
          done: false
        - time: "17:30"
          text: Write tomorrow's seed note
          done: false

  - id: news-morning
    type: news
    data:
      label: Top News
      items:
        - headline: The almanac printer now speaks fluent YAML.
          source: Desk Wire
          time: now
        - headline: A tiny paper ritual beats a crowded dashboard.
          source: Morning Press
          time: 7m
        - headline: Inspect metrics save another roll of thermal paper.
          source: Render Bureau
          time: 11m

  - id: note-morning
    type: note
    data:
      label: Field Note
      text: Pick three important things. Let the rest wait its turn.
      author: Morning Signal
```

The page begins with identity and context, then moves into actionable information. That order makes the paper useful even when the reader only glances at the top half.

## Step 2 — Render the Preview

Run:

```bash
almanach-render-service render \
  --layout morning.yaml \
  --out /tmp/morning.png \
  --output yaml
```

Check the output width:

```yaml
width: 384
selector: .paper-body
```

If the height is much larger than expected, shorten headlines or lower `bodyScale` to `1.35`.

## Step 3 — Inspect the Render

Run:

```bash
almanach-render-service inspect \
  --layout morning.yaml \
  --output yaml
```

Confirm that `.paper-body`, `.canvas`, and `.workspace` all show `overflow: visible`. This is the CLI's way of telling you Chrome captured the whole paper rather than a clipped scroll container.

## Step 4 — Make It More Fun

Swap the note for a tiny ritual prompt:

```yaml
  - id: note-morning
    type: note
    data:
      label: Tiny Ritual
      text: Before opening chat, write one sentence about what would make today satisfying.
      author: Morning Signal
```

Or add a quote after the note:

```yaml
  - id: quote-morning
    type: quote
    data:
      label: Closing Thought
      text: The secret of getting ahead is getting started.
      author: Mark Twain
```

When adding a block, render again before printing. A single quote can add more height than expected at larger font scales.

## Step 5 — Print

After the preview looks good:

```bash
almanach-render-service print \
  --layout morning.yaml \
  --printer-ip 192.168.0.126 \
  --feed-lines 3 \
  --output yaml
```

The host appends blank raster rows for the feed, so you should get tear-off space after the content.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Plan rows wrap too much | Task text is too long. | Use shorter verb-first tasks. |
| News dominates the page | Too many headlines or long headlines. | Use two headlines or reduce `bodyScale`. |
| Weather right column feels cramped | Long condition/high/low text. | Keep `condition` short and use compact temperatures. |
| Print has too little tear-off space | Feed value too small. | Increase `--feed-lines` to `4` or `6`. |

## See Also

- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help layouts-user-guide`
- `almanach-render-service help layout-dsl-reference`
- `almanach-render-service help tutorial-knowledge-strip`
