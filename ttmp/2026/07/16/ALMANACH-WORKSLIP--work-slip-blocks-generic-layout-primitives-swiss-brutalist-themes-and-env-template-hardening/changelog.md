# Changelog

## 2026-07-16

- Initial workspace created


## 2026-07-16

Step 1: ticket created; intern design/implementation guide written and uploaded to reMarkable (Projects/ALMANACH-WORKSLIP-guide); 4 phase tasks added.


## 2026-07-16

Step 2 (Phase 1 complete): removed {{$ENV}} template resolution; boundary tests pin that layouts (even self-activating via their own data: map) can never read process env (commit 1de738e).

### Related Files

- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/template.go — env branch removed from resolveExpr
- /home/manuel/code/wesen/go-go-golems/almanach/internal/app/template_boundary_test.go — security boundary tests


## 2026-07-16

Step 3 (Phase 2 complete): 12-block work-slip pack + registry merge + JSON editor fallback + node tests; smoke render verified all types (commit ddced15).


## 2026-07-16

Step 4 (Phase 3 complete): ThemeTokens proto + display/h1/h2/micro presets + swiss/brutalist/terminal themes + embedded Archivo variable font (commit 08e4b8c).


## 2026-07-16

Step 5 (Phase 4 complete): example layouts 10-14 + screenshots + physical prints (job slip, triage card) + help-entry updates (commit 99502fb). All phases done.


## 2026-07-16

Ticket closed


## 2026-07-16

Step 6: bolder defaults per user feedback — bigger slip type scale, work-theme presetOverrides, theme.padding 10x8, heavier rules/banners; screenshots refreshed, reprinted (commit ada024a).


## 2026-07-16

Step 7: margin:0 experiment exposed stateRef/setMargin bug — headless renders had been ignoring layout margin; fixed (commit 39447f6), edge-to-edge slip printed.


## 2026-07-16

Step 8: 0px default padding + blockGap token + scale up one step; decision sheet converted to brutalist mock form; reprinted (commit a4f3c6d).


## 2026-07-17

Step 9: PR #7 review fixes (content field, page-level raster, bayer/FS, printer speed, density range) + CI repair (stale embed, gosec G115, toolchain 1.26.5, TruffleHog excludes) — commits da54a8e, f282553.


## 2026-07-17

Step 10: merged upstream/main into PR branch (main had independently bumped go to 1.26.5); tidy dropped redundant toolchain directive; pushed b23b854. Lesson: pull_request checks build the merge commit.


## 2026-07-17

Step 11: PR #7 follow-up fixes — heat gap density fallback (38) + wire-format inlineTheme honored (commit 7311cff); replied to review comments.


## 2026-07-17

Step 12: docs publishing wired — release.yaml publish-docs enabled (PR #9), terraform almanach publisher (wesen/terraform#14); apply pending operator.

