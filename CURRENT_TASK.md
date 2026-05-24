# CURRENT TASK

## Release · Feature · Task
v0.5 → Feature 5.2 (Inline AI Completion) → Task 5.2.3

## Status
IN PROGRESS

## What was done last session
Completed Task 5.1.3 by adding a reusable Qdrant HTTP client for collection creation, point upserts, and file-path deletes; scaffolding the `cortado-indexer-updater` HTTP ingestion surface with a per-workspace 5-second batcher and injectable incremental processor; normalizing root-indexed chunk metadata to workspace-relative paths; and wiring the new updater Cloud Run service account plus `cortado-file-changes` Pub/Sub topic/subscription through Terraform for both environments.

## What was done this session
Completed Tasks 5.2.1 and 5.2.2 by adding the control-plane `POST /v1/workspaces/{id}/ai/complete` SSE endpoint with prefix/suffix trimming, Vertex-embedded Qdrant top-3 retrieval, and Secret Manager-backed Gemini provider configuration; then exposing a Dart `CortadoAIService` that posts completion requests, parses SSE token streams, supports bearer/dev auth, and is exported from the package root.

## Remaining work this session
Task 5.2.3:
- wire `CortadoAIService` into `CortadoCodeEditor` so inline completions start from the active document buffer
- render streamed ghost text in the CodeMirror web implementation and keep it out of pointer interaction/layout flow
- handle Tab accept, Escape dismiss, and any other key as immediate cancellation/clear for the in-flight completion stream

## Definition of done
- [ ] ghost text renders inline from streamed completion tokens in the CodeMirror-backed editor
- [ ] Tab accepts the ghost text into the document without duplicating prefix content
- [ ] Escape dismisses, and any non-accept key cancels the in-flight request before stale tokens render
- [ ] editor/widget tests and `flutter analyze` pass for the ghost-text integration

## Next task after this one
Task 5.3.1 — Chat API endpoint with RAG
See _dev/features/feat-5-2.md for the active Feature 5.2 spec

## Blocked on / decisions needed
None currently recorded.
