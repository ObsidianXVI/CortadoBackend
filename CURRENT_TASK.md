# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.2 (Workspace Agent — PTY Core) → Task 1.2.2

## Status
DONE

## What was done last session
Completed Task 1.2.1 by defining the first real workspace-agent gRPC contract in `proto/agent/v1/agent.proto` and regenerating the Go/Dart stubs.

## What was done this session
Implemented `agent/internal/pty/manager.go` with session creation, PTY read/write/resize, process-group signaling, missing-shell validation, and session cleanup semantics suitable for the upcoming gRPC server. Added `agent/internal/pty/manager_test.go` with a live PTY smoke test that spawns `bash`, writes `echo hello_cortado`, reads until the expected output appears, and verifies the descriptive missing-shell error path.

## Remaining work this session
None.

## Definition of done
- [x] `agent/internal/pty/manager.go` implements create/read/write/resize/kill PTY session management
- [x] Missing shell path returns a descriptive `"shell ... not found in image"` error
- [x] Unit test spawns `bash`, writes `echo hello_cortado`, and observes the PTY output
- [x] `GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `agent/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...` passes in `agent/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.2.3 — gRPC server and StreamPty implementation
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.3 for full spec

## Blocked on / decisions needed
None.
