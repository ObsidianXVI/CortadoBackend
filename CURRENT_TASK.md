# CURRENT TASK

## Release · Feature · Task
v0.4 → Feature 4.2 (Editor LSP Integration) → Task 4.2.4

## Status
PENDING

## What was done last session
Completed Task 4.2.3 by wiring `publishDiagnostics` through the active LSP client, bridging diagnostics into CodeMirror lint rendering, surfacing warning/error dots for open files in the editor and file tree, and verifying replacement/rendering with focused Flutter tests, `npm test`, and a clean `flutter analyze`.

## What was done this session
Task 4.2.3 is complete and verified locally.

## Remaining work this session
Implement hover and go-to-definition on top of the active LSP bridge:
- trigger `textDocument/hover` after a 500ms mouse hover delay
- render sanitized markdown hover tooltips in CodeMirror
- trigger `textDocument/definition` on Ctrl+click and open the target file
- treat Dart SDK targets as read-only tabs

## Definition of done
- [ ] `textDocument/hover` requests are issued after a 500ms hover delay
- [ ] hover content is rendered as sanitized markdown in CodeMirror tooltips
- [ ] Ctrl+click requests `textDocument/definition` and opens the target file
- [ ] Dart SDK definitions open read-only
- [ ] relevant Flutter/web tests cover hover tooltip and definition navigation
- [ ] `cd flutter && flutter analyze` passes

## Next task after this one
Feature 4.2 complete — next release task to be selected after Task 4.2.4
See _dev/features/feat-4-2.md for full spec

## Blocked on / decisions needed
None currently recorded.
