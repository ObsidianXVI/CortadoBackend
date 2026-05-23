# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.2 (File Tree & Editor Widget) → Task 3.2.2

## Status
IN PROGRESS

## What was done last session
Completed Task 3.2.1 by adding `WorkspaceManager.listDirectory`, introducing the normalized `VfsNode` `freezed` union and lazy-loading `VfsNotifier`, wiring file-event application around the flat path-keyed VFS map, and covering the new state model with Flutter tests after updating the generated-proto runtime dependencies to `grpc ^5.1.0` and `protobuf ^6.0.0`.

## What was done this session
Advanced the active release pointer to Task 3.2.2 after verifying the Task 3.2.1 Flutter slice with `flutter test` and `flutter analyze`.

## Remaining work this session
Implement the file tree widget for Task 3.2.2: build an indent-aware `CortadoFileTree`, render expandable file and directory rows from the VFS map, trigger lazy child loading on first expansion, and wire the widget to live file-watch updates.

## Definition of done
- [ ] `flutter/lib/src/filesystem/cortado_file_tree.dart` renders an indent-aware list of file tree rows backed by the VFS map
- [ ] Directory rows toggle expansion and lazy-load children on first open via `VfsNotifier`
- [ ] File rows expose a selection callback and directory rows show the correct expand/collapse affordance
- [ ] The tree subscribes to file-watch events so external filesystem updates refresh the visible nodes
- [ ] `cd flutter && flutter test` passes
- [ ] `cd flutter && flutter analyze` passes

## Next task after this one
Task 3.2.3 — CodeMirror 6 editor widget
See _dev/features/feat-3-2.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the open file-API behavior confirmations recorded during Task 3.1.2.
