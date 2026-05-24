# Workspace Agent

The workspace agent is the process that runs inside each workspace pod. It owns the workspace-local state that the control plane should not own directly:

- PTY sessions
- filesystem reads and writes
- recursive file watching
- usage accounting

The agent entry point is [`agent/cmd/agent/main.go`](../agent/cmd/agent/main.go).

## gRPC Service

The contract is defined in [`proto/agent/v1/agent.proto`](../proto/agent/v1/agent.proto). The Go implementation lives in [`agent/internal/server`](../agent/internal/server).

Core RPCs:

- `CreatePty`
- `StreamPty`
- `GetIdleStatus`
- `FlushUsageWAL`
- `ListDir`
- `ReadFile`
- `WriteFile`
- `MakeDir`
- `RenamePath`
- `DeletePath`
- `WatchFiles`
- `Health`

## PTY Behavior

PTY sessions are managed by [`agent/internal/pty/manager.go`](../agent/internal/pty/manager.go).

What this manager does:

- resolves the requested shell binary with `exec.LookPath`
- starts the process in a pseudo-terminal with the requested size
- keeps session state in memory
- forwards stdin bytes, resize requests, and signals
- streams stdout/stderr bytes back to the gRPC stream
- emits an exit code when the process ends

### PTY lifecycle example

```text
CreatePty(cols=80, rows=24, shell="/bin/bash")
  -> returns pty_id="5f5a..."
StreamPty(first message: pty_id)
  -> client sends bytes and resize frames
  -> agent sends bytes until the shell exits
  -> final response contains exit_code
```

The agent treats `syscall.EIO` from the PTY master read as a normal shell termination signal.

## Filesystem Behavior

The filesystem implementation is in [`agent/internal/server/filesystem.go`](../agent/internal/server/filesystem.go).

### Path rules

- All paths are resolved under the workspace root, usually `/workspace`.
- Absolute and relative input paths are both supported.
- The resolver rejects paths that escape the workspace root.

### Directory listing

`ListDir` returns per-entry metadata:

- `name`
- `size`
- `is_dir`
- `mod_time`
- `permissions`

### File reads and writes

Reads and writes are chunked and checksum-verified with xxHash64:

- `ReadFile` streams `ReadFileChunk` messages in sequence order.
- `WriteFile` accepts `WriteFileChunk` messages, writes to a temp file, verifies the final checksum, then renames the temp file into place.

### Directory mutation

- `MakeDir` currently uses `os.Mkdir`.
- `RenamePath` requires an existing destination parent directory and refuses to overwrite an existing destination path.
- `DeletePath` currently uses `os.RemoveAll`.

Those delete and parent-directory semantics are intentionally documented here because they are still tracked as open decisions elsewhere in the repo.

## File Watching

`WatchFiles` uses `fsnotify` and emits normalized file events relative to the workspace root. The watcher:

- recursively watches directories,
- debounces bursts of events,
- hashes changed files before sending a modification event,
- excludes some noisy directories such as `.git`, `build`, and `node_modules`.

File event types are:

- `CREATED`
- `MODIFIED`
- `DELETED`
- `RENAMED`

### Example file event

```json
{
  "path": "src/main.dart",
  "type": "FILE_EVENT_TYPE_MODIFIED",
  "checksum": "..."
}
```

## Usage Accounting

The usage tracker in [`agent/internal/usage/tracker.go`](../agent/internal/usage/tracker.go) writes line-delimited JSON records to a WAL under `/workspace/.cortado/usage.wal`.

On each tick, it:

- appends a usage record,
- publishes it to Pub/Sub if configured,
- marks the record published,
- and replays unpublished records at startup.

The record contains CPU, memory, storage, tenant, user, workspace ID, event time, and duration fields. This lets the control plane account for active workspaces without needing to keep PTY history in memory.

## Idle Reporting

`GetIdleStatus` reports:

- the last observed PTY activity time
- a rolling CPU percentage over the last 60 seconds

The control plane uses that signal to decide whether a running workspace is actually idle.

## Code References

- Agent server: [`agent/internal/server/agent_server.go`](../agent/internal/server/agent_server.go)
- Filesystem: [`agent/internal/server/filesystem.go`](../agent/internal/server/filesystem.go)
- PTY manager: [`agent/internal/pty/manager.go`](../agent/internal/pty/manager.go)
- Usage tracker: [`agent/internal/usage/tracker.go`](../agent/internal/usage/tracker.go)
