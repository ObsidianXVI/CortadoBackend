# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.2 (File Tree & Editor Widget) → Task 3.2.2

## Status
IN PROGRESS

## What was done last session
Completed Task 3.2.1 by adding `WorkspaceManager.listDirectory`, introducing the normalized `VfsNode` `freezed` union and lazy-loading `VfsNotifier`, wiring file-event application around the flat path-keyed VFS map, and covering the new state model with Flutter tests after updating the generated-proto runtime dependencies to `grpc ^5.1.0` and `protobuf ^6.0.0`.

## What was done this session
Implemented the core Task 3.2.2 file tree slice by adding `CortadoFileTree`/`FileTreeRow`, opening the mux file-watch channel on `0x0200`, driving the visible tree from the VFS provider, wiring directory expansion to lazy loading, covering the new widget/watch behavior with Flutter tests, and then unblocking the remaining context-menu work by adding explicit `MakeDir`/`RenamePath` agent RPCs plus matching control-plane HTTP endpoints.

## Remaining work this session
Finish the remaining Task 3.2.2 UI work on top of the new mkdir/rename backend contract: wire `New File`, `New Folder`, `Rename`, and `Delete` into the file tree context menu and add the inline rename editor/focus behavior.

## Definition of done
- [x] `flutter/lib/src/filesystem/cortado_file_tree.dart` renders an indent-aware list of file tree rows backed by the VFS map
- [x] Directory rows toggle expansion and lazy-load children on first open via `VfsNotifier`
- [x] File rows expose a selection callback and directory rows show the correct expand/collapse affordance
- [x] The tree subscribes to file-watch events so external filesystem updates refresh the visible nodes
- [ ] Context menu actions are implemented for New File, New Folder, Rename, and Delete
- [ ] Inline rename auto-focuses/selects-all and commits on Enter or blur
- [x] `cd flutter && flutter test` passes
- [x] `cd flutter && flutter analyze` passes

## Next task after this one
Task 3.2.3 — CodeMirror 6 editor widget
See _dev/features/feat-3-2.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the remaining file-API behavior confirmations recorded during Task 3.1.2.
