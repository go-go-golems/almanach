# Rendered Layout Example Previews

These PNG files were generated from `../layouts/*.yaml` with the local CLI renderer.

Command pattern:

```bash
for layout in examples/layouts/[0-9][0-9]-*.yaml; do
  name=$(basename "$layout" .yaml)
  ./almanach-render-service render \
    --layout "$layout" \
    --out "examples/rendered/${name}.png" \
    --output yaml > "examples/rendered/${name}.render.yaml"
  ./almanach-render-service inspect \
    --layout "$layout" \
    --output yaml > "examples/rendered/${name}.inspect.yaml"
done
```

Validation summary from the initial render pass:

| Layout | PNG size | Selector | Notes |
|---|---:|---|---|
| `01-minimal` | `384x343` | `.paper-body` | Minimal title/date/quote smoke test. |
| `02-daily-briefing` | `384x829` | `.paper-body` | Common weather/plan/news/note daily page. |
| `03-knowledge-strip` | `384x955` | `.paper-body` | Word/history/did/quote content renders cleanly. |
| `04-tracker-journal` | `384x1044` | `.paper-body` | Habits/mood/reading/reflection blocks render without clipping. |
| `05-wrapped-render-request` | `384x607` | `.paper-body` | Wrapped `{ layout, render }` YAML works. |
| `06-paper-shell-preview` | `384x465` | `.paper-shell` | Includes decorative zigzag paper shell edges. |

All inspect outputs reported `overflow: visible` for `.paper-shell`, `.paper-body`, `.canvas`, `.workspace`, and `.almanach-app`, and all rendered widths were `384px`.

`contact-sheet.png` is a visual overview of all six rendered examples.
