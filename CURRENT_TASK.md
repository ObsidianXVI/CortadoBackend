# CURRENT TASK

## Release · Feature · Task
v0.4 → Feature 4.2 (Editor LSP Integration) → Task 4.2.1

## Status
PENDING

## What was done last session
Completed Feature 4.1 by landing the LSP proto contract and optional Dart SDK workspace-image layer, implementing the agent-side LSP manager with CRLF-safe Content-Length framing and restart coverage, wiring LSP mux channels through the control-plane gateway, and cleaning up the answered 23/05 file-API/editor decisions by defaulting writes to auto-create parent directories with an explicit strict opt-out while recording the resolved decisions.

## What was done this session
Feature 4.1.1–4.1.3 and the 23/05 cleanup follow-up are complete and verified.

## Remaining work this session
Implement the Dart-side LSP client:
- add a `CortadoLSPClient` over the mux LSP channel
- implement `initialize`, `initialized`, `textDocument/didOpen`, `textDocument/didChange`, and `textDocument/didClose`
- use full-document sync mode for edits
- subscribe to `textDocument/publishDiagnostics`
- show and dismiss the "Language server starting..." state around `initialized`
- queue requests until the server finishes initialization

## Definition of done
- [ ] `CortadoLSPClient` manages JSON-RPC 2.0 request/response state over an LSP mux channel
- [ ] `initialize` and `initialized` are sent in order when the channel opens
- [ ] `didOpen`, `didChange`, and `didClose` notifications are wired for editor lifecycle events
- [ ] file changes use full-document sync payloads
- [ ] `publishDiagnostics` notifications are surfaced to Dart listeners
- [ ] requests issued before initialization are queued and flushed afterward
- [ ] the loading indicator is shown until initialization completes
- [ ] relevant Flutter tests cover initialization, request queueing, and diagnostics delivery
- [ ] `cd flutter && flutter analyze` passes

## Next task after this one
Feature 4.2 → Task 4.2.2 — Completions in CodeMirror
See _dev/features/feat-4-2.md for full spec

## Blocked on / decisions needed
None currently recorded.
