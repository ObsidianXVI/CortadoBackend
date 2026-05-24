# CURRENT TASK

## Release · Feature · Task
v0.7 → Feature 7.1 (Port Forward Gateway) → Task 7.1.1

## Status
IN PROGRESS

## What was done last session
Completed Task 5.2.3 by wiring `CortadoAIService` into `CortadoCodeEditor`, adding CodeMirror ghost-text decorations plus inline completion debounce/cancel key handling in the web bridge, exposing inline completion interop through the editor platform adapters, trimming echoed prefix overlap before ghost rendering, and covering the integration with Flutter widget tests plus JS-side helper tests/build verification.

## What was done this session
Completed Task 6.1.5 by adding a daemon-side sync command registry for `start_sync` / `stop_sync` / `get_sync_status`, exposing a Flutter `CortadoLocalDaemonBridge` with localhost connect/unavailable handling plus conflict and sync status streams, threading local daemon sync state into `VfsNotifier` and `CortadoFileTree`, and covering the new bridge slice with daemon websocket tests plus Flutter bridge/VFS/file-tree tests.

## Remaining work this session
Task 7.1.1:
- add `ListPorts` and `WatchPorts` RPCs to the agent proto/service surface
- detect bound listen sockets from `/proc/net/tcp` and `/proc/net/tcp6`
- expose filtered port events for non-reserved preview/user ports only

## Definition of done
- [ ] agent proto includes `ListPorts` and `WatchPorts` contracts and generated stubs remain current
- [ ] agent can enumerate currently listening non-reserved ports from `/proc/net/tcp*`
- [ ] agent emits add/remove port events on a polling watcher with reserved-port filtering
- [ ] agent tests/build pass for the new port-detection slice

## Next task after this one
v0.7 → Feature 7.1 → Task 7.1.2 — Port forward HTTP/WS gateway
See _dev/features/feat-7-1.md for the active Feature 7.1 spec

## Blocked on / decisions needed
No active blockers.
