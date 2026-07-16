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
codegen, no behavior change (nothing wired into the render path yet).

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
  (`web/src/pb/layout.roundtrip.test.mjs`, `fromJson`/`toJson`, run with `node`).
- **Wiring.** `make proto`, `make test-proto`, `pnpm --dir web test:proto`.

### What was tricky / gotchas (read before Phase 2+)
- **pnpm store is read-only in this sandbox.** The global `~/.config/pnpm/rc`
  sets `store-dir` to an archived, read-only workspace path, so any install that
  needs a *new* tarball fails with `ERR_PNPM_EROFS`, and `$HOME` itself is
  read-only. Worked around by reinstalling into a repo-local store:
  `pnpm install --store-dir ../.tmp-pnpm-store --config.confirmModulesPurge=false`
  (the dir is git-ignored). `node_modules` is now linked to that store; a plain
  `pnpm install` will error `UNEXPECTED_STORE` until you pass the same
  `--store-dir` (or fix the global config on a normal machine).
- **protobuf-es `tsEnum` strips the enum prefix at runtime**: the value is
  `RasterMode.THRESHOLD`, not `RASTER_MODE_THRESHOLD` (the JSON wire form keeps
  the full name). The `.d.ts` shows full names; runtime keys are short.
- No `int64` in the schema, so no `bigint` round-trip hazard yet — keep it that
  way, or add a `toJson` round-trip before any `JSON.stringify` if you introduce
  one.

### Verified
- `make test-proto` green (Go + TS). `buf generate` is idempotent (re-running
  produces no diff). `GOWORK=off go build ./...` and `go vet ./internal/layoutpb/...`
  clean.

### What should be done next
- Phase 2: the React renderer registry. The generated TS types
  (`web/src/pb/.../layout_pb`) are now importable in the studio; build the
  `type -> adapter` registry against `Block`/`Layout` and a graceful
  unknown-type placeholder.
