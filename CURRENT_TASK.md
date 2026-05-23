# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.1 (File API) → Task 3.1.3

## Status
IN PROGRESS

## What was done last session
Completed Task 2.4.3 by adding the control-plane refresh-token exchange, introducing shared Flutter auth-session state for JWT expiry and refresh handling, switching Flutter HTTP and browser WebSocket auth to bearer-token usage, validating the shortened expiry-based refresh soak, and publishing the `v0.2.0` release tag.

## What was done this session
Completed Task 3.1.1 by extending `proto/agent/v1/agent.proto` with the filesystem RPC surface and chunk/file-event messages, then completed Task 3.1.2 by implementing agent-side directory listing, chunked file reads, atomic same-directory writes with xxHash64 verification, recursive debounced file watching under the workspace root, and bufconn regression coverage for the new RPCs.

## Remaining work this session
Implement the control-plane HTTP file endpoints for directory listing, file read/write streaming, delete proxying, and mux channel `0x0200` file-watch forwarding for Task 3.1.3, including the required control-plane test/build verification loop.

## Definition of done
- [ ] `GET /v1/workspaces/{id}/files?path=` proxies to `ListDir`
- [ ] `GET /v1/workspaces/{id}/files/content?path=` streams `ReadFile` into the HTTP response body
- [ ] `PUT /v1/workspaces/{id}/files/content?path=` streams the HTTP request body into `WriteFile` without full buffering
- [ ] `DELETE /v1/workspaces/{id}/files?path=` proxies to `DeletePath`
- [ ] File watch events flow over mux channel `0x0200`
- [ ] `cd control-plane && GOTOOLCHAIN=local go test ./...` passes
- [ ] `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...` passes

## Next task after this one
Task 3.2.1 — Virtual filesystem model in Dart
See _dev/features/feat-3-1.md and _dev/features/feat-3-2.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the open file-API behavior confirmations recorded during Task 3.1.2.
