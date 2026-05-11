# Almanach ZIP Bundle Examples

ZIP bundles keep layout YAML/JSON readable while storing image assets as separate files. The CLI accepts the ZIP directly via `--layout` and inlines relative image paths as data URLs before rendering.

Run from `almanach`:

```bash
./almanach-render-service render \
  --layout examples/bundles/10-sqlite-browser-animals.zip \
  --out /tmp/almanach-sqlite-animals-zip.png \
  --output yaml
```

Examples:

- `10-sqlite-browser-animals/` — readable bundle source directory with `layout.yaml` and `images/*.png`.
- `10-sqlite-browser-animals.zip` — checked-in bundle artifact accepted by the CLI.

To rebuild a ZIP from its source directory, see the bundle-local README.
