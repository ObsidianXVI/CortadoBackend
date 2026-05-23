# Session Log

22/05/26 12:41 [FEAT] (e281076) `dev-pro-large` Bootstrapped the Cortado monorepo baseline by adding the pinned devcontainer, Buf workspace and generation config, initial proto skeleton, minimal Go module roots for the agent and control plane, Flutter package metadata and analysis exclusions, Terraform environment/module placeholders, repo ignore rules, and task tracking updates; verified proto generation, Go builds, Flutter dependency resolution, and Flutter analysis before creating the initial scaffold commit.
- A .devcontainer/Dockerfile
- A .devcontainer/devcontainer.json
- A .gitignore
- A proto/buf.yaml
- A proto/buf.gen.yaml
- A proto/agent/v1/agent.proto
- A agent/go.mod
- A agent/go.sum
- A control-plane/go.mod
- A flutter/pubspec.yaml
- A flutter/analysis_options.yaml
- A scripts/work.sh
- A CURRENT_RELEASE.md
- M CURRENT_TASK.md
- A README.md
22/05/26 13:33 [FEAT] (c24444c) `dev-pro-large` Added Terraform environment roots and modules for project API enablement, IAM service accounts, GKE Autopilot, Workload Identity, and Artifact Registry; documented and executed the one-time dev backend bucket bootstrap; validated both envs and produced a successful backend-backed dev plan while stopping short of live apply because that would provision billable infrastructure.
- A terraform/README.md
- A terraform/envs/dev/main.tf
- A terraform/envs/dev/outputs.tf
- A terraform/envs/dev/providers.tf
- A terraform/envs/dev/terraform.tfvars
- A terraform/envs/dev/variables.tf
- A terraform/envs/dev/versions.tf
- A terraform/envs/prod/main.tf
- A terraform/envs/prod/outputs.tf
- A terraform/envs/prod/providers.tf
- A terraform/envs/prod/terraform.tfvars
- A terraform/envs/prod/variables.tf
- A terraform/envs/prod/versions.tf
- A terraform/modules/gke/main.tf
- A terraform/modules/gke/outputs.tf
- A terraform/modules/gke/variables.tf
- A terraform/modules/iam/main.tf
- A terraform/modules/iam/outputs.tf
- A terraform/modules/iam/variables.tf
- A terraform/modules/iam/versions.tf
- A scripts/bootstrap.sh
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 16:26 [FEAT] `dev-pro-large` Updated the feature specs and technical report to replace Artifact Registry with Docker Hub for v0.1 image distribution, and recorded the deployment decision so future Terraform and Cloud Run tasks stay aligned with Docker Hub-based image references.
- M CURRENT_TASK.md
- M DECISIONS.md
- M _dev/docs/technical_report.md
- M _dev/features/feat-1-1.md
- M _dev/features/feat-1-2.md
- M _dev/features/feat-1-3.md
23/05/26 16:45 [FEAT] `dev-pro-large` Reverted the temporary Docker Hub decision, restored Artifact Registry across the Terraform and planning docs, and made the registry colocated with the `us-central1` GKE cluster so image storage and pulls stay in the same region.
- M AGENTS.md
- M CURRENT_TASK.md
- M DECISIONS.md
- M _dev/docs/technical_report.md
- M _dev/features/feat-1-1.md
- M _dev/features/feat-1-2.md
- M _dev/features/feat-1-3.md
- M terraform/envs/dev/main.tf
- M terraform/envs/dev/outputs.tf
- M terraform/envs/prod/main.tf
- M terraform/envs/prod/outputs.tf
- M terraform/modules/gke/main.tf
- M terraform/modules/gke/outputs.tf
- M terraform/modules/gke/variables.tf
23/05/26 17:25 [FEAT] `dev-pro-large` Recorded the post-bootstrap Kubernetes identity setup as the next actionable task by updating the active release/task pointers and adding a dedicated feature task for creating the `cortado-workspaces` namespace and `workspace-sa` KSA annotated to the existing workspace-agent GSA.
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
- M _dev/features/feat-1-1.md
23/05/26 04:08 [FEAT] (385323b) `dev-pro-large` Completed Task 1.1.5 by adding and applying a durable Kubernetes bootstrap manifest that creates the `cortado-workspaces` namespace and `workspace-sa` service account with Workload Identity annotations, then verified the live cluster object YAML contains both required annotations and advanced the release pointer to Feature 1.2 Task 1.2.1.
- A scripts/k8s/workspace-bootstrap.yaml
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 05:23 [FEAT] (2e51f1b) `dev-pro-large` Completed Task 1.2.1 by replacing the empty workspace-agent proto skeleton with the first PTY-oriented gRPC API, including unary create/health RPCs and a bidirectional terminal stream contract, then regenerated the Go and Dart stubs and verified lint/build/analyze status stayed green.
- M proto/agent/v1/agent.proto
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 05:44 [FEAT] (8cb210a) `dev-pro-large` Completed Task 1.2.2 by adding the first PTY session manager with creation, I/O, resize, kill, and process-group signaling support, plus a live shell-backed unit test and a missing-shell validation test to lock down the manager behavior before the gRPC server layer is added.
- M agent/go.mod
- M agent/go.sum
- A agent/internal/pty/manager.go
- A agent/internal/pty/manager_test.go
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 05:59 [FEAT] (cef4317) `dev-pro-large` Completed Task 1.2.3 by implementing the first runnable workspace-agent gRPC server, including PTY create/health unary RPCs, bidirectional PTY streaming with resize and signal handling, a `cmd/agent` entrypoint with graceful shutdown, and bufconn tests that exercise the real service surface.
- A agent/cmd/agent/main.go
- M agent/go.mod
- M agent/go.sum
- M agent/internal/pty/manager.go
- A agent/internal/server/agent_server.go
- A agent/internal/server/agent_server_test.go
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 06:11 [FEAT] (781d613) `dev-pro-large` Completed Task 1.2.4 by adding the multi-stage workspace-agent Dockerfile and a GitHub Actions workflow that builds and pushes the agent image to Artifact Registry, then verifying locally that the built container contains a statically linked `cortado-agent` binary and runs with `CORTADO_ENV=production`.
- A agent/Dockerfile
- A .github/workflows/build-agent.yml
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 06:37 [FEAT] (cf808f8) `dev-pro-large` Completed Task 1.2.5 by adding Terraform-managed Kubernetes bootstrap/test-pod manifests, wiring them into the env roots with null resources, pushing the current workspace-agent image into the dev Artifact Registry repository, applying the dev Terraform changes, and verifying that the dev cluster’s workspace bootstrap objects are correct and the test pod reaches Ready after Autopilot scales up capacity.
- M terraform/README.md
- M terraform/envs/dev/.terraform.lock.hcl
- M terraform/envs/dev/main.tf
- M terraform/envs/dev/terraform.tfvars
- M terraform/envs/dev/variables.tf
- M terraform/envs/dev/versions.tf
- M terraform/envs/prod/.terraform.lock.hcl
- M terraform/envs/prod/main.tf
- M terraform/envs/prod/terraform.tfvars
- M terraform/envs/prod/variables.tf
- M terraform/envs/prod/versions.tf
- A terraform/k8s/workspace-namespace.yaml
- A terraform/k8s/workspace-pod-test.yaml
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 05:47 [FEAT] (9d21e3e) `dev-pro-large` Completed Task 1.3.1 by scaffolding the control-plane HTTP service with chi routing, `/health`, and dev-bypass auth, then adding the reusable Cloud Run Terraform module and env-root wiring needed to deploy the control plane from the regional Artifact Registry repository once a container image is published.
- A control-plane/cmd/server/main.go
- A control-plane/go.sum
- A control-plane/internal/api/health.go
- A control-plane/internal/api/router.go
- A control-plane/internal/api/router_test.go
- A control-plane/internal/gateway/doc.go
- A control-plane/internal/middleware/auth.go
- A control-plane/internal/middleware/auth_test.go
- A control-plane/internal/store/doc.go
- A control-plane/internal/workspace/doc.go
- M control-plane/go.mod
- A terraform/modules/cloudrun/main.tf
- A terraform/modules/cloudrun/outputs.tf
- A terraform/modules/cloudrun/variables.tf
- A terraform/modules/cloudrun/versions.tf
- M terraform/envs/dev/main.tf
- M terraform/envs/dev/outputs.tf
- M terraform/envs/dev/terraform.tfvars
- M terraform/envs/dev/variables.tf
- M terraform/envs/prod/main.tf
- M terraform/envs/prod/outputs.tf
- M terraform/envs/prod/terraform.tfvars
- M terraform/envs/prod/variables.tf
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 06:03 [FEAT] (892561f) `dev-pro-large` Completed Task 1.3.2 by adding the first `client-go` workspace pod manager with headless-service creation, pod lifecycle watch hooks, and unit coverage around pod/service CRUD and DNS resolution, then provisioning Firestore plus datastore IAM access for the control plane in both Terraform env roots.
- M control-plane/go.mod
- M control-plane/go.sum
- A control-plane/internal/workspace/pod_manager.go
- A control-plane/internal/workspace/pod_manager_test.go
- M terraform/envs/dev/main.tf
- A terraform/envs/dev/firestore.tf
- M terraform/envs/prod/main.tf
- A terraform/envs/prod/firestore.tf
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 06:18 [FIX] (699a675) `dev-pro-large` Hardened the GitHub Actions agent image pipeline so `build-and-push` now chooses an available Google Cloud auth mode when secrets exist and otherwise performs a local Docker build without attempting Artifact Registry login or push, which prevents secretless runs such as Dependabot from failing at `google-github-actions/auth`.
- M .github/workflows/build-agent.yml
- M CURRENT_TASK.md
- M _dev/session_log.md
- M _dev/test_status.md
23/05/26 06:19 [FEAT] (53de819) `dev-pro-large` Completed Task 1.3.3 by implementing the control-plane WebSocket mux gateway endpoint with a reusable frame codec, a single-writer pump with keepalive handling, terminal-only channel dispatch scaffolding, and integration tests covering authenticated upgrades and unsupported-channel errors.
- M control-plane/cmd/server/main.go
- M control-plane/go.mod
- M control-plane/internal/api/router.go
- A control-plane/internal/gateway/handler.go
- A control-plane/internal/gateway/mux.go
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 06:51 [FEAT] (53de819) `dev-pro-large` Completed Task 1.3.4 by wiring the control-plane terminal bridge from the WebSocket mux to the workspace agent gRPC PTY stream, including cached per-workspace gRPC connections, default bridge-based connect-handler wiring, close-frame surfacing for setup failures, and bufconn-backed integration coverage for open/data/exit and connection reuse.
- M control-plane/go.mod
- M control-plane/go.sum
- A control-plane/internal/gateway/bridge.go
- M control-plane/internal/gateway/handler.go
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 07:00 [FEAT] (f5477ac) `dev-pro-large` Completed Task 1.4.1 by turning the placeholder Flutter package into a usable WebSocket client scaffold with a public `CortadoClient`, a mux-frame codec, platform-aware socket connectors for web versus native headers, and unit coverage for connect/auth/send-receive behavior plus codec compatibility bytes from the Go side.
- M flutter/pubspec.yaml
- M flutter/lib/cortado.dart
- A flutter/lib/src/cortado_client.dart
- A flutter/lib/src/mux_frame.dart
- A flutter/lib/src/platform_web_socket_connector.dart
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
23/05/26 07:12 [FEAT] `dev-pro-large` Completed Task 1.4.2 by finalizing the Flutter mux codec surface with exported protocol constants and stronger interoperability tests, advanced the tracked release pointer to Task 1.4.3, and recorded the unresolved package-web asset and terminal-resize protocol decisions needed before the xterm-based terminal widget can be finished.
- A DECISIONS_NEEDED.md
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
- M flutter/lib/src/mux_frame.dart
23/05/26 07:30 [FEAT] `dev-pro-large` Completed Task 1.4.3 by adding the web-only `CortadoTerminal` widget and platform bridge on the Flutter side, wiring demo-app host HTML to load xterm.js plus the local JS shim, extending the control-plane mux/bridge with a dedicated resize message that maps onto agent PTY resize RPCs, and recording the now-resolved web asset and resize transport decisions before advancing the release pointer to the end-to-end smoke test.
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
- M DECISIONS.md
- M DECISIONS_NEEDED.md
- M control-plane/internal/gateway/bridge.go
- M control-plane/internal/gateway/mux.go
- M demo_app/web/index.html
- M flutter/lib/cortado.dart
- M flutter/lib/src/mux_frame.dart
- A flutter/lib/src/terminal/cortado_terminal.dart
23/05/26 07:44 [FEAT] (ff299bf) `dev-pro-large` Built the Feature 1.4.4 repo-side smoke-test harness by replacing the demo app scaffold with a configurable web terminal shell around the local `cortado` package, adding resizable terminal UI and smoke-test instructions, and adding demo-app coverage for config parsing and connect flow while confirming that live Cloud Run verification remains blocked until the dev control-plane service is actually deployed and exposed in Terraform state.
- M CURRENT_TASK.md
- M demo_app/README.md
- M demo_app/lib/main.dart
- M demo_app/pubspec.yaml
- A demo_app/lib/src/terminal_smoke_config.dart
23/05/26 08:00 [FEAT] `dev-pro-large` Added the missing control-plane container and GitHub Actions image pipeline, fixed the smoke workspace manifest so `workspace-pod-test` now has the headless Service and label contract expected by the bridge, applied that manifest into the dev cluster, and documented the remaining architecture blocker that Cloud Run still cannot reach the in-cluster workspace DNS path without an explicit networking decision.
- M CURRENT_TASK.md
- M DECISIONS_NEEDED.md
- A .github/workflows/build-control-plane.yml
- A control-plane/Dockerfile
- M terraform/k8s/workspace-pod-test.yaml
23/05/26 10:35 [FEAT] `dev-pro-large` Implemented the Cloud Run private-networking path for Task 1.4.4 by wiring GKE additive VPC DNS plus Cloud Run direct VPC egress in Terraform and the control-plane resolver, deployed live dev control-plane/workspace images, fixed missing workspace smoke tooling and default PTY `TERM`, added stale gRPC connection redial logic in the control-plane bridge, and verified authenticated live terminal smoke against `workspace-pod-test` while documenting that final direct-browser verification remains blocked by Cloud Run IAM/public-access permissions and missing Chrome tooling on this workstation.
- M CURRENT_TASK.md
- M DECISIONS.md
- M DECISIONS_NEEDED.md
- M agent/Dockerfile
- M agent/internal/pty/manager.go
- M control-plane/cmd/server/main.go
- M control-plane/internal/gateway/bridge.go
- M control-plane/internal/workspace/pod_manager.go
- M terraform/envs/dev/main.tf
- M terraform/envs/dev/terraform.tfvars
- M terraform/envs/dev/variables.tf
- M terraform/envs/prod/main.tf
- M terraform/envs/prod/terraform.tfvars
- M terraform/envs/prod/variables.tf
- M terraform/modules/cloudrun/main.tf
- M terraform/modules/cloudrun/variables.tf
- M terraform/modules/gke/main.tf
- M terraform/modules/gke/variables.tf
23/05/26 11:02 [FIX] (pending) `dev-pro-large` Closed Task 1.4.4 by importing the existing Firestore database into Terraform state, installing local Chrome/Xvfb for browser automation, fixing browser WebSocket subprotocol negotiation and over-aggressive control-plane gRPC keepalive behavior, redeploying the control plane, and verifying the live Flutter smoke harness in Chrome for `echo hello_v0_1`, `python3`, `vim`, resize propagation, and browser-observed RTT against the Cloud Run dev service.
- M CURRENT_RELEASE.md
- M CURRENT_TASK.md
- M DECISIONS.md
- M _dev/session_log.md
- M _dev/test_status.md
- M control-plane/internal/gateway/bridge.go
- M control-plane/internal/gateway/bridge_test.go
- M control-plane/internal/gateway/handler.go
- M control-plane/internal/gateway/handler_test.go
- M terraform/envs/dev/terraform.tfvars
