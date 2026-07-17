---
Title: Diary
Ticket: ALMANACH-DSL-V2
Status: active
Topics:
    - almanach
    - dsl
    - layout
    - frontend
    - go
    - protobuf
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Diary for ALMANACH-DSL-V2: the layout DSL v2 handoff (protobuf block IR, renderer registry, typography presets)."
WhatFor: "Record how the handoff was produced and decisions taken."
WhenToUse: "Read before resuming DSL v2 work."
---

# Diary

## Goal

Produce a self-contained handoff document for rebuilding the Almanach layout DSL
around a protobuf block IR, a React renderer registry, typography presets, and
per-block render options — for a contributor with no prior project knowledge.

## Step 1: Analyze the DSL/renderer, capture product decisions, write the handoff

The user asked which parts of the YAML DSL and renderer are fixed and how to
improve them; I analyzed the code and reported. They then made concrete product
decisions and asked for a new ticket + a handoff document for a colleague who
starts tomorrow with zero context, to be uploaded to reMarkable.

### Prompt Context

**User prompt (verbatim):** "1. can we do stsyle presets for preset font sizes / styles and only override for custom? and give a set of defaults so users don't need to provide their own?\n2. what does 2. mean concretely?\n3. let's do the registry. Let's also use protobuf (see how other software in goo-go-golems use that to share types between go and typescript). Potentially even look at ~/code/wesen/go-go-golems/rag-evaluation-system for our the widget IR renderer pattern works in the react application.\n4. (see IR renderer above)\n5., 6. ok\n\nCreate new docmgr ticket, and give instructions for your colleague who is going to actually do this stuff tomorrow. But they don't know anything about the project or all the stuff you already figured out, nor the instruction I just gave you. So make a nice handoff document and upload it to the remarkable tablet"

**Assistant interpretation:** Turn my improvement analysis + the user's decisions
(typography presets with defaults; data-driven themes explained; registry;
protobuf Go/TS; imitate the rag-evaluation-system IR renderer) into a
self-contained handoff, in a new ticket, and upload to reMarkable.

**Inferred user intent:** Hand the DSL v2 build to someone else cleanly, without
losing any of the accumulated context or the product decisions.

**Commit (code):** docs-only.

### What I did
- Analyzed `web/src/almanach-studio.jsx` (BLOCK_TYPES, themes, fs()/bodyScale) and
  the Go render path to document what is fixed.
- Ran two Explore agents: (a) the widget IR renderer in
  `rag-evaluation-system` (kind-discriminated node union, adapter+Map registry,
  RenderContext, graceful unknown handling; hand-written TS + duplicated Go type
  list — recommend codegen instead); (b) the org protobuf Go/TS pattern (Buf v2 +
  protoc-gen-go/protoc-gen-es, protojson camelCase <-> fromJson/toJson,
  int64->bigint, Struct handling; copy from `scraper`).
- Loaded the `protobuf-go-ts-schema-exchange` skill for the wire-format rules.
- Wrote the handoff `design-doc/01-...` and 6 phase tasks.

### Decisions captured
- Typography = named presets with built-in defaults + per-block override (shallow
  merge default <- layout preset override <- block style); bake the
  ALMANACH-PIXELFONT recipe into defaults.
- "Data-driven themes" = a theme is data in the proto (colors, font palette,
  preset overrides) selectable or overridable inline by a layout; embed hinted
  fonts (DejaVu Serif/Sans).
- Registry: imitate rag-evaluation-system's IR renderer, but generate the shared
  types from protobuf instead of hand-syncing Go/TS.

### What warrants a second pair of eyes
- Whether to model block content as `google.protobuf.Struct` first (recommended)
  or per-type messages in a `oneof`.

### What should be done next
- The colleague executes Phases 1-6 in the handoff, shipping after each of 2-5.

### Technical details
- Handoff: `design-doc/01-handoff-...md`. Reference repos: rag-evaluation-system
  (IR renderer), scraper (protobuf Go/TS), go-go-goja (publishable TS package).

