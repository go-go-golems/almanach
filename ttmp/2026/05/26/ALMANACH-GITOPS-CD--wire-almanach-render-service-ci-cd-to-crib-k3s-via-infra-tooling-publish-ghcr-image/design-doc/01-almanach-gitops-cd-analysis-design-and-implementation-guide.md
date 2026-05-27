---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/2026-05-03--goja-hosting-site/.github/workflows/publish-image.yaml
      Note: Reference implementation — the pattern to follow
    - Path: ../../../../../../../../../../code/wesen/2026-05-03--goja-hosting-site/deploy/gitops-targets.json
      Note: Reference gitops-targets.json
    - Path: ../../../../../../../../../../code/wesen/crib-k3s/gitops/applications/almanach.yaml
      Note: ArgoCD Application — auto-sync
    - Path: ../../../../../../../../../../code/wesen/crib-k3s/gitops/kustomize/almanach/deployment.yaml
      Note: Current deployment with :latest tag
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/almanach/.github/workflows/release.yaml
      Note: Current release workflow — builds binaries only
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/almanach/Dockerfile
      Note: Existing Dockerfile that needs CMD addition
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml
      Note: Reusable workflow — the core CI/CD engine
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/actions/open-gitops-pr/src/gitops_pr_action/open_gitops_pr.py
      Note: GitOps PR action — manifest patching + PR logic
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Almanach GitOps CD — Analysis, Design, and Implementation Guide

## 1. Executive Summary

The almanach-render-service has no continuous deployment pipeline. When code is pushed to `main`, no Docker image is built, no container registry is updated, and the running k3s pod continues serving the old binary indefinitely. The current deployment in `wesen/crib-k3s` uses `ghcr.io/go-go-golems/almanach-render-service:latest` with `imagePullPolicy: Always`, but `:latest` is never pushed by any CI job — it is either stale from a manual push or simply absent.

This document describes how to wire almanach into the existing infra-tooling GitOps CD pipeline, following the same pattern already used by `2026-05-03--goja-hosting-site`. The change involves three new files in the almanach repository, one small update to the crib-k3s deployment manifest, and optional Vault configuration for the PR token.

The result: every push to `main` builds a Docker image, pushes it to GHCR with an immutable `sha-<short>` tag, and opens a pull request against `wesen/crib-k3s` to bump the deployment's container image reference. ArgoCD auto-syncs the merged PR, rolling out the new version.

---

## 2. Problem Statement and Scope

### Current state

The almanach repository at `go-go-golems/almanach` has five GitHub Actions workflows:

| Workflow | Trigger | What it does |
|---|---|---|
| `push.yml` | push/PR to main | `go test ./...` |
| `lint.yml` | push/PR to main | golangci-lint |
| `release.yaml` | tag push | GoReleaser builds binaries (no Docker) |
| `codeql-analysis.yml` | push/PR/schedule | CodeQL security scan |
| `dependency-scanning.yml` | push/PR/schedule | govulncheck + gosec |
| `secret-scanning.yml` | push/PR | TruffleHog |

None of these workflows build or push a Docker image. The GoReleaser pipeline produces binaries for Linux/macOS (amd64/arm64), publishes them as GitHub Releases, pushes to a Homebrew tap, and publishes deb/rpm packages to fury.io — but it does not produce container images.

Meanwhile, the crib-k3s cluster has a complete ArgoCD Application watching `gitops/kustomize/almanach/` in `wesen/crib-k3s`. The deployment manifest at `gitops/kustomize/almanach/deployment.yaml` references `ghcr.io/go-go-golems/almanach-render-service:latest` with `imagePullPolicy: Always`. This means:

1. **No image is ever pushed** to that GHCR repository by any automated process.
2. **`:latest` is a floating tag** — there is no immutability guarantee, no audit trail of what version is running.
3. **No PR is opened** against crib-k3s when a new image is built, so there is no human review gate before deployment.
4. **ArgoCD will not restart the pod** because the manifest never changes — it always says `:latest`.

### Desired state

1. Every push to `main` builds a multi-arch Docker image and pushes it to `ghcr.io/go-go-golems/almanach-render-service` with an immutable `sha-<7-char-short-hash>` tag.
2. A pull request is automatically opened against `wesen/crib-k3s` to update the deployment manifest's `image:` field to the new immutable tag.
3. After the PR is merged, ArgoCD detects the manifest change and rolls out the new pod.
4. On pull requests, the same workflow builds and tests the image but does **not** push it (dry-run validation only).

### Scope

