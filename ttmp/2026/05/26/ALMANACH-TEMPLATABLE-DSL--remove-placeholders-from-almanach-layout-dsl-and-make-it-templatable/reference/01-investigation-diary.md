---
title: "Investigation Diary"
doc_type: reference
ticket: ALMANACH-TEMPLATABLE-DSL
status: active
intent: long-term
topics:
  - almanach
  - dsl
  - templates
  - diary
created: "2026-05-26"
updated: "2026-05-26"
---

# Investigation Diary

## Goal

Remove all fetchers and hardcoded content from the almanach system. Make the YAML DSL the sole content source. Add a template engine for automation. Update all docs.

## Step 1: Codebase Exploration and Placeholder Identification

Mapped the entire almanach system: Go backend (`internal/app/`), React SPA (`web/src/almanach-studio.jsx`), CLI commands, HTTP API, and Chrome render pipeline. Identified six fetcher files that inject content the user did not write.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to remove any placeholders from the almanach format / DSL, and update the docs (and skill) accordingly by making them templatable. Create a detailed analysis / design / implementation guide..."

**Assistant interpretation:** Research the almanach codebase, identify all placeholder/hardcoded data, design a template system, produce an intern-ready design doc.

**Inferred user intent:** Produce a complete, implementable design that a new team member can follow.

### What I did

- Read all Go source files in `internal/app/`
- Read the React Studio SPA (`almanach-studio.jsx`)
- Read all documentation files in `internal/app/doc/`
- Read the pi skill file
- Created docmgr ticket ALMANACH-TEMPLATABLE-DSL
- Created design doc and diary

### What was tricky to build

- The `fetch_history.go` file mixes real Wikimedia API calls with hardcoded fallbacks in the same function — hard to separate cleanly.
- The React SPA's `DEFAULTS` object serves a different purpose (editor scaffolding) than the Go hardcoded pools. Must not be removed.

## Step 2: Scope Refinement — Remove ALL Fetchers

User clarified: remove ALL fetch functions, not just placeholders. No "serendipity" content at all. The YAML DSL specifies everything.

### Prompt Context

**User prompt (verbatim):** "ok so actually, we don't want any of the fetch things. I want to get rid of these 'serendipity' parts, and instead allow to use YAML DSL to then specify all these fields in the graphical layout"

**Assistant interpretation:** The design should delete ALL six `fetch_*.go` files, not just the four with hardcoded pools. Even the API-based fetchers (weather, history) go away. Content comes from YAML only.

**Inferred user intent:** Clean separation — the DSL is the content source, period. No APIs, no random pools, no generated defaults.

### What I did

- Rewrote the design doc to reflect the broader scope: all 6 fetcher files deleted, `buildDefaultLayout()` replaced with a minimal scaffold
- Simplified the before/after diagram
- Updated file lists: 6 files deleted (was 3), 1 function replaced (was modified)
- Added scaffold layout design (title + date block only, no invented content)

### Why

The user's clarification is cleaner than the original design. Instead of keeping some fetchers and removing others, everything goes. The YAML DSL already supports every field — the fetchers were a parallel content path that bypassed user control.

### What I learned

- The `fetch_date.go` function seemed harmless (just formats `time.Now()`), but removing it is consistent: if the user wants today's date, they put it in YAML or the data context.
- The scaffold layout is the right "empty state" — just a title and date block. No invented content. Visually clear that the user needs to provide more.
- The "no-data-no-templates" backwards-compatibility rule is even more important now: existing layout files must continue to work without any changes.

### What warrants a second pair of eyes

- Is the scaffold (title + date) the right empty state, or should it be just a title? Adding a date block is deterministic and useful, but it's still content the user didn't write.
- The decision to delete `fetch_date.go` even though it was just local formatting. Consistency vs. convenience tradeoff.

### What should be done in the future

- Consider adding a `--date` flag that auto-fills a date block with today's date (explicit opt-in, not implicit)
- Consider a "daily scaffold" command that generates a starter layout file with today's date pre-filled, so users have a starting point

### Code review instructions

- Read the updated design doc: `design-doc/01-templatable-dsl-analysis-design-and-implementation-guide.md`
- Focus on Section 5.1 (fetcher removal) and the before/after diagram in Section 10
- Check that the scaffold layout (`buildScaffoldLayout`) is minimal enough
- Verify that the "no-data-no-templates" rule is clearly documented

### Technical details

Six fetcher files to delete:

```
internal/app/fetch_date.go      — 13 lines  — fetchDate() local date
internal/app/fetch_weather.go   — 72 lines  — fetchWeather() wttr.in API
internal/app/fetch_news.go      — 14 lines  — fetchNews() hardcoded placeholders
internal/app/fetch_quote.go     — 25 lines  — fetchQuote() hardcoded pool of 8
internal/app/fetch_word.go      — 22 lines  — fetchWord() hardcoded pool of 7
internal/app/fetch_history.go   — 78 lines  — fetchHistory() Wikimedia + fallback
```

Total: 224 lines deleted. Replaced by ~15-line `buildScaffoldLayout()` function.

## Step 3: Implementation — Phase 1 (Delete Fetchers, Add Scaffold)

**Commit:** `106486c` — "feat: remove all fetchers and hardcoded content, replace with scaffold layout"

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Inferred user intent:** Implement the design now.

### What I did

- Deleted all 6 `fetch_*.go` files
- Replaced `buildDefaultLayout()` with `buildScaffoldLayout()` (title + date block only)
- Removed `formatDate()` helper
- Updated callers: `render_oneshot.go:81` and `renderer.go:97`
- Updated `layout_test.go`: removed `TestDefaultLocalFetchersUseFrontendSchema`, added `TestBuildScaffoldLayout`

### What didn't work

- First build attempt failed with `undefined: err` in `render_oneshot.go:82` because I left a stale `if err != nil` check from when `buildDefaultLayout` returned an error. `buildScaffoldLayout` doesn't return an error. Fixed by removing the error check.

## Step 4: Implementation — Phase 2 (Template Engine)

**Commit:** `4c821cd` — "feat: add template engine for {{variable}} resolution in layout files"

### What I did

- Created `internal/app/template.go` with `ResolveTemplate()`, `resolveValue()`, `resolveExpr()`, `walkMap()`, `walkValue()`
- Created `internal/app/template_test.go` with 16 test cases

### What didn't work

- First test compilation failed because `contains()` and `containsSubstr()` helper functions in the test file conflicted with identically-named functions in `server.go`. Fixed by replacing with `strings.Contains()`.

## Step 5: Implementation — Phase 3 (Data Context Loading)

**Commit:** `6e0cd1f` — "feat: add data context loading from YAML/JSON files and --define flags"

### What I did

- Created `internal/app/data_context.go` with `LoadDataContext()` and `loadDataCtxFromFlags()`
- Created `internal/app/data_context_test.go` with 9 test cases

## Step 6: Implementation — Phase 4 (CLI Integration)

**Commit:** `d053b04` — "feat: wire --data and --define CLI flags through render pipeline"

### What I did

- Added `Data` and `Define` fields + flags to all 4 CLI commands (render, print, print-remote, inspect)
- Thread `DataContext` through `layoutJSONFromPathOrDefault` → `layoutJSONFromObjectOrDefault`
- Added `ResolveTemplate()` call before JSON marshaling in `layoutJSONFromObjectOrDefault`
- Added data context extraction from wrapped request body `data` key for HTTP API
- Updated `layout_bundle_test.go` to pass `nil` DataContext

### What was tricky

- The `render_oneshot.go` edits required care: the `layoutJSONFromObjectOrDefault` function needed both the new data context parameter AND the template resolution logic, plus extraction of `data` from wrapped request bodies. All three had to land in one coherent edit.
- The `data_context.go` file had a duplicate `import` block after my first edit. Fixed by merging into a single import.

## Step 7: Implementation — Phase 5 (Docs and Examples)

**Commit:** `635b7fb` — "docs: add template syntax to DSL reference, add template examples, update skill file"

### What I did

- Added "Template Syntax" section to `layout-dsl-reference.md` with expression format, types, CLI usage, priority rules
- Updated wrapped render request example to include `data` key
- Added troubleshooting entries for template errors
- Created `examples/templates/` with 4 files: daily-briefing and knowledge-strip template + data pairs
- Updated `~/.pi/agent/skills/almanach-printing/SKILL.md` with template workflow section

### What should be done in the future

- Update `layouts-getting-started.md` and `layouts-user-guide.md` to reference templates
- Add template examples to tutorials
- Upload updated design doc + diary to reMarkable

## Step 8: Phase 6 — Manual Validation and Final Docs

**Commits:** `8c51be8`

### What I did

- Built binary, ran 5 end-to-end tests:
  1. `--layout template.yaml --data data.yaml` → 384×597 PNG, all variables resolved ✓
  2. `--layout template.yaml` (no data) → renders with literal `{{...}}` text ✓
  3. `--layout plain.yaml` (no expressions, no data) → unchanged ✓
  4. No `--layout` at all → scaffold (384×202) ✓
  5. `--data data.yaml --define title=OVERRIDE` → override wins ✓
