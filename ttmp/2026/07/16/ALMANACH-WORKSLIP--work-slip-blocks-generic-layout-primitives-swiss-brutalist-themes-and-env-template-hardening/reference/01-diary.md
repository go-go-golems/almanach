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
RelatedFiles:
    - Path: repo://internal/app/template.go
      Note: '{{$ENV}} removal (commit 1de738e)'
    - Path: repo://internal/app/template_boundary_test.go
      Note: boundary tests (commit 1de738e)
ExternalSources: []
Summary: Implementation diary for the work-slip integration (slip-studio block pack, theme tokens, $ENV removal).
LastUpdated: 2026-07-16T21:20:00-04:00
WhatFor: Review and continuation of the ALMANACH-WORKSLIP implementation.
WhenToUse: Read before continuing or reviewing this ticket's work.
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

## Step 2: Phase 1 — remove {{$ENV}}, pin the env boundary with tests

Removed the environment-variable branch from the template expression resolver
so a layout can never read process env vars, updated the template tests to
assert the *absence* of the behavior, and added boundary tests around
`layoutJSONFromObjectOrDefault`. While writing the boundary test I found the
vulnerability was worse than designed-for: the function merges a layout's own
`data:` map into the data context, so template resolution could be
self-activated by any layout — including one POSTed to the HTTP server — and
`{{$SECRET}}` would then have resolved server-side. The "server never
resolves" assumption from the design review was wrong; removal (not
allowlisting) was the right call.

### Prompt Context

**User prompt (verbatim):** (see Step 1 — "remove it too. Create a new docmgr ticket for all this, ...")

**Assistant interpretation:** Execute Phase 1 of the plan: delete `$ENV`
support, keep `{{key}}`/`{{key:fallback}}` intact, prove the boundary with
tests, update docs.

**Inferred user intent:** Close the env-exfiltration hole before layouts start
arriving from upstream generators.

**Commit (code):** 1de738e — "fix(template): remove {{$ENV}} resolution — layouts can no longer read process env"

### What I did
- `internal/app/template.go`: deleted the `$`-prefix `os.LookupEnv` branch in
  `resolveExpr` (and the `os` import); left a comment stating the security
  rationale. `$FOO` now behaves like any unknown key (fallback or error).
- `internal/app/template_test.go`: replaced the three env tests with
  `TestResolveValue_EnvVarSyntaxDoesNotResolve` (env set → still an error, and
  the error must not leak the value) and
  `TestResolveValue_EnvVarSyntaxUsesFallbackNotEnv`; replaced the
  context-vs-env test with `TestResolveValue_DollarKeyResolvesFromContext`.
- New `internal/app/template_boundary_test.go`:
  `TestLayoutObject_SelfActivatedDataCannotReadEnv` (layout with its own
  `data:` map + `{{$SECRET}}` must error without leaking) and
  `TestLayoutObject_NoContextLeavesMarkersUnresolved` (nil context → markers
  pass through verbatim; this is what the server path relies on).
- `internal/app/doc/layout-dsl-reference.md`: dropped the two `$ENV` rows,
  added the rationale paragraph, fixed the data-context priority section.

### Why
- Decided in this session's design review; the guide records the threat model.

### What worked
- `go build`, `go test ./internal/app/` and `golangci-lint` all green on the
  first full run; lefthook pre-commit re-ran both.

### What didn't work
- N/A — no failed attempts this step.

### What I learned
- `layoutJSONFromObjectOrDefault` (render_oneshot.go:98-109) merges the
  layout's own `data:` map into the data context before resolving. The design
  guide claimed the server path "never resolves templates because ctx is nil"
  — that was false for layouts carrying `data:`. The boundary test now
  encodes the true invariant: resolution may run, but the environment is
  never a source.

### What was tricky to build
- Nothing structurally; the subtlety was test design — asserting both the
  error *and* that the error string does not contain the env value (an error
  that echoed the resolved value would itself be a leak channel).

### What warrants a second pair of eyes
- Behavior change: `{{$NAME}}` in an existing layout now errors (or takes its
  fallback) instead of reading the environment. No repo layouts or docs used
  it outside the reference table, but any private layout relying on it breaks
  loudly with "template variable \"$NAME\" not provided".

### What should be done in the future
- N/A (phases 2-4 are tracked as tasks).

