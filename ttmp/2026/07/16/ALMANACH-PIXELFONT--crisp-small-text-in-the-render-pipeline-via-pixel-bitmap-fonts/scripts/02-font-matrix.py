#!/usr/bin/env python3
"""
02-font-matrix.py — small-text font/size/technique matrix (ALMANACH-PIXELFONT)

Renders the same small text across several fonts and sizes, through the same
supersample+threshold path the render pipeline uses, so we can compare which
(font, size, technique) reads best as a 384px-wide 1-bit thermal bitmap.

Techniques (each renders the whole grid once, then downsamples to 384):
  1x-aaoff : scale 1, anti-aliasing OFF (fontconfig)      -> the AA-off path
  2x/3x/4x : scale N, anti-aliasing ON, box-average down  -> supersample path
  3x-T160  : scale 3, threshold 160 (biases thin strokes to black)

Usage:
  python3 02-font-matrix.py --out ./matrix                 # render all techniques to PNGs
  python3 02-font-matrix.py --print --printer http://192.168.0.126 --tech 3x   # print one technique
"""
import argparse
import base64
import os
import subprocess
import sys
import numpy as np
from PIL import Image

Image.MAX_IMAGE_PIXELS = None  # our own render screenshots, not untrusted input

W = 384
CHROME = "/usr/bin/google-chrome-stable"
HERE = os.path.dirname(os.path.abspath(__file__))
NOAA = os.path.join(HERE, "fonts-noaa.conf")
FONTS_CSS = os.path.abspath(os.path.join(HERE, "..", "..", "..", "..", "..", "web", "dist", "fonts.css"))

SAMPLE = "Illegibility eagso mw 0123456789"
# (label, css font-family, upright sizes, italic sizes) — x-height matched:
# EB Garamond runs ~2-3px smaller than DejaVu at the same nominal size, so it is
# tested larger to reach a comparable effective x-height.
FONTS = [
    ("EB Garamond (SPA serif) BIG", "'EB Garamond', serif", [13, 14, 15, 16], [14, 15, 16]),
    ("DejaVu Serif (hinted)", "'DejaVu Serif', serif", [10, 11, 12, 13], [11, 12, 13]),
    ("DejaVu Sans (hinted)", "'DejaVu Sans', sans-serif", [10, 11, 12], [11, 12]),
]

# scale, aa_off, threshold
TECHNIQUES = {
    "1x-aaoff": (1, True, 128),
    "2x": (2, False, 128),
    "3x": (3, False, 128),
    "4x": (4, False, 128),
    "3x-T160": (3, False, 160),
}


ITALIC_SAMPLE = "We stood enjoying the brief apricity 0123"


def build_html(title="MATRIX"):
    rows = [f'<div class="hdr">{title}</div>']
    for label, fam, sizes, ital_sizes in FONTS:
        rows.append(f'<div class="lbl">{label}</div>')
        for sz in sizes:
            rows.append(f'<div class="s" style="font-family:{fam};font-size:{sz}px;line-height:{sz+2}px">'
                        f'{sz}&nbsp; {SAMPLE}</div>')
        for sz in ital_sizes:
            rows.append(f'<div class="s" style="font-family:{fam};font-style:italic;font-size:{sz}px;line-height:{sz+2}px">'
                        f'{sz}i {ITALIC_SAMPLE}</div>')
        rows.append('<div class="gap"></div>')
    link = f'<link rel="stylesheet" href="file://{FONTS_CSS}">' if os.path.exists(FONTS_CSS) else ""
    html = f"""<!doctype html><meta charset=utf-8>{link}<style>
html,body{{margin:0;background:#fff}}
.p{{width:{W}px;padding:6px 4px;color:#000;box-sizing:border-box}}
.hdr{{font-family:'DejaVu Sans',sans-serif;font-size:15px;font-weight:bold;background:#000;color:#fff;padding:3px 4px;margin-bottom:3px}}
.lbl{{font-family:'DejaVu Sans',sans-serif;font-size:11px;font-weight:bold;border-bottom:2px solid #000;padding:1px 3px;margin-top:5px}}
.s{{color:#000;margin:1px 0;white-space:nowrap;overflow:hidden}}
.gap{{height:5px}}</style><div class="p">{''.join(rows)}</div>"""
    path = os.path.join(HERE, "matrix.html")
    open(path, "w").write(html)
    return path


