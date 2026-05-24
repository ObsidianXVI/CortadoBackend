## Feature 1.3 — Control Plane: WebSocket Gateway

### Task 1.3.1 — Control plane Go app skeleton + dev bypass
**What to do:**
- Initialize `control-plane/` Go module.
- Use `chi` router: `github.com/go-chi/chi/v5`.
- Structure:
  ```
  control-plane/
  ├── cmd/server/main.go
  ├── internal/
  │   ├── api/            # HTTP handlers
  │   ├── middleware/      # auth, logging, CORS
  │   ├── workspace/       # lifecycle logic
  │   ├── gateway/         # WebSocket + gRPC bridge
  │   └── store/           # DB interface (Firestore for now)
  ```
- Implement the dev-bypass middleware first:
  ```go
  // internal/middleware/auth.go
  func DevBypassAuth(next http.Handler) http.Handler {
      if os.Getenv("CORTADO_ENV") != "development" {
          // In production, this middleware is a no-op pass-through;
          // real auth middleware (added in v0.2 Task 2.1.1) replaces it.
          return next
      }
      return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
          token := r.Header.Get("X-Cortado-Dev-Token")
          if token != "dev-bypass" {
              http.Error(w, "missing dev bypass token", http.StatusUnauthorized)
              return
          }
          // Inject a fake tenant/user context for downstream handlers
          ctx := context.WithValue(r.Context(), ctxKeyTenantID, "dev-tenant")
          ctx = context.WithValue(ctx, ctxKeyUserID, "dev-user")
          next.ServeHTTP(w, r.WithContext(ctx))
      })
  }
  ```
- Add `GET /health` → `{"status": "ok", "env": "development"}`.
- Terraform for Cloud Run deployment:
  ```hcl
  # terraform/modules/cloudrun/main.tf
  resource "google_cloud_run_v2_service" "control_plane" {
    name     = "cortado-control-plane-${var.env}"
    location = var.region

    template {
      service_account = var.control_plane_sa_email
      containers {
        image = "${var.region}-docker.pkg.dev/${var.project_id}/cortado-${var.env}/cortado-control-plane:${var.image_tag}"
        env {
          name  = "CORTADO_ENV"
          value = var.env == "dev" ? "development" : "production"
        }
        env {
          name  = "GCP_PROJECT"
          value = var.project_id
        }
      }
    }
    # Allow unauthenticated (Cloud Run IAM) — app-level auth is handled internally
    # Remove this in v0.2 for production
  }

  resource "google_cloud_run_v2_service_iam_member" "public" {
    project  = var.project_id
    location = google_cloud_run_v2_service.control_plane.location
    name     = google_cloud_run_v2_service.control_plane.name
    role     = "roles/run.invoker"
    member   = "allUsers"
  }
  ```

**Key detail**: `image_tag` is a Terraform variable. In CI, `terraform apply -var="image_tag=$GITHUB_SHA"` deploys the specific image built in that pipeline run. Use the regional Artifact Registry host `${var.region}-docker.pkg.dev` and a repository name aligned with the environment, such as `cortado-${var.env}`, so Cloud Run and GKE pull images from the same region as the cluster. This gives you a full audit trail: every deployment maps to a Git commit. Never use `:latest` as the image tag in Terraform.

**Challenge**: Cloud Run requires the container to start and be listening on `$PORT` within 4 minutes or it's killed. Your `main.go` must read `os.Getenv("PORT")` (not hardcoded 8080). A startup that initializes DB connections, loads config, and starts the HTTP server must all complete in under 4 minutes — this is easy to satisfy for a Go HTTP server but becomes relevant in v0.4 when startup includes gRPC connection pooling.

---

### Task 1.3.2 — Workspace pod manager (client-go) + Terraform Firestore
**What to do:**
- Add `k8s.io/client-go` to the control plane.
- Implement `WorkspacePodManager`:
  - `Create(workspaceID, image string, cpu, memGB float64) error`
  - `Delete(workspaceID string) error`
  - `GetStatus(workspaceID string) (corev1.PodPhase, error)`
  - `GetServiceDNS(workspaceID string) string` — returns `{workspaceID}.cortado-workspaces.svc.cluster.local`
- Always create a headless `Service` alongside the pod. The service's `selector` matches `cortado/workspace-id: {id}`. The agent is then reachable at the stable DNS name regardless of pod restarts.
- For the database, use Firestore (free tier, zero setup). Terraform:
  ```hcl
  resource "google_firestore_database" "cortado" {
    project     = var.project_id
    name        = "(default)"
    location_id = "us-central1"
    type        = "FIRESTORE_NATIVE"
  }
  resource "google_project_iam_member" "control_plane_firestore" {
    project = var.project_id
    role    = "roles/datastore.user"
    member  = "serviceAccount:${var.control_plane_sa_email}"
  }
  ```

