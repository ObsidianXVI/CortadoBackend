## Feature 5.3 — AI Chat Panel

### Task 5.3.1 — Chat API endpoint with RAG
- `POST /v1/workspaces/{id}/ai/chat`.
- RAG: Qdrant search with the user's message → top-5 chunks.
- System prompt includes relevant chunks + current file content.
- Summarize conversation history every 10 turns to manage context window.
- Stream SSE response.
- Store conversation history in Firestore per workspace (persists across sessions).

---

### Task 5.3.2 — Chat panel widget
- `CortadoChatPanel`: scrollable message list + text input.
- Render AI markdown with `flutter_markdown`. Code blocks use `flutter_highlight`.
- "Insert at cursor" button on code blocks.
- `ValueNotifier<String>` for in-progress message — only rebuilds the streaming bubble, not the whole list.

---

### Task 5.3.3 — @-mention context injection
- `@filename` and `@SymbolName` mentions auto-complete from file tree + symbol index.
- Parse mentions before sending, inject file contents / symbol definitions into context.
- Truncate large files: if a mentioned file > 100 lines, include only the semantic unit (function/class) nearest to the current cursor.
