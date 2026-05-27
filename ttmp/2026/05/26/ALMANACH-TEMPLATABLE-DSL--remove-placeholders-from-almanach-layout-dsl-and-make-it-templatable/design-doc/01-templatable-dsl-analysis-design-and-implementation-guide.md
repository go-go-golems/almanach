---
title: "Templatable DSL: Analysis, Design, and Implementation Guide"
doc_type: design-doc
ticket: ALMANACH-TEMPLATABLE-DSL
status: active
intent: long-term
topics:
  - almanach
  - dsl
  - templates
  - layout
  - printing
created: "2026-05-26"
updated: "2026-05-26"
---

# Templatable DSL: Analysis, Design, and Implementation Guide

**Ticket:** ALMANACH-TEMPLATABLE-DSL
**Status:** Design — ready for implementation
**Audience:** A new intern joining the project who needs to understand the full system before making changes

---

## 1. Executive Summary

The Almanach system renders thermal paper pages from a YAML-based layout DSL. Currently, when no layout file is provided, the system calls a set of "fetcher" functions that generate content automatically: some pull from hardcoded pools of quotes and words ("serendipity" content), some call external APIs (weather, Wikipedia history), and some return outright placeholders ("Placeholder headline 1"). The result is that the default output contains random, stale, or fake content mixed with real API data — with no indication to the user which is which.

This design proposes a clean separation: **the YAML DSL is the sole content source**. All fetchers are removed. All hardcoded pools are removed. The `buildDefaultLayout()` function is removed. When no layout file is provided, the system produces a minimal empty-scaffold layout (title + date only) rather than inventing content.

On top of this, a **template layer** allows YAML layouts to contain `{{variable}}` expressions that are resolved from a separate data context file or CLI flags. This gives automation scripts (cron jobs, LLMs, pipelines) a way to generate layouts without string manipulation.

The change simplifies the system significantly: six Go files are deleted, two are substantially trimmed, and the content pipeline becomes purely data-driven.

---

## 2. Problem Statement and Scope

### 2.1 What is broken today

There are six fetcher files in the Go backend. All of them inject content that the user did not write:

| File | Function | Source | Problem |
|---|---|---|---|
| `fetch_date.go` | `fetchDate()` | Local computation | Harmless but redundant — user can write dates in YAML |
| `fetch_weather.go` | `fetchWeather()` | External API (wttr.in) | Silent failure: if API is down, block is omitted. User has no control. |
| `fetch_news.go` | `fetchNews()` | Hardcoded array | Returns "Placeholder headline 1" and "Placeholder headline 2". Not real news. |
| `fetch_quote.go` | `fetchQuote()` | Hardcoded pool of 8 | Random quote from a fixed pool. Repeats quickly. |
| `fetch_word.go` | `fetchWord()` | Hardcoded pool of 7 | Random word from a fixed pool. Repeats quickly. |
| `fetch_history.go` | `fetchHistory()` | Wikimedia API + 3-item fallback | Mixes live API with hardcoded fallback. Confusing. |

These are all called by `buildDefaultLayout()` in `layout.go:176`, which constructs a layout automatically when no `--layout` file is provided. The function is called from two places:

1. `render_oneshot.go:81` — CLI render/print/inspect commands when no layout path is given
2. `renderer.go:97` — HTTP server `/api/render` and `/api/render-and-print` when POST body is empty

The core issue is that **the system invents content the user did not ask for**. The YAML DSL already supports specifying every field of every block type. The React Studio UI already lets users edit all these fields interactively. The fetchers are a legacy path from before the DSL was complete.

### 2.2 What we want instead

