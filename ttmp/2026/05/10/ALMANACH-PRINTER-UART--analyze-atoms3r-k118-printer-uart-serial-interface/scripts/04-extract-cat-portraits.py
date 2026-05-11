#!/usr/bin/env python3
"""Extract a 4x4 cat collage into verified square portraits.

This script preserves the source collage and generated portraits inside the
ALMANACH-PRINTER-UART ticket so the extraction survives /tmp cleanup or session
crashes.

Default input is the refreshed clipboard image from the crash-recovery turn:

    /tmp/pi-clipboard-bcf0fca6-54a3-4349-a272-570e6afe0793.png

Default output:

    ../assets/cat-portraits/source-4x4-cat-collage.png
    ../assets/cat-portraits/portraits/cat-portrait-rRR-cCC.png
    ../assets/cat-portraits/contact-sheet.png
    ../assets/cat-portraits/manifest.md
"""

from __future__ import annotations

import argparse
import shutil
from pathlib import Path

from PIL import Image, ImageDraw

SCRIPT_DIR = Path(__file__).resolve().parent
TICKET_DIR = SCRIPT_DIR.parent
DEFAULT_INPUT = Path("/tmp/pi-clipboard-bcf0fca6-54a3-4349-a272-570e6afe0793.png")
DEFAULT_OUTPUT_DIR = TICKET_DIR / "assets" / "cat-portraits"


def extract_portraits(src: Path, output_dir: Path, grid: int = 4, inset: int = 10) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    source_copy = output_dir / "source-4x4-cat-collage.png"
    shutil.copyfile(src, source_copy)

    portrait_dir = output_dir / "portraits"
    portrait_dir.mkdir(parents=True, exist_ok=True)
    for old in portrait_dir.glob("cat-portrait-*.png"):
        old.unlink()

    img = Image.open(source_copy).convert("RGBA")
    width, height = img.size
    cell_w = width / grid
    cell_h = height / grid
    cell = int(min(cell_w, cell_h))

    portraits: list[tuple[Path, tuple[int, int, int, int], tuple[int, int]]] = []
    for row in range(grid):
        for col in range(grid):
            left = int(round(col * cell_w)) + inset
            top = int(round(row * cell_h)) + inset
            right = int(round((col + 1) * cell_w)) - inset
            bottom = int(round((row + 1) * cell_h)) - inset
            crop = img.crop((left, top, right, bottom)).resize((cell, cell), Image.Resampling.LANCZOS)
            path = portrait_dir / f"cat-portrait-r{row + 1:02d}-c{col + 1:02d}.png"
            crop.save(path)
            portraits.append((path, (left, top, right, bottom), crop.size))

    thumb = 180
    label_h = 20
    sheet = Image.new("RGB", (grid * thumb, grid * (thumb + label_h)), "white")
    draw = ImageDraw.Draw(sheet)
    for index, (path, _, _) in enumerate(portraits):
        row, col = divmod(index, grid)
        im = Image.open(path).convert("RGB").resize((thumb, thumb), Image.Resampling.LANCZOS)
        x = col * thumb
        y = row * (thumb + label_h)
        sheet.paste(im, (x, y))
        draw.text((x + 4, y + thumb + 3), path.stem.replace("cat-portrait-", ""), fill=(0, 0, 0))
    contact_sheet = output_dir / "contact-sheet.png"
    sheet.save(contact_sheet)

    manifest = output_dir / "manifest.md"
    manifest.write_text(
        "# Cat portrait extraction manifest\n\n"
        f"- Source: `{source_copy.name}`\n"
        f"- Source size: `{width}x{height}`\n"
        f"- Grid: `{grid}x{grid}`\n"
        f"- Inset per cell: `{inset}px`\n"
        f"- Output portrait size: `{cell}x{cell}`\n"
        f"- Contact sheet: `{contact_sheet.name}`\n\n"
        "## Portraits\n\n"
        + "".join(f"- `{p.relative_to(output_dir)}` crop={box} size={size}\n" for p, box, size in portraits)
    )

    print(f"copied source to {source_copy}")
    print(f"extracted {len(portraits)} portraits to {portrait_dir}")
    print(f"wrote contact sheet to {contact_sheet}")
    print(f"wrote manifest to {manifest}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, default=DEFAULT_INPUT)
    parser.add_argument("--output-dir", type=Path, default=DEFAULT_OUTPUT_DIR)
    parser.add_argument("--grid", type=int, default=4)
    parser.add_argument("--inset", type=int, default=10)
    args = parser.parse_args()
    extract_portraits(args.input, args.output_dir, args.grid, args.inset)


if __name__ == "__main__":
    main()
