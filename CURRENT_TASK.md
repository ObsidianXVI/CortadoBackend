# CURRENT TASK

## Release · Feature · Task
v0.5 → Feature 5.1 (Codebase Indexing Pipeline) → Task 5.1.2

## Status
AWAITING INPUT

## What was done last session
Pulled v0.5 Features 5.1, 5.2, and 5.3 out of the release timeline into dedicated `_dev/features/` specs and scaffolded the new `indexer/` Python job with a chunker module, CLI, Dockerfile, and unit tests.

## What was done this session
Completed Task 5.1.1 by wiring parser-backed semantic chunk extraction for Python, JavaScript, and Go, pinning the corresponding grammar wheels in `indexer/pyproject.toml`, and keeping Dart on the existing fallback chunker path for the first shipping cut. The indexer now degrades cleanly when `tree_sitter` is not installed locally, host tests skip semantic cases instead of failing, and the pinned dependency set verifies in Docker with semantic tests fully passing.

## Remaining work this session
Start Task 5.1.2:
- choose the initial embedding provider path from the feature spec and reflect it in the first indexer/updater scaffolding
- decide where the embedding pipeline code should live relative to the Python indexer job and the future updater service
- map the Qdrant sidecar changes required in workspace runtime specs and Terraform before implementation begins

## Definition of done
- [x] `_dev/features/feat-5-1.md` exists with Task 5.1.1 scoped from the release timeline
- [x] subsequent v0.5 features were split into `_dev/features/feat-5-2.md` and `_dev/features/feat-5-3.md`
- [x] `indexer/` Python microservice scaffold exists for tree-sitter chunking
- [x] tree-sitter packaging/version constraints are pinned per spec
- [x] local indexer unit tests pass
- [x] local indexer Docker build passes
- [x] semantic tree-sitter parser wiring is implemented for the first supported language set
- [x] any new infrastructure or service decisions are recorded if needed

## Next task after this one
Task 5.1.2 — Embedding pipeline + Qdrant sidecar
See _dev/features/feat-5-1.md for the active Feature 5.1 spec

## Blocked on / decisions needed
Task 5.1.2 needs an embedding provider choice:
- `text-embedding-004` on Vertex AI
- `voyage-code-3` on Voyage AI
If there is no product reason to prefer Voyage, defaulting to Vertex AI is the lower-friction path because the stack already runs on GCP.
