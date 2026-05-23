# Decisions Needed

## 23/05/26

- Confirm whether `DeletePath` should remain recursive for directories. The current Task 3.1.2 implementation uses `os.RemoveAll`, which deletes non-empty directories in one call.
- Confirm whether `WriteFile` should auto-create missing parent directories. The current Task 3.1.2 implementation requires the parent directory to already exist and returns `NotFound` otherwise.
- Define the backend/API contract for file-tree mutations beyond write/delete. Task 3.2.2 needs first-class support for `New File`, `New Folder`, and `Rename`, but the current agent/control-plane surface only exposes list/read/write/delete. Decide whether to add explicit mkdir/rename RPCs and HTTP endpoints now or defer the UI actions to a later task.
