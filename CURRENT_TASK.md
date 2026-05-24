# CURRENT TASK

## Release · Feature · Task
v0.4 → Feature 4.2 (Editor LSP Integration) → Task 4.2.2

## Status
PENDING

## What was done last session
Completed Task 4.2.1 by landing the Dart-side `CortadoLSPClient`, wiring JSON-RPC initialization and document lifecycle traffic over mux LSP channels, surfacing `publishDiagnostics`, showing the editor startup overlay while the language server initializes, and covering the behavior with focused Flutter tests plus a clean `flutter analyze`.

## What was done this session
Task 4.2.1 is complete and verified locally.

## Remaining work this session
Implement CodeMirror completions on top of the new LSP client:
- register a JS interop bridge so CodeMirror can ask Dart for completions
- forward `textDocument/completion` through `CortadoLSPClient`
- map LSP `CompletionItem[]` responses into CodeMirror completion entries
- debounce completion requests by 150ms after the last keystroke
- discard stale completion responses when the cursor moves before the result arrives

## Definition of done
- [ ] JS interop exposes a completion request entrypoint from CodeMirror into Dart
- [ ] Dart forwards completion requests through `CortadoLSPClient`
- [ ] LSP completion responses are mapped into CodeMirror completion entries
- [ ] completion requests debounce for 150ms after typing stops
- [ ] stale completion responses are dropped if the cursor position no longer matches
- [ ] relevant Flutter/web tests cover completion bridging and stale-result filtering
- [ ] `cd flutter && flutter analyze` passes

## Next task after this one
Feature 4.2 → Task 4.2.3 — Diagnostics (publishDiagnostics)
See _dev/features/feat-4-2.md for full spec

## Blocked on / decisions needed
None currently recorded.
