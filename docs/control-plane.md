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
- `POST /v1/sessions/exchange/firebase`
- `POST /v1/sessions/refresh`

### Personal API key routes

- `POST /v1/api-keys`
- `GET /v1/api-keys`
- `DELETE /v1/api-keys/{id}`

### Platform tenant routes

- `POST /v1/platform-tenants`
- `GET /v1/platform-tenants`
- `POST /v1/platform-tenants/{id}/api-keys`
- `GET /v1/platform-tenants/{id}/api-keys`
- `DELETE /v1/platform-tenants/{id}/api-keys/{keyID}`

### Development-only Firebase bootstrap route

- `POST /v1/dev/firebase/tenant-claim`

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
- `sub`: user or platform subject
- `act`: actor type (`user` or `platform`)
- standard expiration, issue time, and token ID claims

The middleware in [`control-plane/internal/middleware/auth.go`](../control-plane/internal/middleware/auth.go) also supports the development bypass when `CORTADO_ENV=development` and the request carries `X-Cortado-Dev-Token: dev-bypass` or `?dev_token=dev-bypass` for WebSocket upgrades.

The API key routes now accept a normal Cortado JWT access token first and fall back to Firebase-token auth second. The combined middleware lives in [`control-plane/internal/middleware/api_key_auth.go`](../control-plane/internal/middleware/api_key_auth.go). That keeps the first-party browser flow on normal Cortado sessions while preserving the older Firebase-token bootstrap path where it is still used.

In development, the control plane also exposes a Firebase-authenticated bootstrap route that verifies the Firebase token without requiring an existing tenant claim, then uses the Firebase Admin SDK to assign the configured development tenant claim to that Firebase user.

## Session Flow

`POST /v1/sessions` accepts:

```json
{
  "api_key": "acme-api-key",
  "user_id": "user-42"
}
```

`user_id` is required for personal API keys and must be omitted for platform API keys.

and returns:

```json
{
  "access_token": "eyJ...",
  "refresh_token": "b3c..."
}
```

The auth service looks up the API key against Firestore-backed records, resolves the tenant plus actor type, issues a short-lived access token, and stores the refresh token. `POST /v1/sessions/exchange/firebase` verifies a first-party Firebase ID token and returns the same Cortado session shape without requiring API-key bootstrap. `POST /v1/sessions/refresh` accepts the refresh token and returns a new access token.

If an API key record also stores a `userId`, `POST /v1/sessions` only succeeds when the caller-provided `user_id` matches that bound owner. The validation cache stores both tenant and user identity so repeated session creation preserves the same check on cache hits.

Platform API keys carry `act=platform` in the JWT and use a synthetic `sub` derived from the platform tenant. That keeps workspace and session flows compatible with the existing JWT structure without treating the platform's downstream end users as Cortado identities.

## API Key Issuance Flow

`POST /v1/api-keys` can be called with either a normal Cortado access token from the first-party session flow or a Firebase ID token from the older bootstrap path. It returns the raw Cortado API key once plus its stored metadata:

```json
{
  "apiKey": "cortado_...",
  "record": {
    "id": "key-123",
    "kind": "personal",
    "tenantId": "tenant-acme",
    "userId": "firebase-user-1",
    "revoked": false,
    "createdAt": "2026-05-25T05:00:00Z"
  }
}
```

The raw key is never stored in Firestore. The control plane stores only the bcrypt hash plus `tenantId`, `userId`, `revoked`, and `createdAt`. `GET /v1/api-keys` lists the authenticated user's keys for that tenant, and `DELETE /v1/api-keys/{id}` marks a matching key revoked.

## Platform Tenant Bootstrap

`POST /v1/platform-tenants` is authenticated with a normal Cortado user session and creates a distinct platform tenant owned by that bootstrap user. The route accepts:

```json
{
  "displayName": "Acme IDE"
}
```

It returns the stored platform tenant metadata:

```json
{
  "tenant": {
    "tenantId": "platform-123",
    "displayName": "Acme IDE",
    "kind": "platform",
    "createdAt": "2026-05-26T04:00:00Z",
    "updatedAt": "2026-05-26T04:00:00Z"
  }
}
```

Platform keys are then managed through `/v1/platform-tenants/{id}/api-keys`. Those keys are tenant-bound, store only a bcrypt hash at rest, and are intentionally not tied to a first-party Cortado end-user account. They exist for SaaS backends that already own their own user model.

## Development Firebase Bootstrap

`POST /v1/dev/firebase/tenant-claim` is only mounted when `CORTADO_ENV=development`. It accepts a Firebase ID token in `Authorization: Bearer ...` and optionally a JSON body:

```json
{
  "tenantId": "demo-tenant"
}
```

If `tenantId` is omitted, the route uses `CORTADO_FIREBASE_DEV_TENANT_ID`, falling back to `demo-tenant`. The control plane merges that tenant into the user's existing Firebase custom claims and returns the assignment:

```json
{
  "assignment": {
    "tenantId": "demo-tenant",
    "userId": "firebase-user-1"
  }
}
```

### Firebase env knobs

- `CORTADO_FIREBASE_PROJECT_ID` overrides the Firebase project used by the Admin SDK. It defaults to the same GCP project ID as the control plane.
- `CORTADO_FIREBASE_TENANT_CLAIM` overrides the custom claim name. It defaults to `tenant_id`.
- `CORTADO_FIREBASE_DEV_TENANT_ID` sets the default tenant assigned by the development-only bootstrap route. It defaults to `demo-tenant`.

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
- Firebase auth middleware: [`control-plane/internal/middleware/firebase_auth.go`](../control-plane/internal/middleware/firebase_auth.go)
- Dev bootstrap handler: [`control-plane/internal/api/dev_bootstrap.go`](../control-plane/internal/api/dev_bootstrap.go)
- API key service: [`control-plane/internal/auth/api_keys.go`](../control-plane/internal/auth/api_keys.go)
- Platform tenant handler: [`control-plane/internal/api/platform_tenants.go`](../control-plane/internal/api/platform_tenants.go)
- Platform tenant service: [`control-plane/internal/auth/platform_tenants.go`](../control-plane/internal/auth/platform_tenants.go)
- Workspace service: [`control-plane/internal/workspace/service.go`](../control-plane/internal/workspace/service.go)
- WebSocket mux: [`control-plane/internal/gateway/mux.go`](../control-plane/internal/gateway/mux.go)
- File bridge: [`control-plane/internal/gateway/file_bridge.go`](../control-plane/internal/gateway/file_bridge.go)
- Terminal bridge: [`control-plane/internal/gateway/bridge.go`](../control-plane/internal/gateway/bridge.go)
