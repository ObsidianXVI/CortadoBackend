## Feature 3.2 — File Tree & Editor Widget
**Duration**: Weeks 11–12 (3 tasks, ~4 days)

### Task 3.2.1 — Virtual filesystem model in Dart
**What to do:**
- Implement normalized VFS state: `Map<String, VfsNode>` keyed by path (not nested tree).
- `VfsNode` is a `freezed` union: `VfsFile({path, name, size, modTime})` and `VfsDir({path, name, childPaths, expanded, loaded})`.
- `VfsNotifier` (Riverpod `AsyncNotifier`) loads directory contents lazily: only fetch children when a directory is first expanded.
- `VfsNotifier.applyEvent(FileEvent)` updates the flat map: adds/removes/updates the entry at the given path, and updates the parent directory's `childPaths` list.

**Key detail**: Store `childPaths` as a `List<String>` (paths), not `List<VfsNode>`. When a child changes, only the child's entry in the flat map changes — the parent's `childPaths` list is untouched. This means parent directories don't rebuild when child files are modified, which is the primary performance concern for a large file tree.

---

### Task 3.2.2 — File tree widget
**What to do:**
- `CortadoFileTree`: custom `ListView.builder` with indent-aware rows. Each row is a `FileTreeRow` widget showing an icon, name, and optional status indicator.
- Expand/collapse on tap (directories only). Load children on first expand (triggers `VfsNotifier.loadDirectory`).
- Context menu on secondary tap / long press: New File, New Folder, Rename (inline editing), Delete (with confirmation dialog).
- File status dots: modified (yellow), conflict (red) — populated by the sync system in v0.6.
- WatchFiles events update `VfsNotifier` in real-time.

**Challenge**: Inline rename (clicking F2 or selecting Rename from context menu replaces the file name label with a `TextField`). The `TextField` must auto-focus, select all text, and commit on Enter or blur. In Flutter, managing focus for transient inline editors requires a `FocusNode` with `requestFocus()` called in a `WidgetsBinding.addPostFrameCallback`. If you call `requestFocus()` during the build phase, Flutter ignores it.

---

### Task 3.2.3 — CodeMirror 6 editor widget
**What to do:**
- Embed CodeMirror 6 via `HtmlElementView` (same pattern as xterm.js).
- Add `web/js/cortado_editor.js` that initializes a CodeMirror instance with: basic setup, syntax highlighting where available for JS, Python, Go, YAML, and JSON, line numbers, bracket matching. Dart remains a plaintext fallback until a deliberate Dart language package choice is made.
- Wire: file open → `GET /v1/workspaces/{id}/files/content` → set CodeMirror content.
- Wire: `Ctrl+S` → `PUT /v1/workspaces/{id}/files/content`.
- Modified indicator: compare CodeMirror content hash against last-saved hash. Show a dot in the tab title when different.
- Multi-tab: maintain a `List<OpenTab>` in a `TabsNotifier`. Max 15 open tabs (more than enough for v0.3).

**Challenge**: CodeMirror's `EditorView.dispatch` for setting content replaces the document — this resets scroll position and cursor. For re-loading a file (after an external change), preserve cursor position: save `{line, ch}` before dispatch and restore via a subsequent `dispatch({selection: EditorSelection.cursor(savedPos)})`.

---
