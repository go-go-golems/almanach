# Tasks

## Phase 1: Almanach repository — CI/CD workflow and Docker image

- [x] Create `deploy/gitops-targets.json` pointing at wesen/crib-k3s
- [x] Create `.github/workflows/publish-image.yaml` using infra-tooling reusable workflow
- [x] Add `CMD ["serve"]` to Dockerfile
- [x] Verify Dockerfile builds locally

## Phase 2: Crib-k3s deployment — switch to immutable tags

- [ ] Update `gitops/kustomize/almanach/deployment.yaml`: `:latest` → `:sha-<placeholder>`, `imagePullPolicy: Always` → `IfNotPresent`
- [ ] Verify kustomize builds cleanly with `kubectl kustomize`

## Phase 3: Vault configuration (admin prerequisite)

- [x] Create Vault role `almanach-render-service-gitops-pr` via Terraform (`wesen/terraform`, commit `af3a5c0`)
- [x] Store GitHub PAT at `kv/data/ci/github/almanach-render-service/gitops-pr-token` (copied from Hetzner GitOps PR token)

## Phase 4: Validation

- [ ] Push to main triggers publish-image workflow
- [ ] Image appears in GHCR with `sha-<hash>` tag
- [ ] PR opened against wesen/crib-k3s
- [ ] ArgoCD rolls out new pod after merge