- **In scope**: `publish-image.yaml` workflow, `deploy/gitops-targets.json`, Dockerfile updates, deployment manifest update, Vault configuration.
- **Out of scope**: GoReleaser changes (it continues handling binary releases), ArgoCD application changes (the `applications/almanach.yaml` already exists and is correct), monitoring/alerting.

---

## 3. System Architecture

### 3.1 Repositories involved

Three repositories participate in the pipeline:

```
┌─────────────────────────────────┐     push to main      ┌──────────────────────────────┐
│  go-go-golems/almanach          │ ───────────────────── │  GitHub Actions runner        │
│                                 │                       │                              │
│  .github/workflows/             │                       │  1. Run go test ./...         │
│    push.yml (existing)          │                       │  2. Build Docker image        │
│    publish-image.yaml (NEW)     │                       │  3. Push to ghcr.io           │
│                                 │                       │  4. Open PR to crib-k3s       │
│  deploy/                        │                       │                              │
│    gitops-targets.json (NEW)    │                       └──────────┬───────────────────┘
│                                 │                                  │
│  Dockerfile (exists, may need   │                                  │ push image
│            update)              │                                  ▼
│                                 │                       ┌──────────────────────────────┐
└─────────────────────────────────┘                       │  ghcr.io                     │
                                                          │                              │
                                                          │  go-go-golems/               │
                                                          │    almanach-render-service:  │
                                                          │      sha-a1b2c3d             │
                                                          │      main                    │
                                                          │      latest                  │
                                                          └──────────┬───────────────────┘
                                                                     │ PR #42
                                                                     ▼
┌─────────────────────────────────┐     ArgoCD sync       ┌──────────────────────────────┐
│  wesen/crib-k3s                 │ ◀──────────────────── │  ArgoCD (in-cluster)         │
│                                 │                       │                              │
│  gitops/kustomize/almanach/     │                       │  watches main branch of      │
│    deployment.yaml              │                       │  wesen/crib-k3s at path      │
│      image: sha-a1b2c3d (bump)  │                       │  gitops/kustomize/almanach/   │
│    service.yaml                 │                       │                              │
│    ingress.yaml                 │                       │  auto-sync + self-heal        │
│    kustomization.yaml           │                       └──────────────────────────────┘
│    namespace.yaml               │
│    certificate.yaml             │
│                                 │
│  gitops/applications/           │
│    almanach.yaml (unchanged)    │
└─────────────────────────────────┘
```

### 3.2 The infra-tooling reusable workflow

The pipeline is powered by a reusable workflow in `go-go-golems/infra-tooling`:

```
publish-image.yaml (caller)
       │
       │  uses: go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml@main
       │
       ▼
publish-ghcr-image.yml (reusable)
       │
       ├── Job: publish
       │     ├── Checkout caller repo
       │     ├── Set up Go (from go.mod)
       │     ├── Run tests (go test ./...)
       │     ├── Compute image coordinates
       │     │     image_name = ghcr.io/$GITHUB_REPOSITORY  (or override)
       │     │     image_tag  = sha-${GITHUB_SHA::7}
       │     │     image_ref  = image_name:image_tag
       │     ├── Docker buildx build + push
       │     │     tags: sha-<hash>, main (if default branch), latest (if default branch)
       │     │     cache: gha (GitHub Actions cache)
       │     └── Outputs: image_ref, image_tag, image_name
       │
       └── Job: gitops-pr  (conditional: only on main push)
             ├── Checkout caller repo
             ├── Checkout infra-tooling repo
             ├── Validate credentials (Vault or legacy secret)
             ├── Retrieve token from Vault (hashicorp/vault-action)
             └── Run open-gitops-pr action
                   │
                   ├── Read deploy/gitops-targets.json
                   ├── For each target:
                   │     ├── Clone gitops repo (wesen/crib-k3s)
                   │     ├── Patch deployment.yaml image field
                   │     ├── Git commit + push branch
                   │     └── Open PR via gh CLI
                   └── Outputs: changed, changed_targets, branch_names, pr_numbers
```

### 3.3 The gitops-targets.json contract

The `deploy/gitops-targets.json` file tells the `open-gitops-pr` action where and how to patch. Here is the schema:

```json
{
  "targets": [
    {
      "name": "string — unique identifier for this target",
      "gitops_repo": "string — GitHub org/repo of the GitOps repository",
      "gitops_branch": "string — branch to target (usually 'main')",
      "manifest_path": "string — path within the GitOps repo to the Deployment manifest",
      "container_name": "string — name of the container in the Deployment to patch",
      "patch_strategy": "string — optional, defaults to 'container-image'"
    }
  ]
}
```

**Required keys**: `name`, `gitops_repo`, `gitops_branch`, `manifest_path`, `container_name`.

