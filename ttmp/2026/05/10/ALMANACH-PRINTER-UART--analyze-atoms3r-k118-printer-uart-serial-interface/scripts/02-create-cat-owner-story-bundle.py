#!/usr/bin/env python3
"""Split a 4x4 cat collage into portraits and create a story ZIP bundle.

Input defaults to the clipboard image used during the investigation:

    /tmp/pi-clipboard-684daa14-7016-4dfe-bdf6-432e3fc0d1b1.png

The collage is 4 columns x 4 rows, with thin red guide lines. We crop inside each
cell by a small margin to remove those guide lines and preserve square portraits.

Outputs:

    /tmp/pi-cat-portraits/cat-portrait-R-C.png
    /tmp/cat-owner-story-bundle.zip

Print with:

    go run ./cmd/almanach-render-service print --layout /tmp/cat-owner-story-bundle.zip --printer-ip 192.168.1.242 --output json
"""

from __future__ import annotations

import argparse
import json
import shutil
import zipfile
from pathlib import Path

from PIL import Image

DEFAULT_INPUT = Path("/tmp/pi-clipboard-684daa14-7016-4dfe-bdf6-432e3fc0d1b1.png")
DEFAULT_PORTRAIT_DIR = Path("/tmp/pi-cat-portraits")
DEFAULT_WORK_DIR = Path("/tmp/cat-owner-story-bundle")
DEFAULT_ZIP = Path("/tmp/cat-owner-story-bundle.zip")

STORIES = [
    (
        "Miso keeps the morning schedule",
        "Miso believes every owner needs a punctual supervisor. At seven, she sits by the kettle and blinks until breakfast becomes official.",
    ),
    (
        "Juniper audits the desk",
        "Juniper inspects notebooks, keyboards, and every unattended pen. Her owner calls it chaos; Juniper calls it document control.",
    ),
    (
        "Toast guards the window",
        "Toast watches pigeons with the patience of a tiny astronomer. His owner narrates the sightings, and both pretend this is research.",
    ),
    (
        "Luna closes the day",
        "Luna waits until the room goes quiet, then curls beside the nearest hand. Her owner finally stops working, which was Luna's plan all along.",
    ),
]


def split_grid(src: Path, out_dir: Path, grid: int = 4, inset: int = 10) -> list[Path]:
    image = Image.open(src).convert("RGBA")
    width, height = image.size
    cell_w = width / grid
    cell_h = height / grid
    cell = int(min(cell_w, cell_h))
    out_dir.mkdir(parents=True, exist_ok=True)

    portraits: list[Path] = []
    for row in range(grid):
        for col in range(grid):
            left = int(round(col * cell_w)) + inset
            top = int(round(row * cell_h)) + inset
            right = int(round((col + 1) * cell_w)) - inset
            bottom = int(round((row + 1) * cell_h)) - inset
            crop = image.crop((left, top, right, bottom))
            # Normalize to a square so contain/cover behavior is predictable.
            crop = crop.resize((cell, cell), Image.Resampling.LANCZOS)
            path = out_dir / f"cat-portrait-{row + 1}-{col + 1}.png"
            crop.save(path)
            portraits.append(path)
    return portraits


def create_story_bundle(portraits: list[Path], work_dir: Path, zip_path: Path) -> None:
    if work_dir.exists():
        shutil.rmtree(work_dir)
    (work_dir / "images").mkdir(parents=True)

    # Use three diagonal-ish portraits so each one can print larger while keeping
    # the final bitmap below the current 90 KiB single-request limit.
    selected = [portraits[i] for i in [0, 5, 10]]
    blocks = [
        {
            "id": "title",
            "type": "title",
            "data": {
                "text": "CATS AND THEIR PEOPLE",
                "subtitle": "three large portraits from the household bureau",
            },
        }
    ]

    for i, (portrait, (label, text)) in enumerate(zip(selected, STORIES), 1):
        name = f"portrait-{i}.png"
        shutil.copyfile(portrait, work_dir / "images" / name)
        blocks.append(
            {
                "id": f"img{i}",
                "type": "image",
                "data": {
                    "label": label,
                    "src": f"images/{name}",
                    "alt": label,
                    "caption": portrait.stem,
                    "height": 196,
                    "fit": "contain",
                    "border": False,
                    "grayscale": True,
                    "thermalTone": "light",
                },
            }
        )
        blocks.append({"id": f"note{i}", "type": "note", "data": {"label": "Story note", "text": text}})

    blocks.append(
        {
            "id": "quote",
            "type": "quote",
            "data": {
                "quote": "A cat owns the room by needing nothing from it, then asking for everything.",
                "author": "Almanach cat desk",
            },
        }
    )

    layout = {"theme": "minimal", "paperWidth": 384, "bodyScale": 1.52, "feedLines": 4, "blocks": blocks}
    (work_dir / "layout.json").write_text(json.dumps(layout, indent=2))

    if zip_path.exists():
        zip_path.unlink()
    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for path in sorted(work_dir.rglob("*")):
            if path.is_file():
                zf.write(path, path.relative_to(work_dir).as_posix())


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, default=DEFAULT_INPUT)
    parser.add_argument("--portrait-dir", type=Path, default=DEFAULT_PORTRAIT_DIR)
    parser.add_argument("--work-dir", type=Path, default=DEFAULT_WORK_DIR)
    parser.add_argument("--zip", type=Path, default=DEFAULT_ZIP)
    args = parser.parse_args()

    if args.portrait_dir.exists():
        shutil.rmtree(args.portrait_dir)
    portraits = split_grid(args.input, args.portrait_dir)
    create_story_bundle(portraits, args.work_dir, args.zip)
    print(f"split {len(portraits)} portraits into {args.portrait_dir}")
    print(f"created {args.zip} ({args.zip.stat().st_size} bytes)")


if __name__ == "__main__":
    main()
