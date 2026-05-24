# CURRENT TASK

## Release · Feature · Task
v0.6 → Feature 6.1 (Local Daemon) → Task 6.1.4

## Status
IN PROGRESS

## What was done last session
Completed Task 5.2.3 by wiring `CortadoAIService` into `CortadoCodeEditor`, adding CodeMirror ghost-text decorations plus inline completion debounce/cancel key handling in the web bridge, exposing inline completion interop through the editor platform adapters, trimming echoed prefix overlap before ghost rendering, and covering the integration with Flutter widget tests plus JS-side helper tests/build verification.

## What was done this session
Completed Task 6.1.3 by adding `proto/filesync/v1/filesync.proto`, generating the shared FileSync stream bindings, wiring the control plane to serve an h2c gRPC sync relay alongside the existing HTTP router, computing the initial flat-map state-vector sync plan against the workspace agent, applying inbound daemon file ops onto the workspace filesystem, forwarding workspace watch events back over the sync stream, and covering the new relay slice with control-plane gRPC tests plus proto/build verification. Advanced Task 6.1.4 substantially by exposing daemon-side `localClock`/`remoteClock` state, teaching the watcher to advance `localClock`, and adding a daemon conflict-resolution engine with base snapshots, `diff3`-runner support, binary last-write-wins handling, and merge/conflict logging.

## Remaining work this session
Task 6.1.4:
- wire unresolved `ConflictNotice` events onto the local daemon WebSocket bridge on mux channel `0x0600`
- cover the conflict-notice emission path with targeted tests once the daemon bridge transport is in place

## Definition of done
- [x] daemon state tracks `localClock`, `remoteClock`, and `lastSyncedClock` consistently for synced files
- [x] sync application path detects conflicts instead of silently clobbering concurrent edits
- [x] text conflicts attempt `diff3`, binary conflicts use last-write-wins by mod time, and unresolved conflicts are surfaced
- [ ] merge/conflict logging and any new relay notices are covered by targeted tests/build verification

## Next task after this one
Task 6.1.5 — Flutter package: daemon bridge
See _dev/features/feat-6-1.md for the active Feature 6.1 spec

## Blocked on / decisions needed
No active blockers. `ConflictNotice` ownership is resolved to the local daemon WebSocket bridge; the remaining work is implementation.