1. **No fetchers.** All content comes from the YAML/JSON layout file.
2. **No `buildDefaultLayout()`.** When no layout is provided, produce a minimal scaffold (title + date with today's date), not a full page of invented content.
3. **Template support.** Layout files can contain `{{variable}}` expressions that are resolved from a data context, so automation can produce layouts without generating full YAML each time.
4. **Updated docs and skill file.** Remove all references to fetchers, placeholders, and generated content.

### 2.3 Scope boundaries

**In scope:**

- Delete all six `fetch_*.go` files.
- Remove `buildDefaultLayout()` from `layout.go` and replace with a minimal scaffold.
- Implement template engine for `{{variable}}` resolution.
- Add `--data` / `--define` CLI flags for template data context.
- Update all docs (`layout-dsl-reference.md`, `layouts-getting-started.md`, `layouts-user-guide.md`, tutorials).
- Update the pi skill file (`almanach-printing/SKILL.md`).
- Add template example files.

**Out of scope:**

- Changing the React Studio UI (template resolution is backend-only).
- Adding new block types.
- Modifying ESP32 firmware or BLE provisioning.
- Adding a scheduler/cron system.

---

## 3. Current-State Architecture

### 3.1 High-level data flow

The system has a linear pipeline from content to paper:

```
┌──────────────────────────────────────────────────────────────────┐
│                        DATA SOURCES                              │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐               │
│  │ YAML/JSON│  │ ZIP bundle│  │ fetchers + APIs  │ ← REMOVING    │
│  │ file     │  │ (.zip)    │  │ (wttr.in, wiki,  │    ALL OF     │
│  │          │  │           │  │  hardcoded pools) │    THIS       │
│  └────┬─────┘  └─────┬─────┘  └────────┬─────────┘               │
│       │              │                  │                         │
└───────┼──────────────┼──────────────────┼─────────────────────────┘
        │              │                  │
        ▼              ▼                  ▼
  ┌─────────────────────────────────────────┐
  │      layoutJSONFromPathOrDefault()      │  ← layout_bundle.go:32
  │                                         │
  │  1. Read file or call buildDefaultLayout│
  │  2. Parse YAML/JSON to map[string]any   │
  │  3. Inline ZIP image assets as data:URLs│
  │  4. Marshal to JSON string              │
  └─────────────────┬───────────────────────┘
                    │
                    ▼  layoutJSON (string)
  ┌─────────────────────────────────────────┐
  │       renderWithChrome()                │  ← renderer.go:100
  │                                         │
  │  1. Launch headless Chrome              │
  │  2. Navigate to /almanach SPA          │
  │  3. Call window.almanachLoadLayout()    │
  │  4. Wait for fonts + images             │
  │  5. Screenshot CSS selector             │
  │  6. Convert PNG → 1-bit bitmap          │
  └─────────────────┬───────────────────────┘
                    │
                    ▼  RenderResult { Bitmap, PNG, Metrics }
  ┌─────────────────────────────────────────┐
  │       Output path                       │
  │                                         │
  │  render → write PNG/bitmap to disk      │
  │  print  → POST bitmap to ESP32          │
  │  serve  → HTTP JSON/binary response     │
  └─────────────────────────────────────────┘
```

After this change, the left side simplifies dramatically:

```
┌────────────────────────────────────────────────────┐
│                   DATA SOURCES                      │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐      │
│  │ YAML/JSON│  │ ZIP bundle│  │ Data context │      │
│  │ template │  │ (.zip)    │  │ (--data file)│      │
│  └────┬─────┘  └─────┬─────┘  └──────┬───────┘     │
│       │              │               │              │
└───────┼──────────────┼───────────────┼──────────────┘
        │              │               │
        ▼              ▼               ▼
  ┌─────────────────────────────────────────┐
  │    layoutJSONFromPathOrDefault()        │
  │                                         │
  │  1. Read file                           │
  │  2. Parse YAML/JSON                     │
  │  3. Resolve {{template}} expressions    │  ← NEW
  │  4. Inline ZIP image assets             │
  │  5. Marshal to JSON string              │
  └─────────────────┬───────────────────────┘
                    │
                    ▼  (rest of pipeline unchanged)
```

### 3.2 The layout DSL

A layout is a JSON object with a specific schema. The top-level fields define page settings; the `blocks` array holds ordered content blocks.

**Page-level fields:**

| Field | Type | Default | Purpose |
|---|---|---|---|
| `almanach_studio_version` | int | 1 | Schema version |
| `exported_at` | string | now | Provenance timestamp |
| `theme` | string | "minimal" | Visual theme key |
| `paperWidth` | int | 384 | Paper width in pixels |
| `bodyScale` | float | 1.4–1.6 | Font size multiplier |
| `feedLines` | int | 3 | Blank rows for tear-off |

**Block envelope:**

```json
{
  "id": "unique-block-id",
  "type": "title",
  "data": { /* type-specific fields */ }
}
```

**Supported block types (16 total):**

`title`, `date`, `divider`, `plan`, `news`, `weather`, `note`, `image`, `habits`, `mood`, `reading`, `reflection`, `quote`, `word`, `history`, `did`

Each type has a specific data schema defined in `internal/app/layout.go` as Go structs. The DSL already supports specifying every single field — the fetchers are unnecessary.

### 3.3 Go backend structure

The Go backend lives in `internal/app/`. Here is a map of the key files, with annotations for what this change affects:

```
internal/app/
├── cmd_root.go           # CLI entry point, registers all subcommands
├── cmd_render.go         # `render` subcommand             ← ADD --data/--define
├── cmd_print.go          # `print` subcommand              ← ADD --data/--define
├── cmd_print_remote.go   # `print-remote` subcommand       ← ADD --data/--define
├── cmd_inspect.go        # `inspect` subcommand            ← ADD --data/--define
├── cmd_serve.go          # `serve` subcommand
├── cmd_setup.go          # `setup` subcommand (BLE)
├── cmd_ble_provision.go  # `ble-provision` subcommand
│
├── config.go             # Config struct, env var loading
├── layout.go             # Layout, Block, Data structs     ← REMOVE buildDefaultLayout, keep types
├── layout_bundle.go      # ZIP, YAML/JSON parsing          ← ADD dataCtx parameter
├── layout_test.go        # Layout tests                    ← REMOVE fetcher tests
├── render_oneshot.go     # One-shot render                 ← ADD template resolution
├── renderer.go           # Chrome headless pipeline        ← REMOVE buildDefaultLayout fallback
├── server.go             # HTTP server, routes             ← Handle data key in POST body
├── printer.go            # Bitmap segmentation, ESP32 POST
├── bitmap.go             # PNG → 1-bit bitmap conversion
│
├── fetch_date.go         # DELETE                          ✗
├── fetch_weather.go      # DELETE                          ✗
├── fetch_news.go         # DELETE                          ✗
├── fetch_quote.go        # DELETE                          ✗
├── fetch_word.go         # DELETE                          ✗
├── fetch_history.go      # DELETE                          ✗
│
├── template.go           # NEW: template engine            ✓
├── template_test.go      # NEW: engine tests               ✓
├── data_context.go       # NEW: data context loading       ✓
├── data_context_test.go  # NEW: loading tests              ✓
│
└── doc/                  # Embedded help documents         ← UPDATE
    ├── layout-dsl-reference.md
    ├── layouts-getting-started.md
    ├── layouts-user-guide.md
    ├── tutorial-daily-briefing.md
    └── tutorial-knowledge-strip.md
```

### 3.4 The React Studio SPA

The frontend is a single JSX file at `web/src/almanach-studio.jsx` (~2615 lines). It contains:

- **`DEFAULTS`** (line 126): Default data for each block type, used when creating new blocks in the editor or when imported block data is missing fields. These are *editor scaffolding*, not the Go placeholder pools. **These stay** — they serve a different purpose (interactive editing).
- **`BLOCK_TYPES`** (line 213): Metadata for the 16 block types.
- **`RENDERERS`** (line 756): Maps block type → React component.
- **`parseLayoutJson()`** (line 1456): Parses JSON into normalized blocks, merging missing fields from `DEFAULTS`.
- **`buildLayoutJson()`** (line 1437): Serializes editor state to layout JSON.
- **`window.almanachLoadLayout`** (line 1542): Headless API for Go backend.

The Studio does not need template awareness. Templates are resolved before the layout JSON reaches the SPA.

### 3.5 CLI commands affected

All layout-consuming commands share a common loading path through `layoutJSONFromPathOrDefault()`. The commands that will get new `--data` / `--define` flags:

| Command | File | New flags |
|---|---|---|
| `render` | `cmd_render.go` | `--data`, `--define` |
| `print` | `cmd_print.go` | `--data`, `--define` |
| `print-remote` | `cmd_print_remote.go` | `--data`, `--define` |
| `inspect` | `cmd_inspect.go` | `--data`, `--define` |

---

## 4. Gap Analysis

### 4.1 Content is invented without user consent

`buildDefaultLayout()` calls five fetcher functions and hardcodes one `DidData` block. The user has zero control over what appears unless they provide a full layout file. Even then, the "default" path (no `--layout` flag) produces a page full of random quotes, random words, and fake news.

### 4.2 Placeholder content masquerades as real content

`fetchNews()` returns "Placeholder headline 1" — indistinguishable in structure from real headlines. There is no `stale: true` flag, no log warning, nothing.

### 4.3 The DSL already supports all fields

Every block type's data schema is fully defined in `layout.go` and fully editable in the Studio UI. The fetchers are a parallel content path that bypasses the DSL entirely. Removing them loses no capability.

### 4.4 No template mechanism for automation

Scripts and cron jobs that want to generate daily layouts have no way to parameterize a layout file. They must generate the full YAML each time. A template layer (separate from fetcher removal) fills this gap.

---

## 5. Proposed Architecture

### 5.1 Remove all fetchers and the default layout builder

**Delete these files entirely:**

| File | Lines | What it contained |
|---|---|---|
| `fetch_date.go` | 13 | `fetchDate()` — local date formatting |
| `fetch_weather.go` | 72 | `fetchWeather()` — wttr.in API client |
| `fetch_news.go` | 14 | `fetchNews()` — hardcoded placeholder headlines |
| `fetch_quote.go` | 25 | `fetchQuote()` — hardcoded pool of 8 quotes |
| `fetch_word.go` | 22 | `fetchWord()` — hardcoded pool of 7 words |
| `fetch_history.go` | 78 | `fetchHistory()` — Wikimedia API + hardcoded fallback |

**Remove from `layout.go`:**

- The `buildDefaultLayout()` function (lines 175–215)
- The `formatDate()` helper (line 217)
- The hardcoded `DidData` block (lines 206–211)

**Replace with a minimal scaffold function:**

```go
// buildScaffoldLayout produces a minimal layout with just a title and date.
// Used when no layout file is provided — no content is invented.
func buildScaffoldLayout(cfg Config) *Layout {
    now := time.Now()
    return &Layout{
        Version:    1,
        ExportedAt: now.UTC().Format(time.RFC3339),
        Theme:      cfg.DefaultTheme,
        PaperWidth: cfg.PaperWidth,
        BodyScale:  cfg.BodyScale,
        FeedLines:  cfg.FeedLines,
        Blocks: []Block{
            newBlock("title", TitleData{
                Text:     "ALMANACH",
                Subtitle: now.Format("January 2, 2006"),
            }),
        },
    }
}
```

This scaffold gives the user something to render (a single title block with today's date) rather than erroring. It contains zero invented content — the only text is the word "ALMANACH" and the current date.

**Update callers:**

| File | Line | Change |
|---|---|---|
| `render_oneshot.go` | 81 | `buildDefaultLayout(cfg)` → `buildScaffoldLayout(cfg)` |
| `renderer.go` | 97 | `buildDefaultLayout(s.cfg)` → `buildScaffoldLayout(s.cfg)` |

### 5.2 Template engine

The template engine is a single Go file (`internal/app/template.go`) that resolves `{{variable}}` expressions in layout files using a provided data context.

**Expression syntax:**

| Expression | Meaning |
|---|---|
| `{{key}}` | Look up `key` in the data context. Error if missing. |
| `{{key:fallback value}}` | Look up `key`. Use fallback string if missing. |
| `{{$ENV_VAR}}` | Look up environment variable `ENV_VAR`. Error if unset. |
| `{{$ENV_VAR:fallback}}` | Look up environment variable. Use fallback if unset. |

No conditionals, no loops, no function calls. This is a flat substitution engine with environment variable fallback. It adds zero external dependencies.

**Example:**

Template (`morning.yaml`):
```yaml
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.45
feedLines: 3
blocks:
  - id: title-1
    type: title
    data:
      text: "{{title}}"
      subtitle: "{{subtitle}}"
  - id: date-1
    type: date
    data:
      date: "{{date}}"
      day: "{{day}}"
  - id: weather-1
    type: weather
    data:
      temp: "{{temp}}"
      condition: "{{condition}}"
      high: "{{high}}"
      low: "{{low}}"
  - id: quote-1
    type: quote
    data:
      label: "Quote of the Day"
      text: "{{quote_text}}"
      author: "{{quote_author}}"
```

Data context (`data.yaml`):
```yaml
title: "MORNING SIGNAL"
subtitle: "Coffee, weather, tasks"
date: "May 26, 2026"
day: "Monday"
temp: "22°C"
condition: "Partly cloudy"
high: "25°C"
low: "14°C"
quote_text: "The only way to do great work is to love what you do."
quote_author: "Steve Jobs"
```

Result: a standard layout file with all expressions replaced by data context values.

### 5.3 Data context sources

The data context is built by merging sources in priority order (highest wins):

```
CLI --define flags  >  --data file  >  environment variable fallbacks in expressions
```

| Source | CLI flag | Format | Priority |
|---|---|---|---|
| Inline key-value | `--define key=value` / `-D key=value` | string | Highest |
| Data file | `--data context.yaml` / `-d context.yaml` | YAML or JSON | Medium |
| Environment fallback | `{{$ENV_VAR}}` expression syntax | string | Lowest |

### 5.4 Pipeline integration point

The template engine is called inside `layoutJSONFromObjectOrDefault()` in `render_oneshot.go`, after parsing the raw YAML/JSON but before marshaling to JSON:

```
CURRENT:
  readFile → parseYAML → layoutJSONFromObjectOrDefault() → marshalJSON → render

PROPOSED:
  readFile → parseYAML → resolveTemplate(layout, dataCtx) → layoutJSONFromObjectOrDefault() → marshalJSON → render
```

**Backwards compatibility rule:** When the data context is empty (no `--data` file, no `--define` flags), template resolution is skipped entirely. Existing layout files without `{{...}}` expressions are completely unaffected. This is critical — it means the change is zero-risk for existing users.

### 5.5 HTTP API changes

The `POST /api/render` and `POST /api/render-and-print` endpoints accept a JSON body. After this change, the body can contain a `data` key alongside `layout` and `render`:

```json
{
  "layout": { "blocks": [...] },
  "data": { "title": "HELLO", "date": "May 26, 2026" }
}
```

The server handler extracts `data` and passes it through `resolveTemplate()`.

---

## 6. Detailed Design

### 6.1 Template expression parser

**Pseudocode:**

```go
// resolveValue resolves all {{expr}} markers in a single string.
func resolveValue(s string, ctx map[string]string) (string, error) {
    var result strings.Builder
    i := 0
    for i < len(s) {
        if i+1 < len(s) && s[i] == '{' && s[i+1] == '{' {
            end := strings.Index(s[i:], "}}")
            if end == -1 {
                return "", fmt.Errorf("unclosed {{ in template string")
            }
            expr := strings.TrimSpace(s[i+2 : i+end])
            val, err := resolveExpr(expr, ctx)
            if err != nil {
                return "", fmt.Errorf("{{%s}}: %w", expr, err)
            }
            result.WriteString(val)
            i = i + end + 2
        } else {
            result.WriteByte(s[i])
            i++
        }
    }
    return result.String(), nil
}

// resolveExpr resolves one expression: key, key:fallback, $ENV, $ENV:fallback
func resolveExpr(expr string, ctx map[string]string) (string, error) {
    parts := strings.SplitN(expr, ":", 2)
    key := parts[0]
    hasFallback := len(parts) == 2
    fallback := ""
    if hasFallback {
        fallback = parts[1]
    }

    // Environment variable: $NAME
    if strings.HasPrefix(key, "$") {
        envKey := key[1:]
        if val, ok := os.LookupEnv(envKey); ok {
            return val, nil
        }
        if hasFallback {
            return fallback, nil
        }
        return "", fmt.Errorf("environment variable %s not set", envKey)
    }

    // Regular key lookup
    if val, ok := ctx[key]; ok {
        return val, nil
    }
    if hasFallback {
        return fallback, nil
    }
    return "", fmt.Errorf("template variable %q not provided", key)
}
```

### 6.2 Recursive object walker

The `ResolveTemplate` function walks the layout object recursively, replacing strings that contain `{{...}}`:

```go
func ResolveTemplate(obj map[string]interface{}, ctx map[string]string) error {
    if len(ctx) == 0 {
        return nil // No data context → no-op (backwards compat)
    }
    return walkMap(obj, ctx)
}

func walkMap(m map[string]interface{}, ctx map[string]string) error {
    for k, v := range m {
        resolved, err := walkValue(v, ctx)
        if err != nil {
            return fmt.Errorf("key %q: %w", k, err)
        }
        m[k] = resolved
    }
    return nil
}

func walkValue(v interface{}, ctx map[string]string) (interface{}, error) {
    switch val := v.(type) {
    case map[string]interface{}:
        if err := walkMap(val, ctx); err != nil {
            return nil, err
        }
        return val, nil
    case []interface{}:
        for i, child := range val {
            resolved, err := walkValue(child, ctx)
            if err != nil {
                return nil, fmt.Errorf("index %d: %w", i, err)
            }
            val[i] = resolved
        }
        return val, nil
    case string:
        if strings.Contains(val, "{{") && strings.Contains(val, "}}") {
            return resolveValue(val, ctx)
        }
        return val, nil
    default:
        return v, nil // numbers, bools, nil — pass through
    }
}
```

### 6.3 Data context loading

New file `internal/app/data_context.go`:

```go
package app

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"
)

// DataContext is a flat key-value mapping for template resolution.
type DataContext map[string]string

// LoadDataContext builds a DataContext from a file and/or inline key=value pairs.
// Priority: inline defines > file values.
func LoadDataContext(dataPath string, defines []string) (DataContext, error) {
    ctx := DataContext{}

    if dataPath != "" {
        data, err := os.ReadFile(dataPath)
        if err != nil {
            return nil, fmt.Errorf("read data file %s: %w", dataPath, err)
        }

        var raw map[string]interface{}
        ext := strings.ToLower(filepath.Ext(dataPath))
        if ext == ".json" {
            if err := json.Unmarshal(data, &raw); err != nil {
                return nil, err
            }
        } else {
            if err := yaml.Unmarshal(data, &raw); err != nil {
                return nil, err
            }
        }

        for k, v := range raw {
            ctx[k] = fmt.Sprintf("%v", v)
        }
    }

    for _, d := range defines {
        parts := strings.SplitN(d, "=", 2)
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid --define: %q (expected key=value)", d)
        }
        ctx[parts[0]] = parts[1]
    }

    return ctx, nil
}
```

### 6.4 CLI flag additions

For each layout-consuming command, add:

```go
fields.New("data", fields.TypeString, fields.WithDefault(""),
    fields.WithHelp("YAML/JSON data context file for template resolution")),
fields.New("define", fields.TypeString, fields.WithDefault(""),
    fields.WithHelp("Inline key=value for template variables (repeat -D key=val)")),
```

The settings struct for each command gets:

```go
Data   string `glazed:"data"`
Define string `glazed:"define"`
```

### 6.5 Modified function signatures

```go
// BEFORE:
func layoutJSONFromPathOrDefault(layoutPath string, cfg Config) (*layoutLoadResult, error)
func layoutJSONFromObjectOrDefault(obj map[string]interface{}, cfg Config) (string, map[string]interface{}, error)

// AFTER:
func layoutJSONFromPathOrDefault(layoutPath string, cfg Config, dataCtx DataContext) (*layoutLoadResult, error)
func layoutJSONFromObjectOrDefault(obj map[string]interface{}, cfg Config, dataCtx DataContext) (string, map[string]interface{}, error)
```

### 6.6 Scaffold layout for empty input

When no layout file is provided and no POST body is sent, the system now produces a minimal scaffold instead of calling fetchers:

```go
// In render_oneshot.go, layoutJSONFromObjectOrDefault:
if len(obj) == 0 {
    layout := buildScaffoldLayout(cfg)
    b, err := json.Marshal(layout)
    if err != nil {
        return "", nil, fmt.Errorf("marshal scaffold: %w", err)
    }
    return string(b), nil, nil
}
```

```go
// In renderer.go, layoutJSONFromReader:
if layoutOverride != nil {
    data, err := io.ReadAll(layoutOverride)
    if err != nil {
        return "", fmt.Errorf("read layout: %w", err)
    }
    if len(bytes.TrimSpace(data)) > 0 {
        return string(data), nil
    }
}

// Instead of buildDefaultLayout, use scaffold:
layout := buildScaffoldLayout(s.cfg)
b, err := json.Marshal(layout)
if err != nil {
    return "", fmt.Errorf("marshal scaffold: %w", err)
}
return string(b), nil
```

---

## 7. Implementation Plan

### Phase 1: Delete fetchers and clean up layout.go

**Delete:**

- `internal/app/fetch_date.go`
- `internal/app/fetch_weather.go`
- `internal/app/fetch_news.go`
- `internal/app/fetch_quote.go`
- `internal/app/fetch_word.go`
- `internal/app/fetch_history.go`

**Modify `internal/app/layout.go`:**

- Replace `buildDefaultLayout()` with `buildScaffoldLayout()` (title only, no fetcher calls)
- Remove `formatDate()` helper
- Remove `math/rand/v2` and `time` imports if no longer needed
- Keep all type definitions (`Layout`, `Block`, `TitleData`, etc.) — these are unchanged
- Keep `newBlock()`, `dividerBlock()`, `nextBlockID()` helpers

**Modify `internal/app/render_oneshot.go`:**

- Replace `buildDefaultLayout(cfg)` → `buildScaffoldLayout(cfg)` at line 81

**Modify `internal/app/renderer.go`:**

- Replace `buildDefaultLayout(s.cfg)` → `buildScaffoldLayout(s.cfg)` at line 97

**Update `internal/app/layout_test.go`:**

- Remove `TestDefaultLocalFetchersUseFrontendSchema` (tests deleted functions)
- Keep `TestFrontendBlockSchemaDataKeys` (tests type definitions, not fetchers)

**Verify:**

```bash
cd ~/code/wesen/go-go-golems/almanach
go build ./...
go test ./... -count=1
```

**Definition of done:** Build compiles, tests pass, `almanach-render-service render` (no --layout) produces a single-block scaffold page.

### Phase 2: Implement template engine

**Create:**

- `internal/app/template.go` — `ResolveTemplate()`, `resolveValue()`, `resolveExpr()`, `walkMap()`, `walkValue()`
- `internal/app/template_test.go`

**Test cases:**

| Test | Input | Expected |
|---|---|---|
| Simple substitution | `"hello {{name}}"` + `{"name":"world"}` | `"hello world"` |
| Missing key without fallback | `"{{missing}}"` + `{}` | Error |
| Missing key with fallback | `"{{missing:default}}"` + `{}` | `"default"` |
| Env var | `"{{$USER}}"` + `{}` | Value of `$USER` |
| Env var with fallback | `"{{$NONEXISTENT:anon}}"` + `{}` | `"anon"` |
| Multiple expressions | `"{{a}} and {{b}}"` | Both resolved |
| Nested object | Walk through `blocks[*].data.*` | All strings resolved |
| Empty context + no expressions | Plain layout | Unchanged (no-op) |
| Empty context + expressions | Layout with `{{x}}` | Unchanged (no-op rule) |
| Fallback with colons | `"{{key:a:b:c}}"` | `"a:b:c"` (first colon splits) |

**Definition of done:** All tests pass. No new dependencies. `go vet` clean.

### Phase 3: Implement data context loading

**Create:**

- `internal/app/data_context.go` — `LoadDataContext()`
- `internal/app/data_context_test.go`

**Definition of done:** YAML and JSON data files load correctly. `--define` overrides file values.

### Phase 4: Wire CLI flags

**Modify:**

- `cmd_render.go` — Add `--data`, `--define` flags
- `cmd_print.go` — Add `--data`, `--define` flags
- `cmd_print_remote.go` — Add `--data`, `--define` flags
- `cmd_inspect.go` — Add `--data`, `--define` flags
- `layout_bundle.go` — Thread `DataContext` through `layoutJSONFromPathOrDefault()`
- `render_oneshot.go` — Thread `DataContext` through `layoutJSONFromObjectOrDefault()`, call `ResolveTemplate()`
- `server.go` — Handle `data` key in POST body

**Wiring pseudocode (cmd_render.go example):**

```go
func (c *RenderCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
    s := &RenderSettings{}
    // ... decode settings ...

    cfg := LoadConfig()

    // NEW: Load data context
    var defines []string
    if s.Define != "" {
        defines = strings.Split(s.Define, ",") // or however Glazed handles repeatable flags
    }
    dataCtx, err := LoadDataContext(s.Data, defines)
    if err != nil {
        return err
    }

    layoutSource, err := layoutJSONFromPathOrDefault(s.Layout, cfg, dataCtx)  // dataCtx added
    // ... rest unchanged ...
}
```

**Definition of done:** `almanach-render-service render --layout template.yaml --data data.yaml --out /tmp/test.png` works. `almanach-render-service render --layout plain.yaml --out /tmp/test.png` still works unchanged.

### Phase 5: Update documentation and examples

**Update docs:**

1. `internal/app/doc/layout-dsl-reference.md` — Add "Template Syntax" section; remove any references to generated/fetched content
2. `internal/app/doc/layouts-getting-started.md` — Add template workflow; update for scaffold behavior
3. `internal/app/doc/layouts-user-guide.md` — Add template design patterns; remove fetcher references
4. `internal/app/doc/tutorial-daily-briefing.md` — Add template variant
5. `internal/app/doc/tutorial-knowledge-strip.md` — Add template variant

**Create template examples:**

1. `examples/templates/daily-briefing-template.yaml`
2. `examples/templates/daily-briefing-data.yaml`
3. `examples/templates/knowledge-strip-template.yaml`
4. `examples/templates/knowledge-strip-data.yaml`

**Update skill file:**

- `~/.pi/agent/skills/almanach-printing/SKILL.md` — Add "Template Workflow" section, remove fetcher/placeholder references

### Phase 6: Testing and validation

- `go test ./... -count=1`
- `golangci-lint run ./...`
- Manual: render a template with data context
- Manual: render without data context (no-op)
- Manual: render with missing variable (clear error)
- Print a template-driven layout on the thermal printer

---

## 8. API Reference

### 8.1 New Go types

```go
// DataContext is a flat key-value mapping used for template resolution.
type DataContext map[string]string

// LoadDataContext builds a DataContext from a file and/or inline key=value pairs.
func LoadDataContext(dataPath string, defines []string) (DataContext, error)

// ResolveTemplate walks a layout object and resolves all {{expr}} expressions.
// When ctx is empty, this is a no-op.
func ResolveTemplate(obj map[string]interface{}, ctx DataContext) error
```

### 8.2 New function

```go
// buildScaffoldLayout produces a minimal layout with just a title block.
// Replaces the former buildDefaultLayout which used fetchers.
func buildScaffoldLayout(cfg Config) *Layout
```

### 8.3 Removed functions

| Function | File | Reason |
|---|---|---|
| `fetchDate()` | `fetch_date.go` | Users write dates in YAML |
| `fetchWeather()` | `fetch_weather.go` | Users write weather in YAML |
| `fetchNews()` | `fetch_news.go` | Was placeholder data only |
| `fetchQuote()` | `fetch_quote.go` | Was hardcoded pool |
| `fetchWord()` | `fetch_word.go` | Was hardcoded pool |
| `fetchHistory()` | `fetch_history.go` | Mixed API + hardcoded fallback |
| `fallbackHistory()` | `fetch_history.go` | Hardcoded fallback |
| `buildDefaultLayout()` | `layout.go` | Used all fetchers above |
| `formatDate()` | `layout.go` | Only used by buildDefaultLayout |

### 8.4 New CLI flags

| Flag | Short | Type | Applies to | Description |
|---|---|---|---|---|
| `--data` | `-d` | string (file path) | render, print, print-remote, inspect | YAML/JSON data context file |
| `--define` | `-D` | string (repeatable) | render, print, print-remote, inspect | Inline `key=value` override |

### 8.5 Template expression syntax

```
expression := "{{" expr "}}"
expr       := key [ ":" fallback ]
key        := varname | "$" envname
varname    := [a-zA-Z_][a-zA-Z0-9_]*
envname    := [a-zA-Z_][a-zA-Z0-9_]*
fallback   := any string (first colon splits)
```

---

## 9. File Reference

### 9.1 Files that will be deleted (Phase 1)

| File | Lines | Content |
|---|---|---|
| `internal/app/fetch_date.go` | 13 | Date formatting |
| `internal/app/fetch_weather.go` | 72 | wttr.in API client |
| `internal/app/fetch_news.go` | 14 | Placeholder headlines |
| `internal/app/fetch_quote.go` | 25 | Hardcoded quote pool |
| `internal/app/fetch_word.go` | 22 | Hardcoded word pool |
| `internal/app/fetch_history.go` | 78 | Wikimedia API + fallback |

### 9.2 Files that will be modified

| File | What changes | Phase |
|---|---|---|
| `internal/app/layout.go` | Replace `buildDefaultLayout()` with `buildScaffoldLayout()`, remove `formatDate()` | 1 |
| `internal/app/render_oneshot.go` | Use `buildScaffoldLayout`, add `DataContext` parameter | 1, 4 |
| `internal/app/renderer.go` | Use `buildScaffoldLayout` | 1 |
| `internal/app/layout_test.go` | Remove fetcher tests | 1 |
| `internal/app/layout_bundle.go` | Thread `DataContext` parameter | 4 |
| `internal/app/cmd_render.go` | Add `--data`, `--define` flags | 4 |
| `internal/app/cmd_print.go` | Add `--data`, `--define` flags | 4 |
| `internal/app/cmd_print_remote.go` | Add `--data`, `--define` flags | 4 |
| `internal/app/cmd_inspect.go` | Add `--data`, `--define` flags | 4 |
| `internal/app/server.go` | Handle `data` key in POST body | 4 |
| `internal/app/doc/layout-dsl-reference.md` | Add template syntax, remove fetcher refs | 5 |
| `internal/app/doc/layouts-getting-started.md` | Add template workflow | 5 |
| `internal/app/doc/layouts-user-guide.md` | Add template patterns | 5 |
| `~/.pi/agent/skills/almanach-printing/SKILL.md` | Add template examples | 5 |

### 9.3 Files that will be created

| File | Purpose | Phase |
|---|---|---|
| `internal/app/template.go` | Template engine core | 2 |
| `internal/app/template_test.go` | Engine unit tests | 2 |
| `internal/app/data_context.go` | Data context loading | 3 |
| `internal/app/data_context_test.go` | Loading tests | 3 |
| `examples/templates/daily-briefing-template.yaml` | Example template | 5 |
| `examples/templates/daily-briefing-data.yaml` | Example data | 5 |
| `examples/templates/knowledge-strip-template.yaml` | Example template | 5 |
| `examples/templates/knowledge-strip-data.yaml` | Example data | 5 |

---

## 10. Diagram: Before vs. After

### Before (current)

```
                    ┌─────────────────────────────────┐
                    │     buildDefaultLayout()         │
                    │                                 │
                    │  fetchDate()    → DateData      │
                    │  fetchWeather() → WeatherData   │  ← API call, can fail silently
                    │  fetchNews()    → NewsData       │  ← PLACEHOLDER "Placeholder headline 1"
                    │  fetchQuote()   → QuoteData      │  ← RANDOM from pool of 8
                    │  fetchWord()    → WordData       │  ← RANDOM from pool of 7
                    │  fetchHistory() → HistoryData    │  ← API + hardcoded fallback
                    │  (hardcoded)    → DidData        │  ← Fixed honey fact
                    └────────────┬────────────────────┘
                                 │
                                 ▼
                    Full page of invented + API content
```

### After (proposed)

```
  ┌───────────────┐     ┌───────────────┐
  │ layout.yaml   │     │ data.yaml     │
  │ (template)    │     │ (context)     │
  └───────┬───────┘     └───────┬───────┘
          │                     │
          └─────────┬───────────┘
                    │
                    ▼
          ┌─────────────────────┐
          │  ResolveTemplate()  │  ← Only if --data is provided
          │  Replace {{expr}}   │
          └─────────┬───────────┘
                    │
                    ▼
          Standard layout JSON → render pipeline (unchanged)


  No --layout provided?
                    │
                    ▼
          ┌─────────────────────┐
          │ buildScaffoldLayout │  ← Just title block, no invented content
          └─────────────────────┘
```

---

## 11. Risks, Alternatives, and Open Questions

### 11.1 Risks

1. **No more "zero-config" daily page.** Users who relied on `almanach-render-service render` (no args) getting a full daily page will now get just a title. **Mitigation:** This is intentional — the old behavior printed fake content. The scaffold makes it clear that you need to provide content.

2. **Breaking change for HTTP API callers** that send empty POST bodies. They now get a scaffold instead of a full page. **Mitigation:** Document clearly. The scaffold is visually obvious ("ALMANACH" title only).

3. **Glazed flag type limitations.** `fields.TypeStringList` may not exist for repeatable `-D` flags. **Mitigation:** Use comma-separated strings or single string per flag.

### 11.2 Alternatives considered

1. **Keep fetchers, add "is_generated" flag.** Rejected. The goal is to never invent content. A flag doesn't fix the fundamental problem.

2. **Go `text/template`.** Rejected. Too powerful (conditionals, loops, pipelines), conflicts with common text content (`{{.Key}}` syntax), harder to skip for backwards compatibility.

3. **Frontend template resolution.** Rejected. Would require template awareness in the SPA, doubling complexity. Backend is the right place.

4. **Keep `fetchDate()` only.** Considered — dates are local and deterministic. Rejected for consistency: the YAML DSL already has a `date` block type. If you want today's date, put it in the YAML or the data context.

### 11.3 Open questions

1. Should the scaffold include a date block in addition to the title? **Recommendation:** Yes — add a date block with today's date. It's deterministic (no API, no randomness) and makes the scaffold immediately recognizable as "today's page, waiting for content".

2. Should the scaffold be configurable (e.g., custom title text via config)? **Recommendation:** No for v1. Users who want customization should use a template.

---

## 12. Testing Strategy

### 12.1 Unit tests

| Test | File | Validates |
|---|---|---|
| `TestResolveValue` | `template_test.go` | Expression parser correctness |
| `TestResolveExpr` | `template_test.go` | Key lookup, env vars, fallbacks |
| `TestWalkValue` | `template_test.go` | Recursive object walking |
| `TestLoadDataContext` | `data_context_test.go` | File loading, merging |
| `TestTemplateNoOp` | `template_test.go` | Empty context = no-op |
| `TestTemplateErrors` | `template_test.go` | Missing variable → clear error |
| `TestBuildScaffoldLayout` | `layout_test.go` | Scaffold has title+date only |

### 12.2 Integration tests

| Test | Validates |
|---|---|
| `TestRenderWithTemplate` | `render --layout tmpl.yaml --data data.yaml --out test.png` |
| `TestRenderWithoutData` | `render --layout tmpl.yaml --out test.png` (no-op, expressions stay literal) |
| `TestRenderWithoutExpressions` | `render --layout plain.yaml --out test.png` (unchanged) |
| `TestScaffoldRender` | `render` with no args produces title+date scaffold |

### 12.3 Manual validation

1. Create template + data context
2. Render via CLI
3. Verify PNG content matches data context values
4. Print on thermal printer

---

## 13. Key Terms

| Term | Definition |
|---|---|
| **Layout** | A YAML/JSON document describing a thermal paper page as page settings + ordered blocks |
| **Block** | A content unit within a layout (title, date, weather, etc.) |
| **Template** | A layout file containing `{{variable}}` expressions instead of literal values |
| **Data context** | A flat key-value mapping used to resolve template expressions |
| **Scaffold** | A minimal layout (title + date) produced when no layout file is provided |
| **Fetcher** | A Go function that retrieves data from an API or hardcoded pool — all to be removed |
| **DSL** | Domain-specific language — the YAML-based layout format |
| **Render pipeline** | The path from layout YAML → Chrome headless screenshot → PNG/bitmap |
| **Bitmap** | A 1-bit monochrome image sent to the thermal printer |

---

## 14. References

| Reference | Path | Purpose |
|---|---|---|
| Layout DSL Reference | `internal/app/doc/layout-dsl-reference.md` | Complete field reference |
| Go Layout Types | `internal/app/layout.go` | Go struct definitions for all block data types |
| Layout Bundle Loading | `internal/app/layout_bundle.go` | YAML/JSON/ZIP parsing |
| Render Pipeline | `internal/app/renderer.go` | Chrome headless rendering |
| Print Pipeline | `internal/app/printer.go` | Bitmap segmentation and ESP32 POST |
| React Studio | `web/src/almanach-studio.jsx` | Frontend SPA |
| Pi Skill File | `~/.pi/agent/skills/almanach-printing/SKILL.md` | Printing workflow documentation |
| Example Layouts | `examples/layouts/` | Existing example files |
