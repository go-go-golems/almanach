---
Title: 'Handoff: Layout DSL v2 (protobuf block IR, renderer registry, typography presets)'
Ticket: ALMANACH-DSL-V2
Status: active
Topics:
    - almanach
    - dsl
    - layout
    - frontend
    - go
    - protobuf
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/app/bitmap.go
    - Path: internal/app/render_oneshot.go
      Note: Hand-parsed render options to replace with the proto RenderOptions
    - Path: internal/app/renderer.go
    - Path: web/src/almanach-studio.jsx
      Note: The studio+renderer; BLOCK_TYPES, themes, fs()/bodyScale to be replaced by the registry+presets
ExternalSources: []
Summary: 'Self-contained handoff for rebuilding the Almanach layout DSL: a protobuf-defined block IR shared between Go and TypeScript, a React renderer registry (block type -> component), typography presets with built-in defaults and per-block overrides, data-driven themes with a wider (hinted) font palette, and per-block render/raster options. Orients a contributor with no prior project knowledge, summarizes the rendering findings that motivate the work, gives the two reference patterns to copy (widget IR renderer, protobuf Go/TS exchange), and a phased plan.'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Hand the DSL v2 work to a contributor who has never seen the project.
WhenToUse: Read start-to-finish before touching the layout DSL, the studio, or the renderer.
---


# Handoff: Almanach Layout DSL v2

## 0. How to read this, and what you're being asked to do

You are picking up a feature in a project you have not seen. This document is
written so you do not need any prior context: it explains what the project is
(§1), how its "layout language" and renderer work today (§2), what is wrong with
them (§3), what we learned about print quality that makes this urgent (§4), what
to build (§5), the two code patterns to copy (§6), and an ordered plan (§7).
Read it top to bottom once before writing code.

**The one-sentence version:** the layout language can currently express *which*
blocks and *which* theme, but almost nothing about typography (font, size,
weight per element) or rasterization — and those are exactly the knobs that
control print legibility — so we are rebuilding the DSL around a **protobuf-
defined block IR** shared by Go and TypeScript, a **React renderer registry**,
**typography presets with sensible defaults**, and **per-block render options**.

## 1. What Almanach is

Almanach prints small "almanach" pages (a daily briefing: title, date, word of
the day, today-in-history, a quote, an image, etc.) on a **58 mm thermal
printer** — a monochrome device that can only burn a dot or not, 384 dots wide.

Three parts:

- **A React studio** (`web/src/almanach-studio.jsx`): a browser app where a user
  arranges "blocks" on a virtual paper strip and edits their content. One large
  JSX file. It also runs *headless* during rendering (below).
- **A Go render/print service** (`internal/app/...`, binary
  `cmd/almanach-render-service`): takes a layout, renders it with headless
  Chrome, screenshots the paper element, converts the screenshot to a 1-bit
  bitmap, and POSTs it to the printer firmware.
- **ESP32 firmware** on the printer (AtomS3R + K118 head): receives the packed
  bitmap over HTTP and emits it to the head via ESC/POS.

A **layout** is a YAML or JSON document. The pipeline:

```
layout.yaml
  -> Go parses it, starts an ephemeral web server serving the SPA
  -> headless Chrome loads /almanach, calls window.almanachLoadLayout(layout)
  -> the React studio renders the blocks onto a .paper-body element
  -> Go screenshots .paper-body (device-pixel-ratio 1, 384px wide)
  -> internal/app/bitmap.go thresholds the screenshot to a 1-bit bitmap
  -> internal/app/printer.go POSTs the packed bytes to the firmware
```

So the "DSL" is not interpreted by Go — Go hands the layout to the React app, and
**the React app is the renderer**. Today the layout's meaning is defined entirely
by what `almanach-studio.jsx` chooses to draw.

## 2. The current layout DSL and renderer (as-is)

### 2.1 The layout document

Top-level fields (validated loosely): `theme` (a fixed enum of hardcoded theme
names), `paperWidth` (dots, 280-600, default 384), `bodyScale` (a single global
font-size multiplier, 1-2, default ~1.6), `feedLines`, `blocks` (array of
`{ id, type, data }`), `render:` (optional render options, §2.4), `data:`
(optional template variables).

Real example (`examples/layouts/03-knowledge-strip.yaml`):

