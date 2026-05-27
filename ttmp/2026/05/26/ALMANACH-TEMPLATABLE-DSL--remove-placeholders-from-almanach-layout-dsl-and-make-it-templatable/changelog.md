# Changelog

## 2026-05-26

- Initial workspace created


## 2026-05-26

Created ticket with design doc and investigation diary. Mapped all 4 hardcoded placeholder pools, designed template engine architecture, wrote 6-phase implementation plan.

### Related Files

- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/fetch_history.go — Hardcoded history fallback identified
- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/fetch_news.go — Placeholder headlines identified
- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/fetch_quote.go — Hardcoded quote pool identified
- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/fetch_word.go — Hardcoded word pool identified


## 2026-05-26

Uploaded design doc + diary bundle to reMarkable at /ai/2026/05/26/ALMANACH-TEMPLATABLE-DSL. Doctor passes clean.


## 2026-05-26

Scope expanded: remove ALL 6 fetcher files (not just 3 placeholder ones). buildDefaultLayout() replaced with minimal scaffold. Design doc rewritten, diary updated. Ready for reMarkable re-upload.


## 2026-05-26

Phases 1-5 complete. All fetchers deleted, scaffold added, template engine implemented, --data/--define wired, docs and examples updated. 5 commits: 106486c, 4c821cd, 6e0cd1f, d053b04, 635b7fb.

### Related Files

- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/examples/templates/ — Template + data context examples (commit 635b7fb)
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/internal/app/cmd_render.go — Added --data/--define flags (commit d053b04)
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/internal/app/data_context.go — Data context loading (commit 6e0cd1f)
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/internal/app/layout.go — Replaced buildDefaultLayout with buildScaffoldLayout (commit 106486c)
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/internal/app/template.go — Template engine with ResolveTemplate (commit 4c821cd)


## 2026-05-26

Phase 6 complete. Manual validation: 5 end-to-end tests pass (template+data, no-data, plain, scaffold, override). Updated getting-started and user-guide docs.

