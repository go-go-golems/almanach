---
Title: "Tutorial: Make a Curious Knowledge Strip"
Slug: "tutorial-knowledge-strip"
Short: "Build a playful word-history-facts layout that feels like a tiny paper museum card."
Topics:
- layouts
- tutorial
- knowledge
- yaml
Commands:
- render
- inspect
- print
Flags:
- layout
- out
- selector
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial builds a playful knowledge strip: one word, a few historical events, several facts, and a closing quote. It is a good layout for automated daily generation because each section can come from a different data source.

The design works best when every fact is short. Think museum label, not encyclopedia entry.

## What You Will Build

The stack is:

```text
title -> date -> word -> history -> did -> quote
```

The page uses `bodyScale: 1.35` because knowledge content tends to be text-heavy. Lower scale keeps the page readable without becoming too tall.

## Step 1 — Create the Layout

Create `curiosity.yaml`:

```yaml
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.35
feedLines: 3
blocks:
  - id: title-curiosity
    type: title
    data:
      text: CURIOSITY STRIP
      subtitle: One word, three moments, and odd facts

  - id: date-curiosity
    type: date
    data:
      date: May 8, 2026
      day: Friday

  - id: word-curiosity
    type: word
    data:
      label: Word of the Day
      word: petrichor
      phonetic: pe-tri-kor
      part: noun
      definition: The pleasant smell that accompanies the first rain after warm, dry weather.
      example: The sidewalk released petrichor as the storm arrived.

  - id: history-curiosity
    type: history
    data:
      label: Today in History
      items:
        - year: "1886"
          event: Coca-Cola is first sold at Jacob's Pharmacy in Atlanta.
        - year: "1945"
          event: Victory in Europe Day is celebrated after Germany's surrender.
        - year: "1970"
          event: The Beatles release their final studio album, Let It Be.

  - id: did-curiosity
    type: did
    data:
      label: Did You Know?
      items:
        - Honey never spoils when stored properly.
        - A day on Venus is longer than a Venusian year.
        - Octopuses have three hearts and blue blood.

  - id: quote-curiosity
    type: quote
    data:
      label: Closing Thought
      text: The cure for boredom is curiosity. There is no cure for curiosity.
      author: Dorothy Parker
```

This layout demonstrates the most important rule for generated knowledge pages: each section gets only a few items. Three history events and three facts are plenty for 384px paper.

## Step 2 — Render and Review

Run:

```bash
almanach-render-service render \
  --layout curiosity.yaml \
  --out /tmp/curiosity.png \
  --output yaml
```

Open `/tmp/curiosity.png`. Check that:

- the word block has a large readable word,
- history events wrap without colliding,
- fact bullets are separated,
- the quote still has breathing room.

If the page is too tall, remove one history event before lowering the font scale. Content editing usually beats shrinking text.

## Step 3 — Try a Paper-Shell Preview

Most print output uses `.paper-body`, but a knowledge strip can look nice with the zigzag paper edge in a PNG preview:

```bash
almanach-render-service render \
  --layout curiosity.yaml \
  --selector .paper-shell \
  --out /tmp/curiosity-shell.png
```

Use `.paper-shell` for sharing an image preview. Use `.paper-body` for actual print-oriented bitmap output.

## Step 4 — Generate This Layout from Data

A script or LLM can fill this structure with fresh data each day. The generator should produce only the data fields, not CSS or HTML.

Pseudo-code:

```text
word = fetch_word_of_day()
history = fetch_three_short_events(today)
facts = choose_three_short_facts()
quote = choose_quote()

layout = {
  theme: "minimal",
  paperWidth: 384,
  bodyScale: 1.35,
  blocks: [title, date, word, history, did, quote]
}
write_yaml(layout)
```

Keep a length budget in your generator:

| Section | Budget |
|---|---:|
| Word definition | 120 characters |
| History event | 95 characters |
| Fact | 80 characters |
| Quote | 120 characters |

These limits are not strict, but they keep the strip graceful.

## Step 5 — Print

```bash
almanach-render-service print \
  --layout curiosity.yaml \
  --printer-ip 192.168.0.126 \
  --feed-lines 3 \
  --output yaml
```

If you plan to print a daily knowledge strip automatically, render a PNG copy to an archive directory first. That gives you a visual record of what was sent to paper.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| History section is cramped | Events are too long. | Use fewer events or shorter summaries. |
| Word definition wraps too much | Definition is dictionary-length. | Rewrite it as one concise sentence. |
| Quote pushes page too tall | Quote is too long. | Choose a shorter quote or remove the quote block. |
| Paper-shell preview is taller than body preview | Shell includes decorative zigzag edges. | This is expected; print with `.paper-body`. |

## See Also

- `almanach-render-service help layouts-getting-started`
- `almanach-render-service help layouts-user-guide`
- `almanach-render-service help layout-dsl-reference`
- `almanach-render-service help tutorial-daily-briefing`
