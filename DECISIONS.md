# DECISIONS

## 23/05/26

- `DeletePath` is recursive for directories and may delete empty directories as well.
  Rationale: the file API needs a single delete operation that can remove directory subtrees without requiring callers to empty them first, and the control-plane docs should reflect the same recursive semantics for workspace path deletion.

- `WriteFile` defaults to auto-creating missing parent directories, while the API can expose an explicit opt-out that requires parents to already exist.
  Rationale: this keeps common editor-driven writes ergonomic while still allowing callers to request strict parent-exists behavior when they need it. The implementation detail remains intentionally abstracted at the documentation level.

- CodeMirror's Dart language remains a plain-text fallback rather than selecting a community-maintained Dart package by default.
  Rationale: the repo should avoid claiming a specific CodeMirror Dart integration that is not the canonical upstream path, and the editor bridge should continue to treat Dart as supported via fallback text rendering until a deliberate package choice is made.

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

- Workspace hibernation uses a 20-minute idle timeout by default, records terminal activity to Firestore at most once per minute, and applies a separate 30-minute stale-activity fallback scan.
  Rationale: this matches Task 2.1.2’s intended UX and cost controls while preventing Firestore write amplification and still catching agent-unreachable `RUNNING` workspaces that would otherwise leak.

- Frontend-facing Flutter package work should stay at the bare minimum needed for API/client integration, with minimal styling and widget polish.
  Rationale: this package is intended to be embedded by downstream products that provide their own polished frontends, so effort should focus on middleware/client correctness, input parsing and sanitization, low-bloat integration surfaces, and avoiding unnecessary UI dependencies.

## 24/05/26

- Workspace snapshot access may widen beyond `roles/storage.objectCreator`, but the effective permissions must stay scoped so each workspace agent can only access its own snapshot data.
  Rationale: restic's normal GCS backend flow needs object create/read/list/delete capabilities, but widening the bucket role is only acceptable when the final IAM or ACL model still prevents cross-workspace snapshot access.

- Dart SDK go-to-definition targets stay on the current read-only tab path without relaxing the workspace-root file API to read `/usr/local/dart-sdk/...` sources.
  Rationale: the user explicitly chose the lowest-change path. Keeping SDK definition tabs read-only without widening the file read surface avoids a new absolute-path exception in the agent/control-plane stack and stays closest to the existing blueprint and codebase behavior.

- Task 5.1.1 ships semantic tree-sitter chunking for Python, JavaScript, and Go, while Dart remains on the validated fallback window chunker for the first indexer cut.
  Rationale: the first verified grammar set is enough to close the semantic chunker milestone without introducing an unverified Dart grammar build path into the Docker image. The fallback contract is already part of the feature spec, so keeping Dart on that path minimizes moving parts while preserving correct indexing behavior.
