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

