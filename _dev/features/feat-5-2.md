## Feature 5.2 — Inline AI Completion

### Task 5.2.1 — Completion context builder + AI endpoint
- Control plane: `POST /v1/workspaces/{id}/ai/complete`.
- Context: code prefix (4KB) + suffix (1KB) + RAG top-3 chunks (Qdrant search query = last 3–5 lines before cursor).
- Call AI model with streaming (Claude Haiku or Gemini Flash for latency).
- Stream response as SSE (`data: {"token": "..."}\n\n`).

**Key detail**: The AI API key must never leave the control plane — it's loaded from Secret Manager at startup. The Flutter client calls the control plane's `/ai/complete` endpoint, which proxies to the AI provider. Terraform manages the secret resource:
```hcl
resource "google_secret_manager_secret" "ai_api_key" {
  secret_id = "cortado-ai-api-key-${var.env}"
  replication { auto {} }
}
```

---

### Task 5.2.2 — Streaming completion in Dart
- `CortadoAIService.streamCompletion(context)` returns `Stream<String>` of tokens.
- Uses `http.Client().send()` with `response.stream` for SSE parsing.
- Cancel the stream immediately on any keydown event (before the debounce timer — stale completions are worse than no completions).

---

### Task 5.2.3 — Ghost text in CodeMirror
- `Decoration.widget` with a `GhostTextWidget` that renders a `<span>` with the accumulated tokens, styled `color: rgba(128,128,128,0.6); pointer-events: none`.
- Tab → accept (insert ghost text at cursor, clear decoration).
- Escape → dismiss.
- Any other key → cancel the in-flight request, clear decoration.

**Challenge**: Multi-line ghost text uses a single `<pre>` element in the widget. Test vim-mode editors and editors with custom keymaps — many intercept Tab before CodeMirror sees it. If Tab doesn't accept ghost text, debug the event handler priority (`EditorView.domEventHandlers` vs `keymap.of`).