### Code review instructions
- Start at `internal/app/template.go` `resolveExpr` (the deleted branch), then
  `internal/app/template_boundary_test.go` for the invariant.
- Validate: `GOWORK=off go test ./internal/app/ -run 'TestResolve|TestLayoutObject' -count=1`.

### Technical details
- The self-activation path: `layoutJSONFromObjectOrDefault` → merge
  `obj["data"]` into `wrappedDataCtx` → `ResolveTemplate(layoutMap, wrappedDataCtx)`
  runs whenever that map is non-empty, on both CLI and server code paths.

## Step 3: Phase 2 — the work-slip block pack

Implemented the twelve generic layout primitives from slip-studio as React
block adapters in `web/src/blocks/slip/`, registered them alongside the studio
blocks, and verified all of them end to end with a headless render of a
single smoke layout. The pack is theme-token driven (spacing, rule weights,
banner style) with fallbacks, so every existing theme renders it correctly
before Phase 3 adds token-carrying themes.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Execute Phase 2 of the plan: the block pack,
registry integration, editor support, tests, and a smoke render.

**Inferred user intent:** Make the slip-studio page designs expressible in the
real Almanach pipeline.

**Commit (code):** ddced15 — "feat(slip): work-slip block pack — 12 generic layout primitives"

### What I did
- `web/src/blocks/slip/tokens.js`: `DEFAULT_TOKENS` + `spaceToken`/`ruleToken`/
  `ruleStyleToken`/`bannerStyleToken`/`colWidthStyle` accessors (React-free).
- `web/src/blocks/slip/qr.js`: `buildQrMatrix` over the new bundled
  `qrcode-generator` dep — integer px per module so modules survive 1-bit.
- `web/src/blocks/slip/components.jsx`: SlipText (preset + align + -webkit
  line clamp), Banner (invert/outline via token), Rule (solid/dashed), Space,
  Row (fixed px + fr columns; nested blocks via `ctx.renderBlock`), Kv,
  SlipList, Checks (inline/columns), Writein, Qr (SVG rects,
  crispEdges), Bars, SlipTable.
- `web/src/blocks/slip/adapters.jsx`: `SLIP_ADAPTERS` (module "slip") +
  `SLIP_DEFAULTS` + `SLIP_BLOCK_TYPES`.
- Studio wiring: `mergeBlockRegistries(studio, slip)`; `renderBlock` now
  injects `ctx.renderBlock = (child) => renderBlock(child, ctx)` for container
  blocks; `ALL_DEFAULTS`/`ALL_BLOCK_TYPES` (palette group "Work Slip");
  `GenericJsonEditor` fallback (`EDITORS[type] ?? GenericJsonEditor`) with a
  `lastEmitted` ref so live re-parse doesn't clobber in-progress typing.
- `slip.test.mjs` node test (tokens, widths, QR geometry); wired as
  `test:slip` and into `test:web`. `pnpm add qrcode-generator` (store-dir
  workaround). Rebuilt the SPA; rendered a 14-block smoke layout headless —
  every block correct, and a bogus `repeat` type placeholders as designed.

### Why
- One adapter per primitive keeps the DSL v2 dispatch/registry contract; the
  pack stays a data-styled leaf of the theme system rather than a fork.

### What worked
- The whole pack rendered correctly on the first headless render (382x982
  strip) — banner inversion, kv baseline alignment, QR finder patterns, table
  alignment all right.

### What didn't work
- One test assertion was wrong, not the code: I asserted `dark(1,1) === true`
  for the QR finder pattern; ring row 1 is light except its border, so the
  correct expectations are `dark(1,0)=true, dark(1,1)=false, dark(2,2)=true`.

### What I learned
- `parseLayoutJson` backfills defaults per type, so `SLIP_DEFAULTS` had to
  merge into the same lookup (`ALL_DEFAULTS`) or imported slip layouts would
  lose their default fields.

### What was tricky to build
- Nested block rendering without exposing studio internals to the pack: the
  registry stays React-free and `renderBlock` lives in the studio, so the
  studio injects a `ctx.renderBlock` closure at dispatch time. Container
  blocks (row) call it; unknown child types still get the UnknownBlock
  placeholder for free. The subtlety: the closure recurses with the *parent*
  ctx so the child gets its own `block` reference but shares theme/registry.
