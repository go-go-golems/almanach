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
