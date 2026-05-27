---
title: "Implementation Diary"
doc_type: reference
intent: long-term
status: active
topics:
  - gitops
  - argocd
  - ci-cd
  - docker
  - github-actions
  - almanach
ticket: ALMANACH-GITOPS-CD
---

# Diary

## Goal

Capture the investigation and design process for wiring almanach-render-service CI/CD to crib-k3s via the infra-tooling publish-ghcr-image reusable workflow.

## Step 1: Investigation — Understand the Existing Pipeline Pattern

### Prompt Context

**User prompt (verbatim):** "does the deploy job build open a PR against the crib k3s argcocd? ~/code/wesen/crib-k3s/ and .almanach/.github"

**Assistant interpretation:** Does the almanach repository have a CI/CD pipeline that builds Docker images and opens deployment PRs against the crib-k3s ArgoCD GitOps repo?

**Inferred user intent:** Understand if automated deployment exists, and if not, create one.

### What I did

- Read all 6 workflow files in `go-go-golems/almanach/.github/workflows/`
- Read `release.yaml` — GoReleaser builds binaries only, no Docker
- Read `push.yml` — just `go test ./...`
- Read the deployment manifest in `wesen/crib-k3s/gitops/kustomize/almanach/deployment.yaml` — uses `:latest` with `imagePullPolicy: Always`
- Read the ArgoCD Application at `gitops/applications/almanach.yaml` — auto-sync + self-heal

### What worked

- Quickly identified the gap: no Docker image build, no GHCR push, no GitOps PR automation
- ArgoCD setup is already correct — no changes needed to the Application

### What didn't work

- N/A — investigation only

### What I learned

- The GoReleaser pipeline handles binary releases (GitHub Releases, Homebrew, fury.io) but not container images
- The `:latest` tag in the deployment has never been updated by any CI process
- ArgoCD is configured correctly but has nothing to sync to since the manifest never changes

### What warrants a second pair of eyes

- The `imagePullPolicy: Always` with `:latest` is a risky combination — if someone manually pushed a broken `:latest`, ArgoCD wouldn't catch it because the manifest hasn't changed

### What should be done in the future

- After wiring the pipeline, consider deprecating the `:latest` tag entirely to enforce immutable deployments

## Step 2: Investigation — Study the Reference Implementation

### Prompt Context

**User prompt (verbatim):** "Yes, look at how it's done in /home/manuel/code/wesen/2026-03-27--hetzner-k3s and say, /home/manuel/code/wesen/2026-05-03--goja-hosting-site/.github or so."

**Inferred user intent:** Study the goja-hosting-site pattern for how it builds Docker images and opens GitOps PRs, then apply the same pattern to almanach.

### What I did

- Read `2026-05-03--goja-hosting-site/.github/workflows/publish-image.yaml` — the caller workflow
- Read `2026-05-03--goja-hosting-site/deploy/gitops-targets.json` — the target config
- Read `2026-05-03--goja-hosting-site/Dockerfile` — the Docker build
- Read the full infra-tooling reusable workflow at `go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml` (190 lines)
- Read the full open-gitops-pr action source at `go-go-golems/infra-tooling/actions/open-gitops-pr/src/gitops_pr_action/open_gitops_pr.py` (440 lines)
- Read `go-go-golems/infra-tooling/actions/open-gitops-pr/action.yml`
- Examined the goja-kanban deployment in `wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-kanban/deployment.yaml` to see a patched manifest (`sha-b52aecc` tag)
- Read all 6 kustomize files in `wesen/crib-k3s/gitops/kustomize/almanach/`

### What worked

- The goja-hosting-site pattern is clean and well-structured — a single caller workflow with ~30 lines delegates everything to the reusable workflow
- The open-gitops-pr action uses text-based YAML patching (not a YAML library), which is robust for simple container image bumps
- Vault integration via `hashicorp/vault-action` with JWT OIDC is already set up for other repos

### What I learned

