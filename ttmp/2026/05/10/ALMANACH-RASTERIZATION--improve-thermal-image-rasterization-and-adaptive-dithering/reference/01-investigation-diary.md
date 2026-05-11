---
title: Investigation diary
docType: reference
topics: [almanach, thermal-printer, backend, firmware, go]
status: active
intent: long-term
---

# Diary

## Goal

Capture the creation of the `ALMANACH-RASTERIZATION` research ticket, the current-code investigation, external research, design-document writing, and reMarkable delivery for future adaptive thermal rasterization work.

## Step 1: Create the rasterization research ticket and collect evidence

The user asked for a new ticket that preserves the rasterization research and turns it into a detailed intern-oriented analysis/design/implementation guide. I created `ALMANACH-RASTERIZATION`, added the primary design document and diary, and started by grounding the recommendations in the current Almanach code path instead of writing only from general image-processing knowledge.

The investigation confirmed that Almanach currently uses a simple whole-page screenshot pipeline: Chrome renders the React layout, Go captures a PNG screenshot, `internal/app/bitmap.go` converts pixels using a fixed threshold, and the host posts a packed 1-bit bitmap to the firmware endpoint.

### Prompt Context

**User prompt (verbatim):** "Ok, Createa new ticket to improve rasterization, and store all this research you did in there. You can also use `surf kagi search --query ...` and `surf kagi assistant --help` to do web searches and llm assistant research, and use defuddle to download solid reference material and read and store in the ticket.\n\nCreate  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a structured docmgr ticket for future rasterization improvements, research dithering/halftoning techniques with web sources, write a detailed intern-ready guide tied to the current codebase, and upload the resulting bundle to reMarkable.

**Inferred user intent:** Preserve the design thinking and give a future implementer enough technical context to improve image quality without breaking the working printer path.

**Commit (code):** N/A — documentation/research step only.

### What I did
- Created ticket `ALMANACH-RASTERIZATION` with topics `almanach, thermal-printer, backend, firmware, go`.
- Added documents:
  - `design-doc/01-adaptive-thermal-rasterization-analysis-design-and-implementation-guide.md`
  - `reference/01-investigation-diary.md`
- Added tasks for architecture mapping, external research, guide writing, validation, and upload.
- Inspected current rendering/printing files:
  - `internal/app/bitmap.go`
  - `internal/app/renderer.go`
  - `internal/app/printer.go`
  - `internal/app/cmd_print.go`
  - `internal/app/server.go`
  - `internal/app/render_oneshot.go`
  - `internal/app/layout_bundle.go`
  - `web/src/almanach-studio.jsx`
  - `firmware/atoms3r/main/web_server.c`
- Used `surf kagi search` for thermal printer dithering, halftoning, ImageMagick dithering, and error diffusion references.
- Used `surf kagi assistant --help`, then ran a Kagi Assistant research prompt after discovering the assistant CLI takes the prompt as a positional argument rather than `--prompt`.
- Used `defuddle parse --md` to store external source material in the ticket.

### Why
- The current fixed-threshold rasterizer is too limited for photographs and edge-sensitive images.
- The design needs to be tied to code paths and constraints: screenshot rendering, packed bitmap format, firmware body limit, and the future segmented endpoint.
- Storing sources and assistant research in the ticket makes future implementation work reproducible.

### What worked
- Ticket and docs were created successfully.
- Kagi search and Kagi Assistant produced useful algorithm and implementation references.
- Defuddle successfully downloaded:
  - ImageMagick quantization/dithering material,
  - Tanner Helland dithering algorithms article,
  - Sweetcorn dithering algorithm notes.
- Code inspection produced concrete line references for the current threshold conversion and print contract.

### What didn't work
- `surf kagi assistant --prompt ...` failed with:
  - `Error: unknown flag: --prompt`
- I corrected that by invoking the assistant with the prompt as a positional argument.
- `defuddle parse` for the Adafruit thermal paper article failed with:
  - `Error: Failed to fetch https://blog.adafruit.com/2016/05/05/the-floyd-steinberg-dithering-algorithm-for-thermal-paper-printing/: 403 Forbidden`
- I kept the Adafruit URL in the source index but did not store extracted content for it.

