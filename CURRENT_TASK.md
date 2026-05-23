# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.2 (Workspace Agent — PTY Core) → Task 1.2.1

## Status
DONE

## What was done last session
Completed Task 1.1.5 by creating and applying the `cortado-workspaces` namespace and `workspace-sa` Workload Identity bootstrap manifest for the dev cluster, then verifying the service account annotations through `kubectl`.

## What was done this session
Expanded `proto/agent/v1/agent.proto` from the empty skeleton into the first real gRPC contract for the workspace agent. The service now defines `CreatePty`, bidirectional `StreamPty`, and `Health`, plus the PTY/session/window-size message types needed for the next Go server and PTY manager tasks. Adjusted message names to `*Request` / `*Response` so `buf lint` passes under the repo's standard lint rules, then regenerated the Go and Dart stubs with `buf generate`.

## Remaining work this session
None.

## Definition of done
- [x] `WorkspaceAgentService` defines `CreatePty`, `StreamPty`, and `Health`
- [x] PTY/session/window-size message types are defined in `proto/agent/v1/agent.proto`
- [x] `buf lint` passes
- [x] `buf generate` succeeds
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...` passes in `agent/`
- [x] `flutter analyze` passes in `flutter/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.2.2 — PTY management in Go
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.2 for full spec

## Blocked on / decisions needed
None.