- The reusable workflow does everything: Go setup, tests, Docker buildx, GHCR push, GitOps PR
- The caller only needs to provide: Dockerfile path, test command, gitops-targets.json path, and Vault credentials
- The `open-gitops-pr` action creates a branch named `automation/<app>-<target>-sha-<hash>` and opens a PR with a descriptive title
- The action validates `gitops-targets.json` strictly — missing keys, empty values, duplicate names are all errors
- The `patch_strategy: container-image` (default) finds the container by name and replaces the `image:` line
- Image tags are always `sha-<7-char-short-hash>` — immutable and traceable
- The `platforms` parameter defaults to `linux/amd64`, which matches the k3s cluster

### What was tricky to build

- The Vault configuration requires admin access to create the role and secret — this is a prerequisite that cannot be done from the almanach repository alone
- The GHCR image name (`almanach-render-service`) doesn't match the repo name (`almanach`), so the `image_name` override is needed

### What should be done in the future

- After first successful push, verify GHCR package visibility (may default to private)
- Consider multi-arch builds (arm64) if ARM nodes are added to the cluster

## Step 3: Design — Write the Analysis and Implementation Guide

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to have gitops with almanach to crib-k3s. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and then upload to remarkable. Keep a diary as you work"

**Inferred user intent:** Create a comprehensive, intern-friendly design document that covers the entire system, then deliver it via docmgr ticket and reMarkable.

### What I did

- Created docmgr ticket `ALMANACH-GITOPS-CD`
- Created design doc (35KB, 14 sections) covering:
  - Executive summary
  - Problem statement and scope
  - Full system architecture with ASCII diagrams
  - Reusable workflow internals
  - gitops-targets.json contract
  - open-gitops-pr patching algorithm
  - Vault authentication flow
  - ArgoCD sync behavior
  - Current-state analysis (Dockerfile, deployment, GHCR naming)
  - Reference implementation comparison
  - Gap analysis table
  - Proposed solution with complete pseudocode for all files
  - End-to-end flow diagrams (normal deploy, PR dry-run, rollback)
  - 5-phase implementation plan
  - Risks and mitigations
  - Alternatives considered
  - Open questions
  - Key file reference tables
- Created 6 tasks
- Created this diary

### What worked

- The ASCII architecture diagram clearly shows the three-repository flow
- The end-to-end flow pseudocode is detailed enough to trace every step
- The file reference tables provide quick navigation to all relevant files
- The comparison with the goja-hosting-site reference implementation grounds the design in concrete examples

### What didn't work

- N/A — design phase

### What warrants a second pair of eyes

- The `image_name` override — should we use `ghcr.io/go-go-golems/almanach` (shorter, matches repo) or `ghcr.io/go-go-golems/almanach-render-service` (matches existing deployment)?
- The initial placeholder tag `sha-0000000` — is this acceptable, or should we push a real initial image first?

### What should be done in the future

- Upload to reMarkable (DONE)
- Configure Vault (prerequisite for GitOps PR automation)
- Implement Phase 1 (almanach repo changes)

## Step 4: Implementation — Phase 1 (CI/CD Workflow + Bug Fix)

**Commit:** `b295e45` — "feat: add publish-image CI/CD workflow, gitops targets, Dockerfile CMD"
**Commit:** `c595005` — "fix: add template resolution to HTTP API render path"

### Prompt Context

**User prompt (verbatim):** "we're taking over for your little brother. they might have made some things we need to correct, i'm not sure. Anyway, add phases and tasks to the ticket, and then work on them one by one, committing at appropriate intervals, keeping a detailed diary as you go"

**Inferred user intent:** Review the existing work for issues, implement the GitOps CD pipeline phases, commit incrementally, keep diary.

### What I did

1. Reviewed all existing implementation files from the "little brother's" work:
   - `template.go`, `template_test.go` (16 tests) — solid
   - `data_context.go`, `data_context_test.go` (9 tests) — solid
   - `layout_bundle.go` — correctly threads DataContext
   - `render_oneshot.go` — correctly resolves templates for CLI path
   - `layout.go` — scaffold is clean
   - `renderer.go` — **found a bug** (see below)
   - `cmd_render.go`, `cmd_print.go`, `cmd_print_remote.go`, `cmd_inspect.go` — all wired correctly

2. **Found and fixed a bug**: The HTTP server's `layoutJSONFromReader()` in `renderer.go` was passing raw JSON straight to Chrome without template resolution. The CLI path went through `layoutJSONFromObjectOrDefault()` which resolves templates, but the server path bypassed it entirely. Fixed by making `layoutJSONFromReader` parse the JSON, extract the `data` key for template context, and call `layoutJSONFromObjectOrDefault`.

