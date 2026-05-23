# Decisions Needed

## 23/05/26

- Confirm whether `DeletePath` should remain recursive for directories. The current Task 3.1.2 implementation uses `os.RemoveAll`, which deletes non-empty directories in one call.
- Confirm whether `WriteFile` should auto-create missing parent directories. The current Task 3.1.2 implementation requires the parent directory to already exist and returns `NotFound` otherwise.
- Confirm whether v0.3 should keep CodeMirror's Dart language as a plain-text fallback or adopt a specific community-maintained CM6 Dart package. Task 3.2.3 ships the editor bridge and tabs flow, but CodeMirror does not provide an official Dart language package to select by default.