## Step 2: Phase 1 — protobuf layout schema + codegen + round-trip tests

Executed Phase 1 of the handoff: the shared block-IR schema and its Go/TS
codegen, no behavior change (nothing wired into the render path yet). This
establishes the single source of truth (`layout.proto`) both the Go service and
the TS studio import, replacing the implicit React-defined contract.

It unlocks everything downstream: the registry (Phase 2) dispatches on the
generated `Block.type`, and the typography/theme/render work (Phases 3-5) fills
in messages this schema already declares.

### Prompt Context

**User prompt (verbatim):** "Work on ttmp/2026/07/16/ALMANACH-DSL-V2--layout-dsl-v2-protobuf-block-ir-renderer-registry-and-typography-presets/design-doc/01-handoff-layout-dsl-v2-protobuf-block-ir-renderer-registry-typography-presets.md"

**Assistant interpretation:** Start executing the handoff's phased plan; Phase 0
is orientation, so begin Phase 1 (proto schema + Buf codegen + round-trip test).

**Inferred user intent:** Actually build the DSL v2 described in the handoff,
in order, verifying each phase.

**Commit (code):** d777d80 — "feat(dsl-v2): Phase 1 — protobuf layout block IR + Go/TS codegen"

### What I did
- **Buf setup.** Root `buf.yaml` (v2) scoped to `proto/` via `modules: [{path:
  proto, name: buf.build/local/almanach}]` so it does **not** pick up the
  pre-existing `internal/provisioning/native/proto` module. `buf.gen.yaml` uses
  **local** plugins (`protoc-gen-go` on PATH, `web/node_modules/.bin/protoc-gen-es`)
  rather than scraper's remote plugins — the machine's Buf token is invalid
  (`buf registry login` fails), so remote codegen is unavailable here.
- **Schema.** `proto/almanach/layout/v1/layout.proto`: `Layout` (schema_version,
  paper_width, feed_lines, theme ref + inline `Theme`, `Typography`, repeated
  `Block`, `data` map, `RenderOptions`), `Block` (id, type, `TextStyle` style,
  `google.protobuf.Struct` content, per-block `RenderOptions`), `TextStyle`
  (optional scalars + `min_size` legibility floor), `Typography`
  (map<string,TextStyle>), `Theme`/`ThemeColors`, `RenderOptions`, and
  `TextCase`/`RasterMode` enums. Block content is `Struct` for now (open Q in §9).
- **Codegen.** `buf generate` -> Go in `gen/almanach/layout/v1/` (note: no
  `proto/` path prefix because the module root is `proto/`, so `go_package` is
  `.../gen/almanach/layout/v1;layoutv1`), TS in `web/src/pb/almanach/layout/v1/`.
- **Codec + tests.** `internal/layoutpb/codec.go` mirrors scraper's
  `runtimeevents` (protojson `UseProtoNames:false` camelCase, schema_version
  normalize, JSON + binary). A shared golden `proto/.../testdata/layout_golden.json`
  is read by **both** the Go test (`internal/layoutpb/codec_test.go`, field
  assertions + proto.Equal round-trip) and a runner-free TS test
  (`web/test/layout.roundtrip.test.mjs`, `fromJson`/`toJson`, run with `node`).
- **Wiring.** `make proto`, `make test-proto`, `pnpm --dir web test:proto`.

### Why
- The DSL contract was implicit and lived inside one React file; unknown fields
  vanished silently. A protobuf schema makes it explicit, validated, versioned,
  and shared by both languages — the prerequisite the handoff calls out.

### What worked
- Local-plugin codegen produced clean Go + TS; the shared golden fixture decodes
  identically on both sides; round-trip is stable (`proto.Equal`, `toJson`).

### What didn't work (verbatim)
- Remote Buf plugins: `buf generate` -> `Failure: your Buf API token for
  buf.build is invalid. Run "buf registry login"...`. Switched to local plugins.