3. Created Phase 1 implementation:
   - `.github/workflows/publish-image.yaml` — CI/CD workflow using infra-tooling reusable workflow
   - `deploy/gitops-targets.json` — single target pointing at `wesen/crib-k3s` deployment.yaml container `render`
   - Updated `Dockerfile` — added `CMD ["serve"]` for easier local testing

4. Verified Dockerfile builds locally: `docker build -t almanach-test-build .` → Entrypoint `[almanach-render-service]` + Cmd `[serve]` ✓
5. Verified server template resolution: POST to `/api/render` with `data` key + `{{title}}` expressions → 384×1082 rendered page ✓

### What worked

- The infra-tooling reusable workflow pattern from goja-hosting-site was directly copy-pasteable with minimal adaptation (path filters, image_name override, vault_role)
- Docker build cache was warm from previous builds, so verification was fast
- The server template fix was a small, targeted change — parse + delegate to existing function

### What didn't work

- Initially tried to update the crib-k3s deployment.yaml to use `sha-placeholder` as the image tag. Realized this would break the running pod since no image with that tag exists yet. The correct approach is to keep `:latest` until the first pipeline run creates the image and opens a PR to switch to immutable tags.

### What was tricky to build

- The `layoutJSONFromReader` function needed to handle both the case where the server receives a raw layout JSON (no wrapping) and the wrapped format (`{data: {...}, layout: {...}}`). The existing `layoutJSONFromObjectOrDefault` already handles both, so delegating to it was the right call.
- Needed to add `gopkg.in/yaml.v3` import to `renderer.go` for the YAML fallback parse.

### What warrants a second pair of eyes

- The `layoutJSONFromRaw` method tries JSON first, then YAML. If both fail, it returns the first error. This matches the CLI behavior in `layout_bundle.go`.
- The crib-k3s deployment still uses `imagePullPolicy: Always` — this is fine for now since it references `:latest`, but should change to `IfNotPresent` when the pipeline switches it to immutable tags.

### What should be done in the future

- Phase 2: Update crib-k3s deployment after first pipeline run
- Phase 3: Vault configuration (admin prerequisite)
- Phase 4: End-to-end validation

## Step 5: Terraform Takeover — Move Vault Role Setup into wesen/terraform

This step corrected the previous ad-hoc Vault work. I had manually written the Almanach GitOps PR Vault policy with `vault policy write`; the user clarified that Vault setup must live in `/home/manuel/code/wesen/terraform`. I moved the role/policy definition into the Terraform-managed `vault/github-actions/envs/k3s` environment and applied it there.

The important design correction was repository ownership. Existing GitOps PR roles in Terraform assumed all caller repositories are owned by `wesen`, but Almanach lives at `go-go-golems/almanach`. I made `repository_owner` explicit per role so existing roles remain bound to `wesen/*`, while `almanach-render-service-gitops-pr` is correctly bound to `go-go-golems/almanach`.

### Prompt Context

**User prompt (verbatim):** "we're taking over for our little brothers. We want the vault setup to be done in ~/code/wesen/terraform . 

Examine what was done so far, and let's fix things up"

**Assistant interpretation:** Stop making manual Vault changes; inspect the existing Terraform-managed Vault GitHub Actions setup and move the Almanach GitOps role/policy into that source of truth.

**Inferred user intent:** Keep infrastructure declarative and auditable, remove drift from hand-applied Vault changes, and ensure the Almanach workflow can authenticate with Vault using the same pattern as existing GitOps PR workflows.

**Commit (terraform):** `af3a5c0` — "vault: add almanach GitOps PR GitHub Actions role"

### What I did

- Inspected `/home/manuel/code/wesen/terraform/vault/github-actions/envs/k3s`.
- Found existing Terraform resources:
  - `local.gitops_pr_roles`
  - `vault_policy.gitops_pr`
  - `vault_jwt_auth_backend_role.gitops_pr`
- Verified the earlier manual Vault state:
  - Policy `gha-almanach-render-service-gitops-pr` existed.
  - Role `almanach-render-service-gitops-pr` did not exist.
  - Secret `kv/ci/github/almanach-render-service/gitops-pr-token` did not exist.
