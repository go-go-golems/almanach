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
