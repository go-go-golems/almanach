#!/usr/bin/env python3
"""Create foo-cat ZIP layout bundles for Almanach printer testing.

The script uses the historical foo cat banner assets and creates several ZIP
bundles with relative image paths. The Almanach CLI can print these with:

    go run ./cmd/almanach-render-service print --layout /tmp/foo-cat-6-bundle.zip --printer-ip 192.168.1.242 --output json
"""

from __future__ import annotations

import json
import shutil
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[7]
ASSET_DIR = ROOT / "esp32-s3-m5/ttmp/2026/05/08/ALMANACH-IMAGE-BLOCKS--add-almanach-image-blocks-and-upload-support/various/grid-banners/foo"
OUT_DIR = Path("/tmp")


def build_bundle(count: int, *, light: bool = False) -> Path:
    images = sorted(ASSET_DIR.glob("foo-banner-r*-c01.png"))[:count]
    suffix = f"{count}-light" if light else str(count)
    work = OUT_DIR / f"foo-cat-{suffix}-bundle"
    if work.exists():
        shutil.rmtree(work)
    (work / "images").mkdir(parents=True)

    blocks = [
        {
            "id": "title",
            "type": "title",
            "data": {
                "text": f"FOO CAT {'LIGHT ' if light else ''}NOTES {count}",
                "subtitle": "zip bundle print test",
            },
        }
    ]

    for i, image in enumerate(images, 1):
        name = f"foo-cat-{i:02d}.png"
        shutil.copyfile(image, work / "images" / name)
        data = {
            "label": f"Foo cat plate {i}",
            "src": f"images/{name}",
            "alt": image.name,
            "caption": image.stem,
            "height": 62,
            "fit": "contain",
            "border": True,
            "grayscale": True,
        }
        if light:
            data["thermalTone"] = "light"
        blocks.append({"id": f"img{i}", "type": "image", "data": data})
        blocks.append(
            {
                "id": f"note{i}",
                "type": "note",
                "data": {
                    "label": f"Observation {i}",
                    "text": "A compact cat fact for this longer-but-controlled test print.",
                },
            }
        )

    layout = {"theme": "minimal", "paperWidth": 384, "bodyScale": 1.2, "feedLines": 4, "blocks": blocks}
    (work / "layout.json").write_text(json.dumps(layout, indent=2))

    zip_path = OUT_DIR / f"foo-cat-{suffix}-bundle.zip"
    if zip_path.exists():
        zip_path.unlink()
    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for path in sorted(work.rglob("*")):
            if path.is_file():
                zf.write(path, path.relative_to(work).as_posix())
    return zip_path


def main() -> None:
    for count in (3, 4, 5, 6, 7, 8):
        path = build_bundle(count)
        print(f"{path} {path.stat().st_size} bytes")
    light = build_bundle(6, light=True)
    print(f"{light} {light.stat().st_size} bytes")


if __name__ == "__main__":
    main()
