# CURRENT TASK

## Release · Feature · Task
v0.6 → Feature 6.1 (Local Daemon) → Task 6.1.5

## Status
IN PROGRESS

## What was done last session
Completed Task 5.2.3 by wiring `CortadoAIService` into `CortadoCodeEditor`, adding CodeMirror ghost-text decorations plus inline completion debounce/cancel key handling in the web bridge, exposing inline completion interop through the editor platform adapters, trimming echoed prefix overlap before ghost rendering, and covering the integration with Flutter widget tests plus JS-side helper tests/build verification.

## What was done this session
Completed Task 6.1.3 by adding `proto/filesync/v1/filesync.proto`, generating the shared FileSync stream bindings, wiring the control plane to serve an h2c gRPC sync relay alongside the existing HTTP router, computing the initial flat-map state-vector sync plan against the workspace agent, applying inbound daemon file ops onto the workspace filesystem, forwarding workspace watch events back over the sync stream, and covering the new relay slice with control-plane gRPC tests plus proto/build verification. Completed Task 6.1.4 by exposing daemon-side `localClock`/`remoteClock` state, teaching the watcher to advance `localClock`, adding a daemon conflict-resolution engine with base snapshots, `diff3`-runner support, binary last-write-wins handling, merge/conflict logging, and wiring unresolved conflict notices onto the local daemon WebSocket bridge on mux channel `0x0600` with targeted daemon app/filesync coverage.

## Remaining work this session
Task 6.1.5:
- add a Flutter-side `CortadoLocalDaemonBridge` for connecting to `ws://127.0.0.1:9731`
- expose `startSync(localPath, workspaceId)`, `stopSync`, and `getSyncStatus`
- surface daemon-not-running and conflict/sync status hooks in the package API with minimal UI assumptions

## Definition of done
- [ ] Flutter package exposes a local daemon bridge client API for start/stop/status against `ws://127.0.0.1:9731`
- [ ] daemon bridge surfaces install/offline status in a host-app-friendly way without assuming a full UI shell
- [ ] conflict/sync status data is available to downstream package consumers through package state
- [ ] affected Flutter tests/analyze pass for the new bridge slice

## Next task after this one
v0.7 → Feature 7.1 — Workspace port forwarding
See _dev/features/feat-6-1.md for the active Feature 6.1 spec

## Blocked on / decisions needed
No active blockers.
