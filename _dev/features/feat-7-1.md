## Feature 7.1 — Port Forward Gateway
**Duration**: Weeks 31–32 (3 tasks, ~4 days)

### Task 7.1.1 — Port detection in agent
- Add `ListPorts() returns (ListPortsResponse)` and `WatchPorts() returns (stream PortEvent)` to agent proto.
- Parse `/proc/net/tcp` and `/proc/net/tcp6` for `LISTEN` state ports. Hex-decode addresses (little-endian).
- Poll every 5 seconds for new bindings. Emit `PortEvent` on new port bound or released.
- Security: only expose ports 1024–65535. Block 9090 (agent gRPC) and any port already used by Cortado infrastructure.

---

### Task 7.1.2 — Port forward HTTP/WS gateway
- New Cloud Run service `cortado-portforward`. Terraform:
  ```hcl
  resource "google_cloud_run_v2_service" "portforward" {
    name     = "cortado-portforward-${var.env}"
    location = var.region
    ...
  }
  ```
- URL pattern: `https://portforward.cortado.dev/{workspaceId}/{port}/{path...}`.
- Validates JWT (same middleware as control plane), verifies the requested port through the workspace agent, resolves workspace service DNS, and proxies directly to the detected workspace port.
- WebSocket proxy: use `http.Hijack()` to get the raw TCP connection for WS tunneling.
- Wildcard TLS cert for `*.portforward.cortado.dev` via cert-manager + Let's Encrypt (Terraform `null_resource` to install cert-manager, then a `Certificate` CRD if using GKE Ingress).

**Challenge**: `httputil.ReverseProxy` does not handle WebSocket upgrades. Implement separate code paths for HTTP (`reverseProxy.ServeHTTP`) and WebSocket (`hijackAndTunnel`).

---

### Task 7.1.3 — Flutter web preview
- "Run Preview" button triggers: stream `flutter build web` output to a terminal session, then serve `build/web` via `dart pub global run dhttpd --port 8080`.
- Poll `WatchPorts` until port 8080 is bound, then show "Open Preview" button.
- Preview opens as an `IFrame` widget embedding the port-forward URL.
- Pass `X-Cortado-Preview-Build: flutter-web` header to trigger CORS policy in the portforward gateway that allows `Content-Security-Policy: frame-ancestors *`.

---
