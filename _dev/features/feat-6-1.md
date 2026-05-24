## Feature 6.1 — Local Daemon
**Duration**: Weeks 25–27 (5 tasks, ~7 days)

### Task 6.1.1 — cortado-daemon architecture
- Standalone Go binary, separate from the workspace agent.
- Local WebSocket proxy on `ws://127.0.0.1:9731` (listen on `127.0.0.1` only — never `0.0.0.0`).
- Responds with `Access-Control-Allow-Origin: *` for the CORS preflight (browser requires this).
- Service definition files shipped with the binary (launchd plist for macOS, systemd user unit for Linux, NSSM config for Windows).
- Install script at `https://install.cortado.dev/daemon` (Shell script served from GCS, path managed via Terraform `google_storage_bucket_object`).

**Key detail**: Use `modernc.org/sqlite` (pure-Go SQLite, no CGO) for the local state database (`~/.cortado/state.db`). This allows `CGO_ENABLED=0` for the daemon binary, enabling simple cross-compilation:
```makefile
build-daemon:
    GOOS=darwin  GOARCH=arm64  CGO_ENABLED=0 go build -o dist/cortado-daemon-macos-arm64  ./cmd/daemon
    GOOS=linux   GOARCH=amd64  CGO_ENABLED=0 go build -o dist/cortado-daemon-linux-amd64  ./cmd/daemon
    GOOS=windows GOARCH=amd64  CGO_ENABLED=0 go build -o dist/cortado-daemon-windows.exe ./cmd/daemon
```

**Challenge**: macOS TCC permissions. Watching directories outside of `~/Documents`, `~/Desktop`, and `~/Downloads` requires Full Disk Access on macOS 13+. The install script must guide the user to grant this. Alternatively, use `NSOpenPanel` to have the user explicitly pick the workspace root — this grants access without Full Disk Access. Implement the `NSOpenPanel` approach via a thin macOS helper (SwiftUI or Objective-C) bundled with the daemon for macOS only.

---

### Task 6.1.2 — Filesystem watcher (cross-platform)
- `fsnotify` watcher on user-configured workspace roots.
- Local state index: SQLite table `file_state(path TEXT PK, checksum TEXT, mod_time INT, synced_clock INT)`.
- Process events: compute xxHash64 before and after 50ms debounce; only emit op if checksum changed.
- Default excludes: `node_modules`, `.git`, `.dart_tool`, `build`, `Pods`, `.gradle`, `*.pyc`, `__pycache__`.

**Challenge**: inotify limit (Linux, default 8192 watches). Warn at 80% capacity. Print instructions to increase: `echo fs.inotify.max_user_watches=524288 | sudo tee /etc/sysctl.d/40-inotify.conf && sudo sysctl -p`.

---

### Task 6.1.3 — FileSync gRPC proto + stream
- New service in `proto/filesync/v1/filesync.proto` (separate from agent proto):
  ```protobuf
  service FileSyncService {
    rpc Sync(stream SyncMessage) returns (stream SyncMessage);
  }
  ```
- `SyncMessage.FileOp` carries `op_id` (UUID for dedup), `path`, `OpType`, `content` (for files <256KB) or `patch` (bsdiff for larger files), `local_clock`, `checksum`.
- Control plane acts as relay: receives ops from daemon, forwards to workspace agent, forwards agent ops back to daemon.

**Challenge**: Initial sync (first connect) requires a Merkle-tree diff to determine what to sync. Implement a simplified version: daemon sends a `StateVector` message containing `Map<path, checksum>` for all local files. Control plane compares against workspace state and replies with a list of `{path, direction}` ("send local→cloud" or "send cloud→local") for differing files. Full Merkle tree is over-engineered for v0.6 — this flat-map comparison is O(n) in file count but correct.

---

### Task 6.1.4 — Conflict detection and resolution
- Vector clock per file: `{localClock, remoteClock, lastSyncedClock}`.
- Conflict: both sides modified since `lastSyncedClock`. For text files: attempt `diff3` 3-way merge (call system `diff3` binary or use a pure-Go implementation). On merge failure: emit `ConflictNotice` on WS mux channel `0x0600`.
- Binary files: last-write-wins by `modTime`.
- Log all merge operations to `~/.cortado/merge.log`.

---

### Task 6.1.5 — Flutter package: daemon bridge
- `CortadoLocalDaemonBridge` connects to `ws://127.0.0.1:9731`.
- Exposes `startSync(localPath, workspaceId)`, `stopSync`, `getSyncStatus`.
- If daemon not running: show "Install Cortado Daemon" banner with download link.
- File tree shows sync status (spinner on syncing files, conflict icon on conflicted files) via the `VfsNotifier` state.

---
