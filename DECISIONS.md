# DECISIONS

## 23/05/26

- v0.1 uses Google Artifact Registry for container image distribution.
  Rationale: keep the image registry colocated with GKE in `us-central1` so workspace and control-plane image pulls stay in-region, reducing cross-region transfer risk and keeping GCP IAM-based access control for deploys. Terraform and deployment specs should reference `us-central1-docker.pkg.dev/...` image names.

- The Flutter package's web terminal integration expects consuming apps to add xterm.js script and stylesheet tags in their own `web/index.html`; the in-repo `demo_app` is the reference integration point for local testing.
  Rationale: `flutter/` is a package rather than a runnable app shell, so a package-local `web/index.html` would not be used by downstream consumers. Documenting the host-app requirement keeps the package integration explicit and avoids relying on package-local HTML that consumers never load.

- Terminal resize travels over the WebSocket mux as message type `0x05` with an 8-byte big-endian payload `[cols:uint32][rows:uint32]`.
  Rationale: the dedicated resize message keeps terminal control traffic distinct from data frames while reusing the same big-endian binary conventions as the mux header and the agent's `WindowSize` gRPC contract.

- v0.1 keeps the control plane on Cloud Run and uses Direct VPC egress plus GKE Cloud DNS additive VPC scope for workspace-agent reachability.
  Rationale: this preserves the current release/deployment spec while giving the Cloud Run control plane a supported path to resolve and connect to headless workspace Services over private Pod IPs.

- Browser terminal sessions negotiate the `cortado-v1` WebSocket subprotocol explicitly.
  Rationale: browser WebSocket clients send a non-empty `Sec-WebSocket-Protocol` header for the terminal transport, and the control plane must echo that exact protocol during upgrade or the handshake fails before any terminal traffic starts.

- Workspace PVCs use the `cortado-workspace` StorageClass by default.
  Rationale: Task 2.1.1 bootstraps that class via Terraform in both environments, and the control plane now defaults to the same name so dynamically created workspace PVCs bind without extra per-environment overrides.

- The Cloud Run control plane discovers the GKE API endpoint and cluster CA via the Container API when explicit kubeconfig data is unavailable.
  Rationale: local development can keep using `KUBECONFIG`, but deployed Cloud Run instances do not have a kubeconfig file. Passing cluster identity env vars and resolving endpoint/CA through the Container API gives the workspace CRUD control plane a deployable Kubernetes client path without requiring in-cluster execution.