- Updated Terraform:
  - Added `repository_owner` to every GitOps PR role entry.
  - Added `almanach-render-service-gitops-pr` with `repository_owner = "go-go-golems"` and `repository = "go-go-golems/almanach"`.
  - Changed the JWT role bound claims from hardcoded `repository_owner = "wesen"` to `each.value.repository_owner`.
  - Included `repository_owner` in Terraform outputs.
- Ran:
  - `terraform fmt`
  - `terraform validate`
  - `AWS_PROFILE=manuel terraform plan -out /tmp/almanach-vault-github-actions.tfplan`
  - `AWS_PROFILE=manuel terraform apply -auto-approve /tmp/almanach-vault-github-actions.tfplan`
  - `AWS_PROFILE=manuel terraform plan -detailed-exitcode`
- Verified post-apply state:
  - Role exists with `repository_owner=go-go-golems`, `repository=go-go-golems/almanach`, `ref=refs/heads/main`, `event_name=push`.
  - Policy exists and only reads `kv/data/ci/github/almanach-render-service/gitops-pr-token` plus token self-management paths.
  - Final Terraform plan has **No changes**.

### What worked

- Terraform already had the correct abstraction for GitOps PR roles; it only needed an owner-aware extension.
- The plan was minimal: `2 to add, 0 to change, 0 to destroy`.
- Applying after the earlier manual policy write was safe: Terraform now manages the policy and role, and the final plan is clean.

### What didn't work

- First `terraform plan` failed because AWS backend credentials were missing:
  - `Error: No valid credential sources found`
  - Fixed by running with `AWS_PROFILE=manuel`.
- The GitHub token secret is still not present in Vault. Existing generic and goja tokens are fine-grained PATs for `wesen/2026-03-27--hetzner-k3s`, not `wesen/crib-k3s`; testing them against `wesen/crib-k3s` returned GitHub API `404 Not Found`, which indicates no access to that repository.

### What I learned

- `/home/manuel/code/wesen/terraform/vault/github-actions/envs/k3s` is the canonical source for GitHub Actions → Vault JWT roles.
- The existing GitOps PR implementation had an implicit assumption that every caller repository belongs to `wesen`.
- Almanach is the first GitOps PR caller from the `go-go-golems` org targeting the `wesen/crib-k3s` repo, so this required making repository ownership explicit.

### What was tricky to build

- Avoiding drift: a manually written policy already existed in Vault. The Terraform provider accepted the policy resource and the final plan is clean, so no manual cleanup was needed.
- The GitOps target repository (`wesen/crib-k3s`) is different from the caller repository (`go-go-golems/almanach`), but Vault JWT auth only binds the **caller workflow identity**. The PAT secret controls target-repo permissions separately.

### What warrants a second pair of eyes

- Confirm whether the PAT should be a fine-grained user PAT or a GitHub App installation token. The current infra-tooling pattern uses Vault-stored fine-grained PATs.
- Confirm the PAT should grant only `contents: write`, `pull_requests: write`, and `metadata: read` on `wesen/crib-k3s`.

### What should be done in the future

- Create/store a fine-grained GitHub PAT for `wesen/crib-k3s` at `kv/ci/github/almanach-render-service/gitops-pr-token` with field `token`. (DONE by copying the existing Hetzner GitOps PR token.)
- Re-run the publish workflow after the PAT secret exists.
- Let the workflow open the first crib-k3s PR that moves the deployment from `:latest` to an immutable `sha-<hash>` tag.

## Step 6: Vault Secret — Copy Existing Hetzner GitOps PR Token

The Vault role and policy now exist in Terraform, but the policy only grants read access to a secret path. The workflow also needs an actual GitHub token stored at that path. The user clarified that we should copy the same token used for the Hetzner GitOps PR flow.

I copied the existing token from `kv/ci/github/gitops-pr-token` into the Almanach-specific path `kv/ci/github/almanach-render-service/gitops-pr-token`. This keeps the workflow configuration unchanged: the almanach workflow asks Vault for the almanach-specific path, while the secret value is the same token currently used by the existing GitOps PR automation.

### Prompt Context

**User prompt (verbatim):** "copy the same token as for hetzner-k3s"