- The generic JSON editor's resync loop: applying parsed JSON on each
  keystroke triggers a data-prop change that would re-format the textarea
  under the cursor; a `lastEmitted` ref breaks the cycle by reference
  identity.

### What warrants a second pair of eyes
- `RowBlock` column shorthand: a col without `blocks` becomes a synthetic
  `text` block via `{ ...col, w: undefined }` — check no other col-only field
  should be stripped.
- Line clamp uses `-webkit-box`; fine for Chrome (the only renderer), but
  Firefox studio users would see unclamped text.

### What should be done in the future
- Bespoke inspector forms for the most-used slip blocks (kv, checks) if JSON
  editing proves annoying in practice.

### Code review instructions
- Start at `web/src/blocks/slip/components.jsx`, then the studio diff
  (registry merge, `renderBlock`, `GenericJsonEditor`).
- Validate: `pnpm --dir web test:web`, then a headless render of any layout
  using slip types (`--web-dir web/dist` after `pnpm --dir web build`).

### Technical details
- Smoke render: 384x982 px, threshold 128, crispsans theme, bodyScale 1;
  layout at scratchpad `slip-smoke.yaml` (all 12 types + unknown-type probe).

## Step 4: Phase 3 — theme tokens, work themes, Archivo, slip type scale

Extended the theme model with design tokens (proto `ThemeTokens`: spacing
scale, rule weights, rule/banner style), added the `display/h1/h2/micro`
presets sized for 384-dot paper, gave built-in themes the ability to carry
`presetOverrides`, embedded the Archivo variable font, and shipped the three
work themes: `swiss`, `brutalist`, `terminal`. Verified by rendering the
Phase 2 smoke layout in all three — Archivo renders (distinct grotesque
letterforms), brutalist uppercases everything with slab rules, terminal is
all-mono with dashed rules and an outlined banner.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Execute Phase 3: token model + presets + themes +
fonts.

**Inferred user intent:** Give the block pack the bold slip-studio looks as
data-driven themes.

**Commit (code):** 08e4b8c — "feat(themes): ThemeTokens + work themes (swiss/brutalist/terminal) + slip type scale"

### What I did
- `proto/almanach/layout/v1/layout.proto`: new `ThemeTokens` message
  (`space` map, `rules` map, `rule_style`, `banner_style`); `Theme.tokens = 5`.
  `make proto` regenerated Go + TS; round-trip tests still green.
- `web/src/typography/presets.js`: added `display` (40/900), `h1` (25/900),
  `h2` (19/700), `micro` (10/700 upper, minSize 9) — slip-studio's 576-dot
  scale at ~x0.67.
- `web/src/almanach-studio.jsx`: three new `THEMES` entries with `tokens` and
  `presetOverrides`; `resolveThemeSpec` passes `spec.tokens` through;
  the preset resolver now layers `THEMES[key].presetOverrides` below
  inline-theme overrides and layout typography.
- `web/src/fonts-embedded.css`: appended Archivo as a **single variable-font
  face** (`font-weight: 100 900`, latin subset, woff2 data URI, ~63KB font).

### Why
- Tokens make the pack restylable as data (the whole point of porting themes
  rather than hardcoding looks); built-in `presetOverrides` express
  slip-studio's `forceCase` without a special mechanism.

### What worked
- Google Fonts was reachable, so no fonttools subsetting was needed: the css2
  API serves already-subset woff2 that Chrome loads natively.

### What didn't work
- First download pass produced three identical 35KB payloads for weights
  400/700/900 — Archivo is a variable font and css2 served the same file
  three times. Verified by hashing the base64 payloads (1 distinct), then
  replaced with one `font-weight: 100 900` face. Not a failure that shipped,
  but 2/3 of the font bytes would have been dead weight.

### What I learned
- `fonts-embedded.css` ships verbatim as `web/dist/fonts.css` (103 @font-face
  entries — the DejaVu faces are split per unicode-range).
- Built-in themes previously could NOT override presets — only inline themes
  could (state `themePresets` was fed only from `resolveThemeSpec`). The
  work themes needed it, so the built-in entry's `presetOverrides` now merges
  as the lowest override layer. Resolution chain is now: defaults <- built-in
  theme presetOverrides <- inline-theme presetOverrides <- layout typography
  <- block style.

