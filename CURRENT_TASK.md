# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.2 (File Tree & Editor Widget) → Task 3.2.1

## Status
IN PROGRESS

## What was done last session
Completed Tasks 3.1.1 and 3.1.2 by extending `proto/agent/v1/agent.proto` with the filesystem RPC surface and chunk/file-event messages, then implementing agent-side directory listing, chunked file reads, atomic same-directory writes with xxHash64 verification, recursive debounced file watching under the workspace root, and bufconn regression coverage for the new RPCs.

## What was done this session
Completed Task 3.1.3 by adding authenticated control-plane file list/read/write/delete HTTP endpoints backed by agent gRPC streaming, wiring file-watch events onto mux channel `0x0200` via the WebSocket `Open` flow, and extending the control-plane API/gateway coverage before advancing the release pointer to Feature 3.2 Task 3.2.1.

## Remaining work this session
Implement the normalized Dart VFS model for Task 3.2.1: add `VfsNode` unions, a lazy-loading `VfsNotifier`, and `FileEvent` application logic that updates the flat path-keyed map without rebuilding unchanged parent directories.

## Definition of done
- [ ] `flutter/lib/src/filesystem/vfs_node.dart` defines a normalized `freezed` `VfsNode` union for files and directories
- [ ] `VfsNotifier` stores filesystem state as `Map<String, VfsNode>` keyed by normalized path
- [ ] Directory loading is lazy: children are fetched only when a directory is first expanded
- [ ] `VfsNotifier.applyEvent(FileEvent)` updates node and parent `childPaths` state correctly for create/modify/delete/rename
- [ ] `cd flutter && flutter test` passes
- [ ] `cd flutter && flutter analyze` passes

## Next task after this one
Task 3.2.2 — File tree widget
See _dev/features/feat-3-2.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the open file-API behavior confirmations recorded during Task 3.1.2.
