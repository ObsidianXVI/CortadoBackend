# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.3 (Control Plane — WebSocket Gateway) → Task 1.3.3

## Status
DONE

## What was done last session
Completed Task 1.3.2 by adding the first `client-go` workspace pod manager with headless-service creation, pod lifecycle watch hooks, and unit coverage around pod/service CRUD and DNS resolution, then provisioning Firestore plus datastore IAM access for the control plane in both Terraform env roots.

## What was done this session
Implemented the authenticated `GET /v1/workspaces/{id}/connect` WebSocket upgrade path for the control plane, added the big-endian mux frame codec and `MuxConn` write pump with a bounded 64-frame drop-on-full queue, and configured WebSocket ping/pong keepalive with read-deadline resets. Limited v0.1 dispatch to terminal channel `0x0001`, returned mux error frames for unsupported channels, and added route/integration coverage for dev-bypass WebSocket auth and terminal dispatch behavior.

## Remaining work this session
None.

## Definition of done
- [x] `GET /v1/workspaces/{id}/connect` upgrades to a WebSocket under the authenticated `/v1` router
- [x] The mux frame codec uses `[channel_id:uint16][msg_type:uint8][payload_len:uint32][payload]` with big-endian encoding
- [x] `MuxConn` uses a single write pump with a buffered 64-frame queue and drops when the queue is full
- [x] The gateway sends WebSocket ping control frames, sets a pong handler, and refreshes the read deadline on pong
- [x] Only terminal channel `0x0001` dispatches in v0.1 and unsupported channels receive mux error frames
- [x] WebSocket upgrades work with the dev-bypass auth flow
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `control-plane/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes in `control-plane/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.3.4 — Bridge: WebSocket mux channel ↔ gRPC agent stream
See _dev/docs/release_timeline.md §Feature 1.3 Task 1.3.4 for full spec

## Blocked on / decisions needed
None.
