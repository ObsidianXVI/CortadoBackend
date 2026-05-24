## Feature 3.1 — File API

### Task 3.1.1 — Proto: filesystem operations
**What to do:**
- Extend `proto/agent/v1/agent.proto`:
  ```protobuf
  rpc ListDir(ListDirRequest)              returns (ListDirResponse);
  rpc ReadFile(ReadFileRequest)            returns (stream ReadFileChunk);
  rpc WriteFile(stream WriteFileChunk)     returns (WriteFileResponse);
  rpc DeletePath(DeletePathRequest)        returns (DeletePathResponse);
  rpc WatchFiles(WatchFilesRequest)        returns (stream FileEvent);
  ```
- `ReadFileChunk`: `{data: bytes, seq: int32, is_last: bool, checksum: bytes}`.
- `FileEvent`: `{path: string, type: FileEventType, checksum: bytes}` where `FileEventType = {CREATED, MODIFIED, DELETED, RENAMED}`.

---

### Task 3.1.2 — Implement filesystem operations in agent
**What to do:**
- `ListDir`: `os.ReadDir(path)` → return entries with name, size, isDir, modTime, permissions.
- `ReadFile`: open, read in 256KB chunks, stream. Compute xxHash64 and send in the last chunk.
- `WriteFile`: receive chunks to a temp file (`.cortado-tmp-{uuid}` in the same directory), verify xxHash64, `os.Rename` to target (atomic on Linux). Default behavior should create missing parent directories, with an explicit API opt-out available for strict parent-exists behavior.
- `WatchFiles`: `fsnotify.NewWatcher()`, watch `/workspace` recursively (excluding `node_modules`, `.git`, `build`), debounce 50ms per path, stream events.

**Key detail**: Atomic rename via `os.Rename` only works if source and dest are on the same filesystem (same PVC mount). Since the temp file is in the same directory as the target, this is guaranteed. Never write directly to the target file — partial writes are visible to concurrent readers.

**Challenge**: `fsnotify` sends 2–4 events per save for most editors (write temp, rename, delete temp). Debouncing at 50ms collapses these. But compute the new checksum *after* debounce — reading the file during debounce may catch an intermediate state. After debounce, wait 10ms, then read and hash. This extra 10ms is imperceptible.

---

### Task 3.1.3 — HTTP file endpoints on control plane
**What to do:**
- `GET /v1/workspaces/{id}/files` with `?path=` query param → proxies to `ListDir`.
- `GET /v1/workspaces/{id}/files/content?path=` → proxies to `ReadFile` streaming, streams HTTP response body.
- `PUT /v1/workspaces/{id}/files/content?path=` → streams request body to `WriteFile`.
- `DELETE /v1/workspaces/{id}/files?path=` → proxies to `DeletePath`.
- WatchFiles events flow via the existing WebSocket mux on channel `0x0200`, opened by a new `Open` frame message type.

Add Terraform for the PVC:
```hcl
# PVC is created dynamically by control-plane (client-go)
# but the StorageClass must exist:
resource "null_resource" "storage_class_ssd" {
  depends_on = [null_resource.k8s_base]
  provisioner "local-exec" {
    command = "kubectl apply -f ${path.module}/k8s/storage-class-ssd.yaml"
  }
}
```

**Challenge**: Large file streaming through Cloud Run. Cloud Run has a 32MB in-memory request body limit by default. For `PUT` of large files, this is hit quickly. Set `--max-instances` and increase the memory limit in the Cloud Run Terraform resource, and proxy the file in chunks (stream the HTTP body directly to the gRPC `WriteFile` stream without buffering). Use `io.Pipe()` to bridge the HTTP body reader and the gRPC stream writer.

**DeletePath note**: recursive directory deletion is the settled behavior for workspace paths, including empty directories.

---