**Optional keys**: `patch_strategy` (defaults to `container-image`; also supports `static-publisher-job`).

The action validates this file strictly: missing keys, empty values, and duplicate names are all errors.

### 3.4 How the open-gitops-pr action patches manifests

The action parses the YAML deployment manifest line-by-line (not via a YAML library — it uses text-based indentation tracking). For the `container-image` strategy:

1. It scans for a `containers:` block.
2. Within that block, it looks for a `- name: <container_name>` entry.
3. It replaces the `image:` field immediately below that container entry.
4. If the image is already the target value, it skips the target (no change, no PR).

This means the deployment YAML must follow a standard structure:

```yaml
spec:
  template:
    spec:
      containers:
        - name: render              # ← must match container_name in gitops-targets.json
          image: ghcr.io/go-go-golems/almanach-render-service:sha-a1b2c3d  # ← this gets patched
```

### 3.5 Vault authentication

The reusable workflow supports two token sources for authenticating to the GitOps repo:

1. **Vault** (preferred): Uses `hashicorp/vault-action` with JWT OIDC authentication. The GitHub Actions runner gets an OIDC token, presents it to Vault at `github-actions` auth path, and retrieves a GitHub PAT from `kv/data/ci/github/<repo>/gitops-pr-token`. This PAT needs `repo` scope on the `wesen/crib-k3s` repository.

2. **Legacy secret**: Passes `GITOPS_PR_TOKEN` as a repository or organization secret.

For almanach, we will use Vault. This requires:
- A Vault role: `almanach-render-service-gitops-pr`
- A Vault secret at: `kv/data/ci/github/almanach-render-service/gitops-pr-token`
- The secret value must be a GitHub PAT with push and PR permissions on `wesen/crib-k3s`

### 3.6 ArgoCD sync behavior

The `applications/almanach.yaml` in crib-k3s is already configured with:

```yaml
syncPolicy:
  automated:
    prune: true
    selfHeal: true
  syncOptions:
    - CreateNamespace=true
    - ServerSideApply=true
```

This means:
- **selfHeal: true**: If the live cluster drifts from the desired state, ArgoCD reconciles automatically.
- **prune: true**: Resources removed from Git are removed from the cluster.
- ArgoCD polls the GitOps repo every ~3 minutes (or receives a webhook push).

The flow after merge is:

```
PR merged to main in wesen/crib-k3s
  → ArgoCD detects change (poll/webhook)
  → ArgoCD applies updated Deployment manifest
  → Kubernetes rolling update: new pod with new image
  → Old pod terminated
```

No ArgoCD changes are needed.

---

## 4. Current-State Analysis

### 4.1 Almanach Dockerfile

The existing Dockerfile at the repository root:

```dockerfile
FROM golang:1.26-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -tags=embed -o /almanach-render-service ./cmd/almanach-render-service

FROM chromedp/headless-shell:latest

COPY --from=builder /almanach-render-service /usr/local/bin/almanach-render-service
COPY --from=builder /build/web/ /opt/almanach/web/

ENV ALMANACH_PORT=8199 \
    ALMANACH_WEB_DIR=/opt/almanach/web/dist \
    ALMANACH_PRINTER_IP= \
    ALMANACH_CHROME_PATH=/headless-shell/headless-shell \
    ALMANACH_DEFAULT_THEME=minimal \
    ALMANACH_DEFAULT_FEED=3 \
    ALMANACH_FONT_SCALE=1.6 \
    ALMANACH_PAPER_WIDTH=384
EXPOSE 8199
ENTRYPOINT ["almanach-render-service"]
```

**Issues to address:**

1. **No default CMD**: The deployment passes `args: [serve]` but the Dockerfile doesn't set a default CMD. This is fine — the deployment overrides it — but adding `CMD ["serve"]` would make local testing easier.

2. **Single-arch build**: The Dockerfile uses `CGO_ENABLED=1` with no cross-compilation. The infra-tooling workflow supports `platforms: linux/amd64` by default. Since the k3s cluster runs amd64 nodes, this is fine. Multi-arch (amd64+arm64) would require cross-compilation toolchains in the builder stage.

3. **The web/ directory COPY**: This copies the raw React source. With the `embed` build tag, the Go binary embeds the web assets. The COPY of `/build/web/` is redundant but harmless — it allows overriding the embedded assets at runtime via `ALMANACH_WEB_DIR`.

### 4.2 Deployment manifest (crib-k3s)

The deployment at `gitops/kustomize/almanach/deployment.yaml` currently has:

```yaml
containers:
  - name: render
    image: ghcr.io/go-go-golems/almanach-render-service:latest
    imagePullPolicy: Always
```

**Changes needed:**

