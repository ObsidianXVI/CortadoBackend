# Flutter Package

The `flutter/` directory is a package, not a standalone app shell. Downstream IDE authors embed it and provide their own host app, window chrome, and web integration.

## Main Entry Points

- [`flutter/lib/src/cortado_client.dart`](../flutter/lib/src/cortado_client.dart)
- [`flutter/lib/src/cortado_auth_session.dart`](../flutter/lib/src/cortado_auth_session.dart)
- [`flutter/lib/src/workspace_manager.dart`](../flutter/lib/src/workspace_manager.dart)
- [`flutter/lib/src/cortado_workspace_provider.dart`](../flutter/lib/src/cortado_workspace_provider.dart)
- [`flutter/lib/src/filesystem`](../flutter/lib/src/filesystem)
- [`flutter/lib/src/mux_frame.dart`](../flutter/lib/src/mux_frame.dart)

## Auth Session

`CortadoAuthSession` manages the HTTP session with the control plane:

- `createSession(apiKey, userId)` calls `POST /v1/sessions`
- `refresh()` calls `POST /v1/sessions/refresh`
- it decodes the JWT `exp` claim so it can refresh before expiry
- it automatically schedules refreshes

`CortadoAuthSession` does not mint API keys itself. It expects a raw Cortado API key from a trusted bootstrap flow such as the control-plane Firebase-authenticated `POST /v1/api-keys` route.

The package supports both:

- bearer-token auth for normal HTTP requests
- dev-bypass auth when no session exists and the control plane is running in development mode

## WebSocket Client

`CortadoClient` opens the connect WebSocket to `/v1/workspaces/{id}/connect`.

Important details:

- it always requests the `cortado-v1` WebSocket subprotocol
- it can send the dev token in a header for non-browser clients
- browser clients pass auth through the query string because browser WebSocket APIs cannot set arbitrary headers
- incoming WebSocket binary messages are decoded as `MuxFrame`

### Mux channel usage

- terminal sessions: `0x0001`
- file sync: `0x0200`

## Workspace Manager

`WorkspaceManager` is the HTTP-facing workspace API wrapper. It provides:

- workspace create/list/get/delete and lifecycle transitions
- directory listing
- file read/write
- create directory
- rename path
- delete path
- polling-based workspace status streams

### Path normalization

The file APIs normalize paths before sending them to the control plane:

- `/`, `.`, and empty paths map to the workspace root
- leading slashes are stripped for API calls where the backend expects relative paths

Example:

```dart
manager.listWorkspaces();
manager.getWorkspace('ws-123');
manager.listDirectory('ws-123', path: '/');
manager.readFile('ws-123', path: '/lib/main.dart');
manager.renamePath('ws-123', oldPath: '/lib/main.dart', newPath: '/lib/app.dart');
```

## File Tree and VFS

The file tree is built from a virtual filesystem map managed by `VfsNotifier`.

### Why the VFS exists

The UI needs to:

- lazily fetch directories
- keep track of expansion state
- update in response to file-watch events
- remove stale descendants when a directory changes shape

`VfsNotifier` stores a map of normalized workspace paths to `VfsNode` values:

- `VfsFile` for files
- `VfsDir` for directories

The initial state contains only the root directory entry.

### File tree behavior

`CortadoFileTree`:

- opens the file-watch mux channel automatically by default
- loads the root directory on mount
- expands directories lazily on first open
- listens for `FileEvent` messages and updates the VFS map
- supports selection, context menus, and inline rename editing

## Data Flow Example

1. User expands `/lib`.
2. `CortadoFileTree` asks `VfsNotifier` to load `/lib` if it has not been loaded.
3. `VfsNotifier` calls `WorkspaceManager.listDirectory`.
4. The control plane proxies to the agent `ListDir` RPC.
5. The agent reads the actual workspace filesystem.
6. The response returns to the Flutter package and the file tree updates.

## Riverpod Scope

`CortadoWorkspaceProvider` injects the active `WorkspaceManager` and workspace ID into a scoped provider container so widgets deeper in the tree can call workspace-aware providers without threading those values through every constructor.

## Example Client Setup

```dart
final auth = CortadoAuthSession(baseUrl: 'https://cortado.example.com');
await auth.createSession(apiKey: apiKey, userId: userId);

final manager = WorkspaceManager(
  baseUrl: 'https://cortado.example.com',
  authSession: auth,
);

final client = CortadoClient(
  baseUrl: 'https://cortado.example.com',
  authSession: auth,
);
```

## Code References

- HTTP client: [`flutter/lib/src/workspace_manager.dart`](../flutter/lib/src/workspace_manager.dart)
- WebSocket client: [`flutter/lib/src/cortado_client.dart`](../flutter/lib/src/cortado_client.dart)
- VFS notifier: [`flutter/lib/src/filesystem/vfs_notifier.dart`](../flutter/lib/src/filesystem/vfs_notifier.dart)
- File tree widget: [`flutter/lib/src/filesystem/cortado_file_tree.dart`](../flutter/lib/src/filesystem/cortado_file_tree.dart)