def render(html, scale, aa_off, threshold):
    """Render the page at scale x, box-average to 384-wide, threshold to 1-bit.
    Returns a boolean array (True=black)."""
    png = os.path.join(HERE, "_matrix_tmp.png")
    env = dict(os.environ)
    if aa_off:
        env["FONTCONFIG_FILE"] = NOAA
    else:
        env.pop("FONTCONFIG_FILE", None)
    # window-size is CSS pixels; the device-scale-factor multiplies it into the
    # screenshot. Use the paper width (384) so the capture is exactly W*scale wide.
    subprocess.run([CHROME, "--headless=new", "--disable-gpu", "--no-sandbox",
                    "--hide-scrollbars", f"--force-device-scale-factor={scale}",
                    f"--window-size={W},760",
                    f"--screenshot={png}", f"file://{html}"],
                   env=env, capture_output=True)
    im = Image.open(png).convert("RGB")
    a = np.asarray(im).astype(np.float64)
    gray = (0.299 * a[:, :, 0] + 0.587 * a[:, :, 1] + 0.114 * a[:, :, 2])
    if scale > 1:
        h = (gray.shape[0] // scale) * scale
        w = (gray.shape[1] // scale) * scale
        gray = gray[:h, :w].reshape(h // scale, scale, w // scale, scale).mean(axis=(1, 3))
    return gray < threshold


def pack_bits(black):
    h, w = black.shape
    wpad = ((w + 7) // 8) * 8
    bpr = wpad // 8
    data = bytearray(bpr * h)
    ys, xs = np.nonzero(black)
    for y, x in zip(ys, xs):
        data[y * bpr + (x >> 3)] |= 0x80 >> (x & 7)
    return bytes(data), wpad, h, bpr


def set_heat(printer, density, speed):
    import urllib.request
    for path, key, val in (("density", "density", density), ("speed", "speed", speed)):
        if val is None:
            continue
        req = urllib.request.Request(printer.rstrip("/") + f"/api/printer/{path}",
                                     data=f'{{"{key}":{val}}}'.encode(), method="POST",
                                     headers={"Content-Type": "application/json"})
        urllib.request.urlopen(req, timeout=30).read()


def print_to(printer, black, feed=3):
    import urllib.request
    data, wpad, h, bpr = pack_bits(black)
    seg = 36 * 1024
    off = 0
    while off < h:
        rows = min(max(1, seg // bpr), h - off)
        last = (off + rows) >= h
        req = urllib.request.Request(printer.rstrip("/") + "/api/print/bitmap",
                                     data=data[off * bpr:(off + rows) * bpr], method="POST",
                                     headers={"Content-Type": "application/octet-stream",
                                              "X-Width": str(wpad), "X-Height": str(rows),
                                              "X-Feed": str(feed if last else 0), "Connection": "close"})
        urllib.request.urlopen(req, timeout=120).read()
        off += rows


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default="./matrix")
    ap.add_argument("--print", dest="do_print", action="store_true")
    ap.add_argument("--printer", default="http://192.168.0.126")
    ap.add_argument("--tech", default="", help="single technique to render/print")
    ap.add_argument("--densities", default="", help="comma list; sweep printer density per print")
    ap.add_argument("--speeds", default="", help="comma list; sweep printer speed per print")
    args = ap.parse_args()

    techs = [args.tech] if args.tech else list(TECHNIQUES)
    densities = [int(x) for x in args.densities.split(",")] if args.densities else [None]
    speeds = [int(x) for x in args.speeds.split(",")] if args.speeds else [None]
    os.makedirs(args.out, exist_ok=True)

    for t in techs:
        scale, aa_off, thr = TECHNIQUES[t]
        for d in densities:
            for s in speeds:
                # Bake the technique + heat into the sheet header so every printed
                # strip is self-identifying.
                heat = (f"  d={d}" if d is not None else "") + (f" s={s}" if s is not None else "")
                title = f"{t}  scale={scale} aa_off={int(aa_off)} thr={thr}{heat}"
                html = build_html(title)
                black = render(html, scale, aa_off, thr)
                print(f"[{title}] {black.shape[1]}x{black.shape[0]} black={black.mean()*100:.1f}%", file=sys.stderr)
                tag = t + (f"_d{d}" if d is not None else "") + (f"_s{s}" if s is not None else "")
                Image.fromarray((~black * 255).astype("uint8"), "L").save(os.path.join(args.out, f"{tag}.png"))
                if args.do_print:
                    if d is not None or s is not None:
                        set_heat(args.printer, d, s)
                    print_to(args.printer, black)


if __name__ == "__main__":
    main()
