# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.1 (File API) → Task 3.1.1

## Status
IN PROGRESS

## What was done last session
Completed Task 2.4.3 by adding the control-plane refresh-token exchange, introducing shared Flutter auth-session state for JWT expiry and refresh handling, switching Flutter HTTP and browser WebSocket auth to bearer-token usage, running the full Go and Flutter verification matrix, and validating a shortened expiry-based soak before tagging `v0.2.0`.

## What was done this session
Loaded the Feature 3.1 Task 3.1.1 spec and advanced the active release/task pointers so the next step is the proto contract for filesystem operations instead of more v0.2 auth work.

## Remaining work this session
Extend `proto/agent/v1/agent.proto` with the filesystem RPC surface (`ListDir`, `ReadFile`, `WriteFile`, `DeletePath`, `WatchFiles`) plus the directory entry, chunk, and file-event messages/enums, regenerate stubs, and run the required proto lint/generation checks.

## Definition of done
- [ ] `proto/agent/v1/agent.proto` includes `ListDir`, `ReadFile`, `WriteFile`, `DeletePath`, and `WatchFiles`
- [ ] The file read/write chunk messages capture sequence and checksum semantics needed by later tasks
- [ ] The file watch event messages and enums capture `CREATED`, `MODIFIED`, `DELETED`, and `RENAMED`
- [ ] `cd proto && buf lint` passes
- [ ] `cd proto && buf generate` passes

## Next task after this one
Task 3.1.2 — Implement filesystem operations in agent
See _dev/features/feat-3-1.md for full spec

## Blocked on / decisions needed
None.