1. Replace `:latest` with an initial immutable tag (e.g., `sha-<current-hash>`).
2. Change `imagePullPolicy` from `Always` to `IfNotPresent` — immutable tags never change, so re-pulling is unnecessary.

### 4.3 GHCR image name

The almanach repo is `go-go-golems/almanach` on GitHub. The default image name computed by the reusable workflow is:

```
ghcr.io/go-go-golems/almanach
```

But the deployment currently references:

```
ghcr.io/go-go-golems/almanach-render-service
```

Two options:

- **Option A**: Override `image_name` in the workflow to match the existing `almanach-render-service` name.
- **Option B**: Change the deployment to use `ghcr.io/go-go-golems/almanach` (shorter, matches repo name).

**Recommendation**: Option A — override to `almanach-render-service` for backward compatibility. The deployment already uses this name, and existing documentation references it.

### 4.4 Reference implementation: goja-hosting-site

The goja-hosting-site repo provides the canonical example. Here is exactly what it does:

**`.github/workflows/publish-image.yaml`**:

```yaml
name: publish-image

on:
  pull_request:
    paths:
      - .github/workflows/publish-image.yaml
      - Dockerfile
      - go.mod
      - go.sum
      - cmd/**
      - pkg/**
      - sites/**
      - deploy/**
  push:
    branches:
      - main
    paths:
      - .github/workflows/publish-image.yaml
      - Dockerfile
      - go.mod
      - go.sum
      - cmd/**
      - pkg/**
      - sites/**
      - deploy/**
  workflow_dispatch:

permissions:
  contents: read
  packages: write
  pull-requests: write
  id-token: write

concurrency:
  group: publish-image-${{ github.ref }}
  cancel-in-progress: true

jobs:
  release:
    uses: go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml@main
    secrets: inherit
    with:
      dockerfile: ./Dockerfile
      build_context: .
      test_command: |
        go test ./...
      go_version_file: go.mod
      gitops_target_config: deploy/gitops-targets.json
      push_image: ${{ github.event_name != 'pull_request' }}
      open_gitops_pr: ${{ github.event_name != 'pull_request' && github.ref == 'refs/heads/main' }}
      gitops_pr_token_source: vault
      vault_role: goja-hosting-site-gitops-pr
      vault_secret_path: kv/data/ci/github/goja-hosting-site/gitops-pr-token
      tooling_repository: go-go-golems/infra-tooling
      tooling_ref: main
```

**`deploy/gitops-targets.json`**:

```json
{
  "targets": [
    {
      "name": "goja-kanban-prod",
      "gitops_repo": "wesen/2026-03-27--hetzner-k3s",
      "gitops_branch": "main",
      "manifest_path": "gitops/kustomize/goja-kanban/deployment.yaml",
      "container_name": "goja-kanban"
    }
  ]
}
```

Note: goja-hosting-site targets `wesen/2026-03-27--hetzner-k3s` (the hetzner cluster). Almanach is deployed on the **crib** cluster (`wesen/crib-k3s`).

---

## 5. Gap Analysis

| Area | Current | Needed | Gap |
|---|---|---|---|
| Docker image build | None | Build on push to main | New workflow |
| Docker image push | None | Push to GHCR with immutable tag | New workflow |
| GitOps PR | None | Auto-open PR against crib-k3s | New `deploy/gitops-targets.json` |
| Deployment image tag | `:latest` (floating) | `sha-<hash>` (immutable) | Update deployment.yaml |
| Deployment pull policy | `Always` | `IfNotPresent` | Update deployment.yaml |
| Vault configuration | None | Role + secret for PR token | Vault admin step |
| GHCR image name | `almanach-render-service` | Match via `image_name` override | Workflow config |
| Dockerfile CMD | None | `CMD ["serve"]` | Optional improvement |

---

## 6. Proposed Solution

### 6.1 New file: `.github/workflows/publish-image.yaml`

This is the main CI/CD workflow. It follows the goja-hosting-site pattern exactly.

**Trigger configuration:**

- **On pull_request**: Build and test the image, but do not push. Validates that the Dockerfile compiles and tests pass. Path filters limit runs to relevant changes.
- **On push to main**: Build, test, push to GHCR, open GitOps PR. This is the full deployment pipeline.
- **On workflow_dispatch**: Manual trigger for ad-hoc builds.

**Key parameters:**

