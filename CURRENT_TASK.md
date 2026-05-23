# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.1 (Workspace CRUD API) → Task 2.1.2

## Status
DONE

## What was done last session
Completed Task 2.1.1 by adding the Firestore-backed workspace CRUD API to the control plane, provisioning PVC-backed workspace pods/services, wiring Cloud Run-compatible Kubernetes client bootstrap, and bootstrapping the shared workspace StorageClass in Terraform.

## What was done this session
Implemented Task 2.1.2 end-to-end across the agent and control plane. Added agent-side idle tracking with PTY activity timestamps, a rolling `/proc/stat` CPU sampler, and the new `GetIdleStatus` gRPC method. On the control-plane side, extended workspace persistence to track `lastActiveAt`, rate-limited activity writes to Firestore when terminal data frames arrive, added a background idle monitor that polls `GetIdleStatus` on running workspaces every five minutes, and added the 30-minute Firestore stale-activity fallback stop path. The main server now wraps terminal traffic so activity is recorded as PTY data flows through the mux, and the idle timeout honors `CORTADO_IDLE_TIMEOUT_MINUTES` with the documented default of 20 minutes.

## Remaining work this session
None. Advance to Task 2.2.1.

## Definition of done
- [x] The workspace agent records PTY activity timestamps on terminal input
- [x] The workspace agent exposes `GetIdleStatus()` over gRPC
- [x] `GetIdleStatus()` reports a recent CPU utilization window alongside the last PTY activity timestamp
- [x] The control plane records `lastActiveAt` in Firestore from terminal data frames with write throttling
- [x] A background control-plane monitor polls running workspaces every five minutes and stops workspaces idle beyond the configured timeout
- [x] The control plane also scans Firestore for stale activity older than 30 minutes and stops stale non-stopped workspaces
- [x] `CORTADO_IDLE_TIMEOUT_MINUTES` overrides the default 20-minute idle timeout
- [x] `cd proto && buf lint` passes
- [x] `cd proto && buf generate` passes
- [x] `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes
- [x] `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes
- [x] `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes
- [x] `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes
- [x] `docker build -t cortado-workspace:test agent/` passes
- [x] `docker build -f control-plane/Dockerfile -t cortado-control-plane:test .` passes

## Next task after this one
Task 2.2.1 — WorkspaceManager + status polling
See _dev/docs/release_timeline.md §Feature 2.2 Task 2.2.1 for full spec

## Blocked on / decisions needed
None.