- pnpm adds failed: `ERR_PNPM_EROFS: read-only file system, open
  '/home/manuel/workspaces/2026-05-03/.../.pnpm-store/...'`; then
  `ERR_PNPM_UNEXPECTED_STORE` after pointing at a new store; resolved by a full
  `pnpm install --store-dir ../.tmp-pnpm-store --config.confirmModulesPurge=false`.

### What I learned
- With a v2 `modules: [{path: proto}]` workspace, the module root is `proto/`, so
  `paths=source_relative` output has no `proto/` prefix — `go_package` must be
  `.../gen/almanach/layout/v1`, not `.../gen/proto/...`.
- protobuf-es `tsEnum` strips the enum-name prefix at runtime.

### What was tricky to build
- **pnpm store is read-only in this sandbox.** Cause: the global
  `~/.config/pnpm/rc` sets `store-dir` to an archived, read-only workspace path,
  and `$HOME` is read-only. Symptom: `ERR_PNPM_EROFS` on any new tarball.
  Solution: install into a git-ignored repo-local store
  (`--store-dir ../.tmp-pnpm-store`); `node_modules` is now bound to it, so a
  plain `pnpm install` errors `UNEXPECTED_STORE` until the same `--store-dir` is
  passed. Documented in the `almanach-dsl-v2-build` memory.
- **`tsEnum` prefix stripping**: `RasterMode.THRESHOLD`, not
  `RASTER_MODE_THRESHOLD` (JSON wire form keeps the full name; `.d.ts` shows full
  names but runtime keys are short) — the TS test failed until I used short keys.

### What warrants a second pair of eyes
- Block content is `google.protobuf.Struct` (free-form) for now; the open
  question (Struct vs per-type `oneof`) is deferred. No `int64` in the schema, so
  no `bigint` hazard yet — keep it that way or `toJson`-round-trip before any
  `JSON.stringify`.

### Verified
- `make test-proto` green (Go + TS). `buf generate` is idempotent (re-running
  produces no diff). `GOWORK=off go build ./...` and `go vet ./internal/layoutpb/...`
  clean.

### Code review instructions
- Start at `proto/almanach/layout/v1/layout.proto` (the contract), then
  `internal/layoutpb/codec.go` (protojson options must stay `UseProtoNames:false`
  to match the TS side). Validate: `make test-proto`; regenerate with `make proto`
  and confirm no diff.

### What should be done next
- Phase 2: the React renderer registry. The generated TS types
  (`web/src/pb/.../layout_pb`) are now importable in the studio; build the
  `type -> adapter` registry against `Block`/`Layout` and a graceful
  unknown-type placeholder.

## Step 3: Phase 2 — React renderer registry

Refactored the studio's block dispatch from a bare `RENDERERS` object map into a
proper adapter registry with a graceful unknown-type fallback. Behavior for
known types is unchanged (verified on paper); unknown types now render a visible
placeholder instead of being dropped at parse or crashing.

This de-risks the phases that add block types and styling: adding a block is now
"write a component + register an adapter", and the registry threads a `ctx`
(theme, block) that Phase 3 uses to pass per-block style.

### Prompt Context

**User prompt (verbatim):** "commit all stage worthy (even if not yours). then go into phase 2."

**Assistant interpretation:** Commit everything currently uncommitted (the
pre-existing PIXELFONT ttmp artifacts and my Phase 1 work), then implement
Phase 2, the renderer registry.

**Inferred user intent:** Get a clean tree with Phase 1 recorded in history, then
keep executing the plan.

**Commit (code):** 27f663b — "feat(dsl-v2): Phase 2 — React renderer registry + graceful unknown blocks" (plus 22f1eeb, d777d80 committed first per the instruction).

### What I did
- **`web/src/blocks/registry.js`** — a React-free, side-effect-free registry:
  `defineBlock` (shape validation), `createBlockRegistry` (array -> Map, throws
  on duplicate type), `mergeBlockRegistries`, `resolveBlockAdapter`. Mirrors the
  rag-evaluation-system widget-IR pattern (adapter objects + Map + merge +
  graceful fallback) minus the interactivity plumbing a print DSL doesn't need.
  Kept pure JS (not `.jsx`) so it unit-tests in plain Node.