| Parameter | Value | Reason |
|---|---|---|
| `dockerfile` | `./Dockerfile` | Standard location |
| `build_context` | `.` | Root of repository |
| `test_command` | `go test ./...` | Same as existing push.yml |
| `go_version_file` | `go.mod` | Standard |
| `gitops_target_config` | `deploy/gitops-targets.json` | New file |
| `push_image` | `${{ github.event_name != 'pull_request' }}` | PR = dry-run |
| `open_gitops_pr` | `${{ github.event_name != 'pull_request' && github.ref == 'refs/heads/main' }}` | Only on main |
| `gitops_pr_token_source` | `vault` | Preferred auth |
| `vault_role` | `almanach-render-service-gitops-pr` | New Vault role |
| `vault_secret_path` | `kv/data/ci/github/almanach-render-service/gitops-pr-token` | New Vault secret |
| `image_name` | `ghcr.io/go-go-golems/almanach-render-service` | Match existing deployment |
| `platforms` | `linux/amd64` | k3s cluster is amd64 |

**Concurrency**: `publish-image-${{ github.ref }}` with `cancel-in-progress: true` — ensures only one build per branch at a time.

**Path filters**: Only trigger on changes to Go source, Dockerfile, workflow file, and deploy config. Avoids wasted runs on doc-only changes.

### 6.2 New file: `deploy/gitops-targets.json`

```json
{
  "targets": [
    {
      "name": "almanach-render-prod",
      "gitops_repo": "wesen/crib-k3s",
      "gitops_branch": "main",
      "manifest_path": "gitops/kustomize/almanach/deployment.yaml",
      "container_name": "render"
    }
  ]
}
```

**Key fields explained:**

- `name`: `"almanach-render-prod"` — human-readable identifier, used in branch names and PR titles.
- `gitops_repo`: `"wesen/crib-k3s"` — the crib cluster's GitOps repository.
- `manifest_path`: `"gitops/kustomize/almanach/deployment.yaml"` — path within crib-k3s to the Deployment manifest.
- `container_name`: `"render"` — must match the `- name: render` in the deployment's `containers:` list.

### 6.3 Update: `Dockerfile` (add CMD)

Add a default command to make the image self-documenting and easy to test locally:

```dockerfile
CMD ["serve"]
```

This goes after the `ENTRYPOINT` line. The deployment's `args: [serve]` still works (it overrides CMD).

### 6.4 Update: crib-k3s deployment.yaml

Two changes:

1. Replace `image: ghcr.io/go-go-golems/almanach-render-service:latest` with an initial immutable tag. The first push from the new pipeline will create a PR to update this.
2. Change `imagePullPolicy: Always` to `imagePullPolicy: IfNotPresent`.

```yaml
# Before:
image: ghcr.io/go-go-golems/almanach-render-service:latest
imagePullPolicy: Always

# After:
image: ghcr.io/go-go-golems/almanach-render-service:sha-0000000  # placeholder, first pipeline run will update
imagePullPolicy: IfNotPresent
```

**Note**: The initial `sha-0000000` tag is a placeholder. After the first successful pipeline run, a PR will be opened to bump it to the real SHA. Alternatively, you can manually push an initial image and set the correct SHA.

### 6.5 Vault configuration

This is an administrative step done outside the almanach repository. It requires access to the Vault instance at `vault.yolo.scapegoat.dev`.

**Steps:**

1. Create a GitHub PAT with `repo` scope (or narrower: push + PR on `wesen/crib-k3s`).
2. Store the PAT in Vault at `kv/data/ci/github/almanach-render-service/gitops-pr-token` with field `token`.
3. Create a Vault role `almanach-render-service-gitops-pr` in the `github-actions` auth method that allows the `go-go-golems/almanach` repository to assume it.
4. Configure the bound audience to `https://vault.yolo.scapegoat.dev`.

**Vault CLI pseudocode:**

```bash
# 1. Write the secret
vault kv put kv/ci/github/almanach-render-service/gitops-pr-token \
  token="ghp_<personal-access-token>"

# 2. Create the role (adapt from existing goja-hosting-site role)
vault write auth/github-actions/role/almanach-render-service-gitops-pr \
  bound_repository_ids="<almanach-repo-id>" \
  bound_claims_sub="repo:go-go-golems/almanach" \
  user_claim="repository" \
  policies="ci-gitops-pr" \
  ttl="5m"
```

To find the repository ID:

```bash
gh repo view go-go-golems/almanach --json databaseId --jq '.databaseId'
```

---

## 7. End-to-End Flow

### 7.1 Normal deployment (push to main)