### What I learned
- The current rasterization improvement seam is `PngToBitmap(screenshotBuf, opts.Threshold)` in `internal/app/renderer.go`.
- `RenderOptions` and top-level layout `render` maps already provide an extension path for `rasterMode`, tone curve, and edge-preservation options.
- The firmware and host already enforce a 90 KiB packed bitmap guard, so comparison sheets must account for final bitmap height.
- The screenshot pipeline flattens semantic blocks, so true per-block image/text rasterization probably needs either browser-generated masks or the future segmented print endpoint.

### What was tricky to build
- The main design challenge is architectural, not just algorithmic: if the whole page is already a PNG screenshot, text and images cannot be treated independently without extra metadata.
- The recommendations must avoid destabilizing the proven firmware path. The first phase should keep the same packed bitmap contract and only change host-side black/white conversion.
- External web references vary in authority and implementation detail; the guide distinguishes source material from proposed Almanach-specific choices.

### What warrants a second pair of eyes
- Confirm whether the first implementation should add a small internal raster package or adopt/evaluate a Go dithering dependency such as `esimov/dithergo`.
- Confirm whether default raster mode should remain fixed threshold until hardware comparison sheets pick a better default.
- Review the edge-hybrid design carefully; it can preserve whiskers and contours but may also add noisy speckles if tuned poorly.

### What should be done in the future
- Implement the Phase 1 threshold-preserving raster options and Atkinson mode.
- Print a comparison sheet with cat portraits on the real K118 printer.
- Add debug artifacts for raster layers and black-density metrics.
- Integrate with segmented/chunked printing once that endpoint exists.

### Code review instructions
- Start with the design doc section “Current-state architecture”.
- Then inspect:
  - `internal/app/bitmap.go` for current conversion,
  - `internal/app/renderer.go` for screenshot handoff,
  - `internal/app/printer.go` and `firmware/atoms3r/main/web_server.c` for packed bitmap limits,
  - `web/src/almanach-studio.jsx` for image block tone controls.
- Validate docs with:
  - `docmgr doctor --ticket ALMANACH-RASTERIZATION --stale-after 30`

### Technical details
- Ticket path:
  - `ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/`
- Design doc:
  - `design-doc/01-adaptive-thermal-rasterization-analysis-design-and-implementation-guide.md`
- Sources:
  - `sources/00-source-index.md`
  - `sources/02-imagemagick-quantize-dithering.md`
  - `sources/03-tanner-helland-dithering-eleven-algorithms.md`
  - `sources/04-sweetcorn-dithering-algorithms.md`
  - `sources/05-kagi-assistant-rasterization-research.md`


## Step 2: Build a self-contained retro monochrome browser dithering lab

The user asked for a self-contained macOS 1 style monochrome HTML/JS page for experimenting with dithering algorithms. I created the page inside the rasterization ticket and embedded the verified cat portrait crops as base64 data-URI presets so the page does not depend on `/tmp` files or a running server.

The lab is intentionally standalone: all CSS, JavaScript, and default images live in one HTML file. It supports drag-and-drop image loading and implements the first set of candidate algorithms directly in browser JavaScript so rasterization parameters can be tweaked interactively before any Go implementation work begins.

### Prompt Context

**User prompt (verbatim):** "Create a self contained macos1 retro monochrome looking html + js page where we can try out and tweak all the different algorithms to do dithering, using the cat portraits as efault presets but also allowing me to drop my own pngs. Then open it with firefox"

**Assistant interpretation:** Build an offline single-file interactive dithering playground, preloaded with the ticket-local cat crops, with controls for multiple raster algorithms and parameters, then launch it in Firefox.

**Inferred user intent:** Quickly compare visual effects of rasterization algorithms and parameter choices before implementing them in the production Go rendering pipeline.

**Commit (code):** N/A at diary write time; commit follows after pending docs/assets are reviewed.

### What I did
- Created:
  - `various/raster-dither-lab.html`
- Embedded all 16 verified cat portraits from the `ALMANACH-PRINTER-UART` ticket as data URIs.
- Added drag/drop and file input support for custom PNG/JPEG/WebP images.
- Added macOS 1 inspired monochrome UI styling:
  - menu bar,
  - pixel/dither desktop background,
  - black-and-white windows,
  - title bars,
  - chunky borders and button shadows.
