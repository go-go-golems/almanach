---
Title: Diary
Ticket: ALMANACH-WORKSLIP
Status: active
Topics:
    - layout-dsl
    - thermal-printer
    - typography
    - security
DocType: reference
Intent: long-term
Owners: []
ExternalSources: []
Summary: "Implementation diary for the work-slip integration (slip-studio block pack, theme tokens, $ENV removal)."
LastUpdated: 2026-07-16T21:20:00-04:00
WhatFor: "Review and continuation of the ALMANACH-WORKSLIP implementation."
WhenToUse: "Read before continuing or reviewing this ticket's work."
---

# Diary

## Goal

Capture the implementation journey of ALMANACH-WORKSLIP: analysis of the
slip-studio prototype, removal of the `{{$ENV}}` template feature, the generic
work-slip block pack, swiss/brutalist theme tokens, and the ported example
layouts.

## Step 1: Ticket, analysis, intern-level design guide, reMarkable upload

Analyzed `~/Downloads/slip-studio.html` (a single-file canvas-based receipt
IDE for work/freelance printouts) against the current DSL v2 architecture and
wrote the full integration design: keep Almanach's Chrome/rasterize/print
pipeline, port slip-studio's generic block vocabulary and theme tokens, drop
its canvas renderer and its template-binding language entirely, and remove
`{{$ENV}}` from the Go template engine as hardening. Created the
ALMANACH-WORKSLIP ticket, wrote a ~600-line intern-oriented design/implementation
guide into the ticket, rendered it to PDF (pandoc + mmdc-prerendered mermaid +
headless-Chrome print) and uploaded it to reMarkable (`Projects/`).

### Prompt Context

**User prompt (verbatim):** "remove it too. Create a new docmgr ticket for all this, with a Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.\n\n\nThen add phases and tasks per phase and implement them one by one, committing at appropriate intervals, keeping a etailed diary"

**Assistant interpretation:** Remove `$ENV` template support (in addition to the
already-decided template-binding removal); open a docmgr ticket; write a
detailed intern-level design/implementation guide stored in the ticket; export
it to reMarkable; then create per-phase tasks and implement all phases with
commits and this diary.

**Inferred user intent:** Turn the slip-studio design discussion (this
conversation's earlier turns) into a fully documented, reviewable, implemented
feature, with onboarding-quality documentation readable on the reMarkable.

### What I did
- Read the full current system before designing: `almanach-studio.jsx` (all
  2,791 lines), `blocks/registry.js`, `typography/presets.js`, `layout.proto`,
  `template.go`, `data_context.go`, `layout_bundle.go`, `renderer.go`, and
  `~/Downloads/slip-studio.html` (all 1,328 lines).
- `docmgr ticket create --ticket ALMANACH-WORKSLIP ...`
- `docmgr doc add` (design-doc + this diary); wrote
  `design-doc/01-work-slip-integration-analysis-design-and-implementation-guide.md`:
  Part I system tour (pipeline diagram, proto schema, registry/preset APIs,
  Go rasterize/print internals, `{{key}}` engine + threat model), Part II
  slip-studio dissection (what ports, what's discarded, block semantics),
  Part III per-phase design with pseudocode (row recursion, QR module
  rendering, ThemeTokens proto, token accessors, preset sizes).
- PDF export: stripped frontmatter (awk), `pandoc -f gfm`, pre-rendered the
  mermaid pipeline diagram with `mmdc` (PNG, scale 2), embedded as data URI,
  printed via `google-chrome-stable --headless --print-to-pdf`. Uploaded with
  `rmapi put ... Projects` (14 pages).
- `docmgr task add` × 4 (one per phase, ids s06o/xtye/w279/lbno).

### Why
- The guide is the ticket's contract: the user asked for onboarding-quality
  documentation before implementation, and the phase plan flows from it.
- `$ENV` removal was decided in design review this session: any layout file
  can read arbitrary process env vars once a data context is present, and the
  resolved values leak into PNGs/prints/debug artifacts; no consumer exists.

### What worked
- mmdc is installed and renders mermaid fine with
  `--puppeteerConfigFile <(echo '{"executablePath":"/usr/bin/google-chrome-stable","args":["--no-sandbox"]}')`.
- `rmapi` was already authenticated; upload to `Projects/` succeeded first try.

### What didn't work
- Double-applying `sed '1,/^---$/d'` to strip frontmatter produced an empty
  file (the second pass consumed to EOF since there is no third `---`);
  switched to a single awk frontmatter stripper.
- Mermaid via CDN ESM inside the printed HTML rendered blank in
  `--print-to-pdf` even with `--virtual-time-budget` (module load + async
  render vs. print timing). Pre-rendered SVG *also* drew blank in print
  (mermaid SVG print quirk). PNG via `mmdc -s 2` worked; had to cap it with
  `max-width:70%; max-height:230mm` or the tall flowchart spilled across
  pages and got clipped.

### What I learned
- `docmgr ticket create` scaffolds 9 dirs incl. `design/` but `doc add
  --doc-type design-doc` writes to `design-doc/` — both exist afterwards;
  harmless.
- The studio's on-disk block field is `data:` while the proto calls it
  `content` — documented in the guide as a naming quirk to preserve.

### What was tricky to build
- Getting a mermaid diagram into a Chrome-printed PDF reliably. Symptoms: the
  space is reserved but nothing draws. Cause: mermaid renders asynchronously
  after the print snapshot (CDN case) and its SVG output relies on runtime CSS
  that print rasterization drops (SVG case). Fix: render to PNG offline with
  mmdc and embed as a data URI — pixels always print.

### What warrants a second pair of eyes
- The design decision record in the guide (no template binding, `$ENV`
  removal, top-level-only heat regions) — these constrain all four phases.

### What should be done in the future
- Implement Phases 1–4 (tasks s06o, xtye, w279, lbno).

### Code review instructions
- Read the design guide first:
  `ttmp/2026/07/16/ALMANACH-WORKSLIP--*/design-doc/01-*.md`.
- The PDF on reMarkable (`Projects/ALMANACH-WORKSLIP-guide`) is a render of
  the same file.

### Technical details
- PDF build recipe (scratchpad):
  `awk frontmatter-strip | pandoc -f gfm -t html` → replace
  `<pre class="mermaid">` with mmdc PNG data URI → wrap with print CSS →
  `google-chrome-stable --headless --print-to-pdf --no-pdf-header-footer`.