**Key detail**: Pod creation is async. `Create()` should return as soon as the API server accepts the pod spec (200ms), not wait for the pod to be `Running` (potentially minutes). A background goroutine using `client-go`'s `cache.NewSharedIndexInformer` watches pod status changes and updates Firestore. This is the correct pattern for a Kubernetes controller.

**Challenge**: The control plane (Cloud Run) needs to reach the GKE API server. Cloud Run services run on Google's infrastructure but are not inside your VPC by default. Connecting to a private GKE cluster from Cloud Run requires **VPC Connector** (Terraform: `google_vpc_access_connector`). For a public GKE cluster (acceptable for dev), the API server is internet-accessible and `client-go` uses the cluster's public endpoint. Add VPC Connector for production in v0.8.

---

### Task 1.3.3 — WebSocket mux protocol
**What to do:**
- Implement `GET /v1/workspaces/{id}/connect` as a WebSocket upgrade.
- Frame format (binary, big-endian):
  ```
  [channel_id: uint16][msg_type: uint8][payload_len: uint32][payload: bytes]
  ```
- Channel ranges: `0x0001–0x00FF` = terminal sessions, `0x0100–0x01FF` = LSP (later), `0x0200` = file sync (later).
- Message types: `0x01` = Data, `0x02` = Open, `0x03` = Close, `0x04` = Error, `0xFF` = Ping.
- Implement the write pump (single goroutine owning all `ws.WriteMessage` calls):
  ```go
  type MuxConn struct {
      ws      *websocket.Conn
      writeCh chan []byte   // buffered, capacity 64
      done    chan struct{}
  }

  func (c *MuxConn) startWritePump() {
      defer c.ws.Close()
      for {
          select {
          case frame := <-c.writeCh:
              c.ws.WriteMessage(websocket.BinaryMessage, frame)
          case <-c.done:
              return
          }
      }
  }
  ```
- For v0.1, only dispatch channel `0x0001` (single terminal session).

**Key detail**: `gorilla/websocket` panics if two goroutines call `WriteMessage` concurrently. The write pump pattern above is the standard fix. Set `writeCh` capacity to 64 frames — if the pump is slow and the channel fills, callers should drop (not block) to avoid cascade stalls. Log a metric when drops occur.

**Challenge**: WebSocket ping/pong keepalive. The browser will close idle WebSocket connections after ~30-60 seconds (varies by browser and network). Set a `SetPongHandler` and send pings every 20 seconds from the write pump. In Flutter, respond to ping frames automatically (the `web_socket_channel` package does this). On the Go side, `ws.SetReadDeadline(time.Now().Add(60 * time.Second))` and reset it on each pong.

---

### Task 1.3.4 — Bridge: WebSocket mux channel ↔ gRPC agent stream
**What to do:**
- When an `Open` frame arrives on channel `0x0001`:
  1. Get workspace pod DNS from `WorkspacePodManager.GetServiceDNS(workspaceID)`
  2. Dial gRPC: `grpc.NewClient(podDNS+":9090", grpc.WithTransportCredentials(insecure.NewCredentials()))` — insecure for v0.1, mTLS in v0.2
  3. Call `StreamPty` on the agent
  4. Start two goroutines: WS→gRPC forwarder, gRPC→WS forwarder
- Cache the `*grpc.ClientConn` per workspace ID — don't re-dial on every frame.
- Add latency instrumentation (even without OTel yet): `time.Since(receivedAt)` logged on each gRPC send.

**Key detail**: Use `grpc.WithKeepaliveParams` to prevent the gRPC connection from being closed by GKE's idle timeout:
```go
kaParams := keepalive.ClientParameters{
    Time:                30 * time.Second,
    Timeout:             10 * time.Second,
    PermitWithoutStream: true,
}
grpc.NewClient(addr, grpc.WithKeepaliveParams(kaParams), ...)
```
Without this, the connection is silently dropped after ~10 minutes of idle and the next keystroke gets a `transport is closing` error.

**Challenge**: The gRPC connection is established to the pod's headless service DNS. If the pod hasn't started yet (still `Pending`), the DNS doesn't resolve and gRPC dial fails. The `grpc.WithBlock()` option combined with `grpc.WaitForReady(true)` will retry until the pod is reachable — but this blocks the WebSocket handler goroutine indefinitely. Better: dial without `WithBlock` (non-blocking) and let the first `StreamPty` call fail-fast if not ready. Surface the error as a close frame on the WS channel so the Flutter client can show "workspace starting..."

---
