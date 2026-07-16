#!/usr/bin/env python3
"""
01-thermal-lab.py — thermal rasterization experiment harness (ALMANACH-RASTER-LAB)

Generates a synthetic "mixed card" (title + body text + gray ramp + a real photo
region), rasterizes it through a chosen dither algorithm and host tone curve,
optionally sets the printer heat/density, and prints it — with a printed settings
header so every physical strip is self-describing.

This is EXPERIMENTAL, ticket-local software. It mirrors the firmware contract
(MSB-first 1-bit packing, GS v 0 via /api/print/bitmap, density via
/api/printer/density) so lab findings transfer directly to the Go service later.

Usage:
  # 1) Inspect previews without a printer (writes upscaled PNGs):
  python3 01-thermal-lab.py --dry-run --out ./out \
      --grid --algos threshold,atkinson,floyd,bayer8 --densities 12,20,28,35

  # 2) Print one candidate:
  python3 01-thermal-lab.py --printer http://192.168.1.242 \
      --algo atkinson --density 24 --gamma 1.4 --contrast 0.85

  # 3) Print the full density x algorithm grid (one labeled strip per cell):
  python3 01-thermal-lab.py --printer http://192.168.1.242 --grid \
      --algos threshold,atkinson,floyd,bayer8 --densities 12,20,28,35

Levers:
  --gamma/--brightness/--contrast   host tone curve (lever 1), applied pre-dither
  --algo / --algos                  1-bit conversion (lever 2)
  --density / --densities           printer heat (lever 3), 0..39, set per strip
"""
import argparse
import io
import os
import sys
import time
import urllib.request
import numpy as np
from PIL import Image, ImageDraw, ImageFont

WIDTH_DEFAULT = 384                      # thermal paper width in dots
SAFE_SEGMENT_BYTES = 36 * 1024           # ESP32 httpd reliable receive limit

# ---------------------------------------------------------------------------
# Fonts
# ---------------------------------------------------------------------------
def load_font(size, bold=False):
    candidates = [
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf" if bold
        else "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
    ]
    for p in candidates:
        try:
            return ImageFont.truetype(p, size)
        except OSError:
            continue
    return ImageFont.load_default()

# ---------------------------------------------------------------------------
# Synthetic mixed card: title + body + gray ramp + real photo region
# ---------------------------------------------------------------------------
def find_photo():
    import glob, os
    # ttmp root is four levels up from scripts/ (ttmp/YYYY/MM/DD/TICKET/scripts)
    ttmp = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", ".."))
    pats = glob.glob(os.path.join(
        ttmp, "**", "cat-portraits", "portraits", "*.png"), recursive=True)
    return sorted(pats)[0] if pats else None

def build_card(width, photo_path):
    """Return an 8-bit grayscale PIL image: mixed text + ramp + photo."""
    W = width
    img = Image.new("L", (W, 1), 255)  # height grown as we compose
    # We compose onto a tall white canvas then crop.
    H = 1000
    canvas = Image.new("L", (W, H), 255)
    d = ImageDraw.Draw(canvas)
    y = 6

    # --- Title (large) ---
    ft = load_font(30, bold=True)
    d.text((6, y), "ALMANACH", font=ft, fill=0)
    y += 38
    fs = load_font(16, bold=True)
    d.text((6, y), "raster lab — mixed card", font=fs, fill=40)
    y += 26

    # --- Body paragraph (small, several sizes) ---
    for sz, txt in [
        (13, "Body 13px: the quick brown fox jumps over the lazy dog."),
        (11, "Body 11px: legibility check for small thermal glyphs 0123456789."),
        (9,  "Body 9px: fine print aeiou ABCDEFG .,:;!? — does it survive?"),
    ]:
        fb = load_font(sz)
        d.text((6, y), txt, font=fb, fill=0)
        y += sz + 6
    y += 6

    # --- Horizontal gray ramp 0->255 (tonal calibration band) ---
    ramp_h = 40
    ramp = np.tile(np.linspace(0, 255, W).astype(np.uint8), (ramp_h, 1))
    canvas.paste(Image.fromarray(ramp, "L"), (0, y))
    d.text((6, y + 2), "gray ramp 0->255", font=load_font(9), fill=255)
    y += ramp_h + 8

    # --- Photo region (real continuous-tone content) ---
    if photo_path:
        photo = Image.open(photo_path).convert("L")
        pw = W - 12
        ph = int(photo.height * pw / photo.width)
        ph = min(ph, 300)
        photo = photo.resize((pw, ph), Image.LANCZOS)
        canvas.paste(photo, (6, y))
        y += ph + 6
    else:
        # fallback synthetic radial gradient "photo"
        ph = 220
        yy, xx = np.mgrid[0:ph, 0:W]
        cx, cy = W / 2, ph / 2
        r = np.sqrt((xx - cx) ** 2 + (yy - cy) ** 2)
        g = (255 * (1 - np.clip(r / (W / 2), 0, 1))).astype(np.uint8)
        canvas.paste(Image.fromarray(g, "L"), (0, y))
        y += ph + 6

    return canvas.crop((0, 0, W, min(y + 4, H)))

