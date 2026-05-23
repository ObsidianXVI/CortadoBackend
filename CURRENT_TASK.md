# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.1 (Workspace CRUD API) → Task 2.1.1

## Status
DONE

## What was done last session
Completed Task 1.4.4 by closing the live browser smoke loop for the Flutter web terminal against the Cloud Run control plane, fixing browser WebSocket subprotocol negotiation and gRPC keepalive behavior, and recording the measured browser round-trip latency.

## What was done this session
Implemented the control-plane workspace CRUD surface for Task 2.1.1. Added authenticated `POST /v1/workspaces`, `GET /v1/workspaces`, `GET /v1/workspaces/{id}`, `POST /v1/workspaces/{id}/start`, `POST /v1/workspaces/{id}/stop`, and `DELETE /v1/workspaces/{id}` handlers backed by a new workspace service and Firestore store. Extended the workspace pod manager to manage PVCs alongside the headless Service and pod, mount persistent storage at `/workspace`, keep stop/delete semantics distinct, and translate pod informer events into `RUNNING`, `STOPPING`, and `STOPPED` transitions without clobbering `DELETED`. Wired the server bootstrap to construct Firestore and Kubernetes clients, including a Cloud Run-compatible GKE discovery path via the Container API plus cluster identity env vars, and added Terraform bootstrap for the `cortado-workspace` StorageClass in both dev and prod.

## Remaining work this session
None. Advance to Task 2.1.2.

## Definition of done
- [x] `POST /v1/workspaces` returns `202 Accepted` with a persisted workspace in `CREATING`
- [x] `GET /v1/workspaces` lists workspaces for the current tenant context
- [x] `GET /v1/workspaces/{id}` enforces tenant scoping and returns the current persisted status
- [x] `POST /v1/workspaces/{id}/start` transitions a stopped workspace to `STARTING`
- [x] `POST /v1/workspaces/{id}/stop` transitions a workspace to `STOPPING` and relies on the pod watcher to settle it at `STOPPED`
- [x] `DELETE /v1/workspaces/{id}` marks the workspace `DELETED` and avoids later pod-delete events regressing it to `STOPPED`
- [x] The control-plane workspace service persists workspace records through Firestore
- [x] The pod manager provisions PVC + headless Service + pod, reuses the PVC on restart, and deletes the PVC on permanent delete
- [x] Cloud Run startup has a non-kubeconfig Kubernetes client path via GKE API discovery
- [x] Terraform bootstraps the `cortado-workspace` StorageClass in both environments
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `control-plane/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes in `control-plane/`
- [x] `docker build -f control-plane/Dockerfile -t cortado-control-plane:test .` passes
- [x] `terraform -chdir=terraform/envs/dev validate` passes
- [x] `terraform -chdir=terraform/envs/prod validate` passes

## Next task after this one
Task 2.1.2 — Scale-to-zero: idle detection and hibernation
See _dev/docs/release_timeline.md §Feature 2.1 Task 2.1.2 for full spec

## Blocked on / decisions needed
None.
