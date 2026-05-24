# Protocols and Data Shapes

This page documents the wire-level shapes that connect the three runtime layers.

## WebSocket Mux Frame

The control plane uses a binary frame format over WebSocket.

Header layout:

- bytes `0..1`: channel ID, big-endian `uint16`
- byte `2`: message type, `uint8`
- bytes `3..6`: payload length, big-endian `uint32`
- bytes `7..`: payload

The Flutter package implements the same layout in [`flutter/lib/src/mux_frame.dart`](../flutter/lib/src/mux_frame.dart). The Go control plane implements it in [`control-plane/internal/gateway/mux.go`](../control-plane/internal/gateway/mux.go).

### Message types

- `0x01` data
- `0x02` open
- `0x03` close
- `0x04` error
- `0x05` resize
- `0xFF` ping

### Channel IDs

- `0x0001` terminal sessions
- `0x0200` file sync

### Example frame

Terminal open frame with shell payload:

```text
channelId=0x0001
messageType=0x02
payload="bash"
```

Terminal resize payload:

```text
cols=120 (uint32 big-endian)
rows=40  (uint32 big-endian)
```

## gRPC File Contract

The shared protobuf file is [`proto/agent/v1/agent.proto`](../proto/agent/v1/agent.proto).

### File listing

`ListDir` returns a repeated list of `DirectoryEntry` values:

```json
{
  "name": "main.dart",
  "size": 32,
  "is_dir": false,
  "mod_time": "2026-05-23T22:05:00Z",
  "permissions": 420
}
```

### File read/write

Reads and writes are chunked for large files. Each chunk carries:

- a sequence number
- the raw bytes
- an `is_last` flag
- a checksum on the final chunk

The control plane also uses xxHash64 checksums when proxying file content over HTTP.
`WriteFile` defaults to auto-creating missing parent directories, while an explicit API opt-out can require parents to already exist.

## File Events

`WatchFiles` emits `FileEvent` records.

Fields:

- `path`: path relative to the workspace root
- `type`: created, modified, deleted, or renamed
- `checksum`: present for file content changes where the agent can hash the file

### Example event stream

1. `src/main.dart` modified
2. `src/new_file.dart` created
3. `docs` renamed to `documentation`
4. `README.md` deleted

The Flutter file tree consumes those events to update the in-memory VFS map.

## HTTP File API

The control plane exposes a REST wrapper over the agent file RPCs.

### Requests

- `GET /v1/workspaces/{id}/files?path=...`
- `GET /v1/workspaces/{id}/files/content?path=...`
- `PUT /v1/workspaces/{id}/files/content?path=...`
- `POST /v1/workspaces/{id}/files/directory?path=...`
- `POST /v1/workspaces/{id}/files/rename?path=...&newPath=...`
- `DELETE /v1/workspaces/{id}/files?path=...`

### Response examples

Create directory:

```http
HTTP/1.1 201 Created
```

Write file:

```json
{
  "bytesWritten": 1234,
  "checksum": [1, 2, 3, 4]
}
```

Delete path:

- files are deleted directly
- directories are deleted recursively

## Workspace Status Polling

The Flutter package does not maintain a websocket for workspace status. Instead it polls `GET /v1/workspaces/{id}` and adapts its polling interval based on the current state.

That means:

- transitional states are polled more frequently,
- running workspaces are polled less frequently,
- terminal statuses eventually stop polling.

## Session Tokens

The auth service issues:

- an RS256 access token with tenant ID in the custom `tid` claim
- a random refresh token stored in Firestore

The Flutter package decodes the JWT payload locally only to read the expiry time. It does not verify the signature client-side; the server remains the source of truth.
