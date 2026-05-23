# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.2 (File Tree & Editor Widget) → Task 3.2.2

## Status
COMPLETE

## What was done last session
Completed Task 3.2.1 by adding `WorkspaceManager.listDirectory`, introducing the normalized `VfsNode` `freezed` union and lazy-loading `VfsNotifier`, wiring file-event application around the flat path-keyed VFS map, and covering the new state model with Flutter tests after updating the generated-proto runtime dependencies to `grpc ^5.1.0` and `protobuf ^6.0.0`.

## What was done this session
Completed Task 3.2.2 by wiring the remaining file-tree actions end to end: `WorkspaceManager` now exposes read/write/mkdir/rename/delete helpers against the control-plane file APIs, `CortadoFileTree` now supports directory context-menu actions (`New File`, `New Folder`, `Rename`, `Delete`), inline rename with focus/select-all plus commit-on-submit/blur, and the widget tests cover those flows alongside the existing lazy-load and file-watch behavior. While closing the task out, the Flutter auth/client/workspace-manager tests were also stabilized with fixed clocks/timer factories so wall-clock-dependent token refreshes no longer leak async refresh calls into unrelated test cases.

## Remaining work this session
None. Task 3.2.2 is complete and verified; the next task remains pending and has not been started.

## Definition of done
- [x] `flutter/lib/src/filesystem/cortado_file_tree.dart` renders an indent-aware list of file tree rows backed by the VFS map
- [x] Directory rows toggle expansion and lazy-load children on first open via `VfsNotifier`
- [x] File rows expose a selection callback and directory rows show the correct expand/collapse affordance
- [x] The tree subscribes to file-watch events so external filesystem updates refresh the visible nodes
- [x] Context menu actions are implemented for New File, New Folder, Rename, and Delete
- [x] Inline rename auto-focuses/selects-all and commits on Enter or blur
- [x] `cd flutter && flutter test` passes
- [x] `cd flutter && flutter analyze` passes

## Next task after this one
Task 3.2.3 — CodeMirror 6 editor widget
See _dev/features/feat-3-2.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the remaining file-API behavior confirmations recorded during Task 3.1.2.
