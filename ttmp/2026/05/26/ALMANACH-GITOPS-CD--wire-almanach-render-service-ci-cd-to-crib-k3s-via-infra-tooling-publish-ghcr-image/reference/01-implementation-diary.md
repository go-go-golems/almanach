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

- Upload to reMarkable
- Configure Vault (prerequisite for GitOps PR automation)
- Implement Phase 1 (almanach repo changes) after merging the template work
