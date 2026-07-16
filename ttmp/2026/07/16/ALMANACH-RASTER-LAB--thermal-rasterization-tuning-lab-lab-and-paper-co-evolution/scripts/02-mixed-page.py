#!/usr/bin/env python3
"""
02-mixed-page.py — mixed-page proof (ALMANACH-RASTER-LAB)

Demonstrates the full paper-verified recipe set in ONE print job, using
per-segment printer heat (density + speed) on the existing segmented path —
no firmware change:

  Segment A (TEXT, hot):   header + body, BITMAP font, density 30, speed 37
  Segment B (PHOTO, cool): the cat, Atkinson dither + gamma 0.8, density 20, speed 80
  Segment C (TEXT, hot):   caption + fact, BITMAP font, density 30, speed 37

Text uses bundled X11 PCF bitmap fonts (crisp small glyphs, no AA dropout).
The photo uses Atkinson at a lighter tone. Each segment sets its own density and
speed before its bitmap is sent.

  python3 02-mixed-page.py --printer http://192.168.0.126
  python3 02-mixed-page.py --dry-run --out ./out     # preview PNGs only
"""
import argparse
import os
import sys
import time
import urllib.request
import numpy as np
from PIL import Image, ImageDraw, ImageFont

WIDTH = 384
SAFE_SEGMENT_BYTES = 36 * 1024
VALID_SPEEDS = [25, 30, 37, 50, 56, 62, 70, 80, 90, 100, 120, 150, 180, 200, 220]

# --- Recipes (paper-verified) ----------------------------------------------
TEXT_DENSITY, TEXT_SPEED = 30, 37     # hot + slow: dark, even strokes
PHOTO_DENSITY, PHOTO_SPEED = 20, 80   # cool + fast: light, un-muddy photo
PHOTO_GAMMA = 0.8

# --- Fonts ------------------------------------------------------------------
def vec_font(size, bold=False):
    for p in (["/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"] if bold
              else ["/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"]):
        try:
            return ImageFont.truetype(p, size)
        except OSError:
            pass
    return ImageFont.load_default()

def bmp_font(name, size):
    return ImageFont.truetype(os.path.join(os.path.dirname(__file__), "fonts", f"{name}.pcf"), size)

# --- Photo (Atkinson + gamma) ----------------------------------------------
def find_photo():
    import glob
    ttmp = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", ".."))
    p = glob.glob(os.path.join(ttmp, "**", "cat-portraits", "portraits", "*.png"), recursive=True)
    return sorted(p)[0] if p else None

def atkinson(gray, T=128):
    work = gray.astype(np.float64).copy()
    h, w = work.shape
    out = np.zeros((h, w), dtype=bool)
    taps = [(1, 0, 1), (2, 0, 1), (-1, 1, 1), (0, 1, 1), (1, 1, 1), (0, 2, 1)]
    for y in range(h):
        for x in range(w):
            old = work[y, x]
            new = 0.0 if old < T else 255.0
            if new == 0.0:
                out[y, x] = True
            err = (old - new) / 8.0
            for dx, dy, _ in taps:
                nx, ny = x + dx, y + dy
                if 0 <= nx < w and 0 <= ny < h:
                    work[ny, nx] += err
    return out

def photo_block(width, gamma):
    path = find_photo()
    W = width
    canvas = Image.new("L", (W, 340), 255)
    d = ImageDraw.Draw(canvas)
    y = 0
    if path:
        im = Image.open(path).convert("L")
        pw = W - 12
        ph = min(int(im.height * pw / im.width), 300)
        canvas.paste(im.resize((pw, ph), Image.LANCZOS), (6, 0))
        y = ph
    else:
        y = 240
    card = np.array(canvas.crop((0, 0, W, y)))
    v = np.clip(card / 255.0, 0, 1) ** gamma
    return atkinson(v * 255.0)

# --- Text blocks (bitmap font) ---------------------------------------------
def wrap(draw, text, font, width):
    words, lines, line = text.split(), [], ""
    for w in words:
        t = (line + " " + w).strip()
        if draw.textlength(t, font=font) > width - 12:
            lines.append(line); line = w
        else:
            line = t
    if line:
        lines.append(line)
    return lines

def text_block(width, spec):
    """spec: list of (kind, font, text). kind: 'title'|'line'|'para'|'rule'."""
    W = width
    tall = Image.new("L", (W, 1400), 255)
    d = ImageDraw.Draw(tall)
    d.fontmode = "1"  # no AA for any vector fallback; bitmap fonts are 1-bit anyway
    y = 4
    for kind, font, text in spec:
        if kind == "rule":
            d.line([(6, y + 2), (W - 6, y + 2)], fill=0, width=2); y += 8; continue
        lh = font.size + (5 if kind == "para" else 3)
        lines = wrap(d, text, font, W) if kind == "para" else [text]
        for ln in lines:
            d.text((6, y), ln, font=font, fill=0); y += lh
        if kind == "title":
            y += 4
    black = np.array(tall.crop((0, 0, W, y + 4))) < 128
    return black