```
Developer pushes commit to main
  │
  ▼
GitHub Actions triggers publish-image.yaml
  │
  ├── push.yml also triggers (parallel: runs go test)
  │
  ▼
Reusable workflow: publish job
  │
  ├── Checkout go-go-golems/almanach
  ├── Set up Go 1.26
  ├── Run: go test ./...               ← gate: tests must pass
  ├── Compute: image_name=ghcr.io/go-go-golems/almanach-render-service
  │            image_tag=sha-a1b2c3d
  │            image_ref=ghcr.io/go-go-golems/almanach-render-service:sha-a1b2c3d
  ├── Docker buildx build (linux/amd64)
  │     ├── Tags: sha-a1b2c3d, main, latest
  │     ├── Cache: GHA cache (speeds up subsequent builds)
  │     └── Push: yes (main branch)
  └── Output: image_ref, image_tag, image_name
  │
  ▼
Reusable workflow: gitops-pr job (conditional: main branch only)
  │
  ├── Checkout go-go-golems/almanach (for gitops-targets.json)
  ├── Checkout go-go-golems/infra-tooling
  ├── Validate Vault credentials
  ├── Retrieve PR token from Vault
  └── Run open-gitops-pr action:
        │
        ├── Read deploy/gitops-targets.json
        │     → target: almanach-render-prod
        │     → repo: wesen/crib-k3s
        │     → manifest: gitops/kustomize/almanach/deployment.yaml
        │     → container: render
        │
        ├── Clone wesen/crib-k3s (main branch)
        ├── Read gitops/kustomize/almanach/deployment.yaml
        ├── Find container "render", patch image: line
        │     old: image: ghcr.io/go-go-golems/almanach-render-service:sha-previous
        │     new: image: ghcr.io/go-go-golems/almanach-render-service:sha-a1b2c3d
        │
        ├── If image unchanged → skip (no PR)
        │
        ├── Git commit on branch: automation/almanach-render-service-almanach-render-prod-sha-a1b2c3d
        ├── Git push branch to wesen/crib-k3s
        └── Open PR via gh CLI:
              title: "Deploy almanach-render-prod using ghcr.io/go-go-golems/almanach-render-service:sha-a1b2c3d"
              body:  "Automated image bump for `almanach-render-prod`."
  │
  ▼
PR appears in wesen/crib-k3s (review + merge)
  │
  ▼
ArgoCD detects merge (poll/webhook, ~3 min)
  │
  ▼
ArgoCD applies updated Deployment
  │
  ▼
Kubernetes rolling update:
  ├── New pod pulls ghcr.io/go-go-golems/almanach-render-service:sha-a1b2c3d
  ├── Readiness probe passes (/health → 200)
  └── Old pod terminated
```

### 7.2 Pull request validation (dry-run)

```
Developer opens PR against almanach main
  │
  ▼
GitHub Actions triggers publish-image.yaml
  │
  ▼
Reusable workflow: publish job
  │
  ├── Checkout, set up Go, run tests
  ├── Compute image coordinates
  ├── Docker buildx build
  │     ├── Tags: sha-<hash>
  │     ├── Push: NO (pull_request event)
  │     └── Validates: Dockerfile compiles, binary runs
  └── gitops-pr job: SKIPPED (open_gitops_pr is false for PRs)
```

### 7.3 Rollback

To roll back to a previous version:

1. Revert the merge commit in `wesen/crib-k3s` (which restores the previous image tag).
2. Or: manually open a PR against `wesen/crib-k3s` to set the image tag back to the desired SHA.
3. ArgoCD syncs the reverted manifest and rolls back the pod.

Because every image tag is immutable (`sha-<hash>`), old images remain in GHCR and can always be referenced.

---

## 8. Implementation Plan

### Phase 1: Almanach repository changes

**Files to create/modify in `go-go-golems/almanach`:**

1. **Create `.github/workflows/publish-image.yaml`**

   Copy the goja-hosting-site template, adapt:
   - `vault_role` → `almanach-render-service-gitops-pr`
   - `vault_secret_path` → `kv/data/ci/github/almanach-render-service/gitops-pr-token`
   - Add `image_name: ghcr.io/go-go-golems/almanach-render-service`
   - Add `platforms: linux/amd64`
   - Update path filters for almanach's directory structure

2. **Create `deploy/gitops-targets.json`**

   Single target pointing at `wesen/crib-k3s` → `gitops/kustomize/almanach/deployment.yaml` → container `render`.

3. **Update `Dockerfile`**

   Add `CMD ["serve"]` after the `ENTRYPOINT` line.

### Phase 2: crib-k3s deployment update

**Files to modify in `wesen/crib-k3s`:**

1. **Update `gitops/kustomize/almanach/deployment.yaml`**

   - Change `image:` from `:latest` to `:sha-<initial-hash>` (placeholder)
   - Change `imagePullPolicy` from `Always` to `IfNotPresent`

   This should be committed directly to main (no PR automation yet).

### Phase 3: Vault configuration

**Administrative steps (one-time):**