### What was tricky to build
- Deciding where slip-studio's `forceCase` lives. A theme-level "uppercase
  everything" flag would need plumbing into `resolveStyle`; instead the
  brutalist theme sets `textCase: "upper"` on the affected presets via
  `presetOverrides` — same paper result, zero new mechanism. Deviation from
  the design guide (which sketched a `force_case` token) noted here.

### What warrants a second pair of eyes
- `patch.tokens` replaces the base theme's tokens wholesale for inline themes
  (no deep merge); per-name gaps fall back to pack defaults, not the base
  theme's tokens. Documented in code; revisit if inline themes start patching
  single steps.
- Brutalist `hair` rule = 6px makes kv/table/writein separators heavy slabs —
  intentional look, but check on paper.

### What should be done in the future
- Consider an italic Archivo face if slip layouts ever want italic (only
  normal style is embedded).

### Code review instructions
- Start at the proto diff, then `resolveThemeSpec`/`theme.preset` wiring in
  `almanach-studio.jsx`, then the THEMES entries.
- Validate: `make proto && GOWORK=off go test ./... && pnpm --dir web test:web`,
  then render any slip layout with `theme: swiss|brutalist|terminal`.

### Technical details
- Archivo face: Google Fonts v25 latin subset, variable wght 100-900,
  embedded as `data:font/woff2;base64` (~47KB CSS). All three theme stacks
  fall back to DejaVu Sans.

## Step 5: Phase 4 — example slips, physical prints, documentation

Ported five slip-studio example documents as pre-expanded layouts
(`examples/layouts/10-job-slip.yaml` … `14-morning-digest.yaml`), rendered all
five headless, printed the job slip and the triage card on the K118, copied
the renders into `docs/screenshots/`, and updated both Glazed help entries.
This completes all four phases of the ticket.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Execute Phase 4: examples, verification
(including paper), docs.

**Inferred user intent:** Have ready-to-copy templates for the upstream
job-feed producer, verified on real hardware.

**Commit (code):** 99502fb — "feat(examples): work-slip example layouts 10-14 + docs"

### What I did
- Five example layouts: job slip + decision sheet (swiss), triage card
  (brutalist, includes the payment warning banner + QR), focus card
  (terminal), morning digest (swiss, the slip-studio `repeat` expanded by
  hand into four numbered rows). All `bodyScale: 1`, `paperWidth: 384`.
- Rendered all five headless (all OK); printed 10 and 12 at density 38 on
  192.168.0.126 — the triage card (868 rows) went in 2 segments
  (`ok:true,segments:2`).
- `docs/screenshots/10-*.png … 14-*.png`; `examples/layouts/README.md` gained
  the work-slip section.
- `internal/app/doc/layout-dsl-reference.md`: work-slip block table (12
  types + data fields), theme tokens section, swiss/brutalist/terminal rows
  in the theme table, new presets listed with the `bodyScale: 1` convention.
- `internal/app/doc/layout-typography-and-rendering.md`: "Work slips" section
  (primitives, themes, the two conventions: no binding language, bodyScale 1).
- Verified both help entries load from the built binary.

### Why
- The examples are the contract for whatever generates job JSON upstream —
  each shows the expanded form of what slip-studio expressed with bindings.

### What worked
- All five layouts rendered correctly on the first attempt; the digest's
  nested rows (number column + title/meta stack) compose exactly like the
  slip-studio original.

### What didn't work
- N/A — no failed attempts this step. (Observation, not a failure: the
  brutalist h1 clamps "…Internal Tooling" to 2 lines with an ellipsis at
  384 dots; the slip-studio original has the same `lines: 2` behavior.)

### What I learned
- Per-segment heat kicks in automatically for tall pages (38KiB firmware
  limit), independent of per-block density overrides — `sendBitmapWithHeat`
  reported 2 segments for the 868-row triage card.

### What was tricky to build
- Nothing new mechanically; the work was faithful porting — mapping
  slip-studio's `style:"h1" weight:"black" case:"upper"` onto
  `preset: h1` + block `style: { textCase: upper }` per the design guide's
  naming rule (`style` stays a TextStyle object).

### What warrants a second pair of eyes
- The physical prints: check the brutalist slab rules and the QR scannability
  on paper (QR at 110px, density 38 — scanned fine in the render, verify the
  print).