def build_text_card(width):
    """Text-only card for tuning heat (density/speed): glyph legibility at many
    sizes + a reverse (white-on-black) bar to expose heat bleed / stroke fill-in."""
    W = width
    H = 900
    canvas = Image.new("L", (W, H), 255)
    d = ImageDraw.Draw(canvas)
    y = 6
    d.text((6, y), "ALMANACH", font=load_font(30, bold=True), fill=0); y += 40
    d.text((6, y), "text heat calibration", font=load_font(15, bold=True), fill=0); y += 26

    para = ("The quick brown fox jumps over the lazy dog. "
            "Pack my box with five dozen liquor jugs. 0123456789 "
            "aeiou ACEGO bdpq .,:;!?()-/ — counters: e a g s o m w")
    for sz in [16, 14, 13, 12, 11, 10, 9, 8]:
        f = load_font(sz)
        # wrap to width
        words = (f"{sz}px  " + para).split()
        line = ""
        for w in words:
            test = (line + " " + w).strip()
            if d.textlength(test, font=f) > W - 12:
                d.text((6, y), line, font=f, fill=0); y += sz + 3
                line = w
            else:
                line = test
        if line:
            d.text((6, y), line, font=f, fill=0); y += sz + 6
    y += 4
    # Reverse bar: white text on solid black — heat bleed closes the glyphs first here
    bar_h = 34
    d.rectangle([0, y, W - 1, y + bar_h], fill=0)
    d.text((6, y + 8), "REVERSE 12px white-on-black bleed test 0123", font=load_font(12, bold=True), fill=255)
    y += bar_h + 8
    return canvas.crop((0, 0, W, min(y + 4, H)))

def load_bitmap_font(name, size):
    """Load a bundled PCF bitmap font at its native pixel size."""
    p = os.path.join(os.path.dirname(__file__), "fonts", f"{name}.pcf")
    return ImageFont.truetype(p, size)

