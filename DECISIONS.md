# DECISIONS

## 23/05/26

- v0.1 uses Google Artifact Registry for container image distribution.
  Rationale: keep the image registry colocated with GKE in `us-central1` so workspace and control-plane image pulls stay in-region, reducing cross-region transfer risk and keeping GCP IAM-based access control for deploys. Terraform and deployment specs should reference `us-central1-docker.pkg.dev/...` image names.

- The Flutter package's web terminal integration expects consuming apps to add xterm.js script and stylesheet tags in their own `web/index.html`; the in-repo `demo_app` is the reference integration point for local testing.
  Rationale: `flutter/` is a package rather than a runnable app shell, so a package-local `web/index.html` would not be used by downstream consumers. Documenting the host-app requirement keeps the package integration explicit and avoids relying on package-local HTML that consumers never load.

- Terminal resize travels over the WebSocket mux as message type `0x05` with an 8-byte big-endian payload `[cols:uint32][rows:uint32]`.
  Rationale: the dedicated resize message keeps terminal control traffic distinct from data frames while reusing the same big-endian binary conventions as the mux header and the agent's `WindowSize` gRPC contract.
