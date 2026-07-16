---
Title: Diary
Ticket: ALMANACH-RASTER-LAB
Status: active
Topics:
    - almanach
    - thermal-printer
    - printing
    - go
    - firmware
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: firmware/atoms3r/main/printer_drv.c
    - Path: internal/app/bitmap.go
    - Path: internal/app/printer.go
    - Path: repo://ttmp/2026/07/16/ALMANACH-RASTER-LAB--thermal-rasterization-tuning-lab-lab-and-paper-co-evolution/scripts/01-thermal-lab.py
      Note: 'Experiment harness: mixed card + tone curve + dither + density + print'
ExternalSources: []
Summary: 'Chronological diary of the thermal rasterization tuning-lab work: investigation, the intern guide, and the lab-and-paper co-evolution loop.'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Record what was tried, what worked, what failed, and what to do next.
WhenToUse: Read before resuming ALMANACH-RASTER-LAB work.
---


# Diary

## Goal

Capture the creation of the `ALMANACH-RASTER-LAB` ticket, the investigation of
the current Almanach rasterization pipeline, the writing of the intern-oriented
analysis/design/implementation guide, and the establishment of the
lab-and-paper co-evolution workflow used to tune thermal rasterization,
black/white contrast, and printer heat/density.

## Step 1: Investigate the pipeline and stand up the ticket + intern guide

The user reported that printed output is "a bit hard to read" and wants to
experiment with rasterization, black/white contrast, and heat. They asked me to
analyze the repo (plus prior work in the `DITHER`/rasterization ticket and the
go-go-parc knowledge base), report findings and options without implementing,
and then — after a follow-up — to create a new docmgr ticket, keep a detailed
diary, write an intern-oriented analysis/design/implementation guide, store it
in the ticket, and upload it to reMarkable. A further mid-turn instruction
established the working mode: I optimize rasterization in the browser lab
autonomously, ask the user to print candidates and return photos of the paper,
and evolve lab and hardware in tandem before merging winners into the main Go
service.

I grounded the guide in the actual code paths rather than general theory:
Chrome renders the React SPA, Go screenshots it to one flat PNG,
`internal/app/bitmap.go` converts it with a single fixed grayscale threshold,
and `internal/app/printer.go` POSTs a packed 1-bit bitmap (auto-segmented above
~36 KiB) to the ESP32, which emits `GS v 0` over UART to the K118 head. The
firmware also exposes density/speed/graphics endpoints that the host currently
never coordinates with the image — the underused "heat" lever the user asked
about.

### Prompt Context

**User prompt (verbatim):** "Analyze this repo, and let me know how we could optimize the rasterization and black/white contrast / heat, because currently things are a bit hard to read, and I would like to experiment with different options. look at claw-stuff/ttmp/2026/05/10/DITHER-001--deep-research-report-dithering-and-adaptive-thermal-rasterization/ and potentially ~/code/wesen/go-go-golems/go-go-parc/Projects for work we have already done. \n\nDon't implement, let me know what you find and what you could think of trying out."

**Follow-up (verbatim):** "I would basically like for you to optimize things in the lab by yourself, but ask me to compare / view things on paper (potentially even give you photos) so that we can evolve both in tandem. You are welcome to write new software for the experiment as well, before we merge things back into the main software. \n\nCreate  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Mid-turn (verbatim):** "create a new docmgr ticket, and keep a detailed diary (see skill for the format) as you work, commititng at appropriate itnervals"

**Assistant interpretation:** Analyze the rasterization pipeline; create a new
docmgr ticket; write a clear, technical intern guide covering the whole system,
the problem, the levers, the lab, the firmware API, and the co-evolution
workflow; keep a diary; commit at intervals; upload the guide to reMarkable.

**Inferred user intent:** Set up a durable, reproducible experimentation
workflow (with the agent doing lab tuning and the user providing paper/photo
feedback) to make thermal output legible, and capture enough system knowledge
that a newcomer can drive it.

**Commit (code):** see changelog — docs-only in this step.

### What I did
- Located the real paths (the user's `claw-stuff/DITHER-001` path did not exist;
  the actual assets are the `ALMANACH-RASTERIZATION` ticket and the go-go-parc
  On-Ramp notes `dithering-and-rasterization.md` / `esc-pos-thermal-printer.md`).
- Read the current rasterizer (`bitmap.go`), transport (`printer.go`), render
  seam (`renderer.go`, `render_oneshot.go`), print CLI (`cmd_print.go`), and the
  firmware driver/server (`printer_drv.c`, `web_server.c`).