def build_smalltext_card(width):
    """Compare small-text rendering strategies on one card:
       (A) vector + anti-aliasing  -> gets threshold-dropped strokes,
       (B) vector with AA OFF (fontmode='1'),
       (C) bundled BITMAP (PCF) fonts at native pixel size.
    The whole card is grayscale; the pipeline's threshold then hits (A) only,
    since (B)/(C) are already pure black. This isolates why small text drops."""
    W = width
    canvas = Image.new("L", (W, 1100), 255)
    d = ImageDraw.Draw(canvas)
    y = 6
    d.text((6, y), "SMALL TEXT STRATEGIES", font=load_font(18, bold=True), fill=0); y += 26
    sample = "eagso mw 3809 Illegibility: the quick brown fox"

    def section(title):
        nonlocal y
        d.rectangle([0, y, W - 1, y + 15], fill=0)
        d.text((4, y + 2), title, font=load_font(11, bold=True), fill=255)
        y += 20

    # (A) vector, anti-aliased (default fontmode 'L')
    section("A: vector + anti-aliasing (current, threshold drops strokes)")
    d.fontmode = "L"
    for sz in [11, 10, 9, 8]:
        d.text((6, y), f"{sz}px AA  {sample}", font=load_font(sz), fill=0); y += sz + 5

    # (B) vector, anti-aliasing OFF
    section("B: vector, AA OFF (fontmode=1)")
    d.fontmode = "1"
    for sz in [11, 10, 9, 8]:
        d.text((6, y), f"{sz}px 1b  {sample}", font=load_font(sz), fill=0); y += sz + 5
    d.fontmode = "L"

    # (C) bundled bitmap PCF fonts at native size
    section("C: bitmap PCF fonts (native pixels)")
    for name, sz in [("7x14", 14), ("6x12", 12), ("6x10", 10), ("6x9", 9),
                     ("5x8", 8), ("5x7", 7), ("4x6", 6)]:
        try:
            bf = load_bitmap_font(name, sz)
            d.text((6, y), f"{name}  {sample}", font=bf, fill=0); y += sz + 5
        except Exception as e:
            d.text((6, y), f"{name} load failed: {e}", font=load_font(9), fill=0); y += 14

    return canvas.crop((0, 0, W, min(y + 6, 1100)))

# ---------------------------------------------------------------------------
# Tone curve (lever 1): applied to normalized gray v in [0,1]
#   v' = clamp( ((v^gamma) - 0.5) * contrast + 0.5 + brightness )
# gamma<1 lightens midtones, gamma>1 darkens. Direction is decided on paper.
# ---------------------------------------------------------------------------
def apply_tone(gray, gamma, brightness, contrast):
    v = gray.astype(np.float64) / 255.0
    if gamma and gamma > 0:
        v = np.power(np.clip(v, 0, 1), gamma)
    v = (v - 0.5) * contrast + 0.5 + brightness
    return np.clip(v * 255.0, 0, 255)

# ---------------------------------------------------------------------------
# Dither algorithms (lever 2). Return bool array, True = black dot.
# ---------------------------------------------------------------------------
BAYER8 = np.array([
    [0,48,12,60,3,51,15,63],[32,16,44,28,35,19,47,31],
    [8,56,4,52,11,59,7,55],[40,24,36,20,43,27,39,23],
    [2,50,14,62,1,49,13,61],[34,18,46,30,33,17,45,29],
    [10,58,6,54,9,57,5,53],[42,26,38,22,41,25,37,21]], dtype=np.float64)

DIFFUSION = {
    # name: (divisor, [(dx,dy,weight), ...])
    "floyd":   (16, [(1,0,7),(-1,1,3),(0,1,5),(1,1,1)]),
    "atkinson":(8,  [(1,0,1),(2,0,1),(-1,1,1),(0,1,1),(1,1,1),(0,2,1)]),
    "stucki":  (42, [(1,0,8),(2,0,4),(-2,1,2),(-1,1,4),(0,1,8),(1,1,4),(2,1,2),
                     (-2,2,1),(-1,2,2),(0,2,4),(1,2,2),(2,2,1)]),
    "sierra2": (16, [(1,0,4),(2,0,3),(-2,1,1),(-1,1,2),(0,1,3),(1,1,2),(2,1,1)]),
}

def dith_threshold(gray, T):
    return gray < T