- **`web/src/blocks/registry.test.mjs`** — runner-free node test: defineBlock
  validation, dup guard, lookup hit/miss, merge + merge-dup guard, render
  passthrough of `(data, ctx)`.
- **Studio wiring (`almanach-studio.jsx`)** — replaced the `RENDERERS` object
  with `BLOCK_ADAPTERS` (one `defineBlock` per existing component, each wrapping
  the unchanged `*Block` component) -> `BLOCK_REGISTRY`. Added an `UnknownBlock`
  placeholder component and a `renderBlock(block, ctx)` helper (ctx =
  `{ theme, registry, block }`). The paper render loop now calls
  `renderBlock(b, { theme, registry: BLOCK_REGISTRY })`.
- **`parseLayoutJson`** no longer drops unknown-type blocks; it keeps any block
  with a string `type`, guarding the `DEFAULTS[type]` lookup (unknown -> `{}`).
  Both feed paths (`window.almanachLoadLayout` headless + file import) go through
  this function, so the placeholder shows in the print pipeline too.

### Why
- 16 bespoke components dispatched by a bare object map; unknown types crashed
  the render or were dropped at parse. The registry pattern (from
  rag-evaluation-system) gives a dup-guarded, composable, gracefully-degrading
  dispatch — the handoff's Phase 2.

### What worked
- Keeping the registry pure JS (no JSX) let it unit-test in plain Node; wrapping
  the *unchanged* `*Block` components in adapters meant zero visual change for
  known types, confirmed on paper.

### What I learned
- Both feed paths (`window.almanachLoadLayout` and file import) funnel through
  `parseLayoutJson`, so relaxing its type filter is the single place that makes
  unknown blocks reach the placeholder in the print pipeline too.

### What was tricky to build
- **`buf.gen.yaml` has `clean: true`**, so `buf generate` wipes the entire
  `web/src/pb` output dir. Cause/symptom: the Phase 1 proto round-trip test was
  written *inside* `web/src/pb/`, so `make proto` silently deleted it — and it
  had never made it into the Phase 1 commit (only the generated files did).
  Solution: moved it to **`web/test/layout.roundtrip.test.mjs`** and re-ran
  `make proto` to confirm it survives regen. Rule: never put hand-written files
  under a buf `clean:true` out dir.

### What warrants a second pair of eyes
- `parseLayoutJson` now keeps any block with a string `type`; unknown types get
  `data: {}`. Confirm no downstream code assumes `DEFAULTS[type]` exists.

### Verified
- `make test-web` green (proto round-trip 13 assertions + registry unit test).
  `make proto` re-run confirms the relocated test survives regen.
- End-to-end render through the real pipeline (`google-chrome-stable`, 384px):
  - `examples/layouts/03-knowledge-strip.yaml` renders identically to before
    (384x1000) — no behavior change for known blocks.
  - A layout with an unknown `sparkline` block renders the title and quote
    normally and shows a dashed "Unknown block type 'sparkline' — not registered"
    placeholder between them (previously the block vanished at parse).
- `GOWORK=off go test ./...` and `pnpm run build` (studio bundle) both clean.

### Code review instructions
- Start at `web/src/blocks/registry.js` (pure logic) + `registry.test.mjs`, then
  the `BLOCK_ADAPTERS`/`BLOCK_REGISTRY`/`renderBlock`/`UnknownBlock` region and the
  relaxed `parseLayoutJson` in `web/src/almanach-studio.jsx`. Validate:
  `make test-web`; render the known + an unknown-block layout and diff visually.

### What should be done next
- Phase 3: typography presets. With the registry in place and `Typography`/
  `TextStyle` already in the proto, introduce the preset model + paper-verified
  defaults and migrate components off inline `theme.fs(n)` sizes. Ship + print.

## Step 4: Phase 3 — typography presets (the ship-and-print quality win)

Introduced named typography presets with built-in defaults + a three-layer
override model, baked the ALMANACH-PIXELFONT recipe into the defaults, and
migrated every block component off inline `theme.fs(n)`/font literals. This is
the phase where the print-legibility win lands.