- Implemented interactive algorithms:
  - fixed threshold,
  - adaptive mean threshold,
  - Bayer 4x4,
  - Bayer 8x8,
  - deterministic hash noise mask,
  - Floyd-Steinberg,
  - Atkinson,
  - Stucki,
  - Burkes,
  - Sierra 2-row,
  - Sierra Lite,
  - edge hybrid using Atkinson plus Sobel edge recovery.
- Added controls for:
  - max output width,
  - threshold,
  - brightness,
  - contrast,
  - gamma,
  - dither strength,
  - edge threshold.
- Added output stats:
  - output dimensions,
  - packed printer byte estimate,
  - black pixel density,
  - remaining margin under the current 90 KiB guard.

### Why
- The production Go implementation should be guided by visual tests on real examples.
- A standalone browser lab makes it fast to tune algorithm parameters without rebuilding Go or firmware.
- Embedding cat presets makes the tool reproducible after crashes and usable offline.

### What worked
- The self-contained HTML file was generated successfully.
- It includes 16 cat presets and is about 5.1 MB.
- The page is stored in the ticket, not `/tmp`.

### What didn't work
- N/A for file creation. Firefox launch is recorded in the next command/step if the local environment accepts GUI launch.

### What I learned
- Browser JavaScript is enough for quick experimentation at 384px thermal-printer widths.
- The same controls proposed in the design doc map naturally to an interactive lab: threshold, tone curve, diffusion strength, and edge threshold.

### What was tricky to build
- The page had to be self-contained while still carrying useful default images, so embedding all cat portraits as base64 data URIs was the simplest robust option.
- The “blue-noise” option is currently a deterministic hash-mask approximation, not a true void-and-cluster blue-noise mask; the UI labels it accordingly.

### What warrants a second pair of eyes
- Verify the algorithm implementations against the eventual Go implementation before using lab output as a formal golden reference.
- Review whether the hash-mask mode should be renamed or replaced with an embedded true blue-noise matrix.

### What should be done in the future
- Add a true blue-noise mask.
- Add local black-density cap controls.
- Add export/import of parameter presets as JSON.
- Add a comparison-sheet export suitable for `almanach-render-service print`.

### Code review instructions
- Open `various/raster-dither-lab.html` in Firefox.
- Choose cat presets and compare Atkinson, Floyd-Steinberg, Bayer 8x8, and edge-hybrid.
- Drop a custom PNG and verify stats update.

### Technical details
- Page path:
  - `ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html`
- Source cat assets:
  - `ttmp/2026/05/10/ALMANACH-PRINTER-UART--analyze-atoms3r-k118-printer-uart-serial-interface/assets/cat-portraits/portraits/`


## Step 3: Add browser-side print-to-printer button to the dithering lab

The user asked to add a print button directly to the self-contained dithering lab. I updated the HTML page so it can pack the currently rasterized 1-bit output into the same MSB-first bitmap body used by the firmware endpoint and attempt a direct browser `POST` to the configured printer URL.

Because the page is opened from `file://`, browser CORS/private-network rules may block a direct request to `http://192.168.1.242/api/print/bitmap`. The UI reports that error clearly if it happens. The implementation still uses the real firmware contract (`X-Width`, `X-Height`, `X-Feed: 0`, octet-stream body) so it can work in a permissive browser context or after adding a localhost proxy.

### Prompt Context

**User prompt (verbatim):** "add a print to printer button"

**Assistant interpretation:** Extend the standalone raster lab with a UI action that sends the current dithered bitmap to the K118 printer.

**Inferred user intent:** Make the browser lab a quick physical-output testing tool, not only a visual preview tool.

**Commit (code):** N/A at diary write time; commit follows after pending docs/assets are reviewed.

### What I did
- Updated `various/raster-dither-lab.html`.
- Added a `Printer URL` input defaulting to:
  - `http://192.168.1.242/api/print/bitmap`
- Added a `Feed lines` slider.
- Added a full-width `Print to Printer` button.
- Added browser-side MSB-first bit packing with width padded to a multiple of 8.
- Appended feed rows in the bitmap body, matching the host-side strategy used in `internal/app/printer.go`.
- Added a 90 KiB guard before POSTing.
- Added print result/error text to the stats panel.

### What worked
- The standalone HTML page now contains the print UI and packing logic.
- The stats panel now accounts for feed rows when showing byte size and 90 KiB margin.