- Read the prior design doc and investigation diary in the `ALMANACH-RASTERIZATION`
  ticket, and the browser lab (`raster-dither-lab.js/html`) which already
  implements ~12 algorithms + tone controls + a print path, but is not ported to Go.
- Created ticket `ALMANACH-RASTER-LAB` with a diary and a design-doc.
- Wrote the intern guide:
  `design-doc/01-thermal-rasterization-system-analysis-design-and-implementation-guide.md`.

### Why
- The user wants to experiment across three levers; the guide names them
  explicitly (host tone curve, dither algorithm, firmware heat/density) and
  frames the key unknown as the algorithm × density interaction that only paper
  can resolve.
- A new ticket keeps the experimental co-evolution work separate from the older
  research/design ticket, with its own diary and scripts dir.

### What worked
- Full pipeline reconstructed from source with exact ESC/POS bytes:
  density `ESC ## STDP n` (0x1B 23 23 53 54 44 50 n, 0..39, default 20),
  speed `ESC ## STSP n` (default 80), graphics `ESC ## SPSM n` (default 31),
  bitmap `GS v 0` (0x1D 76 30).
- Confirmed the packing invariant and the ~36 KiB safe-segment / 1 MiB hard cap
  in `printer.go`.

### What didn't work
- The user-referenced path
  `claw-stuff/ttmp/2026/05/10/DITHER-001--deep-research-report-...` does not
  exist in this repo. I substituted the real in-repo `ALMANACH-RASTERIZATION`
  ticket and the go-go-parc On-Ramp notes.

### What I learned
- Nothing from the earlier rasterization design has been merged into Go;
  `bitmap.go` is still fixed-threshold only. The browser lab is the furthest-along
  artifact and is the right fast-loop tool.
- The firmware density/speed knobs are separate endpoints and are never
  coordinated with the image today — this is the "heat" lever and the main
  underused axis.
- A real gamma-convention ambiguity exists: the parc note's prose ("gamma 1.8
  makes midtones lighter") contradicts the formula `v^1.8` (which darkens). The
  guide fixes the convention as `v' = v^γ` and defers direction to a printed
  ramp experiment.

### What was tricky to build
- Deciding scope: an extensive design doc already exists. Rather than duplicate
  it, this guide is positioned as the *working* document — it adds the lab/paper
  workflow and elevates firmware heat/density to a first-class lever, and points
  back to the older doc for algorithm depth.

### What warrants a second pair of eyes
- The gamma direction claim — confirm on paper (§7.4 ramp experiment) before
  baking a default.
- Whether the first physical print should be the full density × algorithm grid
  (§7.3) or a smaller probe to save paper.

### What should be done in the future
- Add density/speed controls to the lab UI (drive lever 3 in the fast loop).
- Build the comparison-sheet exporter for the density × algorithm grid.
- Ask the user to print + photograph the grid; record observations here.
- Port the winning combination into an `internal/app/raster` package (Phase 2),
  keeping `threshold` byte-identical.

### Code review instructions
- Start with the guide's §2 (pipeline) and §6 (Go walkthrough).
- Validate firmware API claims against `firmware/atoms3r/main/printer_drv.c`
  lines for `printer_drv_set_density/speed/graphics_mode` and
  `printer_drv_print_bitmap`.
- Validate transport claims against `internal/app/printer.go`
  (`maxSafePrinterBitmapBodyBytes`, `splitBitmap`, `sendSingleBitmap`).
- Docs hygiene: `docmgr doctor --ticket ALMANACH-RASTER-LAB --stale-after 30`.

### Technical details
- Ticket path:
  `ttmp/2026/07/16/ALMANACH-RASTER-LAB--thermal-rasterization-tuning-lab-lab-and-paper-co-evolution/`
- Guide:
  `design-doc/01-thermal-rasterization-system-analysis-design-and-implementation-guide.md`
- Lab (older ticket):
  `ttmp/2026/05/10/ALMANACH-RASTERIZATION--.../various/raster-dither-lab.{html,js}`

## Step 2: Build the experiment harness (synthetic mixed card + grid + printer client)

The user chose a **synthetic mixed card** as the first comparison sample and
confirmed the printer is ready for physical prints now. I built a self-contained
Python harness, `scripts/01-thermal-lab.py`, that generates a mixed card
(ALMANACH title, body text at 13/11/9 px, a 0→255 gray ramp, and a real cat
portrait for continuous-tone content), applies the host tone curve
(gamma/brightness/contrast), rasterizes through a chosen algorithm, prints a
self-describing settings header above each strip, packs MSB-first exactly like
the firmware contract, sets printer density via `/api/printer/density`, and POSTs
the bitmap via `/api/print/bitmap` (segmenting above ~36 KiB, feed on last
segment only).

