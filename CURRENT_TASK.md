# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.3 (Control Plane — WebSocket Gateway) → Task 1.3.4

## Status
DONE

## What was done last session
Completed Task 1.3.3 by implementing the authenticated `GET /v1/workspaces/{id}/connect` WebSocket upgrade path, the big-endian mux frame codec, the single-writer WebSocket pump, and keepalive-backed terminal-only dispatch coverage for the control plane gateway.

## What was done this session
Implemented the terminal bridge from the WebSocket mux to the workspace agent gRPC stream by adding a default `TerminalBridge` with per-workspace gRPC connection caching, `CreatePty` plus `StreamPty` setup on terminal open, bidirectional WS↔gRPC forwarding, close-frame surfacing for setup/stream failures, and latency logging on gRPC sends. Wired the connect handler defaults to use the bridge automatically and added bufconn-backed integration tests covering terminal open/data/exit behavior, setup failure surfacing, and gRPC connection reuse.

## Remaining work this session
None.

## Definition of done
- [x] An `Open` frame on terminal channel `0x0001` resolves the workspace agent DNS, dials gRPC, creates a PTY, and binds a streaming PTY session
- [x] Terminal data frames forward from WebSocket → gRPC and agent data/exit responses forward from gRPC → WebSocket
- [x] The control plane caches the `*grpc.ClientConn` per workspace ID instead of redialing on every frame
- [x] gRPC client connections use keepalive parameters suitable for long-lived idle workspace streams
- [x] Terminal open/setup failures surface back to the client as mux close frames instead of hanging the WebSocket handler
- [x] The bridge logs `time.Since(receivedAt)` latency on each forwarded gRPC send
- [x] Default connect-handler wiring uses the terminal bridge when no custom terminal handler is injected
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `control-plane/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes in `control-plane/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.4.1 — Package scaffold and WebSocket client
See _dev/docs/release_timeline.md §Feature 1.4 Task 1.4.1 for full spec

## Blocked on / decisions needed
None.