### What didn't work
- I did not physically click the browser button from the agent. Direct `file://` to printer fetches may hit CORS/private-network restrictions depending on Firefox settings and firmware headers.

### What I learned
- The lab can reproduce the firmware bitmap contract entirely in browser JavaScript.
- A future localhost proxy endpoint would make browser printing from the lab more reliable than direct `file://` cross-origin POSTs.

### What was tricky to build
- The printer endpoint requires custom headers (`X-Width`, `X-Height`), which are exactly the kind of headers that can trigger browser CORS preflight.
- Feed rows need to be included in the packed body because the host currently bakes feed spacing into the bitmap and sends `X-Feed: 0`.

### What warrants a second pair of eyes
- Verify direct browser print behavior in Firefox. If blocked by CORS, add a localhost proxy endpoint to the Go setup server.
- Confirm whether the lab should target the printer directly or always target localhost.

### What should be done in the future
- Add `/api/print/bitmap` proxy support to the localhost setup server for browser lab output.
- Add an export button for raw packed `.bin` plus metadata JSON.

### Code review instructions
- Open the lab in Firefox, choose a portrait and algorithm, then click `Print to Printer`.
- If it fails with CORS/private-network text, use that as evidence for the localhost proxy follow-up.

### Technical details
- Updated page:
  - `ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html`
- Browser packing function:
  - `packBitsWithFeed(bits,w,h,feedLines)`


## Step 4: Add a localhost server and printer proxy for the dithering lab

The browser reported a CORS failure when the `file://` dithering lab tried to POST directly to the printer at `http://192.168.1.242/api/print/bitmap`. Serving the HTML from an HTTP server helps, but by itself it does not fully solve the problem if the page still posts to the printer IP: `localhost` and `192.168.1.242` are still different origins, and custom headers such as `X-Width` and `X-Height` trigger preflight behavior.

I added a small ticket-local Python server that serves the lab from localhost and proxies same-origin `POST /api/print/bitmap` requests to the printer. The lab now defaults its Printer URL to `/api/print/bitmap`, so the browser posts to localhost and the Python process forwards the raw bitmap body to the printer.

### Prompt Context

**User prompt (verbatim):** "Cross-Origin Request Blocked: The Same Origin Policy disallows reading the remote resource at http://192.168.1.242/api/print/bitmap. (Reason: CORS request did not succeed). Status code: (null).

do we need to serve the html from a http server?"

**Assistant interpretation:** Diagnose the CORS failure and provide a practical way to print from the browser lab without direct cross-origin printer requests.

**Inferred user intent:** Make the `Print to Printer` button actually usable from Firefox.

**Commit (code):** N/A at diary write time; commit follows after pending docs/assets are reviewed.

### What I did
- Added:
  - `scripts/01-serve-raster-dither-lab.py`
- The script serves:
  - `http://localhost:18301/raster-dither-lab.html`
- The script proxies:
  - browser `POST http://localhost:18301/api/print/bitmap`
  - to `http://192.168.1.242/api/print/bitmap`
- Updated the lab default Printer URL from the printer IP to:
  - `/api/print/bitmap`
- Started the server:
  - PID file `/tmp/almanach-raster-lab-server.pid`
  - log `/tmp/almanach-raster-lab-server.log`
- Opened the localhost URL in Firefox.

### What worked
- The lab is now reachable over HTTP at:
  - `http://localhost:18301/raster-dither-lab.html`
- A GET request returned the HTML page.
- Firefox was launched with the localhost URL.

### What didn't work
- A `curl -I` HEAD request returned `501 Unsupported method ('HEAD')` because the tiny Python handler only implements GET/POST/OPTIONS. This does not affect normal browser GET loading.

### What I learned
- The right browser architecture is not merely “serve the HTML over HTTP”; it is “serve the HTML and provide a same-origin localhost proxy”.
- Direct browser requests to the printer are fragile because of CORS/private-network restrictions and custom headers.

### What was tricky to build
- The proxy must preserve the firmware bitmap contract exactly: raw octet-stream body, `X-Width`, `X-Height`, and `X-Feed` headers.
- The proxy should keep the 90 KiB guard so the browser path does not bypass host-side safety.

### What warrants a second pair of eyes
- If the lab becomes permanent, replace the Python proxy with a Go setup-server route so all local tooling lives in the Almanach binary.
- Consider adding `do_HEAD` to the Python script for nicer diagnostics.