I validated it in `--dry-run` mode (writes upscaled PNG previews). The previews
make the problem visible on screen before we spend paper: at gamma 1.0 the
**threshold** card crushes the cat into black/white blobs (all midtone detail
lost — the "hard to read" failure), while **Atkinson** preserves fur, eyes, and
whiskers as tonal dot patterns. Black density at density-20/gamma-1.0 was
threshold 32.7%, atkinson 30.0%, floyd 30.7%, bayer8 31.0%.

### Prompt Context

**User prompt (verbatim):** (see Step 1; plus the AskUserQuestion answers:
sample = "Synthetic mixed card", printer = "Yes, ready now")

**Assistant interpretation:** Build experimental (non-production) software that
lets me tune levers 1–3 on a synthetic mixed card and print candidates for the
user to photograph.

**Inferred user intent:** Start the lab→paper loop with a controlled,
text+ramp+photo sample so the text-vs-photo conflict and the density interaction
are both exercised.

**Commit (code):** see changelog.

### What I did
- Wrote `scripts/01-thermal-lab.py` (card generator, tone curve, dither
  algorithms threshold/atkinson/floyd/stucki/sierra2/bayer8, MSB packing,
  density + bitmap HTTP client, `--grid` and `--dry-run` modes).
- Fixed the cat-portrait lookup to search the ttmp root recursively (the
  portraits live under the 2026/05/10 UART ticket, a different date dir).
- Dry-ran the 4-algo grid and visually inspected threshold vs. atkinson.

### Why
- Paper is the only place dot gain is real, but a screen dry-run first avoids
  wasting paper on obviously-broken candidates and confirms the card composition.

### What worked
- Harness runs; previews render correctly; the threshold↔atkinson contrast is
  exactly the expected illustration of the midtone-collapse problem.
- Packing stays a single segment (~26 KiB card, under the 36 KiB safe limit).

### What didn't work
- First dry-run fell back to a synthetic radial-gradient "photo" because the
  cat-portrait glob used the wrong date directory; fixed with a recursive
  search from the ttmp root.

### What I learned
- Card width 384 is already a multiple of 8, so no padding surprises.
- At gamma 1.0 the black density is ~30–33%; with thermal dot gain this will
  print darker, so the density sweep should include values below 20.

### What warrants a second pair of eyes
- The header is packed as pure threshold and stacked in bit-space above the
  dithered card so the header text stays crisp — confirm this stacking is what
  reviewers expect vs. dithering the whole strip.
- Gamma direction is still unresolved until the paper ramp test (§7.4).

### What should be done in the future
- Confirm the printer URL and print the first grid (algorithms × densities).
- Get photos back; record paper observations; narrow the sweep.
- Add density/speed UI to the browser lab too (parity with this harness).

### Code review instructions
- `scripts/01-thermal-lab.py`: start at `build_card()` (composition),
  `apply_tone()` (lever 1), `DIFFUSION`/`dith_*` (lever 2), `set_density()` +
  `print_bitmap()` (lever 3 + transport).
- Validate: `python3 01-thermal-lab.py --dry-run --out ./out --grid` then inspect PNGs.

### Technical details
- Grid default: algos `threshold,atkinson,floyd,bayer8` × densities
  `12,20,28,35`; each cell a labeled strip.
- Printer client mirrors `internal/app/printer.go` segmentation (~36 KiB) and
  firmware headers `X-Width`/`X-Height`/`X-Feed`.

## Step 3: First physical batch — baseline algorithm comparison on paper

Printed the 4-algorithm baseline (threshold, atkinson, floyd, bayer8) at
density 20, neutral tone (gamma 1.0, contrast 1.0, brightness 0). Printer was at
`http://192.168.0.126` (NOT the older lab's `.242`, which timed out — the user
corrected the address; status endpoint confirmed paper present, not overheated).
The user photographed the four strips (Downloads IMG_2703..2706, HEIC →
converted to JPEG with `heif-convert`).

The paper results are decisive and match the on-screen dry-run prediction: this
is the first real data point in the lab→paper loop.

### Prompt Context

**User prompt (verbatim):** "look at the 4 last files in ~/Downloads/"
(preceded by "i think the printer is 192.168.0.126 ? that's the printer hardware")

**Assistant interpretation:** Read the four photographed thermal strips and
extract paper observations to drive the next tuning round.

**Inferred user intent:** Close the first lab→paper loop and decide what to
print next.

**Commit (code):** N/A (analysis + next-batch prints).