# --- Packing + transport ----------------------------------------------------
def pack_bits(black):
    h, w = black.shape
    wpad = ((w + 7) // 8) * 8
    bpr = wpad // 8
    data = bytearray(bpr * h)
    ys, xs = np.nonzero(black)
    for y, x in zip(ys, xs):
        data[y * bpr + (x >> 3)] |= 0x80 >> (x & 7)
    return bytes(data), wpad, h, bpr

def _post(url, data, headers, timeout=120):
    req = urllib.request.Request(url, data=data, method="POST", headers=headers)
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return r.read()

def set_density(printer, n):
    _post(printer.rstrip("/") + "/api/printer/density",
          f'{{"density":{n}}}'.encode(), {"Content-Type": "application/json"}, 30)

def set_speed(printer, n):
    _post(printer.rstrip("/") + "/api/printer/speed",
          f'{{"speed":{n}}}'.encode(), {"Content-Type": "application/json"}, 30)

def print_bitmap(printer, black, feed_lines=0):
    data, wpad, h, bpr = pack_bits(black)
    url = printer.rstrip("/") + "/api/print/bitmap"
    max_rows = max(1, SAFE_SEGMENT_BYTES // bpr)
    off = 0
    while off < h:
        rows = min(max_rows, h - off)
        last = (off + rows) >= h
        _post(url, data[off * bpr:(off + rows) * bpr], {
            "Content-Type": "application/octet-stream",
            "X-Width": str(wpad), "X-Height": str(rows),
            "X-Feed": str(feed_lines if last else 0), "Connection": "close"})
        off += rows
    return h

# --- Compose the page -------------------------------------------------------
def build_segments(width):
    body = bmp_font("6x10", 10)
    small = bmp_font("6x9", 9)
    date = bmp_font("6x12", 12)
    title = vec_font(34, bold=True)  # large -> vector is fine

    seg_a = text_block(width, [
        ("title", title, "ALMANACH"),
        ("line", date, "Thursday - 16 July 2026 - Cat of the Day"),
        ("rule", None, ""),
        ("para", body, "Good morning. Today's page mixes crisp bitmap-font text "
                       "printed at a hotter density with a photo dithered at a "
                       "lighter one - all in a single job."),
    ])
    seg_b = photo_block(width, PHOTO_GAMMA)
    seg_c = text_block(width, [
        ("line", small, "Above: tabby house cat. Atkinson dither, gamma 0.8, density 20."),
        ("rule", None, ""),
        ("para", body, "On this day in 1945 the first atomic device was tested. "
                       "Almanach fact: a cat has 32 muscles in each ear. "
                       "Quote: \"Time is the longest distance between two places.\""),
        ("line", small, "eagso mw 0123456789 - counters open at 6x9 and 6x10."),
    ])
    return [("A-text-hot", seg_a, TEXT_DENSITY, TEXT_SPEED),
            ("B-photo-cool", seg_b, PHOTO_DENSITY, PHOTO_SPEED),
            ("C-text-hot", seg_c, TEXT_DENSITY, TEXT_SPEED)]

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--printer", default="")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--out", default="./out")
    ap.add_argument("--feed", type=int, default=4)
    args = ap.parse_args()

    segments = build_segments(WIDTH)

    if args.dry_run:
        os.makedirs(args.out, exist_ok=True)
        combined = np.vstack([s[1] for s in segments])
        Image.fromarray((~combined * 255).astype(np.uint8), "L").save(
            os.path.join(args.out, "mixed-page.png"))
        for name, blk, dens, spd in segments:
            print(f"[dry-run] {name}: {blk.shape[1]}x{blk.shape[0]} density={dens} speed={spd}", file=sys.stderr)
        print(f"[dry-run] wrote {args.out}/mixed-page.png", file=sys.stderr)
        return

    if not args.printer:
        print("ERROR: --printer required (or --dry-run)", file=sys.stderr); sys.exit(2)
    for i, (name, blk, dens, spd) in enumerate(segments):
        last = (i == len(segments) - 1)
        print(f"[print] {name} density={dens} speed={spd} rows={blk.shape[0]}", file=sys.stderr)
        set_density(args.printer, dens)
        set_speed(args.printer, spd)
        time.sleep(0.3)
        print_bitmap(args.printer, blk, feed_lines=args.feed if last else 0)
        time.sleep(1.2)
    print("[print] mixed page done", file=sys.stderr)

if __name__ == "__main__":
    main()
