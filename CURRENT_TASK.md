# CURRENT TASK

## Release · Feature · Task
v0.6 → Feature 6.1 (Local Daemon) → Task 6.1.3

## Status
IN PROGRESS

## What was done last session
Completed Task 5.2.3 by wiring `CortadoAIService` into `CortadoCodeEditor`, adding CodeMirror ghost-text decorations plus inline completion debounce/cancel key handling in the web bridge, exposing inline completion interop through the editor platform adapters, trimming echoed prefix overlap before ghost rendering, and covering the integration with Flutter widget tests plus JS-side helper tests/build verification.

## What was done this session
Completed Tasks 6.1.1 and 6.1.2 by scaffolding a standalone `daemon/` Go module with a loopback-only WebSocket proxy, SQLite state bootstrap via `modernc.org/sqlite`, shipped service-definition assets plus a hosted install-script path wired through Terraform, then adding a cross-platform `fsnotify` watcher with 50ms debounce, checksum dedupe against persisted file state, default excludes, and Linux inotify-capacity warnings.

## Remaining work this session
Task 6.1.3:
- add `proto/filesync/v1/filesync.proto` with the bidirectional sync stream contract and regeneration workflow updates
- wire the control plane to relay daemon sync messages between the local daemon and workspace agent
- keep the initial sync path intentionally flat-map based rather than inventing a Merkle-tree implementation early

## Definition of done
- [ ] `proto/filesync/v1/filesync.proto` exists with the stream contract for `SyncMessage`, `FileOp`, and initial state-vector exchange
- [ ] control-plane plumbing can accept daemon sync connections and relay messages toward the workspace side without breaking existing mux traffic
- [ ] generated code / protocol validation passes (`buf lint` and regeneration as needed)
- [ ] affected Go module tests/builds pass for the new relay slice

## Next task after this one
Task 6.1.4 — Conflict detection and resolution
See _dev/features/feat-6-1.md for the active Feature 6.1 spec

## Blocked on / decisions needed
None currently recorded.
