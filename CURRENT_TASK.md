# CURRENT TASK

## Release · Feature · Task
v0.5 → Feature 5.2 (Inline AI Completion) → Task 5.2.1

## Status
IN PROGRESS

## What was done last session
Completed Task 5.1.3 by adding a reusable Qdrant HTTP client for collection creation, point upserts, and file-path deletes; scaffolding the `cortado-indexer-updater` HTTP ingestion surface with a per-workspace 5-second batcher and injectable incremental processor; normalizing root-indexed chunk metadata to workspace-relative paths; and wiring the new updater Cloud Run service account plus `cortado-file-changes` Pub/Sub topic/subscription through Terraform for both environments.

## What was done this session
Started Task 5.2.1 by reviewing the inline-completion spec, confirming the next control-plane surface is `POST /v1/workspaces/{id}/ai/complete`, and identifying the three immediate implementation seams: Secret Manager-backed AI provider credentials, a control-plane context builder that combines prefix/suffix with Qdrant retrieval, and SSE token streaming back to the Flutter client.

## Remaining work this session
Task 5.2.1:
- add the control-plane `POST /v1/workspaces/{id}/ai/complete` endpoint and request/response model
- build the completion context assembler with prefix/suffix extraction plus top-3 Qdrant retrieval hooks
- add Secret Manager and runtime config plumbing so the AI provider key stays server-side only
- stream provider tokens back as SSE without buffering the full completion first

## Definition of done
- [ ] the control plane exposes `POST /v1/workspaces/{id}/ai/complete`
- [ ] completion requests assemble prefix/suffix context plus top-3 Qdrant matches
- [ ] the provider call streams tokens through the control plane as SSE
- [ ] the AI API key remains scoped to Secret Manager / control-plane runtime only
- [ ] control-plane tests and relevant infrastructure/config verification pass for the task changes

## Next task after this one
Task 5.2.2 — Streaming completion in Dart
See _dev/features/feat-5-2.md for the active Feature 5.2 spec

## Blocked on / decisions needed
None currently recorded.
