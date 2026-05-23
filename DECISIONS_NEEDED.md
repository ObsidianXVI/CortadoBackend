# Decisions Needed

## 23/05/26

- Confirm whether `DeletePath` should remain recursive for directories. The current Task 3.1.2 implementation uses `os.RemoveAll`, which deletes non-empty directories in one call.
- Confirm whether `WriteFile` should auto-create missing parent directories. The current Task 3.1.2 implementation requires the parent directory to already exist and returns `NotFound` otherwise.
