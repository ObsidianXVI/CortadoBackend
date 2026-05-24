# CURRENT TASK

## Release · Feature · Task
v0.6 → Feature 6.1 (Local Daemon) → Task 6.1.1

## Status
IN PROGRESS

## What was done last session
Completed Tasks 5.2.1 and 5.2.2 by adding the control-plane `POST /v1/workspaces/{id}/ai/complete` SSE endpoint with prefix/suffix trimming, Vertex-embedded Qdrant top-3 retrieval, and Secret Manager-backed Gemini provider configuration; then exposing a Dart `CortadoAIService` that posts completion requests, parses SSE token streams, supports bearer/dev auth, and is exported from the package root.

## What was done this session
Completed Task 5.2.3 by wiring `CortadoAIService` into `CortadoCodeEditor`, adding CodeMirror ghost-text decorations plus inline completion debounce/cancel key handling in the web bridge, exposing inline completion interop through the editor platform adapters, trimming echoed prefix overlap before ghost rendering, and covering the integration with Flutter widget tests plus JS-side helper tests/build verification.

## Remaining work this session
Task 6.1.1:
- scaffold the standalone `cortado-daemon` Go binary as a repo peer to `agent/` and `control-plane/`
- design the local `ws://127.0.0.1:9731` proxy surface, CORS/preflight handling, and local SQLite state layout with `modernc.org/sqlite`
- add the first install/service-definition packaging scaffolding without inventing unsupported platform integration details

## Definition of done
- [ ] standalone daemon module/binary scaffolding exists and builds with `CGO_ENABLED=0`
- [ ] local proxy listens on `127.0.0.1:9731` only and responds correctly to WebSocket/CORS preflight traffic
- [ ] service definition/install-script artifacts are scaffolded in-repo and wired to the intended distribution path
- [ ] daemon-focused tests/build verification pass for the initial architecture slice

## Next task after this one
Task 6.1.2 — Filesystem watcher (cross-platform)
See _dev/features/feat-6-1.md for the active Feature 6.1 spec

## Blocked on / decisions needed
None currently recorded.
