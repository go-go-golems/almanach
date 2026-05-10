# SQLite Browser Animals ZIP Bundle

This example demonstrates Almanach ZIP layout bundle support. The layout keeps
image sources as short relative paths, while the CLI inlines those image members
as data URLs before rendering.

Render the checked-in ZIP bundle:

```bash
./almanach-render-service render \
  --layout ./examples/bundles/10-sqlite-browser-animals.zip \
  --out /tmp/almanach-sqlite-animals-zip.png \
  --output yaml
```

Rebuild the ZIP from the readable source directory:

```bash
python3 - <<'PY'
from pathlib import Path
import zipfile
root = Path('examples/bundles/10-sqlite-browser-animals')
out = Path('examples/bundles/10-sqlite-browser-animals.zip')
with zipfile.ZipFile(out, 'w', compression=zipfile.ZIP_DEFLATED) as zf:
    for path in sorted(root.rglob('*')):
        if path.is_file():
            info = zipfile.ZipInfo(path.relative_to(root).as_posix())
            info.date_time = (2026, 5, 8, 12, 0, 0)
            info.external_attr = 0o644 << 16
            zf.writestr(info, path.read_bytes(), compress_type=zipfile.ZIP_DEFLATED)
PY
```
