## Feature 4.2 — Editor LSP Integration

### Task 4.2.1 — CortadoLSPClient in Dart
- Full JSON-RPC 2.0 client over the WS mux LSP channel.
- Implement: `initialize`, `initialized`, `textDocument/didOpen`, `textDocument/didChange`, `textDocument/didClose`.
- Use `full` sync mode (`TextDocumentSyncKind.Full`) — send entire file content on every `didChange`.
- Subscribe to `textDocument/publishDiagnostics` notifications.
- Show "Language server starting..." indicator; dismiss when `initialized` received from server.

**Challenge**: `initialize` for the Dart LS takes 3–10 seconds. During this period, queue any requests (`completion`, `hover`) and flush them after `initialized`. A simple list-based queue (`List<PendingRequest>`) flushed in the `initialized` handler is sufficient.

---

### Task 4.2.2 — Completions in CodeMirror
- JS interop bridge: Dart registers `window._cortadoLSPRequest` callback; JS calls it when CodeMirror's completion source fires.
- Dart calls `textDocument/completion`, maps `CompletionItem[]` to CodeMirror `Completion[]`, resolves the Promise via `window._cortadoLSPResult(requestId, items)`.
- Debounce: trigger completion 150ms after last keystroke. Cancel in-flight request if cursor moves before response arrives.

**Challenge**: Completion latency: Dart LS can take 200–800ms for large projects. If the user types faster than completions arrive, verify the cursor position when results arrive matches the position when the request was sent — discard stale results.

---

### Task 4.2.3 — Diagnostics (publishDiagnostics)
- `CortadoLSPClient` exposes `Stream<Map<String, List<Diagnostic>>> diagnosticsStream`.
- In the CodeMirror JS bridge, call `setDiagnostics` (`@codemirror/lint`) when diagnostics arrive.
- Propagate to the file tree: add status dot on files with errors/warnings.
- `publishDiagnostics` replaces (not appends) — store as `_diagnostics[uri] = newList`, never `addAll`.

---

### Task 4.2.4 — Hover and go-to-definition
- `textDocument/hover` on 500ms mouse hover delay: render markdown tooltip in CodeMirror using `hoverTooltip`.
- Sanitize markdown HTML with `DOMPurify` before inserting into the DOM.
- `textDocument/definition` on Ctrl+click: open the target file in a new tab. For Dart SDK files (URI starts with `/usr/local/dart-sdk`), open read-only.

---

---
