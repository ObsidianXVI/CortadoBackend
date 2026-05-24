# CURRENT TASK

## Release · Feature · Task
v0.5 → Feature 5.1 (Codebase Indexing Pipeline) → Task 5.1.1

## Status
IN PROGRESS

## What was done last session
Settled the remaining SDK-definition architecture question by keeping the existing low-change read-only placeholder behavior for Dart SDK targets instead of widening file reads outside the workspace root. Feature 4.2 is now fully closed.

## What was done this session
Pulled v0.5 Features 5.1, 5.2, and 5.3 out of the release timeline into dedicated `_dev/features/` specs and scaffolded the new `indexer/` Python job with a chunker module, CLI, Dockerfile, and unit tests. The scaffold pins `tree-sitter==0.22.0`, uses fallback overlapping line windows today, and verifies cleanly with `python3 -m unittest discover -s tests`, `PYTHONPATH=src python3 -m cortado_indexer --help`, and `docker build -t cortado-indexer:test indexer`.

## Remaining work this session
Finish Task 5.1.1 itself:
- replace the current fallback-only `tree_sitter_chunk_file()` path with verified semantic parser wiring for the supported languages
- add chunk metadata extraction beyond windowed line ranges once parser loading is wired
- decide whether Dart should stay on fallback windows for the first shipping cut or whether a verified Dart grammar build path should be added in the Docker build stage now

## Definition of done
- [x] `_dev/features/feat-5-1.md` exists with Task 5.1.1 scoped from the release timeline
- [x] subsequent v0.5 features were split into `_dev/features/feat-5-2.md` and `_dev/features/feat-5-3.md`
- [x] `indexer/` Python microservice scaffold exists for tree-sitter chunking
- [x] tree-sitter packaging/version constraints are pinned per spec
- [x] local indexer unit tests pass
- [x] local indexer Docker build passes
- [ ] semantic tree-sitter parser wiring is implemented for the first supported language set
- [ ] any new infrastructure or service decisions are recorded if needed

## Next task after this one
Continue Task 5.1.1 — semantic parser wiring and chunk metadata extraction
See _dev/features/feat-5-1.md for the active Feature 5.1 spec

## Blocked on / decisions needed
None currently recorded.
