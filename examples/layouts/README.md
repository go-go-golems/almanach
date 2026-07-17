# Almanach CLI Layout Examples

These YAML files are inputs for the `almanach-render-service render`, `inspect`, and `print` CLI verbs. ZIP bundle examples with relative image assets live in `../bundles/`.

Run from `almanach`:

```bash
make build

./almanach-render-service render \
  --layout examples/layouts/01-minimal.yaml \
  --out /tmp/almanach-minimal.png \
  --output yaml

./almanach-render-service inspect \
  --layout examples/layouts/01-minimal.yaml \
  --output yaml
```

Examples:

- `01-minimal.yaml` — minimal title/date/quote smoke test.
- `02-daily-briefing.yaml` — daily weather, plan, news, and note blocks.
- `03-knowledge-strip.yaml` — word, history, did-you-know, divider, and quote blocks.
- `04-tracker-journal.yaml` — habits, mood, reading, and reflection blocks.
- `05-wrapped-render-request.yaml` — supported `{ layout, render }` wrapper shape.
- `06-paper-shell-preview.yaml` — `.paper-shell` capture for decorative zigzag preview.
- `07-analog-photography.yaml` — analog photography themed field card.
- `08-image-block.yaml` — embedded image-block smoke test.
- `09-picocalc-uf2-nerd-card.yaml` — PicoCalc UF2 loader technical card with embedded SVG.

Work-slip examples (ALMANACH-WORKSLIP; generic layout primitives + the
`swiss`/`brutalist`/`terminal` work themes, `bodyScale: 1`; pre-expanded — the
producer fills in concrete values, there is no binding language):

- `10-job-slip.yaml` — one incoming job at a glance (banner, h1 title, rate row).
- `11-decision-sheet.yaml` — summary, market facts (kv), star/skip decision checks.
- `12-triage-card.yaml` — fit rating, action checkboxes, write-in notes, QR link (brutalist).
- `13-focus-card.yaml` — daily one-thing + time slots (terminal/mono).
- `14-morning-digest.yaml` — numbered digest of the day's new jobs (a pre-expanded "repeat").

Use `.paper-body` for print-oriented output and `.paper-shell` when you explicitly want the decorative zigzag edges included in the preview image. For layouts with separate image files, use a ZIP bundle from `examples/bundles/`.