```yaml
theme: minimal
paperWidth: 384
bodyScale: 1.35
blocks:
  - id: title-1
    type: title
    data: { text: KNOWLEDGE STRIP, subtitle: Word, history, facts }
  - id: word-1
    type: word
    data: { label: Word of the Day, word: apricity, definition: The warmth of the sun in winter. }
  - id: quote-1
    type: quote
    data: { label: Closing Thought, text: The cure for boredom is curiosity., author: Dorothy Parker }
```

### 2.2 Blocks

Block types are a **closed, hardcoded set** in `almanach-studio.jsx`
(`BLOCK_TYPES`, ~line 213): `title, date, divider, plan, news, weather, note,
image, habits, mood, reading, reflection, quote, word, history, did`. Each type
is a bespoke React component with its own hardcoded layout. Unknown types are
silently dropped at parse. A block's `data` has no declared schema — each
component reads the fields it wants.

### 2.3 Themes and typography

Themes are hardcoded JS objects (~line 20+): each sets `fontDisplay`, `fontBody`,
`titleSize`, `titleWeight`, `titleSpacing`, `titleCase`. Fonts are limited to ~6
Google fonts embedded as base64 in `web/src/fonts-embedded.css` (EB Garamond,
Cormorant Garamond, DM Sans, Caveat, Kalam, JetBrains Mono). Body/label sizes are
written inline in each component as `theme.fs(<literal>)` (e.g. `theme.fs(11)`);
`fs(n)` multiplies by `bodyScale`. **The only typography lever a layout has is
`bodyScale` (global) and the theme choice** — no per-block or per-layout control
of font, size, weight, letter-spacing, or line-height.

### 2.4 Render options and the renderer

Go reads five keys from `render:`: `selector`, `threshold`, `viewportWidth`,
`viewportHeight`, `supersample`, each via hand-written
`intFromRenderOptions`/`stringFromRenderOptions` calls
(`internal/app/render_oneshot.go`, `cmd_*.go`). No schema, no validation, no
per-block settings.

The renderer (`internal/app/renderer.go`) is welded to the SPA: navigates to
`/almanach`, waits for `window.almanachReady`, calls `window.almanachLoadLayout`,
screenshots `.paper-body`. Viewport, device-scale-factor, anti-aliasing, selector
are Go constants. It produces **one flat screenshot -> one 1-bit bitmap**, with
no per-block awareness at raster time (it collects per-element bounding boxes via
`collectMetricsJS` but does not use them for rasterization).

## 3. What is fixed today (the problem to solve)

1. **Closed block vocabulary.** 16 bespoke components; adding a block = edit JSX +
   rebuild. No generic/rich-text block, no registry.
2. **Typography is baked into components, not the DSL.** Sizes are literals
   (`fs(11)`), family/weight from the theme. A layout cannot set font, size,
   weight, spacing, or a minimum size — globally or per block.
3. **Fixed themes and a tiny font palette.** No hinted fonts; themes hardcoded; a
   layout cannot define or override a theme inline.
4. **Thin, hand-parsed, global-only render options.** No schema, no validation, no
   per-block raster/heat/font settings.
5. **The DSL contract is implicit and lives inside one React file.** No declared
   schema, no shared types between Go and the SPA, unknown fields vanish silently.

## 4. Why this is urgent: what we learned about print quality

Two prior tickets established, by printing and reading real paper, exactly which
knobs control legibility — and almost none are expressible in the current DSL.

- **`ALMANACH-RASTER-LAB`** (images/heat): photographs need Atkinson dithering +
  a gamma ~0.8 tone curve at printer density ~20; text wants a hotter density
  (~30-38); text and photos want *different* printer heat, so heat should be
  settable per print segment. A pluggable rasterizer (threshold/Atkinson/Floyd/
  Bayer + tone curve) and a printer density/speed client already exist on a
  branch.
- **`ALMANACH-PIXELFONT`** (small text): the pipeline emits 1-bit, so
  anti-aliased sub-pixel strokes get thresholded away and small text loses
  strokes. Paper-verified fixes:
  - **Render with anti-aliasing OFF** (fontconfig `antialias=false` on the render
    browser) so FreeType's hinted monochrome rasterizer makes the 1-bit decision.
    Now the default.
  - **Font hinting and size dominate.** Well-hinted fonts (DejaVu Serif/Sans) are
    crisp small; the current EB Garamond/DM Sans are not. EB Garamond has a small
    x-height and must be set ~+3px larger (Garamond 16-17 ~ DejaVu 11-12).
  - **Weight is the biggest lever:** bold/medium survives the threshold, and
    **bold italic is legible where normal italic is not.**
  - **Printer heat:** density ~38 reads best; speed barely matters.