The recipe (bigger sizes, minSize floors, heavier body/small, bold-italic
quotes) is now a data change, and a layout can override any preset globally or a
block inline — turning what previously required editing React into a few lines
of layout JSON.

### Prompt Context

**User prompt (verbatim):** "Do it."

**Assistant interpretation:** Proceed into Phase 3 (typography presets), which I
had just proposed as the next step.

**Inferred user intent:** Land the print-quality win the handoff flagged as the
phase to "ship and print".

**Commit (code):** d25e17b — "feat(dsl-v2): Phase 3 — typography presets with paper-verified defaults"

### What I did
- **`web/src/typography/presets.js`** — `DEFAULT_PRESETS` (sectionLabel, overline,
  word, metric, body, bodyStrong, emphasis, caption, small, meta) using the same
  field names as the proto `TextStyle`; `resolveStyle(name, {presets, theme,
  bodyScale, overrides})` merges `default <- layout preset <- block/component
  overrides`, scales `size` by bodyScale, floors at `minSize`, and maps
  `textCase`/`italic`/`letterSpacing` to CSS; `makePresetResolver` binds it.
  Recipe baked in: bigger sizes, absolute minSize floors, weight 500-700 on
  body/small, and **bold italic** for `emphasis` (quotes/notes) — the
  paper-verified legible italic. Unit test in `presets.test.mjs`.
- **Studio wiring** — `theme.preset(name, ...overrides)` bound to the layout's
  `typography.presets` + bodyScale; `theme.fontMono` added. `parseLayoutJson`
  now reads `typography.presets` and per-block `style`; `typography` is app state
  set on headless load + file import; `buildLayoutJson` round-trips both. Block
  adapters pass `blockStyle={ctx.block?.style}` to components.
- **Component migration** — all 16 block components resolve text styles from
  presets instead of `fs(n)` literals; the block's primary text takes the
  per-block `style` (e.g. a quote's text = `emphasis`, its attribution =
  `caption`). Only decorative glyphs/icons still use `fs()`.

### Why
- Typography was baked into components as literals (`fs(11)`) with family/weight
  from the theme; a layout couldn't set font/size/weight/spacing/min-size. Presets
  make the paper-verified recipe the default and let layouts/blocks override.

### What worked
- Modeling preset style objects with the **proto `TextStyle` field names** means
  a decoded layout's `typography.presets`/`Block.style` plug straight into the
  resolver with no adapter. On-paper diff shows the intended legibility win.

### What I learned
- `size` must be a *base* px value scaled by `bodyScale`, with `minSize` an
  *absolute* floor applied after scaling — otherwise the floor scales too and
  stops being a legibility guarantee.

### What was tricky to build
- **Font clobbering.** Cause: `resolveStyle` always derived a font from `role`,
  so spreading an ad-hoc override with neither `role` nor `font` (the `<h1>`
  title-override case) forced `fontBody` and wiped the theme's display font.
  Symptom: the title flipped to the body font. Solution: only set `fontFamily`
  when the merged style actually names a `font` or a `role`; added a test
  asserting no font is injected otherwise.
- **Which element gets `Block.style`.** Chose the block's *primary* text element
  per type (quote text = `emphasis`, plan container = `body`, etc.), passed last
  in the override array so it wins over layout + component tweaks.

### What warrants a second pair of eyes
- The preset-per-element mapping across 16 components is judgment-heavy; sizes
  were tuned once on paper and may need per-theme adjustment. Decorative glyphs
  intentionally still use `fs()`.

### Verified (on paper, google-chrome-stable, 384px)
- `examples/layouts/03-knowledge-strip.yaml`: markedly more legible than
  pre-Phase-3 — heavier body text, bold-italic closing quote and subtitle,
  bold display years. Page grew 1000 -> 1071px (intentionally bigger text).
- An override layout (`typography.presets.body = {weight:700,size:15}` + a block
  with `style:{italic:true,weight:500}`) confirms all three layers: the first
  `did` block is bold/enlarged from the layout override, the second flips to
  italic/lighter from its per-block style.