1. Create GitHub PAT with push + PR scope on `wesen/crib-k3s`.
2. Store in Vault: `kv put kv/ci/github/almanach-render-service/gitops-pr-token token=<PAT>`.
3. Create Vault role: `almanach-render-service-gitops-pr` bound to `go-go-golems/almanach`.

### Phase 4: Validation

1. Push a test commit to `main` in almanach.
2. Verify the `publish-image` workflow runs successfully.
3. Verify the image appears in GHCR: `ghcr.io/go-go-golems/almanach-render-service:sha-<hash>`.
4. Verify a PR is opened against `wesen/crib-k3s`.
5. Merge the PR and verify ArgoCD rolls out the new pod.
6. Check `kubectl get pods -n almanach` to confirm the new image is running.

### Phase 5: GHCR package visibility

After the first image push, the GHCR package is created. By default it may be private. Ensure it is accessible to the k3s cluster:

1. Go to `github.com/go-go-golems/almanach/pkgs/container/almanach-render-service`
2. Package settings → Manage Actions access → ensure the repository is linked
3. Set visibility to "Public" or ensure the cluster has appropriate credentials

---

## 9. Pseudocode and API References

### 9.1 publish-image.yaml (complete)

```yaml
name: publish-image

on:
  pull_request:
    paths:
      - .github/workflows/publish-image.yaml
      - Dockerfile
      - go.mod
      - go.sum
      - cmd/**
      - internal/**
      - pkg/**
      - deploy/**
  push:
    branches:
      - main
    paths:
      - .github/workflows/publish-image.yaml
      - Dockerfile
      - go.mod
      - go.sum
      - cmd/**
      - internal/**
      - pkg/**
      - deploy/**
  workflow_dispatch:

permissions:
  contents: read
  packages: write
  pull-requests: write
  id-token: write

concurrency:
  group: publish-image-${{ github.ref }}
  cancel-in-progress: true

jobs:
  release:
    uses: go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml@main
    secrets: inherit
    with:
      dockerfile: ./Dockerfile
      build_context: .
      test_command: |
        go test ./...
      go_version_file: go.mod
      image_name: ghcr.io/go-go-golems/almanach-render-service
      platforms: linux/amd64
      gitops_target_config: deploy/gitops-targets.json
      push_image: ${{ github.event_name != 'pull_request' }}
      open_gitops_pr: ${{ github.event_name != 'pull_request' && github.ref == 'refs/heads/main' }}
      gitops_pr_token_source: vault
      vault_role: almanach-render-service-gitops-pr
      vault_secret_path: kv/data/ci/github/almanach-render-service/gitops-pr-token
      tooling_repository: go-go-golems/infra-tooling
      tooling_ref: main
```

### 9.2 deploy/gitops-targets.json (complete)

```json
{
  "targets": [
    {
      "name": "almanach-render-prod",
      "gitops_repo": "wesen/crib-k3s",
      "gitops_branch": "main",
      "manifest_path": "gitops/kustomize/almanach/deployment.yaml",
      "container_name": "render"
    }
  ]
}
```

### 9.3 Deployment manifest patch (before → after)

```yaml
# BEFORE (current state)
containers:
  - name: render
    image: ghcr.io/go-go-golems/almanach-render-service:latest
    imagePullPolicy: Always

# AFTER (proposed)
containers:
  - name: render
    image: ghcr.io/go-go-golems/almanach-render-service:sha-0000000
    imagePullPolicy: IfNotPresent
```

### 9.4 Dockerfile addition

```dockerfile
# Existing last lines:
EXPOSE 8199
ENTRYPOINT ["almanach-render-service"]

# ADD:
CMD ["serve"]
```

---

## 10. Key File References

### Almanach repository (`go-go-golems/almanach`)

| File | Role | Status |
|---|---|---|
| `Dockerfile` | Multi-stage build (Go + headless Chrome) | Exists, needs CMD |
| `.github/workflows/push.yml` | Unit tests on push/PR | Exists, unchanged |
| `.github/workflows/publish-image.yaml` | Docker build + GitOps PR | **NEW** |
| `deploy/gitops-targets.json` | GitOps target config | **NEW** |
| `.goreleaser.yaml` | Binary releases | Exists, unchanged |

### Crib-k3s repository (`wesen/crib-k3s`)

