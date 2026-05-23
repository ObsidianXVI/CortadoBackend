# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.3 (Control Plane — WebSocket Gateway) → Task 1.3.2

## Status
DONE

## What was done last session
Completed Task 1.3.1 by scaffolding the control-plane HTTP service with chi routing, `/health`, and dev-bypass auth, then adding the Cloud Run Terraform module and env-root wiring for the control plane service.

## What was done this session
Added the first `client-go`-based `WorkspacePodManager` to the control plane, including pod/service creation, deletion, status lookup, stable headless-service DNS generation, and a background watch loop that can push pod lifecycle changes into a downstream status sink. Provisioned Firestore in both Terraform env roots by enabling the Firestore API, creating the default native database in `us-central1`, and granting the control-plane service account `roles/datastore.user`.

## Remaining work this session
None.

## Definition of done
- [x] `control-plane/` depends on `client-go`
- [x] `WorkspacePodManager` implements create/delete/status/DNS methods for workspace pods
- [x] A headless Service is created alongside each workspace pod with selector `cortado/workspace-id`
- [x] The pod manager exposes a background watch path for propagating pod phase changes
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `control-plane/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes in `control-plane/`
- [x] Firestore is provisioned in both Terraform env roots
- [x] The control-plane service account has `roles/datastore.user`
- [x] `terraform validate` passes in `terraform/envs/dev`
- [x] `terraform validate` passes in `terraform/envs/prod`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.3.3 — WebSocket mux protocol
See _dev/docs/release_timeline.md §Feature 1.3 Task 1.3.3 for full spec

## Blocked on / decisions needed
None.