### What should be done in the future
- A `15-pipeline-stats.yaml` (bars-heavy) example if the bars block earns its
  keep on 58mm paper.
- Upstream: point the Upwork scraper at `10-job-slip.yaml`/`11-decision-sheet.yaml`
  as its output templates.

### Code review instructions
- Render any example:
  `almanach-render-service render --layout examples/layouts/12-triage-card.yaml --out /tmp/t.png --format png --web-dir web/dist --chrome-path /usr/bin/google-chrome-stable`
- Docs: `almanach-render-service help layout-dsl-reference` (Work-Slip Blocks
  section) and `help layout-typography-and-rendering` (Work slips section).

### Technical details
- Prints: density 38, 384 dots wide; job slip 395 rows single segment, triage
  card 868 rows in 2 segments.

## Step 6: Bolder defaults — arm's-length type scale, tight work-theme padding

User feedback on the first physical prints: default fonts should be bigger,
margins much smaller, and the result should feel properly brutalist/bold.
Bumped the slip type scale, gave the work themes size+weight preset overrides
and a tight default page padding, thickened the rule tokens and banners, and
reprinted.

### Prompt Context

**User prompt (verbatim):** "defaults shouldbe bigger fonts and there should be much less margin. [image] Here's an example of what it should look like" — followed mid-turn by "right now the printout doesn't feel brutalist / bold enough"

**Assistant interpretation:** Tune the *defaults* (presets + work themes), not
the example layouts, so any slip gets the big/tight/bold look without
per-layout settings.

**Inferred user intent:** Slips are glanced at from arm's length; the visual
target is the slip-studio mock (near edge-to-edge type, heavy everything).

**Commit (code):** ada024a — "feat(slip): bolder defaults — bigger type scale, tight 10x8 work-theme padding"

### What I did
- Presets: `h1` 25→30 (lineHeight 1.08), `h2` 19→22, `display` 40→46,
  `micro` 10→11.
- Work themes now carry size/weight `presetOverrides`: swiss body 15/600,
  bodyStrong 15/800, caption 13; brutalist nothing under weight 800, h1 32
  and h1/h2/display forced uppercase (mixed-case headline looked wrong on the
  triage card); terminal display 36/h1 23/body 14.
- New `theme.padding` field: `ThermalPaper` resolves
  `layout margin ?? theme.padding ?? legacy defaults`; work themes set
  `"10px 8px"`. Examples dropped their explicit `margin:` lines.
- Rule tokens heavier (swiss hair 2/thick 5/heavy 9; brutalist hair 7/thick
  10/heavy 14); banner padMap 7/11/18 and weight 800.
- Triage-card title clamp 2→3 lines (bigger h1 pushed "Tooling" into the
  ellipsis). Rebuilt SPA, re-rendered all five examples, refreshed
  `docs/screenshots/`, reprinted job slip + triage card (density 38).
- Doc: margin section notes the 10×8 work-theme default.

### Why
- The look belongs in theme data, not in every layout — that is exactly what
  presetOverrides/tokens/padding exist for.

### What worked
- The re-render matches the user's reference mock closely (title fills the
  width in 3 lines, banner nearly edge to edge).

### What didn't work
- First bold pass left the brutalist h1 mixed-case (forceCase had only been
  mapped onto body-role presets) and clamped the title mid-word — both fixed
  in the same commit.

### What warrants a second pair of eyes
- The new prints on paper — banner at 10px page padding sits ~8px from the
  strip edge; check the head doesn't clip the leftmost column of the inverted
  banner.

### What should be done in the future
- If almanac themes also want the tighter look someday, `theme.padding` is
  now the lever (kept almanac defaults untouched).

### Code review instructions
- Diff of `THEMES` (swiss/brutalist/terminal), `DEFAULT_PRESETS`, and the
  `ThermalPaper` padding line in `web/src/almanach-studio.jsx`.
- Validate: `pnpm --dir web build`, render `examples/layouts/10-job-slip.yaml`
  and compare with `docs/screenshots/10-job-slip.png`.

## Step 7: margin: 0 experiment — found and fixed a silent margin bug