- `make test-web` (13 + registry + presets assertions), `go test ./...`, and the
  studio bundle build all green.

### Code review instructions
- Start at `web/src/typography/presets.js` (`DEFAULT_PRESETS`, `resolveStyle`
  merge order + minSize/font rules) + `presets.test.mjs`. Then the `theme.preset`
  binding and per-component preset calls in `web/src/almanach-studio.jsx`, and the
  `typography`/`style` plumbing in `parseLayoutJson`/`buildLayoutJson`. Validate:
  `make test-web`; render the knowledge-strip + an override layout and eyeball.

### What should be done next
- Phase 4: data-driven themes + embed the hinted DejaVu Serif/Sans fonts in the
  palette, then let presets/themes select them (the presets already accept an
  explicit `font`; only the embedded families + theme-as-data are missing).

## Step 5: Phase 4 — data-driven themes + hinted DejaVu fonts

Made "theme" a piece of data a layout can carry, not just a built-in name, and
embedded the hinted DejaVu families so the *font* half of the ALMANACH-PIXELFONT
recipe (crisp small text) is finally expressible. A layout's `theme` can now be
an object that patches a base built-in with colors, a font palette, and preset
overrides — no code change, no redeploy.

This closes the "what does data-driven mean concretely" question from the
handoff: built-in themes are seed data, and a user theme is the same shape
supplied inline.

### Prompt Context

**User prompt (verbatim):** "phase 4 5 6, continue. Keep a detailed diary as usual (see skill), and commit at appropriate intervals"

**Assistant interpretation:** Implement Phases 4, 5, and 6 in order, maintaining
the strict-format diary and committing at sensible boundaries. (Also, mid-turn:
"retroactively update the diary to get the proper formatting" — reformatted
Steps 2-4 first, commit d96edb6.)

**Inferred user intent:** Finish the DSL v2 build to completion with reviewable
history and a faithful diary.

**Commit (code):** <this step's commit>

### What I did
- **Embedded DejaVu.** Subset DejaVu Serif (regular/bold/italic/bold-italic) +
  DejaVu Sans (regular/bold) to Latin + punctuation/symbol/dingbat ranges via
  `fonttools subset --flavor=woff` and base64-appended them to
  `web/src/fonts-embedded.css` (~422KB woff -> ~564KB base64, 6 faces).
- **Built-in DejaVu themes.** Added `crisp` (DejaVu Serif) and `crispsans`
  (DejaVu Sans) to `THEMES`.
- **Data-driven `theme`.** `resolveThemeSpec(spec)` accepts a string (built-in
  name, unchanged) or an object `{ base, colors, fontPalette|fontDisplay/
  fontBody/fontMono, titleSize/…, presetOverrides }`, returning
  `{ themeKey, patch, presetOverrides }`. The patch merges over `THEMES[base]`;
  `presetOverrides` becomes a preset layer.
- **Wiring.** `parseLayoutJson` returns `themePatch`/`themePresets`; new
  `inlineTheme`/`themePresets` state set on headless load + file import; theme is
  built as `{...THEMES[themeKey], ...inlineTheme}`; the resolver's preset map is
  `mergePresetMaps(themePresets, typography)` (default <- theme <- layout <-
  block). Selecting a built-in theme in the UI clears the inline patch.
- **`mergePresetMaps`** added to `presets.js` (+ test): per-preset, per-field
  deep merge so theme `body.size` and layout `body.weight` both survive.

### Why
- Previously a theme was JavaScript compiled into the SPA; adding/tweaking one
  meant editing and redeploying React. Making it data lets a layout define or
  patch a theme inline, and embedding hinted fonts makes the crisp-small recipe
  achievable without code.

### What worked
- On paper (384px): built-in `crisp` renders DejaVu Serif with visibly heavier,
  crisper strokes than EB Garamond and legible bold-italic quotes; an inline
  theme patching `minimal` to DejaVu Sans with `body.weight:700` prints a fully
  sans page with bold body — all from layout JSON.
