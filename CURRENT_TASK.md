# CURRENT TASK

## Release · Feature · Task
v0.4 → Feature 4.2 (Editor LSP Integration) → Task 4.2.4

## Status
COMPLETE

## What was done last session
Completed Task 4.2.3 by wiring `publishDiagnostics` through the active LSP client, bridging diagnostics into CodeMirror lint rendering, surfacing warning/error dots for open files in the editor and file tree, and verifying replacement/rendering with focused Flutter tests, `npm test`, and a clean `flutter analyze`.

## What was done this session
Completed Task 4.2.4 by wiring hover and go-to-definition through the active LSP bridge, adding sanitized CodeMirror hover tooltips, opening definition targets in editor tabs, and treating SDK targets as read-only tabs. Verification passed with focused Flutter widget tests, `npm test`, `npm run build`, and a clean `flutter analyze`.

## Remaining work this session
Task 4.2.4 is complete. Real SDK source loading for read-only definition tabs remains a separate architecture decision and is captured in `DECISIONS_NEEDED.md`; the current implementation opens SDK targets as read-only placeholder tabs.

## Definition of done
- [x] `textDocument/hover` requests are issued after a 500ms hover delay
- [x] hover content is rendered as sanitized markdown in CodeMirror tooltips
- [x] Ctrl+click requests `textDocument/definition` and opens the target file
- [x] Dart SDK definitions open read-only
- [x] relevant Flutter/web tests cover hover tooltip and definition navigation
- [x] `cd flutter && flutter analyze` passes

## Next task after this one
Feature 4.2 complete — next release task to be selected after Task 4.2.4
See _dev/features/feat-4-2.md for full spec

## Blocked on / decisions needed
- Decide whether to relax the workspace-root read constraint with a read-only SDK whitelist so SDK definition tabs can show real `/usr/local/dart-sdk/...` source instead of the current read-only placeholder content.