| File | Role | Status |
|---|---|---|
| `gitops/kustomize/almanach/deployment.yaml` | Kubernetes Deployment | Exists, needs image tag update |
| `gitops/kustomize/almanach/service.yaml` | ClusterIP Service | Exists, unchanged |
| `gitops/kustomize/almanach/ingress.yaml` | Traefik Ingress + TLS | Exists, unchanged |
| `gitops/kustomize/almanach/namespace.yaml` | Namespace definition | Exists, unchanged |
| `gitops/kustomize/almanach/certificate.yaml` | cert-manager Certificate | Exists, unchanged |
| `gitops/kustomize/almanach/kustomization.yaml` | Kustomize resources list | Exists, unchanged |
| `gitops/applications/almanach.yaml` | ArgoCD Application | Exists, unchanged |

### Infra-tooling repository (`go-go-golems/infra-tooling`)

| File | Role | Status |
|---|---|---|
| `.github/workflows/publish-ghcr-image.yml` | Reusable CI/CD workflow | Exists, unchanged |
| `actions/open-gitops-pr/action.yml` | GitOps PR action definition | Exists, unchanged |
| `actions/open-gitops-pr/src/gitops_pr_action/open_gitops_pr.py` | Manifest patching + PR logic | Exists, unchanged |

### Vault

| Path | Content | Status |
|---|---|---|
| `kv/data/ci/github/almanach-render-service/gitops-pr-token` | GitHub PAT for crib-k3s PRs | **NEW** |
| `auth/github-actions/role/almanach-render-service-gitops-pr` | JWT role for almanach repo | **NEW** |

---

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| GHCR package is private after first push | Medium | Cluster cannot pull image | Check package visibility after first push |
| Vault role not configured before first run | Medium | GitOps PR job fails (image still pushed) | Configure Vault first, or use `workflow_dispatch` after Vault is ready |
| Docker build fails due to CGO cross-compilation | Low | Build fails | Single-arch `linux/amd64` avoids cross-compilation |
| goja-hosting-site pattern doesn't match exactly | Low | Minor differences | Adapt path filters and image_name |
| `imagePullPolicy: IfNotPresent` causes stale image on node | Low | Wrong binary runs | Use `imagePullPolicy: Always` for first deployment, switch to `IfNotPresent` after confirming immutable tags work |
| ArgoCD doesn't detect PR merge fast enough | Low | Delayed rollout | ArgoCD polls every ~3 min; acceptable for non-critical service |

---

## 12. Alternatives Considered

### 12.1 Build Docker image in GoReleaser

GoReleaser supports Docker builds natively. However:
- The infra-tooling pipeline is already battle-tested and handles GitOps PRs.
- GoReleaser runs on tag push, not on every push to main.
- Adding Docker to GoReleaser would duplicate the image-building concern.

**Decision**: Use infra-tooling reusable workflow. GoReleaser continues handling binary releases.

### 12.2 Use `:latest` tag + ArgoCD image updater

ArgoCD Image Updater can watch for new `:latest` tags and auto-update manifests. However:
- `:latest` is mutable — no audit trail.
- Requires installing and configuring the Image Updater controller.
- The infra-tooling approach gives human review via PRs.

**Decision**: Use immutable `sha-<hash>` tags with PR-based updates.

### 12.3 Push directly to main in crib-k3s (no PR)

The `open-gitops-pr` action can be configured to push directly to the target branch instead of opening a PR. However:
- No human review gate.
- No easy rollback (no PR to revert).
- Risk of breaking the cluster if a bad image is pushed.

**Decision**: Open PRs by default. Can be changed to direct push for hotfixes if needed.

---

## 13. Open Questions

1. **Who has Vault admin access** to create the role and secret? This is a prerequisite before the pipeline can open PRs.
2. **Should we build for arm64 too?** The k3s cluster is amd64, but multi-arch would support future ARM nodes. Requires cross-compilation setup in Dockerfile.
3. **Should `push.yml` be merged into `publish-image.yaml`?** Currently they run in parallel (both trigger on push to main). Could consolidate to reduce workflow runs, but they serve different purposes (unit tests vs Docker build).
4. **Initial image tag**: The deployment will have a placeholder `sha-0000000`. Should we manually push an initial image first, or accept the brief period where the deployment references a nonexistent tag?

---

## 14. References

- `go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml` — reusable workflow source
- `go-go-golems/infra-tooling/actions/open-gitops-pr/` — GitOps PR action source
- `2026-05-03--goja-hosting-site/.github/workflows/publish-image.yaml` — reference implementation
- `2026-05-03--goja-hosting-site/deploy/gitops-targets.json` — reference target config
- `wesen/crib-k3s/gitops/kustomize/almanach/` — current deployment manifests
- `wesen/crib-k3s/gitops/applications/almanach.yaml` — ArgoCD Application
- `go-go-golems/almanach/Dockerfile` — current Dockerfile
- `go-go-golems/almanach/.goreleaser.yaml` — binary release config (unchanged)