- Subsetting kept the decorative glyphs (✦ ❦ ❀ ☾ ●) so DejaVu body text still
  shows them.

### What didn't work (verbatim)
- woff2 failed: `ImportError: No module named brotli` (fonttools woff2 writer
  needs brotli; not installable to the read-only store). Fell back to
  `--flavor=woff` (zlib), still ~4x smaller than raw TTF after subsetting.
- First subset attempt: `ERROR: Unknown option 'no-hinting=false'` — hinting is
  preserved by default; dropped the flag.

### What was tricky to build
- **Preset layering across theme + layout.** A single replace-merge of preset
  maps would drop fields (theme sets `body.size`, layout sets `body.weight` —
  both needed). Solved with `mergePresetMaps` doing a per-preset shallow merge,
  applied theme-first so layout wins per field, then the block style wins last in
  `resolveStyle`'s `overrides`.
- **Stale inline theme.** Loading an inline-theme layout then clicking a built-in
  theme card would keep the old patch; fixed by clearing `inlineTheme`/
  `themePresets` in the theme-card onClick.

### What warrants a second pair of eyes
- Embedded font weight: +564KB base64 in `fonts-embedded.css` (copied to
  `dist/fonts.css`, not the JS bundle). Fine for the render service; confirm it's
  acceptable for any ESP32-served path. Could drop DejaVu Sans (~230KB) if size
  matters, since the serif is the primary crisp-small win.
- `buildLayoutJson` still exports `theme` as the built-in `themeKey` string; an
  inline theme loaded from JSON is applied but not re-serialized on export
  (authoring feature, not a studio-edited one). Round-trip of inline themes is a
  follow-up if needed.

### Verified
- `make test-web` (round-trip + registry + presets incl. `mergePresetMaps`),
  `go test ./...`, and the studio build all green. Two on-paper renders above.

### Code review instructions
- Start at `resolveThemeSpec` + the `crisp`/`crispsans` entries and the
  `theme`/`theme.preset` construction in `web/src/almanach-studio.jsx`, and
  `mergePresetMaps` in `web/src/typography/presets.js`. Font faces are the tail of
  `web/src/fonts-embedded.css`. Validate: `make test-web`; render a layout with
  `"theme": {"base":"minimal","fontPalette":["'DejaVu Sans',sans-serif"],
  "presetOverrides":{"body":{"weight":700}}}` and eyeball the sans + bold.

### What should be done next
- Phase 5: replace the five hand-parsed Go `render:` keys with the typed
  `RenderOptions` proto message (validated once) and add per-block render
  overrides.

## Step 6: Phase 5 — typed RenderOptions (+ printed examples, + margin control)

Replaced the Go side's hand-parsed `render:` keys with the typed, validated
`layoutv1.RenderOptions` message, added per-block render-option parsing (for
Phase 6), and made printer density a real render option applied at print time.
This closes the "thin, hand-parsed, global-only render options" gap from the
handoff's problem list. Two user interjections landed here too: I printed a
couple of examples on the thermal printer, and added a layout-level page margin.

### Prompt Context

**User prompt (verbatim):** (see Step 5 — "phase 4 5 6, continue...")

**Mid-turn user prompt (verbatim):** "print out a few examples btw, or tell me if i can edit them in my browser for testing (update the presets of the webapp I guess?)"

**Mid-turn user prompt (verbatim):** "can you control how much margin there is?"

**Assistant interpretation:** Continue Phase 5; also (1) print sample pages and
explain the browser-editing story, and (2) make the page margin controllable.

**Inferred user intent:** See real output on paper, understand how to iterate in
the browser, and gain a margin knob for tuning.

**Commit (code):** margin = c46d921; Phase 5 typed render options = <this step's commit>.

