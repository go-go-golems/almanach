# Changelog

## 2026-07-16

- Initial workspace created


## 2026-07-16

Step 1: Created ticket + self-contained handoff (protobuf block IR, renderer registry from rag-evaluation-system, typography presets+defaults, data-driven themes, per-block render options). Researched org IR-renderer and protobuf Go/TS patterns. 6 phase tasks.

Step 2 (Phase 1 complete): Added Buf v2 setup (`buf.yaml` scoped to `proto/`, `buf.gen.yaml` with local plugins), authored `proto/almanach/layout/v1/layout.proto` (Layout, Block, TextStyle, Typography, Theme, ThemeColors, RenderOptions + TextCase/RasterMode enums, `schema_version`, Struct block content). Generated Go (`gen/`) + TS (`web/src/pb/`). Added Go codec `internal/layoutpb` and a shared golden fixture with round-trip decode tests on both sides (Go `go test`, TS `node`). Wired `make proto` / `make test-proto` and `pnpm test:proto`. No behavior change — types only.