Rendering the job slip with `margin: 0` produced a page identical to the
default — which exposed a real bug rather than a styling question: the
`stateRef.current` reassignment in the studio omitted `setMargin`, so
`window.almanachLoadLayout` threw a TypeError mid-load in headless renders
and silently dropped every assignment after it (margin, selection). All
headless renders had been falling back to theme padding regardless of the
layout's `margin:`. Restored the setter, rebuilt, and the margin-0 slip now
runs truly edge to edge (banner and rules full-bleed across all 384 dots);
printed one for paper inspection.

### Prompt Context

**User prompt (verbatim):** "try one with margins 0px on each side"

**Assistant interpretation:** Render/print the job slip with zero page margin.

**Inferred user intent:** See how far the edge-to-edge look can be pushed on
the physical strip.

**Commit (code):** 39447f6 — "fix(studio): setMargin dropped from stateRef — headless renders ignored layout margin"

### What worked / didn't
- The experiment itself was trivial; the identical-output render was the
  tell. The `useRef` initializer had `setMargin` but the per-render
  reassignment (the one that actually matters) didn't — a keep-in-sync
  hazard now called out in a comment.

### What warrants a second pair of eyes
- Whether any earlier "margin verified" renders were actually exercising this
  path (browser imports were unaffected; only headless `almanachLoadLayout`
  was broken).
- On paper: whether the printer head reaches the outermost dots or clips the
  full-bleed banner edge.

## Step 8: Edge-to-edge defaults, one step bigger, the brutalist decision sheet

Two more feedback rounds folded into the defaults: work themes now print
truly edge to edge (`padding: "0px"` — the paper strip has physical margin
already) with the type scale up one more step, and the decision-sheet example
was converted to its brutalist slip-studio form to match the user's reference
mock. Matching the mock's tight vertical rhythm surfaced that the paper's
fixed 14px flex gap between blocks was stacking on top of explicit `space`
blocks, so themes gained a `blockGap` token (work themes: 2px) and `space`
blocks now own the rhythm.

### Prompt Context

**User prompt (verbatim):** "alright, 0px margin by default, the paper has enough margin already anyway. make font size one step bolder still." — followed mid-turn by the brutalist decision-sheet mock image with its original slip-studio JSON: "maybe even something like that (which was the original json ...)"

**Assistant interpretation:** Bake 0 margin + a bigger scale into the theme
defaults; make example 11 reproduce the brutalist mock.

**Inferred user intent:** The slips should look like the slip-studio mocks —
dense, loud, edge to edge.

**Commit (code):** a4f3c6d — "feat(slip): edge-to-edge defaults, one step bigger, brutalist decision sheet"

### What I did
- Work themes: `padding: "0px"`, `blockGap: 2` (new `theme.blockGap`,
  consumed in `ThermalPaper`'s block column gap with the old 12/14 fallback).
- Scale: h1 33 (brutalist 35), h2 24/800, display 50 (brutalist 54), micro 12;
  swiss/brutalist body 16, caption 14; terminal display 40/h1 25/body 15.
- `11-decision-sheet.yaml` → brutalist, exact mock block sequence, `#3790`
  (correct last4 of the job id — the old placeholder said `#5353`).
- Refreshed all five screenshots; printed job slip, triage card, and the
  decision sheet.

### What worked
- The decision-sheet render now matches the mock closely: same rhythm, same
  hierarchy, uppercase 800 body, kv slab separators.

### What didn't work
- N/A this step (the earlier margin-0 render in Step 7 already flushed out
  the setMargin bug).

### What I learned
- Vertical whitespace on the paper is the sum of three sources: the paper's
  flex gap between `.block-wrap`s, the capture CSS's `.block-wrap { padding:
  4px 0 }`, and explicit `space` blocks. Only the first is now theme-tunable;
  the capture padding still adds 8px per block everywhere.

### What warrants a second pair of eyes
- Almanac themes are untouched (`blockGap` falls back to 12/14), but any slip
  layout relying on the old inter-block air will tighten up — that is the
  point, but re-render anything saved earlier.
- The digest title now clamps harder ("…FOR BLE…") at h2 24; producers should
  budget ~2 words per h2 line at 384 dots.

### Code review instructions
- Diff `ThermalPaper` (blockGap), the three THEMES entries, DEFAULT_PRESETS,
  and `11-decision-sheet.yaml`; compare `docs/screenshots/11-decision-sheet.png`
  against the user's mock.