**The punchline:** the recommended production recipe — *bigger text, especially
serif; a hinted font (or a larger delicate one); bold for small and italic text;
density ~38* — **cannot be written in a layout today.** It would require editing
React components and themes and redeploying. DSL v2 turns that recipe into a few
lines of YAML.

## 5. What to build (v2 design, with the product decisions baked in)

### 5.1 A protobuf-defined layout/block IR, shared between Go and TypeScript

Define the layout as a **protobuf schema** (`.proto`) that is the single source
of truth, code-generated into **Go types** (service) and **TypeScript types**
(studio). The layout becomes a typed **block IR**: a `Layout` message with
metadata (paper width, theme ref, typography), a repeated `Block`, and each
`Block` carrying `id`, `type`, typed `style` overrides, and content. For block
content that varies per type, start with `google.protobuf.Struct` (free-form JSON
object); migrate hot block types to per-type messages in a `oneof` later.

This replaces the implicit, React-defined contract with an explicit, validated,
versioned one both sides import. Include a `schema_version` field. JSON stays the
transport (Go `protojson` camelCase; TS `fromJson`). Use the org toolchain (§6.2).

### 5.2 A React renderer registry (block type -> component)

Replace the hardcoded per-type components with an **IR renderer**: a component
that walks `blocks` and, for each, looks its `type` up in a **registry**
(`Map<string, BlockAdapter>`) and renders the matching component with the block's
typed data + resolved style. Unknown types render a visible placeholder instead
of vanishing. Adding a block type = write a component + register it + (optionally)
add its content message to the proto. Copy the pattern in §6.1.

### 5.3 Typography presets, with built-in defaults and per-block overrides (decision 1)

The explicit product decision. Model typography as **named presets**, not raw
per-element sizes:

- Ship a **built-in default set of presets** — e.g. `title`, `subtitle`,
  `heading`, `body`, `caption`, `small`, `mono` — each mapping to a concrete
  `TextStyle { font, size, weight, lineHeight, letterSpacing, case }`. **Users
  get good typography without providing anything.**
- A block references a preset by name (usually implied by its type: a `quote`'s
  body text uses `body`, its attribution uses `caption`).
- A layout may **override** a preset globally (e.g. bump `body` to a hinted font
  at a larger size), and a block may **override** specific properties inline for
  the custom case. Resolution is a shallow merge:
  `built-in default <- layout preset override <- block style`.

In the proto: `Typography` = `map<string, TextStyle>` (preset name -> style),
plus a per-block optional `TextStyle style`. The studio resolves the three layers
into final CSS. **Bake the paper-verified recipe into the defaults**: bigger
sizes, a hinted body font, bold for small/italic. That turns the whole
print-quality recipe into a data change.

### 5.4 Data-driven themes and a wider font palette (decision 2, concretely)

You asked what "data-driven themes" means concretely. Today a theme is JavaScript
compiled into the SPA; to add or tweak one you edit and redeploy the React app.
"Data-driven" means a **theme is just data** — part of the protobuf schema
(colors, the font palette it uses, and its preset overrides) — so:

- A layout can **select** a built-in theme by name (as now), **or** supply/patch
  a theme inline in the layout, with no code change.
- The theme's fonts come from an **expanded, embedded font palette** that includes
  **hinted fonts (DejaVu Serif, DejaVu Sans)** — the ones that print crisp small —
  not just the current display fonts. Adding a font = add it to the embedded font
  CSS + list it in the palette, not edit every component.
- A theme, in the schema, is essentially `Theme { name, colors, font_palette,
  preset_overrides }`. Built-in themes are seed data; user themes are the same
  shape supplied inline. That is the concrete meaning of "data-driven": the
  theme's definition lives in data the layout can carry, not in compiled JSX.

### 5.5 Per-block render options and block-aware rasterization (decisions 4-6)

- Replace the five hand-parsed render keys with a **typed `RenderOptions`
  message** (validated once), and allow **per-block render overrides** (raster
  mode, threshold, printer density/speed, supersample).
