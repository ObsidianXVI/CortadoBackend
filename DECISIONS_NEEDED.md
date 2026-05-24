# Decisions Needed

## 23/05/26

- Confirm whether `DeletePath` should remain recursive for directories. The current Task 3.1.2 implementation uses `os.RemoveAll`, which deletes non-empty directories in one call.
- Confirm whether `WriteFile` should auto-create missing parent directories. The current Task 3.1.2 implementation requires the parent directory to already exist and returns `NotFound` otherwise.
- Confirm whether v0.3 should keep CodeMirror's Dart language as a plain-text fallback or adopt a specific community-maintained CM6 Dart package. Task 3.2.3 ships the editor bridge and tabs flow, but CodeMirror does not provide an official Dart language package to select by default.

## 24/05/26

- Confirm whether Task 3.3.2 should keep `roles/storage.objectCreator` on the snapshot bucket, or widen the workspace-agent bucket role for restic's GCS backend. The current spec says `objectCreator`, but restic's documented normal GCS operation appears to require `storage.objects.{create,delete,get,list}` on the bucket.