- Verified rendered PNGs visually via VLM: "MORNING SIGNAL" title, weather, plan, note all correct
- Scaffold shows "ALMANACH" title + date, no invented content
- Updated `layouts-getting-started.md` with "Using Templates" section
- Updated `layouts-user-guide.md` with template recipe and troubleshooting entry

### What worked

- The no-op rule (empty data context = no template resolution) saved us: test #2 and #3 would have failed without it
- Debug artifacts (`layout.json`) made it trivial to verify variable substitution
- The `--define` override correctly takes priority over `--data` file values

### What didn't work

- `-D` shorthand doesn't work because Glazed doesn't auto-create short flags. Must use `--define` instead.

### What warrants a second pair of eyes

- The scaffold layout includes a date block with today's date — this is deterministic but still auto-generated content. Acceptable?

### What should be done in the future

- Consider adding `-D` as an explicit short flag alias
- Consider a `scaffold` command that generates a starter YAML file
- Print a template-driven layout on the physical thermal printer to validate paper output

## Step 9: Documentation Quality Pass — DSL Reference and Supporting Docs

### Prompt Context

**User prompt (verbatim):** "We've updated the yaml spec significantly ... Update @internal/app/doc/layout-dsl-reference.md and potentially other documentation. Study things well before updating, so we get high quality info"

**Assistant interpretation:** Review all current implementation code and the ticket workspace to understand what changed, then bring all docs up to date with the actual state of the codebase.

**Inferred user intent:** Ensure documentation accurately reflects the current DSL spec, including the image block (which was missing), scaffold behavior (undocumented), and correct CLI syntax (--define, not -D).

### What I did

- Thoroughly studied the current Go structs (`layout.go`), React SPA (`almanach-studio.jsx`), template engine (`template.go`), data context (`data_context.go`), CLI flags, and existing docs
- Updated `layout-dsl-reference.md`:
  - Added `image` to supported block types list (was missing — 15 → 16 types)
  - Added full `image` block section with all 9 data fields, defaults, and image source guidance (data URLs, HTTP URLs, ZIP bundles)
  - Added "Scaffold Layout" section explaining what happens when no --layout is provided
  - Fixed CLI template examples: changed `-D` to `--define` (Glazed doesn't create short flags)
  - Fixed `--define` syntax documentation: comma-separated `key=value` pairs, not repeatable
  - Expanded Data Context Priority section with clear syntax description
  - Fixed troubleshooting entry: `-D` → `--define`
- Updated `layouts-getting-started.md`: Added "No Layout File? The Scaffold" section
- Updated `layouts-user-guide.md`: Added "Scaffold Layout" section and "Photo Card" style recipe

### Why

The DSL reference was out of date in several ways: the image block was completely undocumented despite being fully implemented in both Go and React; the scaffold behavior (fetcher removal) was undocumented; the template CLI examples used `-D` which doesn't actually work with Glazed.

### What was tricky to build

- Determining which Go struct fields are actually rendered in the SPA vs. just accepted by the JSON schema. Weather has `humidity` and `wind` fields in Go with `omitempty`, but the React WeatherBlock doesn't render them. Quote has `source` in Go but not in the React renderer. Decided to document only fields that produce visible output.
- The image block has many fields (9) with subtle defaults that differ between the Go struct and the React DEFAULTS object. Had to cross-reference both carefully.

### What warrants a second pair of eyes

- The decision to omit `humidity`, `wind` (weather) and `source` (quote) from the reference because the SPA doesn't render them — should these be documented as "accepted but not rendered" for forward compatibility?
- The image block's `thermalTone` field is SPA-specific (controls a CSS filter). It works for headless rendering but isn't meaningful without the SPA's CSS pipeline.

### What should be done in the future

- Consider adding tutorials that demonstrate the image block
- Consider documenting weather `humidity`/`wind` and quote `source` as optional fields that are accepted but not currently rendered
- Add `-D` as an explicit short flag alias for `--define`

### What I did (follow-up: skill file)

- Updated `~/.pi/agent/skills/almanach-printing/SKILL.md` image block section:
  - Added `thermalTone` field to the table (was missing — DSL reference and SPA both support it)
  - Fixed `fit` values: removed `fill` (SPA only supports `cover` and `contain`)
  - Added defaults column to match DSL reference (`"Image Plate"`, `""`, `160`, `"cover"`, `true`, `true`, `"normal"`)
  - Fixed `src` description: marked as Required
  - Updated YAML example: `height: 100` → `160` (correct default), added `thermalTone: normal`
- The skill file already had correct template workflow, scaffold mention, and `--define` syntax — no other changes needed
