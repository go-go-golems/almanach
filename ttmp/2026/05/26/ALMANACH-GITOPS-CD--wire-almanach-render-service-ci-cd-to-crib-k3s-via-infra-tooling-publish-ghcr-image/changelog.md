# Changelog

## 2026-05-26

- Initial workspace created


## 2026-05-26

Ticket created. Design doc written (35KB, 14 sections). 6 tasks defined. Diary Step 1-3 recorded.

### Related Files

- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/ttmp/2026/05/26/ALMANACH-GITOPS-CD--wire-almanach-render-service-ci-cd-to-crib-k3s-via-infra-tooling-publish-ghcr-image/design-doc/01-almanach-gitops-cd-analysis-design-and-implementation-guide.md — Primary design document


## 2026-05-26

Phase 1 complete. Created publish-image.yaml, gitops-targets.json, Dockerfile CMD. Found and fixed template resolution bug in HTTP API renderer. Commits: b295e45, c595005.

### Related Files

- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/.github/workflows/publish-image.yaml — CI/CD workflow using infra-tooling reusable workflow
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/Dockerfile — Added default CMD serve
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/deploy/gitops-targets.json — GitOps target config for crib-k3s
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/internal/app/renderer.go — Fixed template resolution in HTTP API path


## 2026-05-26

Phase 3 partial complete. Moved Almanach Vault GitHub Actions role/policy into /home/manuel/code/wesen/terraform and applied it. Terraform commit af3a5c0. Remaining blocker: store a crib-k3s-capable GitHub PAT in Vault.

### Related Files

- /home/manuel/code/wesen/terraform/vault/github-actions/envs/k3s/main.tf — Added almanach-render-service GitOps PR JWT role/policy and made repository_owner explicit
- /home/manuel/code/wesen/terraform/vault/github-actions/envs/k3s/outputs.tf — Exposes repository_owner in gitops_pr_roles output
- /home/manuel/workspaces/2026-05-26/fix-almanach-templates/almanach/ttmp/2026/05/26/ALMANACH-GITOPS-CD--wire-almanach-render-service-ci-cd-to-crib-k3s-via-infra-tooling-publish-ghcr-image/reference/01-implementation-diary.md — Recorded Terraform takeover and remaining PAT blocker