- Use the **per-block bounding boxes** the renderer already collects
  (`collectMetricsJS`) for **block-aware rasterization**: threshold text, dither
  images, set printer heat per segment in one page — the "real fix for mixed
  pages" from `ALMANACH-RASTER-LAB`. Later phase; depends on 5.1.
- Treat the protobuf layout schema as **the contract**, validated server-side,
  with the React studio as one renderer among possibly several.

## 6. Reference patterns to copy (with concrete file paths)

### 6.1 The widget IR renderer — `rag-evaluation-system`

The whole reusable pattern is in
`~/code/wesen/go-go-golems/rag-evaluation-system/packages/rag-evaluation-site/src/widgets/`.
Three parts — IR types (`ir/`), a renderer (`WidgetRenderer.tsx`), a registry
(`registry.ts` + `defaultRegistry.ts`) — with React components kept separate and
wired in through thin adapters. Imitate this shape for the block IR.

**The IR node** (`ir/core.ts`) is a discriminated union on `kind`; only the
`component` variant carries a dispatch `type` plus `props` + `children`:

```ts
export type WidgetNode = TextNode | ElementNode | ComponentNode;
export interface ComponentNode {
  kind: "component";
  type: RagWidgetType | string;   // dispatch key; union for autocomplete, | string stays open
  props?: WidgetProps;            // all JsonValue -> the tree serializes cleanly to the wire
  children?: WidgetNode[];
}
```
Everything is JSON-serializable (important — your IR crosses a wire). Props may
themselves hold nodes (slots), and the renderer exposes `ctx.renderValue()` so a
widget can render node-valued props into named slots, not just children.

**The renderer** (`WidgetRenderer.tsx`) recurses, switching on `kind`, then
looking `type` up in the registry:

```ts
function renderComponentNode(node, ctx, registry) {
  const adapter = registry.get(node.type);
  if (!adapter) return <UnknownWidget node={node} />;    // graceful: shows "type not registered", siblings still render
  return adapter.render(node.props ?? {}, renderChildren(node.children, ctx, registry), ctx, node);
}
```
Children are pre-rendered and passed in as `ReactNode[]`. A `RenderContext` is
threaded to every adapter (`renderNode`/`renderChildren`/`renderValue`) so nested
widgets recurse without importing the renderer.

**The registry** (`registry.ts`) is a `Map` built from an array of adapter
objects (not a switch, not a global `register()` side effect):

```ts
export interface WidgetAdapter<P> { type: string; module: string;
  render(props: P, children: ReactNode[], ctx: RenderContext, node: ComponentNode): ReactNode; }
export function defineWidget<P>(a: WidgetAdapter<P>) { return a; }       // identity, for typing
export function createWidgetRegistry(adapters: readonly WidgetAdapter[]): WidgetRegistry { /* Map, throws on dup type */ }
export function mergeWidgetRegistries(...regs): WidgetRegistry { /* flatten + create */ }
```
Each adapter lives next to its component in a `*.widget.tsx` file, e.g.
`components/atoms/Button/Button.widget.tsx` calls `defineWidget({ type: "Button",
module, render })`. `defaultRegistry.ts` composes domain sub-registries via
`mergeWidgetRegistries(...)`. The renderer takes the registry as a **prop** (no
global singleton), so you can use different registries for different surfaces
(e.g. an on-screen preview vs. the print target). Usage in
`web/src/components/pages/DslPreviewPage/DslPreviewPage.tsx`.

**Add a widget type** = (1) write the React component; (2) add `Foo.widget.tsx`
with `defineWidget`; (3) add its adapter to a `createWidgetRegistry([...])` array;
(4) optionally add `"Foo"` to the type union and a prop interface.

**What to imitate:** the `kind`-discriminated node with a `component` variant
(`type`+`props`+`children`), all JsonValue; adapter objects + Map registry built
from arrays + merge helper, passed as a prop; a threaded `RenderContext`; graceful
unknown-type fallback + duplicate-registration guard; node-valued props (slots).
**What to skip:** the `element` raw-HTML variant (replace with your primitives:
text/qr/barcode/hr/image); the toast/`ActionSpec`/`bindAction` interactivity
machinery (a print DSL is non-interactive — strip the action plumbing).

