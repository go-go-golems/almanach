#!/usr/bin/env python3
"""Create a ZIP layout bundle that prints one large cat portrait.

Defaults use a corrected 4x4 portrait crop stored in this ticket by
`04-extract-cat-portraits.py`:

    ../assets/cat-portraits/portraits/cat-portrait-r02-c02.png

Output:

    /tmp/single-large-cat-bundle.zip

Print with:

    go run ./cmd/almanach-render-service print --layout /tmp/single-large-cat-bundle.zip --printer-ip 192.168.1.242 --output json
"""

from __future__ import annotations

import argparse
import json
import shutil
import zipfile
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
TICKET_DIR = SCRIPT_DIR.parent
DEFAULT_IMAGE = TICKET_DIR / "assets" / "cat-portraits" / "portraits" / "cat-portrait-r02-c02.png"
DEFAULT_WORK_DIR = Path("/tmp/single-large-cat-bundle")
DEFAULT_ZIP = Path("/tmp/single-large-cat-bundle.zip")


def create_bundle(image: Path, work_dir: Path, zip_path: Path, height: int) -> None:
    if work_dir.exists():
        shutil.rmtree(work_dir)
    (work_dir / "images").mkdir(parents=True)

    image_name = "large-cat.png"
    shutil.copyfile(image, work_dir / "images" / image_name)

    layout = {
        "theme": "minimal",
        "paperWidth": 384,
        "bodyScale": 1.25,
        "feedLines": 4,
        "blocks": [
            {
                "id": "cat",
                "type": "image",
                "data": {
                    "label": "Large Cat Portrait",
                    "src": f"images/{image_name}",
                    "alt": "Large cat portrait",
                    "caption": "",
                    "height": height,
                    "fit": "contain",
                    "border": False,
                    "grayscale": True,
                    "thermalTone": "light",
                },
            }
        ],
    }
    (work_dir / "layout.json").write_text(json.dumps(layout, indent=2))

    if zip_path.exists():
        zip_path.unlink()
    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for path in sorted(work_dir.rglob("*")):
            if path.is_file():
                zf.write(path, path.relative_to(work_dir).as_posix())


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--image", type=Path, default=DEFAULT_IMAGE)
    parser.add_argument("--work-dir", type=Path, default=DEFAULT_WORK_DIR)
    parser.add_argument("--zip", type=Path, default=DEFAULT_ZIP)
    parser.add_argument("--height", type=int, default=360)
    args = parser.parse_args()

    create_bundle(args.image, args.work_dir, args.zip, args.height)
    print(f"created {args.zip} ({args.zip.stat().st_size} bytes) from {args.image} at height={args.height}")


if __name__ == "__main__":
    main()