### What I did
- **Typed render options** (`internal/app/renderopts.go`): `parseRenderOptions`
  decodes a `render:` map into `layoutv1.RenderOptions` via protojson
  (DiscardUnknown) + range validation; `applyRenderOptions` overlays only set
  (non-nil / non-UNSPECIFIED) fields onto the internal `RenderOptions` struct;
  `perBlockRenderOptions` extracts each block's `render:` keyed by id;
  `rasterModeString` maps the enum. Removed `intFromRenderOptions` /
  `stringFromRenderOptions`.
- **Struct + wiring**: added `RasterMode/Gamma/PrinterDensity/PrinterSpeed` to the
  internal `RenderOptions`; refactored `renderOptionsFromSettings` (now returns an
  error), `cmd_render`, `cmd_inspect`, and `cmd_print` to the typed path.
- **Printer density** applied at print time via new `setPrinterDensity`
  (POST `/api/printer/density`), non-fatal on failure.
- **Flat `render:` fix**: `layoutJSONFromObjectOrDefault` now extracts a
  top-level `render:` (flat layouts), not only the wrapped `{layout, render}`
  form.
- **Margin** (studio, commit c46d921): `marginToPadding` (number | {x,y} |
  {t,r,b,l}) threaded through parse/state/ThermalPaper/export; added
  `Layout.margin` (EdgeInsets) + `Layout.body_scale` to the proto with golden +
  round-trip test coverage.
- **Prints**: set density 38 and printed the `crisp` knowledge strip and the
  inline DejaVu-Sans theme (both `printer_ok:true`).
- **Tests**: `internal/app/renderopts_test.go` (parse/validate/overlay/per-block).

### Why
- The old render options were an unschema'd `map[string]interface{}` read via
  ad-hoc helpers — no validation, no per-block support, only the wrapped form.
  The typed message makes them validated, per-block-capable, and shared with the
  studio's contract.

### What worked
- `render.threshold: 90` now flows to the bitmap threshold (reported 90);
  `threshold: 300` is rejected with a clear error. Per-block parsing returns the
  right map. Printer density endpoint returns `{"ok":true,"density":38}`.

### What didn't work (then fixed)
- First end-to-end test showed `render.threshold: 90` ignored (reported 128) and
  `300` not rejected. Cause: `layoutJSONFromObjectOrDefault` only pulled `render`
  from the wrapped form, so a flat layout's top-level `render:` never reached the
  parser. Fixed with the flat-form `else if` + `delete(obj, "render")`.

### What was tricky to build
- **Signature change ripple.** Making `renderOptionsFromSettings` return an error
  (so validation can surface) broke two call sites (`cmd_render`, `cmd_inspect`);
  updated both, plus refactored `cmd_print`'s inline construction to the shared
  `applyRenderOptions` overlay.
- **proto `optional` presence.** Overlaying "only set fields" relies on the
  generated pointer fields (`*uint32` etc.) for scalars and `!= UNSPECIFIED` for
  the `RasterMode` enum — mixing the two presence models in one overlay function.

### What warrants a second pair of eyes
- `perBlockRenderOptions` is parsed and tested but not yet consumed — Phase 6
  wires it into block-aware rasterization. Confirm the keying by block `id`
  matches what the metrics/bounding-box path will use.
- `setPrinterDensity` derives the base URL by trimming `/api/print/bitmap`;
  verify against any non-standard `--printer-url`.

### Verified
- `go test ./...` (incl. new renderopts tests), `make test-web` (16 + registry +
  presets), Go build with `-tags embed`. On-paper: two prints; `threshold:90`
  applies, `:300` errors.

### Code review instructions
- Start at `internal/app/renderopts.go` (+ `renderopts_test.go`), then the
  refactors in `cmd_render.go`/`cmd_print.go`/`cmd_inspect.go`, the flat-`render`
  fix in `render_oneshot.go`, and `setPrinterDensity` in `printer.go`. Validate:
  `go test ./internal/app/...`; `render --layout <flat yaml with render.threshold>`.

### What should be done next
- Phase 6: block-aware rasterization + per-segment heat. Combine
  `perBlockRenderOptions` with the per-element bounding boxes from
  `collectMetricsJS` to threshold text vs dither images and set per-segment
  printer density in one page.