**Important divergence:** rag-eval's IR is **hand-written TypeScript with no
protobuf**, and it **duplicates** the type-name list in Go
(`pkg/widgetschema/schema.go`) and TS — kept in sync by hand. That duplication is
its main fragility. **For us, do NOT hand-sync**: generate the TS IR types from
the protobuf schema (§6.2) so the block schema has one source of truth.

### 6.2 Protobuf Go/TS schema exchange — the org pattern (copy from `scraper`)

The clean, end-to-end example is `~/code/wesen/go-go-golems/scraper`
(`go-go-goja` is the example if you want a *publishable* TS package). Toolchain:
**Buf v2 + remote plugins protoc-gen-go + protoc-gen-es (`@bufbuild/protobuf`
v2)**, JSON transport via `protojson` <-> `fromJson`/`toJson`.

Note: `~/code/wesen/go-go-golems/almanach/internal/provisioning/native/proto/`
already has a buf setup, but it uses a **local** plugin and generates **Go only**
(vendored ESP-IDF protos) — do **not** copy that for the layout schema; copy
scraper's.

**`buf.gen.yaml`** (copy verbatim, adjust out dirs):

```yaml
version: v2
clean: true
plugins:
  - remote: buf.build/bufbuild/es          # TypeScript (protoc-gen-es v2)
    out: web/src/pb
    opt: [target=js+dts, import_extension=none]
  - remote: buf.build/protocolbuffers/go   # Go (protoc-gen-go)
    out: gen
    opt: [paths=source_relative]
```

**`buf.yaml`**: `version: v2`, `name: buf.build/local/almanach`,
`deps: [buf.build/googleapis/googleapis]` (for `google.protobuf.Struct`/
`Timestamp`).

**Proto layout:** `proto/almanach/layout/v1/layout.proto`, `package
almanach.layout.v1;`, `option go_package =
"github.com/go-go-golems/almanach/gen/proto/almanach/layout/v1;layoutv1";`. Use
`google.protobuf.Struct`/`Value` for free-form block props, `oneof` for block
variants, `int64` only where truly needed (it becomes a JS `bigint`).

**Go side** — centralize a codec (like `scraper/pkg/runtimeevents/codec.go`):

```go
protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: false}.Marshal(msg)
// UseProtoNames:false -> camelCase JSON keys, which protobuf-es fromJson expects.
```

**TS side** — protobuf-es v2 generates, per message, a **type** and a runtime
**schema object**; decode at the API/load boundary and re-encode with `toJson`
(bigint-safe):

```ts
import { fromJson, toJson } from "@bufbuild/protobuf";
import { LayoutSchema, type Layout } from "../pb/proto/almanach/layout/v1/layout_pb";
const layout: Layout = fromJson(LayoutSchema, wireJson);   // JSON -> typed message
```
`package.json`: `"@bufbuild/protobuf": "^2.11.x"`, `"type": "module"`.

