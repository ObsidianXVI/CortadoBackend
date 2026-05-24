# Control Plane

## Entry Point

The main server starts in [`control-plane/cmd/server/main.go`](../control-plane/cmd/server/main.go). It:

- creates a cancellable process context,
- resolves the GCP project ID,
- builds Firestore and Kubernetes clients,
- constructs the auth service,
- constructs the workspace service,
- constructs the file bridge and terminal bridge,
- starts the idle monitor,
- and serves the HTTP router.

## HTTP Surface

The router is assembled in [`control-plane/internal/api/router.go`](../control-plane/internal/api/router.go).

### Public routes

- `GET /health`
- `GET /.well-known/jwks.json`
- `POST /v1/sessions`
- `POST /v1/sessions/refresh`

### Authenticated workspace routes

- `GET /v1/workspaces`
- `POST /v1/workspaces`
- `GET /v1/workspaces/{id}`
- `POST /v1/workspaces/{id}/start`
- `POST /v1/workspaces/{id}/stop`
- `DELETE /v1/workspaces/{id}`
- `GET /v1/workspaces/{id}/files`
- `GET /v1/workspaces/{id}/files/content`
- `PUT /v1/workspaces/{id}/files/content`
- `POST /v1/workspaces/{id}/files/directory`
- `POST /v1/workspaces/{id}/files/rename`
- `DELETE /v1/workspaces/{id}/files`
- `GET /v1/workspaces/{id}/connect`

## Auth Model

Requests normally authenticate with an RS256 JWT access token. The token contains:

- `tid`: tenant ID
- `sub`: user ID
- standard expiration, issue time, and token ID claims

The middleware in [`control-plane/internal/middleware/auth.go`](../control-plane/internal/middleware/auth.go) also supports the development bypass when `CORTADO_ENV=development` and the request carries `X-Cortado-Dev-Token: dev-bypass` or `?dev_token=dev-bypass` for WebSocket upgrades.

## Session Flow

`POST /v1/sessions` accepts:

```json
{
  "api_key": "acme-api-key",
  "user_id": "user-42"
}
```

and returns:

```json
{
  "access_token": "eyJ...",
  "refresh_token": "b3c..."
}
```

The auth service looks up the API key against Firestore-backed records, resolves the tenant, issues a short-lived access token, and stores the refresh token. `POST /v1/sessions/refresh` accepts the refresh token and returns a new access token.

## Workspace Lifecycle

The workspace service is a state machine over `CREATING`, `STARTING`, `RUNNING`, `STOPPING`, `STOPPED`, and `DELETED`.

Important behavior:

- `CreateWorkspace` persists the record before asking the provisioner to create the pod.
- `StartWorkspace` and `StopWorkspace` are idempotent at the API level for already-transitioned states.
- `DeleteWorkspace` marks the record deleted before deleting the pod.
- The idle monitor periodically stops running workspaces when agent-reported idle time and CPU usage stay below the configured threshold.

## File Proxying

The file handlers in [`control-plane/internal/api/files.go`](../control-plane/internal/api/files.go) never touch the workspace filesystem directly. They:

1. validate the tenant context,
2. verify the workspace exists,
3. call the workspace file service,
4. and translate the agent response into JSON or raw bytes.

### Current HTTP file semantics

- `GET /files` returns JSON directory listings.
- `GET /files/content` streams file bytes as `application/octet-stream`.
- `PUT /files/content` streams bytes to the agent, auto-creates missing parent directories by default, and returns bytes-written plus checksum JSON. The API can also expose an explicit strict mode that requires parent directories to already exist.
- `POST /files/directory` creates a directory.
- `POST /files/rename` renames a path using the `newPath` query parameter.
- `DELETE /files` deletes a file or recursively deletes a directory subtree.

## WebSocket Connect Flow

`/v1/workspaces/{id}/connect` upgrades to a WebSocket and speaks the Cortado mux protocol. The control plane multiplexes:

- terminal traffic on channel `0x0001`,
- file sync traffic on channel `0x0200`.

The handler sends frames to the client, receives frames back, and bridges each channel to the correct gRPC stream against the workspace agent.

## Example Response Shapes

Workspace list:

```json
{
  "workspaces": [
    {
      "id": "ws-8f3d2b6d",
      "tenantId": "tenant-acme",
      "status": "RUNNING"
    }
  ]
}
```

File listing:

```json
{
  "entries": [
    {
      "name": "lib",
      "size": 0,
      "isDir": true,
      "modTime": "2026-05-23T22:00:00Z",
      "permissions": 493
    }
  ]
}
```

## Code References

- API router: [`control-plane/internal/api/router.go`](../control-plane/internal/api/router.go)
- Auth middleware: [`control-plane/internal/middleware/auth.go`](../control-plane/internal/middleware/auth.go)
- Workspace service: [`control-plane/internal/workspace/service.go`](../control-plane/internal/workspace/service.go)
- WebSocket mux: [`control-plane/internal/gateway/mux.go`](../control-plane/internal/gateway/mux.go)
- File bridge: [`control-plane/internal/gateway/file_bridge.go`](../control-plane/internal/gateway/file_bridge.go)
- Terminal bridge: [`control-plane/internal/gateway/bridge.go`](../control-plane/internal/gateway/bridge.go)
