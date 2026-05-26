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
- `exchangeFirebaseSession(firebaseIdToken)` calls `POST /v1/sessions/exchange/firebase`
- `refresh()` calls `POST /v1/sessions/refresh`
- it decodes the JWT `exp` claim so it can refresh before expiry
- it automatically schedules refreshes

For the browser-first product path, the package can now own the Firebase sign-in flow directly through [`flutter/lib/src/auth/cortado_firebase_auth.dart`](../flutter/lib/src/auth/cortado_firebase_auth.dart). `CortadoFirebaseAuthClient` signs the user into Cortado-managed Firebase Auth, exchanges the Firebase ID token for a normal Cortado session, and keeps the resulting `CortadoAuthSession` ready for the existing workspace and WebSocket clients.

`CortadoAuthSession` still supports the older API-key bootstrap route for headless or power-user flows. It does not mint API keys itself; API-key management now lives beside the session object so apps can mint personal keys after a normal first-party sign-in.

## Personal API Keys

After the host app finishes the first-party Firebase sign-in flow and has a normal `CortadoAuthSession`, it can manage long-lived personal API keys through `CortadoPersonalApiKeysClient`.

```dart
final authResult = await authClient.signInWithEmailPassword(
  email: 'user@example.com',
  password: 'correct horse battery staple',
);

final personalApiKeys = CortadoPersonalApiKeysClient(
  baseUrl: 'https://cortado.example.com',
  authSession: authResult.session,
);

final issued = await personalApiKeys.issue();
final listed = await personalApiKeys.list();
await personalApiKeys.revoke(issued.record.id);
```

The raw `issued.apiKey` value is returned only from the issuance call. Later list and revoke calls work only on stored metadata, matching the control-plane behavior for hashed-at-rest personal keys.

## First-Party Firebase Auth

The package now exposes two auth entry points for the zero-backend browser path:

- **Low-level helper**: `CortadoFirebaseAuthClient`
- **Drop-in UI**: `CortadoEmbeddedAuth`

### Low-level helper

Use `CortadoFirebaseAuthClient` when the host app wants to keep full control over its own UI:

```dart
final authClient = CortadoFirebaseAuthClient(
  baseUrl: 'https://cortado.example.com',
  firebaseOptions: const FirebaseOptions(
    apiKey: '...',
    appId: '...',
    messagingSenderId: '...',
    projectId: '...',
  ),
);

final result = await authClient.signInWithEmailPassword(
  email: 'user@example.com',
  password: 'correct horse battery staple',
);

final manager = WorkspaceManager(
  baseUrl: 'https://cortado.example.com',
  authSession: result.session,
);

final client = CortadoClient(
  baseUrl: 'https://cortado.example.com',
  authSession: result.session,
);
```

`signInWithGoogle()` uses the Firebase web popup flow. On native platforms, the host app should complete Firebase sign-in itself and then call `exchangeCurrentUser()` on the client instead.

### Drop-in UI

Use `CortadoEmbeddedAuth` when the host app wants a minimal package-owned auth form:

```dart
CortadoEmbeddedAuth(
  authClient: authClient,
  onAuthenticated: (result) {
    // Reuse result.session with WorkspaceManager and CortadoClient.
  },
)
```

The widget intentionally stays low-opinion: email, password, optional Google sign-in, basic busy/error state, and a callback that hands the authenticated Cortado session back to the host app.

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
- First-party Firebase auth client: [`flutter/lib/src/auth/cortado_firebase_auth.dart`](../flutter/lib/src/auth/cortado_firebase_auth.dart)
- Embedded auth widget: [`flutter/lib/src/auth/cortado_embedded_auth.dart`](../flutter/lib/src/auth/cortado_embedded_auth.dart)
- Personal API key client: [`flutter/lib/src/auth/cortado_personal_api_keys.dart`](../flutter/lib/src/auth/cortado_personal_api_keys.dart)
- WebSocket client: [`flutter/lib/src/cortado_client.dart`](../flutter/lib/src/cortado_client.dart)
- VFS notifier: [`flutter/lib/src/filesystem/vfs_notifier.dart`](../flutter/lib/src/filesystem/vfs_notifier.dart)
- File tree widget: [`flutter/lib/src/filesystem/cortado_file_tree.dart`](../flutter/lib/src/filesystem/cortado_file_tree.dart)
