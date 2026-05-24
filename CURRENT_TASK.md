# CURRENT TASK

## Release · Feature · Task
v0.5 → Feature 5.1 (Codebase Indexing Pipeline) → Task 5.1.3

## Status
IN PROGRESS

## What was done last session
Completed Task 5.1.1 by wiring parser-backed semantic chunk extraction for Python, JavaScript, and Go, pinning the corresponding grammar wheels in `indexer/pyproject.toml`, and keeping Dart on the existing fallback chunker path for the first shipping cut.

## What was done this session
Completed Task 5.1.2 by selecting Vertex AI as the first embedding provider, adding a batched `text-embedding-004` embedding client to the Python indexer scaffold, extending the CLI with `--embed`, recording the `ws-{workspaceID}` collection naming contract, and injecting the Qdrant sidecar plus persistent subpath mount into workspace pod creation and the workspace test manifest. The Python side verifies both as a source checkout and as an installed package, and the control-plane workspace pod package continues to build and test cleanly after the sidecar change.

## Remaining work this session
Start Task 5.1.3:
- add the first reusable Qdrant HTTP client code for collection creation, upsert, and delete-by-file-path operations
- scaffold the `cortado-file-changes` Pub/Sub topic/subscription resources and the updater service surface
- define the debounce batching path for per-workspace file change events before wiring control-plane triggers

## Definition of done
- [x] Vertex AI is selected as the initial embedding provider path
- [x] the Python indexer can emit Vertex-backed embeddings in batch mode
- [x] the workspace pod spec includes the Qdrant sidecar and persistent storage subpath mount
- [x] the `ws-{workspaceID}` collection naming and 768-dimension contract are recorded in code/docs
- [x] indexer and control-plane verification passed for the task changes
- [x] any new infrastructure or service decisions are recorded if needed

## Next task after this one
Task 5.1.3 — Incremental index updates via Pub/Sub
See _dev/features/feat-5-1.md for the active Feature 5.1 spec

## Blocked on / decisions needed
None currently recorded.