**Assistant interpretation:** Use the already-provisioned Hetzner GitOps PR token as the Almanach GitOps PR token, rather than creating a new PAT.

**Inferred user intent:** Unblock the publish-image workflow quickly by reusing the existing deployment automation credential.

### What I did

- Read the existing token from:
  - `kv/ci/github/gitops-pr-token`
- Wrote it to:
  - `kv/ci/github/almanach-render-service/gitops-pr-token`
- Verified the destination secret exists and contains key `token`.
- Marked the ticket's PAT storage task complete.

### What worked

- Vault accepted the write and created version 1 of the Almanach-specific secret.
- Verification showed the secret contains the expected `token` key.

### What didn't work

- N/A for the copy operation.

### What I learned

- The infra-tooling workflow does not care whether the token is unique per app; it only requires that the Vault role can read a secret containing a `token` field.
- Keeping an app-specific Vault path still preserves least-privilege on the Vault side, even if the token value is reused.

### What was tricky to build

- The token value must never be printed. I used JSON extraction to pipe it from one Vault read into one Vault write and only printed the destination key list for verification.

### What warrants a second pair of eyes

- Confirm that the copied token actually has permission to push branches and open PRs against `wesen/crib-k3s`. Earlier API probing suggested the Hetzner token may be scoped to `wesen/2026-03-27--hetzner-k3s`; the workflow run will be the definitive validation.

### What should be done in the future

- Trigger the `publish-image` workflow and observe whether the GitOps PR step can access `wesen/crib-k3s`. (DONE)
- If GitHub returns 404/permission denied, replace this copied token with a crib-k3s-scoped PAT. (Not needed: PR creation succeeded.)

## Step 7: End-to-End GitOps Validation and Thermal Print

This step pushed the implementation to `main`, watched the `publish-image` workflow, fixed CI failures, merged the crib-k3s deployment PR, verified the live service, printed a small status almanach, and updated the pinned almanach-printing skill to match the new template CLI format.

The final state is that the almanach publish workflow succeeds, publishes `ghcr.io/go-go-golems/almanach:sha-*`, opens PRs against `wesen/crib-k3s`, and crib-k3s now runs an immutable almanach image. The live service answered `/health`, and the thermal printer accepted the printed mini almanach in two bitmap segments.

### Prompt Context

**User prompt (verbatim):** "print a little almanach about the whole thing, and make sure the skill is now up to date to the new template format"

**Assistant interpretation:** Produce a short printed Almanach page summarizing this deployment/template work, and correct the almanach-printing skill documentation so future agents use `--data`/`--define` correctly.

**Inferred user intent:** Validate the whole stack physically on the printer and leave the operational skill accurate for future use.

**Commits (almanach):**
- `b05965b` — "fix: publish container under repository-scoped GHCR package"
- `89bd1ba` — "fix: address CI lint and secret scanning push range"
- `321cfa6` — "fix: unblock dependency scanning on Go 1.26.3"

**Commit (crib-k3s):** `9b3d525` — merged PR #3, "Deploy almanach-render-prod using ghcr.io/go-go-golems/almanach:sha-321cfa6"

### What I did

- Pushed the almanach branch and then pushed `HEAD` to `main`.
- Watched the `publish-image` workflow.
- First workflow failed on GHCR push because `ghcr.io/go-go-golems/almanach-render-service` is an existing private package not linked to the repo:
  - `403 Forbidden` on layer push.
- Fixed workflow image name to repository-scoped package:
  - `ghcr.io/go-go-golems/almanach`
- Re-ran workflow via push; image publish succeeded and GitOps PR opened.
- Fixed CI follow-ups:
  - gofmt on `renderer.go`
  - checked env mutation errors in `template_test.go`
  - removed unnecessary nil-map check in `data_context.go`
  - removed unused `dividerBlock`
  - fixed TruffleHog push ranges from `base: main/head: HEAD` to event SHAs
  - ran dependency scanning on Go `1.26.3`
  - upgraded `golang.org/x/net` to `v0.55.0` and related `x/*` modules
  - added `#nosec G402` for the explicit `--insecure-skip-verify` development flag
- Verified latest main checks:
  - `publish-image`: success
  - `golang-pipeline`: success
  - `golangci-lint`: success
  - `Secret Scanning`: success
  - `CodeQL`: success
  - `Dependency Scanning`: success