**Transport gotchas** (verified in the org's tests): `int64` -> JSON string ->
`bigint` in TS (never `JSON.stringify` a decoded message with bigint fields —
round-trip via `toJson`); `google.protobuf.Timestamp` -> `{seconds: bigint,
nanos: number}`; `google.protobuf.Struct`/`Value` -> plain JSON object on
`toJson`. Add a golden-fixture decode test (like
`go-go-goja/.../replapi_decode.test.ts`) to lock the contract.

**Workflow:** edit `.proto` -> `buf generate` (from the dir with the buf configs)
-> Go regenerates into `gen/`, TS into `web/src/pb/`. Add a `make proto: buf
generate` target. In CI, `go install github.com/bufbuild/buf/cmd/buf@<ver>` before
`buf generate` if `go generate` drives it.

**Files to copy from:** `scraper/buf.gen.yaml`, `scraper/buf.yaml`,
`scraper/pkg/runtimeevents/codec.go`, `scraper/web/src/api/runtimeEventsApi.ts`;
plus `go-go-goja/web/packages/replapi-types/{package.json,src/index.ts,
src/replapi_decode.test.ts}` if you want a publishable shared TS package instead
of in-repo generated code.

## 7. Phased plan (do these in order; ship after each of 2-5)

- **Phase 0 — Orient.** Read this doc; skim `almanach-studio.jsx` (`BLOCK_TYPES`,
  two block components, the theme objects); run the render pipeline once (§8).
- **Phase 1 — Proto schema + codegen wiring.** Add scraper's Buf setup; define
  `Layout`, `Block`, `TextStyle`, `Typography`, `Theme`, `RenderOptions` with a
  `schema_version`. `buf generate` Go + TS. No behavior change yet — just get
  types generated and importable on both sides, with a round-trip decode test.
- **Phase 2 — Renderer registry.** Refactor the studio so blocks render through a
  `type -> adapter` registry (§6.1). Behavior identical; unknown types show a
  placeholder. De-risks everything after.
- **Phase 3 — Typography presets + defaults + overrides (§5.3).** Introduce the
  preset model with the paper-verified defaults (bigger, hinted, bold small/
  italic). Migrate components to resolve styles from presets instead of `fs(n)`.
  **Ship this and print a real page — this is where the quality win lands.**
- **Phase 4 — Data-driven themes + hinted fonts (§5.4).** Embed DejaVu Serif/Sans;
  make themes data (schema) selectable/overridable by a layout.
- **Phase 5 — Typed + per-block render options (§5.5).** Replace the hand-parsed
  render keys; add per-block overrides.
- **Phase 6 — Block-aware rasterization + per-segment heat (§5.5).** Use the
  bounding boxes; coordinate with the `ALMANACH-RASTER-LAB` rasterizer/heat code.

## 8. Pointers (code, commands, prior work)

Code:

- `web/src/almanach-studio.jsx` — studio + renderer; `BLOCK_TYPES` ~L213, theme
  objects ~L20, block components ~L258+, `fs`/`bodyScale` ~L1481.
- `web/src/fonts-embedded.css` — embedded base64 fonts (add hinted fonts here).
- `internal/app/renderer.go` — headless Chrome, screenshot, allocator, AA (see
  `fontconfig.go`, `supersample.go` from ALMANACH-PIXELFONT).
- `internal/app/render_oneshot.go` — layout parse + `render:` extraction
  (`intFromRenderOptions` — replace with the proto).
- `internal/app/bitmap.go` — the 1-bit threshold.
- `internal/provisioning/native/proto/` — existing buf setup (Go-only; not the
  pattern to copy, but shows buf is already in the repo).

Run the pipeline locally (needs a non-snap Chrome — the default snap chromium is
blocked in the dev sandbox):

```
go build -o /tmp/svc ./cmd/almanach-render-service
/tmp/svc render --layout examples/layouts/03-knowledge-strip.yaml \
  --out /tmp/o.png --format png --debug-dir /tmp/r \
  --web-dir web/dist --chrome-path /usr/bin/google-chrome-stable
# printer, if present, is http://192.168.0.126 ; density via
#   curl -X POST -d '{"density":38}' http://192.168.0.126/api/printer/density
```

Reference repos:

- `~/code/wesen/go-go-golems/rag-evaluation-system` — widget IR renderer (§6.1).
- `~/code/wesen/go-go-golems/scraper` — protobuf Go/TS exchange (§6.2).
- `~/code/wesen/go-go-golems/go-go-goja` — publishable shared TS proto package.

Prior tickets (read the diaries — they hold the paper evidence):

- `ttmp/2026/07/16/ALMANACH-RASTER-LAB--.../` — dithering, tone curve, printer
  heat, per-segment printing, the rasterizer/heat code.
- `ttmp/2026/07/16/ALMANACH-PIXELFONT--.../` — small-text investigation, the
  font/heat matrix harness (`scripts/02-font-matrix.py`), the typography recipe to
  bake into the presets.
- Vault deep-dive article: `go-go-parc` ->
  `Projects/2026/07/16/ARTICLE - Thermal Rasterization - Dithering, Heat, and Bitmap Fonts.md`.

## 9. Open questions to resolve early

- Block content in the proto: start with `google.protobuf.Struct` per block (fast,
  flexible) or per-type messages in a `oneof` (typed, stricter)? **Recommend
  Struct first, migrate hot block types to messages later.**
- Preset names + exact default values: seed from the `ALMANACH-PIXELFONT` recipe,
  then tune on paper.
- Backward compatibility: keep parsing existing YAML layouts, or ship a converter?
  **Recommend a converter + a `schema_version` gate.**
- One source of truth for block type names: they must not be hand-duplicated Go/TS
  (rag-eval's mistake) — generate from the proto.
