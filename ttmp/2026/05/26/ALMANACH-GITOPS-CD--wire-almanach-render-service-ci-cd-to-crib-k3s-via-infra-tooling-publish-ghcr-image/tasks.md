# Tasks

## Phase 1: Almanach repository — CI/CD workflow and Docker image

- [x] Create `deploy/gitops-targets.json` pointing at wesen/crib-k3s
- [x] Create `.github/workflows/publish-image.yaml` using infra-tooling reusable workflow
- [x] Add `CMD ["serve"]` to Dockerfile
- [x] Verify Dockerfile builds locally

## Phase 2: Crib-k3s deployment — switch to immutable tags

- [x] Update `gitops/kustomize/almanach/deployment.yaml`: `:latest` → immutable `sha-*`, `imagePullPolicy: Always` → `IfNotPresent` (crib-k3s PR #2/#3)
- [x] Verify kustomize builds cleanly with `kubectl kustomize`

## Phase 3: Vault configuration (admin prerequisite)

- [x] Create Vault role `almanach-render-service-gitops-pr` via Terraform (`wesen/terraform`, commit `af3a5c0`)
- [x] Store GitHub PAT at `kv/data/ci/github/almanach-render-service/gitops-pr-token` (copied from Hetzner GitOps PR token)

## Phase 4: Validation

- [x] Push to main triggers publish-image workflow
- [x] Image appears in GHCR with `sha-<hash>` tag
- [x] PR opened against wesen/crib-k3s
- [x] ArgoCD rolls out new pod after merge (`sha-321cfa6`; app remains OutOfSync only due pre-existing cert-manager Certificate drift)

## Phase 5: Post-rollout cleanup

- [x] Resolve cert-manager Certificate drift with explicit Certificate ownership (`crib-k3s` commits `06e2ab4`, `4f106fc`, `3329312`)
- [x] Verify ArgoCD reports `Synced Healthy`
- [x] Import and manage `argocd/repo-crib-k3s` via Terraform (`wesen/terraform` commit `059b9b1`)
- [x] Document old GHCR package `ghcr.io/go-go-golems/almanach-render-service` as obsolete/private and use `ghcr.io/go-go-golems/almanach`
- [ ] Rotate the GitHub token backing `kv/ci/github/gitops-pr-token` if local transcript exposure is considered sensitive
