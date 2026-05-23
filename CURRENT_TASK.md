# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.2 (Workspace Agent — PTY Core) → Task 1.2.3

## Status
DONE

## What was done last session
Completed Task 1.2.2 by adding the first PTY session manager with create/read/write/resize/kill behavior, process-group signaling support, and live shell-backed unit tests.

## What was done this session
Implemented `agent/internal/server/agent_server.go` with `CreatePty`, `StreamPty`, and `Health`, wired it into a runnable `agent/cmd/agent/main.go` entrypoint, and added bufconn-based tests for health, PTY creation/streaming, and invalid stream handshakes. The gRPC server now listens on `CORTADO_AGENT_GRPC_PORT` with a default of `9090`, bridges PTY output to the bidi stream, handles resize/signal input, and maps manager/session failures to gRPC status codes.

## Remaining work this session
None.

## Definition of done
- [x] `agent/internal/server/agent_server.go` satisfies `WorkspaceAgentServiceServer`
- [x] `StreamPty` bridges PTY output to gRPC and handles data/resize/signal input
- [x] `agent/cmd/agent/main.go` starts the gRPC server on `:9090` by default
- [x] `CORTADO_AGENT_GRPC_PORT` overrides the listen port
- [x] Bufconn tests cover health, create/stream PTY, and invalid handshake behavior
- [x] `GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `agent/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...` passes in `agent/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.2.4 — Dockerfile for workspace agent
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.4 for full spec

## Blocked on / decisions needed
None.