### What I did
- Converted the four HEIC photos to JPEG and read them.
- Scored each algorithm on: photo tonal detail, text legibility, gray-ramp
  reproduction, overall darkness.

### What worked / findings
- **Atkinson = best photo.** Fur, whiskers, eye catchlights, smooth midtones;
  not muddy, not too dark. black≈33.9%.
- **Floyd = close 2nd**, slightly denser/darker in shadows. black≈33.3%.
- **Bayer8**: midtones present but a distracting cross-hatch grid in flat areas.
- **Threshold** (current production): photo collapses to black blobs + blown
  highlights; chest is a solid black mass; ramp = black bar then white. This IS
  the "hard to read" the user reported.
- **Text survives dithering.** Body text renders pure black, so error diffusion
  has no error to spread there — 13/11 px stay crisp under Atkinson/Floyd; only
  9 px anti-aliased edges soften slightly. => whole-page dithering is far less
  harmful to text than the guide warned; segmented printing is lower priority.
- **Gray ramp crushes in the darkest ~20%** in every mode (dot gain). Midtones/
  shadows want LIGHTENING, i.e. gamma < 1 in the `v'=v^γ` convention. This is
  the direction to test next.
- **Density 20 is roughly right**, maybe slightly light for small text; worth
  nudging up for text punch while watching photo mud.

### What didn't work
- Printer `.242` (from the old lab) is dead here; correct address is
  `192.168.0.126`. Recorded so future runs don't repeat the timeout.

### What I learned
- The headline win is simply **threshold → Atkinson**: it converts the exact
  failure the user described into a legible, detailed print with no other change.
- For our content (pure-black text), the text/photo conflict is milder than
  theory predicts, which de-risks whole-page dithering as a near-term default.

### What warrants a second pair of eyes
- Confirm gamma direction on paper before baking a default (Step 4 sweep).

### What should be done in the future
- Step 4: Atkinson gamma sweep {0.7, 1.0, 1.4} at density 24 to settle the tone
  direction and pick a shadow-opening gamma.
- Then a density sweep around the chosen gamma for text punch vs. photo mud.
- Carry Atkinson (and Floyd as alternate) forward; drop Bayer8 and threshold for
  photos.

### Technical details
- Photos: `~/Downloads/IMG_2703.HEIC` (threshold), `IMG_2704` (atkinson),
  `IMG_2705` (floyd), `IMG_2706` (bayer8); JPEG copies in the session scratchpad.
- Print cmd: `python3 01-thermal-lab.py --printer http://192.168.0.126 --grid
  --algos threshold,atkinson,floyd,bayer8 --densities 20 --gamma 1.0
  --contrast 1.0 --brightness 0.0 --feed 4`.

## Step 4: Gamma sweep result + text heat calibration (density/speed)

Printed the Atkinson gamma sweep {0.7, 1.0, 1.4} at density 24. User feedback:
"the sweet spot is between the first and the second (for the cat)" — i.e. the
cat wants **gamma ≈ 0.8** (between 0.7 and 1.0), confirming the `v'=v^γ`
convention with γ<1 lightening/opening shadows. γ=1.4 (darken) was the wrong
direction, as predicted from the ramp crush.

The user then raised text: it needs its own experimentation, "including setting
the heat contrast on the printer too." I confirmed the printer's "heat" is the
density register (`ESC ## STDP`, 0..39) — there is no separate contrast register,
density *is* the contrast/darkness knob — and that **print speed**
(`ESC ## STSP`) is a second heat lever (slower = longer dwell = darker). Because
our print path already sends the page as separate segment commands, density can
be set *between* segments, so a text segment can print hotter than a photo
segment in one job **without firmware changes** — the real mixed-page answer.

I extended the harness with a **text card** (glyph legibility 16→8 px + a
reverse white-on-black bleed bar) and **speed control/sweep**, then printed a
text density sweep {16, 24, 32, 39} at speed 80.

### Prompt Context

**User prompt (verbatim):** "i think the sweet spot is between the first and the second (for the cat). \n\nI think for text we should experiment more, including setting the heat contrast on the printer too? What do you think"

**Assistant interpretation:** Lock the cat near gamma 0.8; investigate text
legibility as a function of printer heat (density) and speed, which are separate
from the dithering algorithm.

**Inferred user intent:** Get crisp, dark, legible text — tuning the printer's
heat, not just the host-side rasterization — while keeping the photo good.

**Commit (code):** see changelog.

### What I did
- Added `build_text_card()` (sizes 16/14/13/12/11/10/9/8 with counter-heavy
  glyphs + a reverse-video bleed bar) and `set_speed()` + `--card`,
  `--speed`/`--speeds` to `scripts/01-thermal-lab.py`.
