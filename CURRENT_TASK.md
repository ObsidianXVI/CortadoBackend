# CURRENT TASK

## Release · Feature · Task
v0.5 → Feature 5.1 (Codebase Indexing Pipeline) → Task 5.1.1

## Status
PENDING

## What was done last session
Completed Task 4.2.4 by wiring hover and go-to-definition through the active LSP bridge, adding sanitized CodeMirror hover tooltips, opening definition targets in editor tabs, and treating SDK targets as read-only tabs.

## What was done this session
Settled the remaining SDK-definition architecture question by keeping the existing low-change read-only placeholder behavior for Dart SDK targets instead of widening file reads outside the workspace root. Feature 4.2 is now fully closed.

## Remaining work this session
Prepare v0.5 kickoff for the indexing pipeline:
- split Feature 5.1 from `_dev/docs/release_timeline.md` into a dedicated `_dev/features/feat-5-1.md` task spec
- scaffold the `indexer/` service for Task 5.1.1
- confirm dependency/tooling choices for tree-sitter packaging before implementation

## Definition of done
- [ ] `_dev/features/feat-5-1.md` exists with Task 5.1.1 scoped from the release timeline
- [ ] `indexer/` Python microservice scaffold exists for tree-sitter chunking
- [ ] tree-sitter packaging/version constraints are pinned per spec
- [ ] any new infrastructure or service decisions are recorded if needed

## Next task after this one
Task 5.1.2 — Embedding pipeline + Qdrant sidecar
See _dev/docs/release_timeline.md for the current Feature 5.1 source spec

## Blocked on / decisions needed
None currently recorded.
