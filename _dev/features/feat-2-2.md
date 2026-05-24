## Feature 2.2 — Flutter Workspace Manager

### Task 2.2.1 — WorkspaceManager + status polling
**What to do:**
- Implement `WorkspaceManager`:
  ```dart
  class WorkspaceManager {
    Future<Workspace> create({required String image, WorkspaceResources? resources}) async { ... }
    Future<void> stop(String id) async { ... }
    Future<void> start(String id) async { ... }
    Stream<WorkspaceStatus> watchStatus(String id) { ... }
  }
  ```
- `watchStatus` polls `GET /v1/workspaces/{id}` every 3 seconds while status is `CREATING` or `STARTING`, then drops to every 30 seconds when `RUNNING`.
- Use `freezed` for `Workspace` and `WorkspaceStatus` data classes.
- Implement `CortadoWorkspaceProvider` (an `InheritedNotifier` wrapping a Riverpod provider) so child widgets can call `CortadoWorkspaceProvider.of(context).workspaceId`.

**Challenge**: The polling loop must cancel when the widget is disposed (memory leak otherwise). Use Riverpod's `ref.onDispose` to cancel the `StreamSubscription`. A common mistake: the `StreamController` backing `watchStatus` is never closed when the provider is disposed, leading to a `StreamController` leak that causes a Dart analyzer warning `Unclosed instance of type 'StreamController'` in debug mode.

---

### Task 2.2.2 — Reconnection after cold start
**What to do:**
- When `CortadoClient` detects a broken WebSocket (stream done or error), it should:
  1. Show "Reconnecting..." overlay on `CortadoTerminal`
  2. Call `workspaceManager.start(workspaceId)` (no-op if already `RUNNING`)
  3. Poll `watchStatus` until `RUNNING` (with exponential backoff: 2s, 4s, 8s, max 15s)
  4. Re-call `client.connect(workspaceId)`
  5. Re-send `Open` frame on terminal channel
  6. Remove overlay, show "--- Workspace resumed ---" in xterm
- Inject the resume banner: `xterm.write('\r\n\x1b[33m--- Workspace resumed ---\x1b[0m\r\n')` (yellow text).

**Challenge**: Re-open races. The client reconnects and immediately sends an `Open` frame, but the new shell process hasn't started yet (agent just restarted). Implement a small retry on the `Open` frame: if the agent returns an error on `CreatePty`, wait 500ms and retry up to 5 times before showing a permanent error state.

---