- Merged crib-k3s PR #2, then superseded it with PR #3 after the final almanach CI/security fix.
- Merged crib-k3s PR #3 and annotated ArgoCD for a hard refresh.
- Verified crib deployment:
  - image: `ghcr.io/go-go-golems/almanach:sha-321cfa6`
  - ready replicas: `1/1`
  - remote health: `https://almanach.crib.scapegoat.dev/health` returned `ok: true`
- Noted that the ArgoCD Application remains `OutOfSync` only because `cert-manager` mutates/owns the `Certificate`; Deployment is healthy and current.
- Updated `/home/manuel/.pi/agent/skills/almanach-printing/SKILL.md`:
  - replaced invalid `-D` examples with `--define`
  - documented comma-separated inline defines in one `--define` flag
  - documented `{{$ENV_VAR:fallback}}`
  - clarified that no data context means no template resolution
  - clarified YAML/data contexts are the only content source; no invented/fetched content
  - updated Docker image reference to `ghcr.io/go-go-golems/almanach:sha-<commit>`
- Created `/tmp/almanach-gitops-mini-template.yaml` and `/tmp/almanach-gitops-mini-data.yaml`.
- Rendered preview:
  - `384x763`, `/tmp/almanach-gitops-mini.png`
- Printed remotely:
  - `./dist/almanach-render-service print-remote --layout /tmp/almanach-gitops-mini-template.yaml --data /tmp/almanach-gitops-mini-data.yaml --output yaml`
  - remote returned `ok: true`, `printed: true`, `segments: 2`, `384x783`.

### What worked

- The copied Hetzner GitOps PR token could in fact create PRs against `wesen/crib-k3s`.
- Vault OIDC auth worked from GitHub Actions.
- The repo-scoped GHCR package name solved the push authorization issue.
- The remote crib service rendered and printed the template-driven mini almanach successfully.

### What didn't work

- Initial GHCR target `ghcr.io/go-go-golems/almanach-render-service` failed with `403 Forbidden` because it is a private org-scoped package not linked to this repo.
- Initial CI after pushing main exposed lint/security issues from prior changes and old workflow assumptions.
- ArgoCD was initially stuck with `ComparisonError` because it lacked repository credentials for private `wesen/crib-k3s`; I created the `repo-crib-k3s` ArgoCD repository Secret using the Vault token, after which ArgoCD could compare and sync.
- ArgoCD still reports `OutOfSync` due the `cert-manager.io/Certificate` resource, but the Deployment itself is current and healthy.

### What I learned

- For GHCR, repository-scoped package names avoid permissions friction with `GITHUB_TOKEN`.
- The existing TruffleHog workflow was broken for direct pushes to main because `base: main` and `head: HEAD` resolve to the same commit after push.
- Dependency scanning should pin setup-go to `1.26.3` until the runner default catches up to the patched Go toolchain.
- The remote print path is an excellent full-stack validation because it exercises local template resolution, remote render+print, the crib service, and the physical printer.

### What was tricky to build

- The almanach branch was pushed directly to `main`, so every fix commit created another deployment PR. I closed superseded crib-k3s PR #1 and merged the latest PR #3.
- The generated GitOps PR only changes the image line. I manually added `imagePullPolicy: IfNotPresent` on the crib-k3s PR branch before merging, because immutable tags no longer need `Always`.
- ArgoCD repository credentials are cluster state, not part of the application manifest. The deployment was healthy only after adding the private GitHub repo secret.

### What warrants a second pair of eyes

- Decide whether the ArgoCD repo credential Secret should also be represented declaratively in crib-k3s or Terraform instead of being manually created in-cluster.
- Decide whether the `Certificate` should be removed from Git or annotated/ignored so ArgoCD stops reporting OutOfSync on a cert-manager-owned resource.
- Decide whether future GitOps PR automation should also patch `imagePullPolicy` if the old manifest still says `Always`.

### What should be done in the future

- Make ArgoCD private repo credentials declarative.
- Resolve or ignore the cert-manager `Certificate` drift.
- Consider cleaning up the old private `ghcr.io/go-go-golems/almanach-render-service` package or documenting it as obsolete.
