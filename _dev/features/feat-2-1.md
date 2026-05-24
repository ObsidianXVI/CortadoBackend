## Feature 2.1 — Workspace CRUD API

### Task 2.1.1 — Workspace CRUD endpoints
**What to do:**
- `POST /v1/workspaces` → creates pod + PVC (Terraform-managed PVC storage class), returns `{id, status: "CREATING"}` immediately (202 Accepted).
- `GET /v1/workspaces/{id}` → returns current status from Firestore.
- `GET /v1/workspaces` → lists workspaces for current tenant (from context, injected by auth middleware — dev-bypass injects `"dev-tenant"`).
- `POST /v1/workspaces/{id}/start` → creates pod for a stopped workspace (PVC already exists).
- `POST /v1/workspaces/{id}/stop` → sends DELETE to pod, marks `STOPPING` → `STOPPED`.
- `DELETE /v1/workspaces/{id}` → deletes pod + PVC, marks `DELETED`.

Add Terraform for the PVC storage class configuration:
```hcl
# PVC is created dynamically by the control plane (client-go),
# but the StorageClass must exist first
resource "null_resource" "storage_class" {
  depends_on = [null_resource.k8s_base]
  provisioner "local-exec" {
    command = "kubectl apply -f ${path.module}/k8s/storage-class.yaml"
  }
}
```

**Key detail**: The background pod-watcher goroutine (using `client-go` `SharedIndexInformer`) updates Firestore when pod phase transitions. Specifically: `Pending → Running` triggers a Firestore write of `status: "RUNNING"`, and pod deletion triggers `status: "STOPPED"`. This is eventually consistent — a client polling `GET /v1/workspaces/{id}` will see `CREATING` for up to 2-3 minutes. This is expected and communicated to the Flutter client as a loading state.

**Challenge**: Pod deletion in Kubernetes is asynchronous (finalizers, grace period). After `kubectl delete pod`, the pod enters `Terminating` state for up to 30 seconds (configurable via `terminationGracePeriodSeconds`). During this period, the control plane should show `STOPPING`, not `STOPPED`. The pod watcher must distinguish between `Terminating` (phase is still `Running` but `DeletionTimestamp` is set) and fully deleted (watch event type `DELETED`).

---

### Task 2.1.2 — Scale-to-zero: idle detection and hibernation
**What to do:**
- Add idle tracking to the workspace agent. Record `lastActivityAt` timestamp (updated on every PTY write). Expose via a new gRPC method `GetIdleStatus() returns (IdleStatus)`.
- In the control plane, a background goroutine polls `GetIdleStatus` on all `RUNNING` workspaces every 5 minutes. If idle > `CORTADO_IDLE_TIMEOUT_MINUTES` (default 20), call the workspace's stop flow.
- Belt-and-suspenders: also scan Firestore for workspaces where `lastActiveAt` hasn't been updated for >30 minutes, regardless of agent status (covers OOM-killed pods where the agent can't report).

**Key detail**: Store `lastActiveAt` in Firestore (updated by the control plane when it receives PTY data frames via the WebSocket mux). Don't rely solely on the agent's in-memory timestamp — if the agent crashes, the control plane loses visibility. Every PTY data frame flowing through the gateway should update `lastActiveAt` in Firestore (batch updates, max once per minute to avoid Firestore write costs).

**Challenge**: A workspace running a long build job (no PTY input for 30+ minutes) looks idle by the keystroke metric. Add CPU-based override: the agent's `GetIdleStatus` also reports CPU usage over the last 60 seconds. If CPU > 5%, report not-idle regardless of keystroke time. Read CPU from `/proc/stat` (simple subtraction of idle ticks between two samples).

---
