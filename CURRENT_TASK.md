# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.2 (File Tree & Editor Widget) → Task 3.2.3

## Status
COMPLETE

## What was done last session
Completed Task 3.2.1 by adding `WorkspaceManager.listDirectory`, introducing the normalized `VfsNode` `freezed` union and lazy-loading `VfsNotifier`, wiring file-event application around the flat path-keyed VFS map, and covering the new state model with Flutter tests after updating the generated-proto runtime dependencies to `grpc ^5.1.0` and `protobuf ^6.0.0`.

## What was done this session
Completed Task 3.2.3 by adding the new `CortadoCodeEditor` package surface, `OpenTab`/`TabsNotifier` multi-tab state with a 15-tab cap and saved-vs-current hash dirty tracking, and the web/stub platform bridge for a single CodeMirror-backed `HtmlElementView`. The editor now opens files through `WorkspaceManager.readFile`, saves through `WorkspaceManager.writeFile` on `Ctrl+S`, preserves selection when reloading an active tab from external file events, and ships with focused Flutter tests for the save/reload flow plus tab-state unit coverage. The demo host app was also extended with a bundled `demo_app/web/cortado_editor.js` bridge, a local `esbuild` bundle setup, and the matching host-page/README wiring required for Flutter Web consumers.

## Remaining work this session
None. Task 3.2.3 is complete and verified; the next task remains pending and has not been started.

## Definition of done
- [x] `flutter/lib/src/editor/cortado_code_editor.dart` embeds a web editor platform view with a clear non-web fallback
- [x] File open reads `/v1/workspaces/{id}/files/content` through `WorkspaceManager.readFile`
- [x] `Ctrl+S` writes `/v1/workspaces/{id}/files/content` through `WorkspaceManager.writeFile`
- [x] Multi-tab state is implemented with `OpenTab` + `TabsNotifier` and capped at 15 tabs
- [x] Dirty-state tracking compares the current content hash to the last-saved hash and surfaces it on the tab strip
- [x] External file events can reload the active tab while preserving selection when a caller supplies a shared file-event stream
- [x] The demo host app loads the bundled `cortado_editor.js` bridge for Flutter Web validation
- [x] `cd flutter && flutter test` passes
- [x] `cd flutter && flutter analyze` passes
- [x] `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter build web` passes

## Next task after this one
Task 3.3.1 — PVC lifecycle in control plane
See _dev/features/feat-3-3.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the remaining file-API behavior confirmations recorded during Task 3.1.2 plus the unresolved CodeMirror Dart language-package choice for the editor bridge.