### What should be done in the future
- Add a proper `/api/print/bitmap` localhost proxy route to `almanach-render-service setup`.
- Keep the standalone Python script as a ticket-local experiment until the Go route exists.

### Code review instructions
- Start the lab server with:
  - `python scripts/01-serve-raster-dither-lab.py --port 18301 --printer http://192.168.1.242`
- Open:
  - `http://localhost:18301/raster-dither-lab.html`
- Keep Printer URL as:
  - `/api/print/bitmap`
- Click `Print to Printer` and inspect both the browser stats panel and `/tmp/almanach-raster-lab-server.log`.

### Technical details
- Server script:
  - `ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/scripts/01-serve-raster-dither-lab.py`
- Updated HTML:
  - `ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html`
- Running URL:
  - `http://localhost:18301/raster-dither-lab.html`


## Step 5: Split lab JavaScript and print settings header with each raster image

The user asked for the printed output to include the selected algorithm and settings, and said it was fine to separate the JavaScript to make the lab easier to manage. I split the large inline script out of the HTML file into a neighboring `raster-dither-lab.js` file and changed the print path so it prints a monochrome settings header above the dithered image.

The browser preview remains the same: it shows source and 1-bit output. The print path now builds a temporary offscreen print canvas that contains the settings text plus the dithered output, thresholds that composite canvas to 1-bit, packs it MSB-first, appends feed rows, and posts it through the localhost proxy.

### Prompt Context

**User prompt (verbatim):** "print the settings + algorithm when printing. Feel free to separate the js out so that it's easier to edit and manage."

**Assistant interpretation:** Make the physical print self-documenting and refactor the lab so future edits do not require working inside one huge HTML file.

**Inferred user intent:** When comparing physical dither outputs, each strip should identify exactly which algorithm and parameters produced it.

**Commit (code):** N/A at diary write time; commit follows after pending docs/assets are reviewed.

### What I did
- Split the inline lab JavaScript into:
  - `various/raster-dither-lab.js`
- Kept the HTML shell in:
  - `various/raster-dither-lab.html`
- Added a print-only settings header with:
  - `ALMANACH RASTER LAB`,
  - image/preset name,
  - algorithm,
  - threshold,
  - brightness,
  - contrast,
  - gamma,
  - dither strength,
  - edge threshold,
  - bitmap dimensions and bytes.
- Replaced direct bit packing from `lastBits` with a print-composite flow:
  - `makePrintCanvas(c)` draws the settings header and current output canvas,
  - `packCanvasWithFeed(canvas, feedLines)` thresholds and packs the composite canvas.
- Ran a JavaScript syntax check:
  - `node --check .../raster-dither-lab.js`
- Re-opened the localhost lab URL in Firefox.

### What worked
- The JS syntax check passed.
- The lab server served the split JS file from `http://localhost:18301/raster-dither-lab.js`.
- Firefox was reopened to `http://localhost:18301/raster-dither-lab.html`.

### What didn't work
- N/A. I have not clicked the browser print button from the agent; the user should test the physical output in Firefox.

### What I learned
- Separating the JS makes future algorithm edits much easier, even though the JS file still contains embedded base64 cat presets.
- A print-only offscreen canvas is the cleanest way to include labels without changing the visual comparison preview.

### What was tricky to build
- The printed header must be included before 1-bit packing; otherwise it would require a separate printer text call and ordering would be less reliable.
- The header canvas width is padded to at least 384 pixels so the text has enough room and remains aligned with thermal paper width.

### What warrants a second pair of eyes
- Review whether the settings header should include more fields, such as max output width, feed lines, black density, and timestamp.
- Verify physical readability of the 12px monospace header on the K118.

### What should be done in the future
- Add a “print comparison sheet” mode that prints multiple algorithms, each with its own header.
- Move embedded cat presets into a separate generated JS/data file if the lab grows further.

### Code review instructions
- Open `http://localhost:18301/raster-dither-lab.html`.
- Choose an algorithm and click `Print to Printer`.
- Confirm the physical print includes the settings header above the image.

### Technical details
- HTML shell:
  - `ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.html`
- JavaScript:
  - `ttmp/2026/05/10/ALMANACH-RASTERIZATION--improve-thermal-image-rasterization-and-adaptive-dithering/various/raster-dither-lab.js`