def dith_bayer8(gray, T):
    h, w = gray.shape
    m = (BAYER8 + 0.5) / 64.0 * 255.0
    tiled = np.tile(m, (h // 8 + 1, w // 8 + 1))[:h, :w]
    return gray < tiled

def dith_diffuse(gray, T, name):
    div, taps = DIFFUSION[name]
    work = gray.astype(np.float64).copy()
    h, w = work.shape
    out = np.zeros((h, w), dtype=bool)
    for y in range(h):
        for x in range(w):
            old = work[y, x]
            if old < T:
                out[y, x] = True
                new = 0.0
            else:
                new = 255.0
            err = old - new
            for dx, dy, wgt in taps:
                nx, ny = x + dx, y + dy
                if 0 <= nx < w and 0 <= ny < h:
                    work[ny, nx] += err * wgt / div
    return out

def rasterize(gray8, algo, T, gamma, brightness, contrast):
    toned = apply_tone(gray8, gamma, brightness, contrast)
    if algo == "threshold":
        return dith_threshold(toned, T)
    if algo == "bayer8":
        return dith_bayer8(toned, T)
    if algo in DIFFUSION:
        return dith_diffuse(toned, T, algo)
    raise ValueError(f"unknown algo: {algo}")

# ---------------------------------------------------------------------------
# Settings header rendered into the bitmap (self-describing strips)
# ---------------------------------------------------------------------------
def header_image(width, lines):
    fh = load_font(12)
    lh = 15
    img = Image.new("L", (width, lh * len(lines) + 8), 255)
    d = ImageDraw.Draw(img)
    for i, ln in enumerate(lines):
        d.text((4, 3 + i * lh), ln, font=fh, fill=0)
    return img

# ---------------------------------------------------------------------------
# Packing (must match firmware: width padded to /8, MSB-first, black=bit set)
# ---------------------------------------------------------------------------
def pack_bits(black):
    h, w = black.shape
    wpad = ((w + 7) // 8) * 8
    bpr = wpad // 8
    data = bytearray(bpr * h)
    ys, xs = np.nonzero(black)
    for y, x in zip(ys, xs):
        data[y * bpr + (x >> 3)] |= 0x80 >> (x & 7)
    return bytes(data), wpad, h, bpr

# ---------------------------------------------------------------------------
# Printer client (lever 3 + bitmap POST)
# ---------------------------------------------------------------------------
def set_density(printer, density):
    body = f'{{"density":{density}}}'.encode()
    req = urllib.request.Request(printer.rstrip("/") + "/api/printer/density",
                                 data=body, method="POST",
                                 headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.read().decode()

# Valid K118 speeds (firmware web_speed_supported); lower = more heat dwell = darker
VALID_SPEEDS = [25, 30, 37, 50, 56, 62, 70, 80, 90, 100, 120, 150, 180, 200, 220]

def set_speed(printer, speed):
    body = f'{{"speed":{speed}}}'.encode()
    req = urllib.request.Request(printer.rstrip("/") + "/api/printer/speed",
                                 data=body, method="POST",
                                 headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.read().decode()

def print_bitmap(printer, data, wpad, h, bpr, feed_lines=3):
    url = printer.rstrip("/") + "/api/print/bitmap"
    # segment by rows to stay under the safe limit; feed only on last segment
    max_rows = max(1, SAFE_SEGMENT_BYTES // bpr)
    off = 0
    seg = 0
    while off < h:
        rows = min(max_rows, h - off)
        chunk = data[off * bpr:(off + rows) * bpr]
        last = (off + rows) >= h
        req = urllib.request.Request(url, data=chunk, method="POST", headers={
            "Content-Type": "application/octet-stream",
            "X-Width": str(wpad), "X-Height": str(rows),
            "X-Feed": str(feed_lines if last else 0), "Connection": "close"})
        with urllib.request.urlopen(req, timeout=120) as r:
            r.read()
        seg += 1
        off += rows
    return seg

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def candidate_lines(name, algo, density, speed, gamma, brightness, contrast, T, black):
    dens = black.mean() * 100.0
    return [
        f"ALMANACH RASTER LAB  {name}",
        f"algo={algo}  density={density}  speed={speed}",
        f"gamma={gamma}  con={contrast:.2f}  bri={brightness:+.2f}  T={T}",
        f"black={dens:.1f}%  {black.shape[1]}x{black.shape[0]}",
    ]

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--printer", default="", help="printer base URL, e.g. http://192.168.0.126")
    ap.add_argument("--card", default="mixed", choices=["mixed", "text", "smalltext"], help="which test card")
    ap.add_argument("--algo", default="atkinson")
    ap.add_argument("--algos", default="", help="comma list for --grid")
    ap.add_argument("--density", type=int, default=20)
    ap.add_argument("--densities", default="", help="comma list for --grid")
    ap.add_argument("--speed", type=int, default=80, help="print speed; lower = more heat/darker")
    ap.add_argument("--speeds", default="", help="comma list to sweep speed")
    ap.add_argument("--gamma", type=float, default=1.0)
    ap.add_argument("--brightness", type=float, default=0.0)
    ap.add_argument("--contrast", type=float, default=1.0)
    ap.add_argument("--threshold", type=int, default=128)
    ap.add_argument("--width", type=int, default=WIDTH_DEFAULT)
    ap.add_argument("--feed", type=int, default=3)
    ap.add_argument("--grid", action="store_true")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--out", default="./out")
    args = ap.parse_args()

    if args.card == "text":
        card = np.array(build_text_card(args.width))
        print(f"[card] text {card.shape[1]}x{card.shape[0]}", file=sys.stderr)
    elif args.card == "smalltext":
        card = np.array(build_smalltext_card(args.width))
        print(f"[card] smalltext {card.shape[1]}x{card.shape[0]}", file=sys.stderr)
    else:
        photo = find_photo()
        card = np.array(build_card(args.width, photo))
        print(f"[card] mixed {card.shape[1]}x{card.shape[0]} photo={photo or 'synthetic'}", file=sys.stderr)

    algos = [a.strip() for a in (args.algos.split(",") if args.algos else [args.algo])]
    densities = [int(x) for x in (args.densities.split(",") if args.densities else [args.density])]
    speeds = [int(x) for x in (args.speeds.split(",") if args.speeds else [args.speed])]
    for s in speeds:
        if s not in VALID_SPEEDS:
            print(f"ERROR: speed {s} not in {VALID_SPEEDS}", file=sys.stderr); sys.exit(2)
    combos = [(a, d, s) for s in speeds for d in densities for a in algos]

    import os
    if args.dry_run:
        os.makedirs(args.out, exist_ok=True)

    for algo, density, speed in combos:
        black = rasterize(card, algo, args.threshold, args.gamma, args.brightness, args.contrast)
        lines = candidate_lines(args.card, algo, density, speed,
                                args.gamma, args.brightness, args.contrast, args.threshold, black)
        # Header packed as pure threshold (crisp text); card region keeps its
        # dithered bits. Stack in bit-space so the header is never re-dithered.
        hdr_black = np.array(header_image(card.shape[1], lines)) < 128
        full_black = np.vstack([hdr_black, black])
        data, wpad, h, bpr = pack_bits(full_black)
        tag = f"{args.card}_{algo}_d{density}_s{speed}_g{args.gamma}"
        if args.dry_run:
            prev = Image.fromarray((~full_black * 255).astype(np.uint8), "L")
            path = os.path.join(args.out, f"{tag}.png")
            prev.save(path)
            print(f"[dry-run] {path}  bytes={len(data)}  black={black.mean()*100:.1f}%", file=sys.stderr)
        else:
            if not args.printer:
                print("ERROR: --printer required to print (or use --dry-run)", file=sys.stderr)
                sys.exit(2)
            print(f"[print] {tag} bytes={len(data)}", file=sys.stderr)
            set_density(args.printer, density)
            set_speed(args.printer, speed)
            time.sleep(0.3)
            segs = print_bitmap(args.printer, data, wpad, h, bpr, args.feed)
            print(f"[print]   sent in {segs} segment(s)", file=sys.stderr)
            time.sleep(1.5)

if __name__ == "__main__":
    main()