- Verified the text card via dry-run (black≈16.5%).
- Printed text density sweep {16, 24, 32, 39} at speed 80.

### Why
- Text is pure black, so the dither algorithm is irrelevant to it; its quality
  is governed by heat (density × speed) and font size. A dedicated text card
  isolates those.

### What worked
- Harness now sweeps algo × density × speed and switches card type; dry-run of
  the text card confirms clean multi-size rendering.

### What I learned / reasoning
- Cat locks near **Atkinson, gamma ~0.8**, density low-20s.
- The mixed-page conflict is addressable via **per-segment density** on the
  existing segmented print path — no firmware work needed.

### What warrants a second pair of eyes
- The text density ceiling: where dot gain closes 8–9 px counters (need paper).

### What should be done in the future
- Read the text density sweep photos; pick the text heat sweet spot.
- If text wants a hotter density than the photo, prototype per-segment density
  in the harness (text segment hot, photo segment cooler) and print a full
  mixed page to prove the mixed-page fix.
- Optionally a speed sweep at the chosen density (e.g. 80 vs 50 vs 100).

### Technical details
- Text sweep cmd: `python3 01-thermal-lab.py --printer http://192.168.0.126
  --card text --algo threshold --densities 16,24,32,39 --speed 80 --feed 4`.
- Valid speeds (firmware): 25,30,37,50,56,62,70,80,90,100,120,150,180,200,220.

## Step 5: Text density ladder result + power-droop finding + density×speed matrices

Read the text density ladder {16, 24, 32, 39} at speed 80. Result: d16 too light
(9–10 px breaks up), d24 good, **d32 best punch** (solid to ~11 px, 9–10 px just
starting to fill), d39 darkest but the smallest lines begin to bleed/close
counters. So the **text heat sweet spot is ~28–32** — clearly hotter than the
cat's low-20s, confirming that text and photo want different heat and that a
per-segment density split is worthwhile.

The user noticed grey tone varying **across a single line**, most on the two
lower-density strips, and guessed speed. That is **print-head power droop**:
a raster row firing many dots sags the shared power rail, so dots print lighter
across parts of the line; it shows at low density (marginal dots) and is masked
at higher density. Slower speed gives more dwell/recovery and more even heating,
so speed is the right lever. Also learned the "Print depth: N level" banners
between strips are printed by the *firmware* on each density change, not by the
harness.

Per the user's "make extensive matrix work," I printed two density×speed
matrices.

### Prompt Context

**User prompt (verbatim):** "ok, last 4 images again. What's weird is grey tone changing over a single line, at least on the first 2, maybe it the's the speed? anyway, you know best, make extensive matrix work"

**Assistant interpretation:** Diagnose the within-line tone variation (power
droop; speed-related) and run a comprehensive density×speed matrix for text and
photo to find the sweet spots and the evenness behavior.

**Inferred user intent:** Nail down the printer heat settings (density + speed)
that give even, legible text and a good photo, with data across the full grid.

**Commit (code):** N/A (analysis + matrix prints); harness already committed.

### What I did
- Analyzed the 4 text strips (IMG_2707..2710 = d16/24/32/39).
- Printed TEXT matrix: density {20,28,36} × speed {37,62,100} (9 strips).
- Printed PHOTO matrix: Atkinson γ0.8, density {18,24,30} × speed {37,100} (6).

### What worked / findings
- Text sweet spot ~28–32 density; d39 over-bleeds small glyphs.
- Within-line tone variation = power droop, expected to ease at slower speed.
- Text wants more heat than the photo → per-segment density split justified.

### What warrants a second pair of eyes
- Confirm on paper that slower speed (37) visibly evens out the within-line tone
  vs. fast (100), at matched density.

### What should be done in the future
- Read the 15 matrix strips; pick (density, speed) for text and for photo.
- Prototype per-segment density (text hot, photo cooler) and print a full mixed
  Almanach page to prove the mixed-page fix.
- Then port the locked recipe(s) into `internal/app/raster` + a density/speed
  client for `internal/app/printer.go`.

### Technical details
- TEXT matrix cmd: `... --card text --algo threshold --densities 20,28,36
  --speeds 37,62,100 --feed 4` (combos ordered speed-outer, density-inner).
- PHOTO matrix cmd: `... --card mixed --algo atkinson --gamma 0.8
  --densities 18,24,30 --speeds 37,100 --feed 4`.
- Text strips IMG_2707=d16, 2708=d24, 2709=d32, 2710=d39 (all speed 80).
