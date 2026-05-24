# CURRENT TASK

## Release · Feature · Task
v0.4 → Feature 4.2 (Editor LSP Integration) → Task 4.2.3

## Status
PENDING

## What was done last session
Completed Task 4.2.2 by wiring CodeMirror completions through the new Dart LSP client, adding the web-side request/result bridge with 150ms debounce and stale-result dropping, mapping LSP completion payloads into CodeMirror entries, and verifying the flow with demo web and Flutter tests plus a clean analyzer pass.

## What was done this session
Task 4.2.2 is complete and verified locally.

## Remaining work this session
Implement diagnostics propagation on top of the active LSP client:
- expose a diagnostics stream keyed by document URI
- update the CodeMirror bridge to push `publishDiagnostics` results into the editor
- render diagnostics through the lint extension
- surface file-level warning/error state for open files

## Definition of done
- [ ] `CortadoLSPClient` exposes diagnostics updates keyed by document URI
- [ ] the CodeMirror bridge forwards diagnostics into the lint integration
- [ ] `publishDiagnostics` replaces prior diagnostics for the same document
- [ ] file-level warning/error state is surfaced for open files
- [ ] relevant Flutter/web tests cover diagnostics replacement and rendering
- [ ] `cd flutter && flutter analyze` passes

## Next task after this one
Feature 4.2 → Task 4.2.4 — Hover and go-to-definition
See _dev/features/feat-4-2.md for full spec

## Blocked on / decisions needed
None currently recorded.
