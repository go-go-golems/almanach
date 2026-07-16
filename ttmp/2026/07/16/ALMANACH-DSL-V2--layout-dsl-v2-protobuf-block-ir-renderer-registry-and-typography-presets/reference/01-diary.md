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
