# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.3 (Control Plane — WebSocket Gateway) → Task 1.3.1

## Status
DONE

## What was done last session
Completed Task 1.2.5 by adding Terraform-managed Kubernetes bootstrap/test-pod manifests, wiring them into the env roots, pushing the current workspace-agent image into Artifact Registry, applying the dev Terraform changes, and verifying the workspace bootstrap objects plus the test pod in the dev cluster.

## What was done this session
Initialized the control-plane HTTP service under `control-plane/` with a chi router, `/health`, graceful shutdown, and dev-bypass auth that accepts either `X-Cortado-Dev-Token: dev-bypass` or `?dev_token=dev-bypass` in development. Added the minimal Cloud Run Terraform module and wired both env roots to deploy `cortado-control-plane-${var.env}` from the regional Artifact Registry repository using a commit-pinned image tag, with `run.googleapis.com` and `cloudbuild.googleapis.com` enabled and the current control-plane service account reused for the service.

## Remaining work this session
None. Live Cloud Run apply is deferred until a control-plane container image exists in Artifact Registry.

## Definition of done
- [x] `control-plane/` contains the chi-based app skeleton and package layout for API, middleware, gateway, store, and workspace code
- [x] `GET /health` returns the expected JSON payload
- [x] Dev-bypass auth middleware enforces `dev-bypass` in development and injects fake tenant/user context
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `control-plane/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes in `control-plane/`
- [x] A reusable Terraform Cloud Run module exists for the control plane
- [x] Dev/prod env roots wire the Cloud Run module with commit-pinned image tags
- [x] `terraform validate` passes in `terraform/envs/dev`
- [x] `terraform validate` passes in `terraform/envs/prod`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.3.2 — Workspace pod manager (client-go) + Terraform Firestore
See _dev/docs/release_timeline.md §Feature 1.3 Task 1.3.2 for full spec

## Blocked on / decisions needed
None.
