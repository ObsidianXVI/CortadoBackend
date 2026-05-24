# Cortado: Release Timeline & Task Breakdown
## Solo Developer

---

## Reading This Document

Each **Release** is a shippable milestone producing something testable end-to-end.
Each **Feature** within a release is a coherent unit of functionality.
Each **Task** is a 2–6 hour block of work (~1–2 days at 3h/day).

**Task format:**
- What to build/configure/verify
- Key implementation details or decisions
- Foreseeable challenge(s) with mitigation notes

**On language choices**: The Flutter package and all UI-facing code is Dart. The backend has two Go components with specific justifications — the workspace agent needs direct PTY syscall access (`posix_openpt`, `ioctl TIOCSWINSZ`, `SysProcAttr`) that would require brittle `dart:ffi` reimplementation in Dart, and the control plane uses `client-go` (the reference Kubernetes client) and `httputil.ReverseProxy` for WebSocket tunneling. Everything else (API design, business logic, billing) could be Dart; these two are Go for hard technical reasons.

**On authentication**: A hardcoded dev-bypass token (`X-Cortado-Dev-Token: dev-bypass`) is in effect throughout v0.1 and the first half of v0.2. Real JWT auth is wired at the very end of v0.2, immediately before the v0.2 tag, so that the entire lifecycle feature set can be developed and debugged without auth noise. The bypass is gated on `CORTADO_ENV=development` and will fail to compile if that env var is set in the production Docker build.

**On infrastructure**: Every GCP resource is defined in Terraform. No `gcloud` commands are used for resource creation — only for authentication (`gcloud auth`) and querying (`gcloud ... describe/list`). A `null_resource` with `local-exec` bridges the few GKE features not yet in the Terraform provider.

**Realistic output at 3h/day**: deep-focus work yields about one medium-complexity task per session. Factor in ~20% debugging/research overhead not reflected in estimates.

---

## Releases at a Glance

| Release | Name | Weeks | Outcome |
|---------|------|-------|---------|
| v0.1 | Hello Terminal | 1–5 | Single workspace, PTY terminal, dev-bypass auth |
| v0.2 | Workspace Lifecycle | 6–9 | Create/stop/resume, real JWT auth at end |
| v0.3 | Cloud Filesystem | 10–14 | File tree, read/write, cloud-canonical mode |
| v0.4 | Language Intelligence | 15–19 | LSP completions, diagnostics, Dart support |
| v0.5 | AI Completion | 20–24 | Inline ghost text, codebase chat, RAG index |
| v0.6 | Local Mirror & Sync | 25–30 | Local daemon, bidirectional file sync |
| v0.7 | Port Forwarding | 31–34 | HTTP/WS port forward, Flutter web preview |
| v0.8 | Public Beta | 35–39 | Multi-tenant, Stripe billing, pub.dev publish |

Total: ~39 weeks (~9–10 months at 3h/day).

---

---

# RELEASE v0.1 — "Hello Terminal"
### Weeks 1–5 | ~75 hours
**Exit criterion**: A Flutter Web app imports the package, creates a workspace, and types into a live shell running in a GKE pod. All GCP resources exist as Terraform state. Dev environment is fully reproducible via `devcontainer up`.

---

## Feature 1.1 — Repository & Dev Environment Bootstrap
**Duration**: Week 1 (4 tasks, ~3 days)

### Task 1.1.1 — Monorepo scaffold and devcontainer
**What to do:**
- Create the Git monorepo with this layout:
  ```
  cortado/
  ├── agent/              # Go: workspace agent (PTY, gRPC server)
  ├── control-plane/      # Go: HTTP gateway, workspace orchestration
  ├── flutter/            # Dart: the pub package
  ├── proto/              # .proto files shared across Go and Dart
  ├── terraform/          # All GCP infrastructure as code
  │   ├── modules/
  │   │   ├── gke/
  │   │   ├── iam/
  │   │   └── storage/
  │   ├── envs/
  │   │   ├── dev/        # dev workspace: terraform.tfvars, backend.tf
  │   │   └── prod/
  │   └── main.tf
  ├── scripts/            # Utility scripts (not infrastructure creation)
  └── .devcontainer/
      ├── devcontainer.json
      └── Dockerfile
  ```
- Write `.devcontainer/Dockerfile`:
  ```dockerfile
  FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04

  # Go
  ARG GO_VERSION=1.23.4
  RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
      | tar -C /usr/local -xz
  ENV PATH="/usr/local/go/bin:${PATH}"

  # Dart + Flutter
  ARG FLUTTER_VERSION=3.27.0
  RUN git clone --depth 1 --branch ${FLUTTER_VERSION} \
      https://github.com/flutter/flutter.git /opt/flutter
  ENV PATH="/opt/flutter/bin:${PATH}"
  RUN flutter precache --web

  # Terraform
  ARG TF_VERSION=1.9.8
  RUN curl -fsSL https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_linux_amd64.zip \
      -o tf.zip && unzip tf.zip -d /usr/local/bin && rm tf.zip

  # Buf (proto generation)
  ARG BUF_VERSION=1.47.2
  RUN curl -fsSL https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-Linux-x86_64 \
      -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf

  # kubectl + helm + k9s + gcloud CLI
  RUN curl -fsSL https://dl.k8s.io/release/stable.txt | xargs -I{} \
      curl -fsSL https://dl.k8s.io/release/{}/bin/linux/amd64/kubectl \
      -o /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl

  RUN curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

  RUN curl -fsSL https://github.com/derailed/k9s/releases/latest/download/k9s_Linux_amd64.tar.gz \
      | tar -C /usr/local/bin -xz k9s

  RUN curl -fsSL https://sdk.cloud.google.com | bash -s -- --disable-prompts \
      --install-dir=/opt/google-cloud-sdk
  ENV PATH="/opt/google-cloud-sdk/bin:${PATH}"
  ```

- Write `.devcontainer/devcontainer.json`:
  ```json
  {
    "name": "cortado-dev",
    "build": { "dockerfile": "Dockerfile" },
    "features": {
      "ghcr.io/devcontainers/features/docker-outside-of-docker:1": {}
    },
    "postCreateCommand": "buf generate && flutter pub get --directory=flutter/",
    "remoteEnv": {
      "CORTADO_ENV": "development",
      "GOPATH": "/go"
    },
    "mounts": [
      "source=${localWorkspaceFolder},target=/workspace,type=bind"
    ],
    "forwardPorts": [8080, 9090, 9731],
    "extensions": [
      "golang.go",
      "dart-code.dart-code",
      "dart-code.flutter",
      "hashicorp.terraform"
    ]
  }
  ```

**Key detail**: Use **Docker-outside-of-Docker** (mounting the host Docker socket) rather than Docker-in-Docker. DinD requires `--privileged` which is rejected by most CI systems and many corporate environments. DooD mounts `/var/run/docker.sock` from the host — images built inside the devcontainer appear in the host's Docker daemon. This is a privilege escalation vector, which is acceptable for a solo dev project; document it explicitly if this ever becomes a team repo.

**Challenge**: `flutter precache --web` downloads ~500MB of web renderer artifacts. The devcontainer build will take 5–10 minutes on first run. This is once-per-machine (layers are cached). Pin all version `ARG`s exactly — if `FLUTTER_VERSION` drifts between developers, `buf generate` produces different Dart stubs (different Flutter SDK = different `dart` version = potentially different codegen output). The `ARG` pins in the Dockerfile are the source of truth.

---

### Task 1.1.2 — Buf proto toolchain
**What to do:**
- Initialize `proto/` with `buf.yaml`:
  ```yaml
  version: v2
  modules:
    - path: .
  lint:
    use: [DEFAULT]
  breaking:
    use: [FILE]
  ```
- Write `proto/buf.gen.yaml`:
  ```yaml
  version: v2
  plugins:
    - remote: buf.build/protocolbuffers/go:v1.34.2
      out: ../agent/gen
      opt: paths=source_relative
    - remote: buf.build/grpc/go:v1.4.0
      out: ../agent/gen
      opt: paths=source_relative,require_unimplemented_servers=false
    - remote: buf.build/grpc/dart:v1.0.1
      out: ../flutter/lib/src/gen
  ```
- Write the first proto: `proto/agent/v1/agent.proto` — just the package declaration and empty `WorkspaceAgentService` for now. Verify `buf generate` runs without errors and produces stubs in the correct locations.
- Add `buf generate` to the GitHub Actions workflow and fail CI if generated files are out of date (`buf generate && git diff --exit-code`).

**Key detail**: The `require_unimplemented_servers=false` option in the Go gRPC plugin prevents compilation failures when you add new RPC methods to the proto but haven't implemented them in Go yet. Without it, every proto change requires immediate Go implementation — too constraining early on. Remove this option before v0.8 to catch accidental unimplemented methods.

**Challenge**: The Dart gRPC plugin (`buf.build/grpc/dart`) generates `.pb.dart` files that import `package:grpc`. The Flutter package's `pubspec.yaml` must include `grpc: ^4.0.0` as a dependency. On Flutter Web, gRPC uses gRPC-Web (HTTP/1.1 transport) rather than true HTTP/2 gRPC. For the agent↔control-plane communication (server-side, Go↔Go), full HTTP/2 gRPC works fine. The Dart client uses the WebSocket mux (not raw gRPC) to talk to the control plane, so gRPC-Web limitations don't affect the terminal path — but be aware of this boundary if you ever expose gRPC directly to the Flutter client.

---

### Task 1.1.3 — Terraform: GCP project and IAM
**What to do:**
- Write `terraform/envs/dev/main.tf` (and mirror for `prod`):
  ```hcl
  terraform {
    required_providers {
      google = { source = "hashicorp/google", version = "~> 6.0" }
    }
    backend "gcs" {
      bucket = "cortado-tf-state-dev"
      prefix = "terraform/state"
    }
  }

  provider "google" {
    project = var.project_id
    region  = var.region
  }
  ```
- Enable required APIs via `google_project_service`:
  ```hcl
  locals {
    apis = [
      "container.googleapis.com",
      "artifactregistry.googleapis.com",
      "secretmanager.googleapis.com",
      "pubsub.googleapis.com",
      "bigquery.googleapis.com",
      "cloudresourcemanager.googleapis.com",
      "iam.googleapis.com",
    ]
  }
  resource "google_project_service" "apis" {
    for_each = toset(local.apis)
    service  = each.value
    disable_on_destroy = false
  }
  ```
- Define service accounts in `terraform/modules/iam/main.tf`:
  ```hcl
  resource "google_service_account" "control_plane" {
    account_id   = "cortado-control-plane"
    display_name = "Cortado Control Plane"
  }
  resource "google_service_account" "workspace_agent" {
    account_id   = "cortado-workspace-agent"
    display_name = "Cortado Workspace Agent"
  }
  # Minimal IAM bindings — expand per feature as needed
  resource "google_project_iam_member" "control_plane_container_dev" {
    project = var.project_id
    role    = "roles/container.developer"
    member  = "serviceAccount:${google_service_account.control_plane.email}"
  }
  ```
- Create the Terraform state bucket manually (the bootstrap problem — this is the one `gcloud` command you must run by hand):
  ```bash
  gcloud storage buckets create gs://cortado-tf-state-dev \
    --location=us-central1 --uniform-bucket-level-access
  ```
  Document this in `terraform/README.md` as the one-time bootstrap step.

**Key detail**: The Terraform state bucket cannot itself be managed by Terraform (circular dependency). Accept this single manual step. Everything from this point on is `terraform apply`. Add a `scripts/bootstrap.sh` that contains only this bucket creation command, clearly commented as "run once, never again."

**Challenge**: `google_project_service` for `container.googleapis.com` can take 60–120 seconds to propagate. If `terraform apply` proceeds to create the GKE cluster resource before the API is fully enabled, the cluster creation fails. Add `depends_on = [google_project_service.apis]` to all resources that depend on enabled APIs. Terraform's dependency graph handles this automatically if you use references (e.g., `google_project_service.apis["container.googleapis.com"]`), but explicit `depends_on` is clearer during early development.

---

### Task 1.1.4 — Terraform: GKE cluster and Artifact Registry
**What to do:**
- Add `terraform/modules/gke/main.tf`:
  ```hcl
  resource "google_container_cluster" "cortado" {
    name     = "cortado-${var.env}"
    location = var.region

    enable_autopilot = true

    release_channel {
      channel = "RAPID"  # Needed for latest Autopilot features (CRIU, etc.)
    }

    workload_identity_config {
      workload_pool = "${var.project_id}.svc.id.goog"
    }

    # CRIU (checkpoint/restore) requires a null_resource until
    # google_container_cluster supports it natively:
    # tracked at github.com/hashicorp/terraform-provider-google/issues/XXXX
  }

  # Workload Identity binding: Kubernetes SA → GCP SA
  resource "google_service_account_iam_member" "workspace_agent_wi" {
    service_account_id = var.workspace_agent_sa_id
    role               = "roles/iam.workloadIdentityUser"
    member = "serviceAccount:${var.project_id}.svc.id.goog[cortado-workspaces/workspace-sa]"
  }

  # null_resource for features not yet in the Terraform provider
  resource "null_resource" "enable_criu" {
    triggers = { cluster = google_container_cluster.cortado.id }
    provisioner "local-exec" {
      command = <<-EOT
        gcloud container clusters update ${google_container_cluster.cortado.name} \
          --region ${var.region} \
          --enable-checkpoint-restore \
          --project ${var.project_id}
      EOT
    }
  }
  ```
- Add Artifact Registry:
  ```hcl
  resource "google_artifact_registry_repository" "cortado" {
    location      = var.region
    repository_id = "cortado"
    format        = "DOCKER"
  }
  resource "google_artifact_registry_repository_iam_member" "control_plane_writer" {
    location   = google_artifact_registry_repository.cortado.location
    repository = google_artifact_registry_repository.cortado.name
    role       = "roles/artifactregistry.writer"
    member     = "serviceAccount:${var.control_plane_sa_email}"
  }
  ```
- Run `terraform apply` and verify: cluster appears in GCP console, registry is accessible.

**Key detail**: GKE Autopilot clusters provision slowly (5–15 minutes for first `terraform apply`). The `null_resource` for CRIU runs after cluster creation; it will fail silently if the `gcloud` binary isn't in `$PATH` at `terraform apply` time. Since you're running Terraform from inside the devcontainer (which has `gcloud` installed), this is fine. Add a comment in the `null_resource` to make this dependency explicit.

**Challenge**: The Workload Identity binding references a Kubernetes namespace and service account (`cortado-workspaces/workspace-sa`) that don't exist yet — they'll be created when you deploy the first workspace pod. Terraform will apply the IAM binding regardless (GCP accepts it even if the KSA doesn't exist yet), but `terraform plan` will show no drift, which is correct. Create the namespace and KSA in a Kubernetes manifest applied via `kubectl apply` (or via the `kubernetes` Terraform provider if you want to keep everything in Terraform — acceptable but adds provider complexity for v0.1).

---

## Feature 1.2 — Workspace Agent (PTY Core)
**Duration**: Weeks 1–3 (5 tasks, ~7 days)

### Task 1.2.1 — Proto definition: agent gRPC service
**What to do:**
- Write `proto/agent/v1/agent.proto`:
  ```protobuf
  syntax = "proto3";
  package agent.v1;
  option go_package = "github.com/your-org/cortado/agent/gen/agent/v1";

  service WorkspaceAgentService {
    rpc CreatePty(PtyRequest)            returns (PtyResponse);
    rpc StreamPty(stream PtyInput)       returns (stream PtyOutput);
    rpc Health(HealthRequest)            returns (HealthResponse);
  }

  message PtyRequest {
    uint32 cols  = 1;
    uint32 rows  = 2;
    string shell = 3;  // defaults to /bin/bash
    repeated string env = 4;
  }
  message PtyResponse { string pty_id = 1; }

  message PtyInput {
    string pty_id = 1;
    oneof payload {
      bytes      data   = 2;
      WindowSize resize = 3;
      int32      signal = 4;
    }
  }
  message PtyOutput {
    oneof payload {
      bytes data     = 1;
      int32 exit_code = 2;  // sent when process exits
    }
  }
  message WindowSize { uint32 cols = 1; uint32 rows = 2; }
  message HealthRequest {}
  message HealthResponse { string status = 1; }
  ```
- Run `buf lint` (must pass), then `buf generate`.

**Key detail**: `StreamPty` takes a stream of `PtyInput` rather than a single `PtyId` at open time. This means the first message on the stream is always a `PtyInput` with `pty_id` set and no `payload`. The Go server reads this first message to identify which PTY session to bind to. The alternative (a separate `OpenStream(pty_id)` RPC that returns a bidirectional stream) isn't expressible in proto3 — bidirectional streaming RPCs always start with the client's first message.

**Challenge**: Dart gRPC bidirectional streaming (`ClientCall`) has subtly different cancellation semantics from Go. When the Dart client calls `call.cancel()`, the Go server's `stream.Recv()` returns a non-EOF error (status `CANCELLED`). Your Go recv loop must handle this explicitly — don't treat it as an unexpected error worth logging loudly.

---

### Task 1.2.2 — PTY management in Go
**What to do:**
- Initialize `agent/` Go module: `go mod init github.com/your-org/cortado/agent`
- Add dependency: `github.com/creack/pty` (Go's canonical PTY library — this is the core reason the agent is Go).
- Implement `internal/pty/manager.go`:
  ```go
  package pty

  import (
      "errors"
      "os"
      "os/exec"
      "sync"
      "syscall"

      "github.com/creack/pty"
      "github.com/google/uuid"
  )

  type Session struct {
      ID  string
      ptm *os.File    // PTY master fd
      cmd *exec.Cmd
      mu  sync.Mutex
  }

  type Manager struct {
      sessions sync.Map
  }

  func (m *Manager) Create(shell string, cols, rows uint16, env []string) (*Session, error) {
      if shell == "" { shell = "/bin/bash" }
      cmd := exec.Command(shell)
      cmd.Env = append(os.Environ(), env...)
      cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

      ptm, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
      if err != nil { return nil, err }

      s := &Session{ID: uuid.NewString(), ptm: ptm, cmd: cmd}
      m.sessions.Store(s.ID, s)
      return s, nil
  }

  func (m *Manager) Write(id string, data []byte) error {
      s, ok := m.sessions.Load(id)
      if !ok { return errors.New("session not found") }
      _, err := s.(*Session).ptm.Write(data)
      return err
  }

  func (m *Manager) Read(id string, buf []byte) (int, error) {
      s, ok := m.sessions.Load(id)
      if !ok { return 0, errors.New("session not found") }
      return s.(*Session).ptm.Read(buf)
  }

  func (m *Manager) Resize(id string, cols, rows uint16) error {
      s, ok := m.sessions.Load(id)
      if !ok { return errors.New("session not found") }
      return pty.Setsize(s.(*Session).ptm, &pty.Winsize{Cols: cols, Rows: rows})
  }

  func (m *Manager) Kill(id string) {
      if s, ok := m.sessions.LoadAndDelete(id); ok {
          sess := s.(*Session)
          sess.cmd.Process.Kill()
          sess.ptm.Close()
      }
  }
  ```
- Write a unit test that spawns `bash`, sends `echo hello_cortado\n`, reads until it finds `hello_cortado` in the output, then kills the session.

**Key detail**: `EIO` from `ptm.Read()` is not an error — it's the normal signal that the slave PTY has no more writers (the shell exited). Your read loop in the gRPC handler must treat `syscall.EIO` as clean termination:
```go
n, err := sess.ptm.Read(buf)
if err != nil {
    if errors.Is(err, syscall.EIO) { return nil } // shell exited cleanly
    return err
}
```
This is the single most important correctness detail in the PTY code. Missing it causes the read goroutine to spin on errors after every shell exit, logging noise and leaking goroutines.

**Challenge**: `pty.StartWithSize` requires the shell binary to exist in the image. During local devcontainer testing, `/bin/bash` exists. In the workspace Docker image, verify the shell is present before attempting to start it — return a descriptive error (`"shell /bin/bash not found in image"`) rather than a raw `exec` error.

---

### Task 1.2.3 — gRPC server and StreamPty implementation
**What to do:**
- Implement `internal/server/agent_server.go` satisfying the generated `WorkspaceAgentServiceServer` interface.
- `StreamPty` implementation:
  ```go
  func (s *AgentServer) StreamPty(stream pb.WorkspaceAgentService_StreamPtyServer) error {
      // First message must identify the session
      first, err := stream.Recv()
      if err != nil { return err }
      sessionID := first.PtyId

      ctx := stream.Context()
      g, ctx := errgroup.WithContext(ctx)

      // PTY → gRPC: read output and send downstream
      g.Go(func() error {
          buf := make([]byte, 4096)
          for {
              n, err := s.ptyMgr.Read(sessionID, buf)
              if err != nil {
                  if errors.Is(err, syscall.EIO) { return nil }
                  return err
              }
              if err := stream.Send(&pb.PtyOutput{
                  Payload: &pb.PtyOutput_Data{Data: buf[:n]},
              }); err != nil { return err }
          }
      })

      // gRPC → PTY: receive input and write to PTY
      g.Go(func() error {
          for {
              msg, err := stream.Recv()
              if err != nil { return err }
              switch p := msg.Payload.(type) {
              case *pb.PtyInput_Data:
                  s.ptyMgr.Write(sessionID, p.Data)
              case *pb.PtyInput_Resize:
                  s.ptyMgr.Resize(sessionID, uint16(p.Resize.Cols), uint16(p.Resize.Rows))
              case *pb.PtyInput_Signal:
                  // Handle SIGINT (2), SIGTERM (15), etc.
                  // Find the shell's process group and signal it
              }
          }
      })

      return g.Wait()
  }
  ```
- Start the gRPC server on `:9090`. Add `CORTADO_AGENT_GRPC_PORT` env var.

**Key detail**: The two goroutines in `StreamPty` share a lifetime — if one exits (PTY read returns EIO after shell exit), the other should also stop. `errgroup` + a derived context handles this: when the PTY-read goroutine returns `nil` (EIO), `errgroup` cancels the context but doesn't propagate the nil return. The gRPC-receive goroutine, however, is blocked on `stream.Recv()` which doesn't respect context cancellation. Unblock it by calling `stream.Context().Done()` — but the stream's context is already the parent context, not the errgroup context. The cleanest fix: use a separate `context.CancelFunc` that closes a channel, and select between `stream.Recv()` (via a wrapper goroutine that sends to a channel) and the cancel channel.

**Challenge**: gRPC stream `Send()` is not goroutine-safe. Only one goroutine may call `Send()` at a time. In the `StreamPty` design above, only the PTY-read goroutine calls `Send()` (the other goroutine only calls `Recv()`), so this is safe. If you later add a separate "send exit code" path, ensure it goes through the same goroutine that owns `Send()`.

---

### Task 1.2.4 — Dockerfile for workspace agent
**What to do:**
- Write `agent/Dockerfile` as a multi-stage build:
  ```dockerfile
  FROM golang:1.23-alpine AS builder
  WORKDIR /build
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -ldflags="-s -w" -o cortado-agent ./cmd/agent

  FROM ubuntu:22.04
  RUN apt-get update && apt-get install -y \
      bash zsh git curl ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && useradd -m -u 1000 workspace
  COPY --from=builder /build/cortado-agent /usr/local/bin/cortado-agent
  EXPOSE 9090
  ENV CORTADO_ENV=production
  ENTRYPOINT ["/usr/local/bin/cortado-agent"]
  ```
- The Docker image build and push is defined as a GitHub Actions job (not a Terraform resource):
  ```yaml
  # .github/workflows/build-agent.yml
  - name: Build and push workspace agent
    run: |
      IMAGE="us-central1-docker.pkg.dev/${{ vars.GCP_PROJECT }}/cortado/workspace"
      docker build -t $IMAGE:${{ github.sha }} agent/
      docker push $IMAGE:${{ github.sha }}
  ```
- `CGO_ENABLED=0` is required for the static binary. Verify: `docker run --rm $IMAGE file /usr/local/bin/cortado-agent` should print `statically linked`.

**Key detail**: The `CORTADO_ENV=production` in the Dockerfile means any code guarded by `if os.Getenv("CORTADO_ENV") == "development"` will not run in production containers. The dev-bypass auth check is one such guard. This is belt-and-suspenders — the bypass should also check that the incoming token is literally `dev-bypass` and reject anything else, but the env var ensures it can never be reached in a production-built image.

**Challenge**: The Ubuntu base image adds ~80MB. Consider `debian:bookworm-slim` (~30MB) if you don't need Ubuntu-specific packages. The main constraint is that `bash`, `git`, and `curl` are needed for user shell sessions. `debian:bookworm-slim` has all of these available via `apt`. Switch before v0.3 when the image starts growing with language runtimes — the difference compounds.

---

### Task 1.2.5 — Terraform: Kubernetes manifests for workspace pod
**What to do:**
- Use the `kubernetes` Terraform provider (or plain YAML applied via `null_resource`) for the workspace pod's Kubernetes resources.

  For simplicity in v0.1, use plain YAML committed to `terraform/k8s/`:
  ```yaml
  # terraform/k8s/workspace-namespace.yaml
  apiVersion: v1
  kind: Namespace
  metadata:
    name: cortado-workspaces
  ---
  apiVersion: v1
  kind: ServiceAccount
  metadata:
    name: workspace-sa
    namespace: cortado-workspaces
    annotations:
      iam.gke.io/gcp-service-account: cortado-workspace-agent@PROJECT.iam.gserviceaccount.com
  ```

  Apply via `null_resource` in Terraform:
  ```hcl
  resource "null_resource" "k8s_base" {
    depends_on = [google_container_cluster.cortado]
    triggers   = { manifest_hash = filesha256("${path.module}/k8s/workspace-namespace.yaml") }
    provisioner "local-exec" {
      command = <<-EOT
        gcloud container clusters get-credentials ${google_container_cluster.cortado.name} \
          --region ${var.region} --project ${var.project_id}
        kubectl apply -f ${path.module}/k8s/workspace-namespace.yaml
      EOT
    }
  }
  ```
- For the test pod (to validate agent deployment), also add a `workspace-pod-test.yaml` that creates a single named pod using the workspace image from Artifact Registry. This is a one-off test artifact, not a permanent resource.

**Key detail**: `triggers = { manifest_hash = filesha256(...) }` means the `null_resource` re-runs when the YAML file changes. Without this, Terraform won't re-apply Kubernetes manifests after the initial apply. This is a known limitation of the `null_resource` pattern — the `kubernetes` Terraform provider is cleaner for this but adds provider setup overhead. Migrate to the `kubernetes` provider in v0.3 when you have more manifests.

**Challenge**: `gcloud container clusters get-credentials` modifies `~/.kube/config` on the machine running `terraform apply`. Inside the devcontainer, this is `~/.kube/config` in the container's home directory, not the host's. This means `kubectl` commands run from the devcontainer will work, but the host machine won't have cluster access unless you separately run `get-credentials` there. Document this — it causes confusion when switching between terminal and devcontainer.

---

## Feature 1.3 — Control Plane: WebSocket Gateway
**Duration**: Weeks 2–3 (4 tasks, ~5 days)

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
        image = "${var.region}-docker.pkg.dev/${var.project_id}/cortado/control-plane:${var.image_tag}"
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

**Key detail**: `image_tag` is a Terraform variable. In CI, `terraform apply -var="image_tag=$GITHUB_SHA"` deploys the specific image built in that pipeline run. This gives you a full audit trail: every deployment maps to a Git commit. Never use `:latest` as the image tag in Terraform.

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

## Feature 1.4 — Flutter Package: Terminal Widget
**Duration**: Weeks 4–5 (4 tasks, ~5 days)

### Task 1.4.1 — Package scaffold and WebSocket client
**What to do:**
- `flutter create --template=package cortado` inside `flutter/`
- Add to `pubspec.yaml`:
  ```yaml
  dependencies:
    web_socket_channel: ^3.0.0
    riverpod: ^2.5.0
    freezed_annotation: ^2.4.0
  dev_dependencies:
    build_runner: ^2.4.0
    freezed: ^2.4.0
    riverpod_generator: ^2.4.0
  ```
- Implement `CortadoClient`:
  ```dart
  class CortadoClient {
    final String baseUrl;
    // In v0.1: always sends X-Cortado-Dev-Token: dev-bypass
    // In v0.2: sends Authorization: Bearer {jwt}
    final String _devToken = 'dev-bypass';

    late WebSocketChannel _ws;
    final _frames = StreamController<MuxFrame>.broadcast();

    Future<void> connect(String workspaceId) async {
      final uri = Uri.parse('$baseUrl/v1/workspaces/$workspaceId/connect')
          .replace(scheme: 'wss');
      _ws = WebSocketChannel.connect(uri,
          protocols: ['cortado-v1'],
          headers: {'X-Cortado-Dev-Token': _devToken});
      await _ws.ready;
      _ws.stream.listen(_onFrame, onError: _onError, onDone: _onDone);
    }

    void _onFrame(dynamic raw) {
      final bytes = raw as Uint8List;
      final frame = MuxFrame.decode(bytes);
      _frames.add(frame);
    }

    void sendFrame(int channelId, int msgType, Uint8List payload) {
      _ws.sink.add(MuxFrame(channelId, msgType, payload).encode());
    }

    Stream<MuxFrame> framesForChannel(int channelId) =>
        _frames.stream.where((f) => f.channelId == channelId);
  }
  ```

**Key detail**: `WebSocketChannel.connect` on Flutter Web uses the browser's native WebSocket API, which does not support custom HTTP headers (the `headers` parameter is silently ignored on web). Passing `X-Cortado-Dev-Token` as a header will not work in the browser. For v0.1 in dev, pass the token as a query parameter instead: `?dev_token=dev-bypass`. The control plane middleware checks for either the header (for non-browser clients) or the query param (for browser clients).

**Challenge**: The `web_socket_channel` package throws `WebSocketChannelException` (not standard Dart `Exception`) on connection failure. Both `_ws.ready` (a Future) and `_ws.stream` can produce errors independently. Wrap `await _ws.ready` in try/catch AND add `onError` to the stream listener — missing either path gives silent connection failures that appear as a frozen terminal with no error message.

---

### Task 1.4.2 — Mux frame codec in Dart
**What to do:**
- Implement `MuxFrame`:
  ```dart
  class MuxFrame {
    final int channelId;  // uint16
    final int msgType;    // uint8
    final Uint8List payload;

    const MuxFrame(this.channelId, this.msgType, this.payload);

    Uint8List encode() {
      final bd = ByteData(7 + payload.length);
      bd.setUint16(0, channelId, Endian.big);
      bd.setUint8(2, msgType);
      bd.setUint32(3, payload.length, Endian.big);
      final out = bd.buffer.asUint8List();
      out.setRange(7, 7 + payload.length, payload);
      return out;
    }

    static MuxFrame decode(Uint8List bytes) {
      assert(bytes.length >= 7, 'Frame too short');
      final bd = ByteData.sublistView(bytes);
      final channelId = bd.getUint16(0, Endian.big);
      final msgType = bd.getUint8(2);
      final payloadLen = bd.getUint32(3, Endian.big);
      assert(bytes.length == 7 + payloadLen, 'Frame length mismatch');
      return MuxFrame(channelId, msgType,
          Uint8List.sublistView(bytes, 7, 7 + payloadLen));
    }
  }
  ```
- Endianness must match the Go side exactly (`binary.BigEndian`). Write a cross-language test: produce a known frame in Go (`channel=0x0001, type=0x01, payload=[0x41,0x42,0x43]`), hardcode its bytes, and assert that Dart's `decode` produces the same values.

**Challenge**: `Uint8List.sublistView(bytes, 7)` creates a *view* (no copy). Operations on this view affect the original buffer. If the caller modifies `bytes` after calling `decode`, the payload inside the returned `MuxFrame` is silently corrupted. For the terminal hot path (high-frequency frames), views are worth the risk. Add a comment: `// View, not copy — do not mutate source bytes after decode.`

---

### Task 1.4.3 — Terminal widget (xterm.js via HtmlElementView)
**What to do:**
- Download `xterm.js` v5.x and `xterm-addon-fit.js`, place in `flutter/web/js/`.
- Add to `flutter/web/index.html`:
  ```html
  <link rel="stylesheet" href="js/xterm.css"/>
  <script src="js/xterm.js"></script>
  <script src="js/xterm-addon-fit.js"></script>
  <script src="js/cortado_xterm.js"></script>
  ```
- Write `flutter/web/js/cortado_xterm.js`:
  ```javascript
  window.CortadoXterm = {
    _terms: {},
    init(container, id, onDataCallback) {
      const term = new Terminal({
        fontFamily: '"JetBrains Mono", "Fira Code", monospace',
        fontSize: 14,
        cursorBlink: true,
      });
      const fit = new FitAddon.FitAddon();
      term.loadAddon(fit);
      term.open(container);
      fit.fit();
      this._terms[id] = { term, fit };
      term.onData(data => onDataCallback(id, data));
      new ResizeObserver(() => fit.fit()).observe(container);
    },
    write(id, data) { this._terms[id]?.term.write(data); },
    getSize(id) {
      const t = this._terms[id]?.term;
      return t ? { cols: t.cols, rows: t.rows } : null;
    },
    dispose(id) {
      this._terms[id]?.term.dispose();
      delete this._terms[id];
    }
  };
  ```
- Implement `CortadoTerminal` widget using `HtmlElementView` and `dart:js_interop` to call the JS bridge.

**Key detail**: `ResizeObserver` on the container element automatically calls `fit.fit()` when the widget is resized by Flutter layout. The fit addon calculates the correct `cols` and `rows` from the container's pixel dimensions, then you must send a resize frame to the server. Wire: `ResizeObserver callback → JS calls Dart → Dart sends resize MuxFrame → server resizes PTY`.

**Challenge**: Flutter Web's CanvasKit renderer renders Flutter widgets on a canvas, and `HtmlElementView` creates a "platform view hole" in that canvas. In Flutter 3.19+ this generally works, but the hole's z-ordering can conflict with Flutter dialogs and overlays that render *above* the canvas. If you render a Flutter dropdown or tooltip over the terminal, it disappears behind the xterm canvas. Workaround: use Flutter's `Overlay` system for all IDE overlays (file tree popups, completion dropdowns) rather than regular `Stack` children.

---

### Task 1.4.4 — End-to-end smoke test
**What to do:**
- Manually verify the full chain: Flutter Web (Chrome) → WebSocket → Cloud Run control plane → gRPC → GKE workspace pod → PTY → shell.
- Type `echo hello_v0_1` in the terminal widget, see the output.
- Type `vim` — verify that vim's TUI renders correctly (tests proper PTY/VT100 handling).
- Type `python3` — verify an interactive REPL works (tests that the PTY handles input prompts correctly without echoing back incorrectly).
- Test resize: drag the terminal widget to a different size, run `tput cols` in the shell — should reflect the new width.
- Record the round-trip latency: type a character, measure time until it appears back on screen. Target: <150ms from `us-central1` with a Singapore client.

**Challenge**: The latency target may not be met from Singapore to `us-central1` (~200ms RTT). If it's consistently above 200ms, note it as a known issue for v0.1 and plan regional deployment (`asia-southeast1`) for v0.2. The latency measurement itself is done with Chrome DevTools Network tab (WebSocket frames tab shows send/receive timestamps per frame).

---

---

# RELEASE v0.2 — "Workspace Lifecycle"
### Weeks 6–9 | ~60 hours
**Exit criterion**: Workspaces can be created, listed, stopped, and resumed via the Flutter API. Scale-to-zero shuts down idle workspaces. Real JWT auth is wired at the very end of this release, immediately before tagging v0.2.

---

## Feature 2.1 — Workspace CRUD API
**Duration**: Week 6 (2 tasks, ~3 days)

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

## Feature 2.2 — Flutter Workspace Manager
**Duration**: Week 7 (2 tasks, ~3 days)

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

## Feature 2.3 — Basic Billing Events
**Duration**: Week 8 (2 tasks, ~3 days)

### Task 2.3.1 — Pub/Sub + BigQuery via Terraform
**What to do:**
- All infrastructure as Terraform:
  ```hcl
  resource "google_pubsub_topic" "usage_events" {
    name = "cortado-usage-events-${var.env}"
  }

  resource "google_bigquery_dataset" "billing" {
    dataset_id = "cortado_billing_${var.env}"
    location   = var.region
  }

  resource "google_bigquery_table" "usage_events" {
    dataset_id = google_bigquery_dataset.billing.dataset_id
    table_id   = "usage_events"
    schema     = file("${path.module}/schemas/usage_events.json")
    time_partitioning {
      type  = "DAY"
      field = "event_time"
    }
  }

  resource "google_pubsub_subscription" "usage_to_bq" {
    name  = "cortado-usage-to-bq-${var.env}"
    topic = google_pubsub_topic.usage_events.name
    bigquery_config {
      table            = "${var.project_id}.${google_bigquery_table.usage_events.dataset_id}.${google_bigquery_table.usage_events.table_id}"
      write_metadata   = true
    }
  }

  # Dead-letter topic for failed deliveries
  resource "google_pubsub_topic" "usage_events_dlq" {
    name = "cortado-usage-events-dlq-${var.env}"
  }
  ```
- Grant Pub/Sub service account BigQuery write access:
  ```hcl
  resource "google_bigquery_dataset_iam_member" "pubsub_writer" {
    dataset_id = google_bigquery_dataset.billing.dataset_id
    role       = "roles/bigquery.dataEditor"
    member     = "serviceAccount:service-${var.project_number}@gcp-sa-pubsub.iam.gserviceaccount.com"
  }
  ```

**Key detail**: `var.project_number` (not `project_id`) is needed for the Pub/Sub service account email. Add it as a Terraform variable: `data "google_project" "current" {}` and use `data.google_project.current.number`. This is not the same as `project_id` — forgetting this causes silent BigQuery write failures that are very difficult to diagnose.

---

### Task 2.3.2 — Usage event emission from agent + WAL
**What to do:**
- The workspace agent publishes a usage event to the `cortado-usage-events` Pub/Sub topic every 10 seconds while a PTY session is active.
- Event WAL: before publishing, append the event to `/workspace/.cortado/usage.wal` (a simple newline-delimited JSON file). After successful Pub/Sub acknowledgement, mark the event with `"published": true`. On agent startup, replay any unpublished events from the WAL.
- Grant workspace agent SA `pubsub.publisher` role:
  ```hcl
  resource "google_pubsub_topic_iam_member" "agent_publisher" {
    topic  = google_pubsub_topic.usage_events.name
    role   = "roles/pubsub.publisher"
    member = "serviceAccount:${google_service_account.workspace_agent.email}"
  }
  ```

**Challenge**: The WAL file at `/workspace/.cortado/` is on the persistent volume — it survives pod restarts. But if the workspace is permanently deleted (PVC deleted), the WAL is lost without replay. Add a final "flush" step in the delete flow: the control plane calls a `FlushUsageWAL` gRPC method on the agent before deleting the pod, which publishes all pending WAL events synchronously with a 10-second timeout.

---

## Feature 2.4 — Real JWT Authentication (End of v0.2)
**Duration**: Week 9 (3 tasks, ~4 days)

*Auth is implemented last in v0.2 so that all prior features were debugged without auth complexity. After these tasks, the dev-bypass is still available in dev environments but the production path requires real JWTs.*

### Task 2.4.1 — JWT issuance and JWKS endpoint
**What to do:**
- Generate an RSA-2048 keypair. Store the private key in GCP Secret Manager (Terraform manages the *resource*, not the value):
  ```hcl
  resource "google_secret_manager_secret" "jwt_private_key" {
    secret_id = "cortado-jwt-private-key-${var.env}"
    replication { auto {} }
  }
  resource "google_secret_manager_secret_iam_member" "control_plane_reader" {
    secret_id = google_secret_manager_secret.jwt_private_key.id
    role      = "roles/secretmanager.secretAccessor"
    member    = "serviceAccount:${var.control_plane_sa_email}"
  }
  ```
  The key *value* is added manually: `gcloud secrets versions add cortado-jwt-private-key-dev --data-file=private_key.pem`.
- Implement `POST /v1/sessions`: accepts `{api_key, user_id}`, validates API key against Firestore (bcrypt hash), returns `{access_token (JWT, 8h), refresh_token (opaque UUID, 30d)}`.
- JWT claims: `{sub: user_id, tid: tenant_id, exp, jti}`. Workspaces are authorized per-request (not embedded in the JWT).
- Expose `GET /.well-known/jwks.json` with the public key.

**Key detail**: API keys are stored hashed (`bcrypt.GenerateFromPassword(key, 12)`). The raw key is shown to the tenant once at creation time (via the dashboard, implemented in v0.8). For dev, generate a test API key manually and insert its bcrypt hash into Firestore directly. The control plane never stores or logs the raw key.

**Challenge**: bcrypt hash comparison takes ~100ms (by design, to resist brute force). This makes `POST /v1/sessions` slow. Cache the validation result in Dragonfly/Redis (key: hash of the API key, value: tenant_id, TTL 5 minutes). On cache hit, skip bcrypt. On cache miss, bcrypt and store in cache. This reduces p99 from 100ms to <5ms for repeat sessions.

---

### Task 2.4.2 — JWT validation middleware
**What to do:**
- Replace `DevBypassAuth` middleware with a chain: try JWT first, fall back to dev-bypass *only if* `CORTADO_ENV=development`.
  ```go
  func AuthMiddleware(jwks *keyfunc.JWKS) func(http.Handler) http.Handler {
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              // Dev bypass (compile-time guarded)
              if os.Getenv("CORTADO_ENV") == "development" {
                  if r.Header.Get("X-Cortado-Dev-Token") == "dev-bypass" {
                      next.ServeHTTP(w, injectDevContext(r))
                      return
                  }
              }
              // Real JWT validation
              tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
              token, err := jwt.Parse(tokenStr, jwks.Keyfunc)
              if err != nil || !token.Valid {
                  http.Error(w, "unauthorized", 401)
                  return
              }
              claims := token.Claims.(jwt.MapClaims)
              ctx := context.WithValue(r.Context(), ctxKeyTenantID, claims["tid"])
              ctx = context.WithValue(ctx, ctxKeyUserID, claims["sub"])
              next.ServeHTTP(w, r.WithContext(ctx))
          })
      }
  }
  ```
- Use `github.com/MicahParks/keyfunc` for automatic JWKS key rotation.

**Challenge**: The WebSocket upgrade (`/v1/workspaces/{id}/connect`) can't set `Authorization` headers from browsers (browser WebSocket API doesn't support custom headers). Pass the JWT as a query parameter: `?token={jwt}` and extract it in the middleware specifically for WebSocket upgrade requests. Validate the token the same way. Note: query parameters appear in server access logs — consider the JWT expiry window (8h is long for a URL-embedded token; use a short-lived connection token (5 min TTL) for WebSocket URLs specifically.

---

### Task 2.4.3 — JWT refresh in Flutter client + tag v0.2
**What to do:**
- `CortadoClient` stores the JWT and its `exp` claim.
- A background timer fires 5 minutes before expiry: calls `POST /v1/sessions/refresh` with the refresh token, receives a new JWT, stores it.
- Replace `X-Cortado-Dev-Token` with `Authorization: Bearer {jwt}` in all requests (the dev bypass remains as a fallback in `CORTADO_ENV=development`).
- Verify end-to-end with real JWTs: create workspace, open terminal, let it run for 9 hours, confirm the refresh fires and the session continues without interruption.
- Tag the release: `git tag v0.2.0 && git push --tags`.

**Challenge**: The refresh timer must survive Flutter app backgrounding (on mobile targets) and browser tab suspension (on web). Flutter Web's timer continues to run if the tab is active but is throttled by the browser when the tab is in the background. Handle this by also checking the JWT expiry on every request — if expired (tab was suspended), refresh synchronously before proceeding.

---

---

# RELEASE v0.3 — "Cloud Filesystem"
### Weeks 10–14 | ~75 hours
**Exit criterion**: Flutter package exposes a full virtual filesystem. Users browse, create, edit, delete files via a file tree widget. Files persist across stop/start cycles.

---

## Feature 3.1 — File API
**Duration**: Week 10 (3 tasks, ~4 days)

### Task 3.1.1 — Proto: filesystem operations
**What to do:**
- Extend `proto/agent/v1/agent.proto`:
  ```protobuf
  rpc ListDir(ListDirRequest)              returns (ListDirResponse);
  rpc ReadFile(ReadFileRequest)            returns (stream ReadFileChunk);
  rpc WriteFile(stream WriteFileChunk)     returns (WriteFileResponse);
  rpc DeletePath(DeletePathRequest)        returns (DeletePathResponse);
  rpc WatchFiles(WatchFilesRequest)        returns (stream FileEvent);
  ```
- `ReadFileChunk`: `{data: bytes, seq: int32, is_last: bool, checksum: bytes}`.
- `FileEvent`: `{path: string, type: FileEventType, checksum: bytes}` where `FileEventType = {CREATED, MODIFIED, DELETED, RENAMED}`.

---

### Task 3.1.2 — Implement filesystem operations in agent
**What to do:**
- `ListDir`: `os.ReadDir(path)` → return entries with name, size, isDir, modTime, permissions.
- `ReadFile`: open, read in 256KB chunks, stream. Compute xxHash64 and send in the last chunk.
- `WriteFile`: receive chunks to a temp file (`.cortado-tmp-{uuid}` in the same directory), verify xxHash64, `os.Rename` to target (atomic on Linux).
- `WatchFiles`: `fsnotify.NewWatcher()`, watch `/workspace` recursively (excluding `node_modules`, `.git`, `build`), debounce 50ms per path, stream events.

**Key detail**: Atomic rename via `os.Rename` only works if source and dest are on the same filesystem (same PVC mount). Since the temp file is in the same directory as the target, this is guaranteed. Never write directly to the target file — partial writes are visible to concurrent readers.

**Challenge**: `fsnotify` sends 2–4 events per save for most editors (write temp, rename, delete temp). Debouncing at 50ms collapses these. But compute the new checksum *after* debounce — reading the file during debounce may catch an intermediate state. After debounce, wait 10ms, then read and hash. This extra 10ms is imperceptible.

---

### Task 3.1.3 — HTTP file endpoints on control plane
**What to do:**
- `GET /v1/workspaces/{id}/files` with `?path=` query param → proxies to `ListDir`.
- `GET /v1/workspaces/{id}/files/content?path=` → proxies to `ReadFile` streaming, streams HTTP response body.
- `PUT /v1/workspaces/{id}/files/content?path=` → streams request body to `WriteFile`.
- `DELETE /v1/workspaces/{id}/files?path=` → proxies to `DeletePath`.
- WatchFiles events flow via the existing WebSocket mux on channel `0x0200`, opened by a new `Open` frame message type.

Add Terraform for the PVC:
```hcl
# PVC is created dynamically by control-plane (client-go)
# but the StorageClass must exist:
resource "null_resource" "storage_class_ssd" {
  depends_on = [null_resource.k8s_base]
  provisioner "local-exec" {
    command = "kubectl apply -f ${path.module}/k8s/storage-class-ssd.yaml"
  }
}
```

**Challenge**: Large file streaming through Cloud Run. Cloud Run has a 32MB in-memory request body limit by default. For `PUT` of large files, this is hit quickly. Set `--max-instances` and increase the memory limit in the Cloud Run Terraform resource, and proxy the file in chunks (stream the HTTP body directly to the gRPC `WriteFile` stream without buffering). Use `io.Pipe()` to bridge the HTTP body reader and the gRPC stream writer.

---

## Feature 3.2 — File Tree & Editor Widget
**Duration**: Weeks 11–12 (3 tasks, ~4 days)

### Task 3.2.1 — Virtual filesystem model in Dart
**What to do:**
- Implement normalized VFS state: `Map<String, VfsNode>` keyed by path (not nested tree).
- `VfsNode` is a `freezed` union: `VfsFile({path, name, size, modTime})` and `VfsDir({path, name, childPaths, expanded, loaded})`.
- `VfsNotifier` (Riverpod `AsyncNotifier`) loads directory contents lazily: only fetch children when a directory is first expanded.
- `VfsNotifier.applyEvent(FileEvent)` updates the flat map: adds/removes/updates the entry at the given path, and updates the parent directory's `childPaths` list.

**Key detail**: Store `childPaths` as a `List<String>` (paths), not `List<VfsNode>`. When a child changes, only the child's entry in the flat map changes — the parent's `childPaths` list is untouched. This means parent directories don't rebuild when child files are modified, which is the primary performance concern for a large file tree.

---

### Task 3.2.2 — File tree widget
**What to do:**
- `CortadoFileTree`: custom `ListView.builder` with indent-aware rows. Each row is a `FileTreeRow` widget showing an icon, name, and optional status indicator.
- Expand/collapse on tap (directories only). Load children on first expand (triggers `VfsNotifier.loadDirectory`).
- Context menu on secondary tap / long press: New File, New Folder, Rename (inline editing), Delete (with confirmation dialog).
- File status dots: modified (yellow), conflict (red) — populated by the sync system in v0.6.
- WatchFiles events update `VfsNotifier` in real-time.

**Challenge**: Inline rename (clicking F2 or selecting Rename from context menu replaces the file name label with a `TextField`). The `TextField` must auto-focus, select all text, and commit on Enter or blur. In Flutter, managing focus for transient inline editors requires a `FocusNode` with `requestFocus()` called in a `WidgetsBinding.addPostFrameCallback`. If you call `requestFocus()` during the build phase, Flutter ignores it.

---

### Task 3.2.3 — CodeMirror 6 editor widget
**What to do:**
- Embed CodeMirror 6 via `HtmlElementView` (same pattern as xterm.js).
- Add `web/js/cortado_editor.js` that initializes a CodeMirror instance with: basic setup, syntax highlighting (import language packages for Dart, JS, Python, Go, YAML, JSON), line numbers, bracket matching.
- Wire: file open → `GET /v1/workspaces/{id}/files/content` → set CodeMirror content.
- Wire: `Ctrl+S` → `PUT /v1/workspaces/{id}/files/content`.
- Modified indicator: compare CodeMirror content hash against last-saved hash. Show a dot in the tab title when different.
- Multi-tab: maintain a `List<OpenTab>` in a `TabsNotifier`. Max 15 open tabs (more than enough for v0.3).

**Challenge**: CodeMirror's `EditorView.dispatch` for setting content replaces the document — this resets scroll position and cursor. For re-loading a file (after an external change), preserve cursor position: save `{line, ch}` before dispatch and restore via a subsequent `dispatch({selection: EditorSelection.cursor(savedPos)})`.

---

## Feature 3.3 — Persistent Volume and Snapshots
**Duration**: Week 13 (2 tasks, ~3 days)

### Task 3.3.1 — PVC lifecycle in control plane
**What to do:**
- `WorkspacePodManager.Create` now also creates a PVC:
  ```go
  pvc := &corev1.PersistentVolumeClaim{
      ObjectMeta: metav1.ObjectMeta{
          Name: "ws-" + workspaceID,
          Namespace: "cortado-workspaces",
      },
      Spec: corev1.PersistentVolumeClaimSpec{
          StorageClassName: ptr("premium-rwo"),
          AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
          Resources: corev1.ResourceRequirements{
              Requests: corev1.ResourceList{
                  corev1.ResourceStorage: resource.MustParse("10Gi"),
              },
          },
      },
  }
  ```
- On `start`, use `VolumeClaimTemplates` → reuse the existing PVC by name.
- On permanent `delete`, delete the PVC explicitly (PVCs are not garbage-collected with pods by default in GKE).

**Challenge**: `ReadWriteOnce` means only one pod can mount the PVC at a time. The stop flow must confirm pod deletion before the start flow creates a new pod. Add a 30-second wait + retry loop in the start path: if PVC is `Bound` to a terminating pod, wait for it to release before scheduling the new pod.

---

### Task 3.3.2 — Workspace snapshots (restic to GCS via Terraform)
**What to do:**
- Terraform: create a GCS snapshot bucket:
  ```hcl
  resource "google_storage_bucket" "workspace_snapshots" {
    name          = "cortado-snapshots-${var.project_id}-${var.env}"
    location      = var.region
    storage_class = "NEARLINE"
    lifecycle_rule {
      condition { age = 30 }
      action    { type = "Delete" }
    }
  }
  resource "google_storage_bucket_iam_member" "agent_writer" {
    bucket = google_storage_bucket.workspace_snapshots.name
    role   = "roles/storage.objectCreator"
    member = "serviceAccount:${var.workspace_agent_sa_email}"
  }
  ```
- Add `restic` to the workspace Dockerfile.
- Add `CreateSnapshot(context.Context, *pb.SnapshotRequest) returns (*pb.SnapshotResponse)` to the agent gRPC service.
- Control plane calls `CreateSnapshot` in the stop flow with a 30-second timeout (fire-and-forget if timeout exceeded).

---

---

# RELEASE v0.4 — "Language Intelligence"
### Weeks 15–19 | ~75 hours
**Exit criterion**: LSP-powered completions, error squiggles, hover docs, and go-to-definition for Dart code.

---

## Feature 4.1 — LSP Gateway
**Duration**: Week 15 (3 tasks, ~4 days)

### Task 4.1.1 — Proto: LSP service
- Add to `agent.proto`:
  ```protobuf
  rpc OpenLSP(OpenLSPRequest)           returns (OpenLSPResponse);
  rpc StreamLSP(stream LSPMessage)      returns (stream LSPMessage);
  ```
- `OpenLSPRequest`: `{language: string}`.
- `LSPMessage`: `{data: bytes}` — raw JSON-RPC frames (Content-Length framing is stripped by the agent; the gateway sees raw JSON).

**Challenge**: The Dart language server is invoked as `dart language-server --protocol=lsp`. Add the Dart SDK to the workspace Docker image — this adds ~300MB but is required for Dart LSP. Use a build arg so non-Dart workspaces can skip this layer:
```dockerfile
ARG INCLUDE_DART_SDK=false
RUN if [ "$INCLUDE_DART_SDK" = "true" ]; then \
    wget https://storage.googleapis.com/.../dart-sdk.zip && ...; fi
```

---

### Task 4.1.2 — Agent-side LSP process manager
- Implement `LSPManager` in the Go agent: spawns `dart language-server --protocol=lsp` as a subprocess, wraps its stdin/stdout in a Content-Length frame parser/unparser, bridges to the gRPC `StreamLSP` bidirectional stream.
- Lazy start: spawn the language server process on first `OpenLSP` call, not at agent startup.
- Restart on crash: watch the process with `cmd.Wait()`, restart up to 3 times, emit a close event to the gRPC stream if all retries exhausted.

**Key detail**: The Content-Length framing parser must handle `\r\n` (CRLF) line endings — `bufio.Scanner` splits on `\n` and leaves a trailing `\r` in the parsed Content-Length value. Use `strings.TrimSpace` after scanning the header line.

---

### Task 4.1.3 — LSP routing in control plane + WS mux channel
- WebSocket mux channel range `0x0100–0x01FF` for LSP (one channel per language).
- On `Open` frame for an LSP channel: call `OpenLSP(language)` on the agent, establish `StreamLSP` gRPC stream, bind to WS mux channel.
- The gateway is a transparent proxy — it does not parse LSP JSON content.
- Increase mux max frame size for LSP channels to 4MB (completion lists for large projects can exceed the 16KB default).

---

## Feature 4.2 — Editor LSP Integration
**Duration**: Weeks 16–17 (4 tasks, ~5 days)

### Task 4.2.1 — CortadoLSPClient in Dart
- Full JSON-RPC 2.0 client over the WS mux LSP channel.
- Implement: `initialize`, `initialized`, `textDocument/didOpen`, `textDocument/didChange`, `textDocument/didClose`.
- Use `full` sync mode (`TextDocumentSyncKind.Full`) — send entire file content on every `didChange`.
- Subscribe to `textDocument/publishDiagnostics` notifications.
- Show "Language server starting..." indicator; dismiss when `initialized` received from server.

**Challenge**: `initialize` for the Dart LS takes 3–10 seconds. During this period, queue any requests (`completion`, `hover`) and flush them after `initialized`. A simple list-based queue (`List<PendingRequest>`) flushed in the `initialized` handler is sufficient.

---

### Task 4.2.2 — Completions in CodeMirror
- JS interop bridge: Dart registers `window._cortadoLSPRequest` callback; JS calls it when CodeMirror's completion source fires.
- Dart calls `textDocument/completion`, maps `CompletionItem[]` to CodeMirror `Completion[]`, resolves the Promise via `window._cortadoLSPResult(requestId, items)`.
- Debounce: trigger completion 150ms after last keystroke. Cancel in-flight request if cursor moves before response arrives.

**Challenge**: Completion latency: Dart LS can take 200–800ms for large projects. If the user types faster than completions arrive, verify the cursor position when results arrive matches the position when the request was sent — discard stale results.

---

### Task 4.2.3 — Diagnostics (publishDiagnostics)
- `CortadoLSPClient` exposes `Stream<Map<String, List<Diagnostic>>> diagnosticsStream`.
- In the CodeMirror JS bridge, call `setDiagnostics` (`@codemirror/lint`) when diagnostics arrive.
- Propagate to the file tree: add status dot on files with errors/warnings.
- `publishDiagnostics` replaces (not appends) — store as `_diagnostics[uri] = newList`, never `addAll`.

---

### Task 4.2.4 — Hover and go-to-definition
- `textDocument/hover` on 500ms mouse hover delay: render markdown tooltip in CodeMirror using `hoverTooltip`.
- Sanitize markdown HTML with `DOMPurify` before inserting into the DOM.
- `textDocument/definition` on Ctrl+click: open the target file in a new tab. For Dart SDK files (URI starts with `/usr/local/dart-sdk`), open read-only.

---

---

# RELEASE v0.5 — "AI Completion"
### Weeks 20–24 | ~75 hours
**Exit criterion**: AI inline ghost text and RAG-backed chat panel, powered by a per-workspace Qdrant index.

---

## Feature 5.1 — Codebase Indexing Pipeline
**Duration**: Week 20 (3 tasks, ~4 days)

### Task 5.1.1 — Tree-sitter chunker (Python microservice)
- Build `indexer/` Python service (Cloud Run Job, not a service).
- Tree-sitter chunking per language (Dart, Python, JS, Go). Fallback: 50-line windows with 10-line overlap.
- Terraform:
  ```hcl
  resource "google_cloud_run_v2_job" "indexer" {
    name     = "cortado-indexer-${var.env}"
    location = var.region
    template {
      template {
        service_account = var.indexer_sa_email
        containers {
          image = "${var.region}-docker.pkg.dev/${var.project_id}/cortado/indexer:${var.image_tag}"
        }
      }
    }
  }
  ```
- The control plane triggers indexer jobs via the Cloud Run Jobs API (on workspace create and on file change events from Pub/Sub).

**Challenge**: tree-sitter Python package requires compiled C extensions. Pin `tree-sitter==0.22.0` — the 0.23.x API is breaking. Build in Docker's build stage, not at runtime.

---

### Task 5.1.2 — Embedding pipeline + Qdrant sidecar
- Use `text-embedding-004` (Vertex AI) or `voyage-code-3` (Voyage AI). Batch in groups of 100 chunks.
- Deploy Qdrant as a sidecar container in the workspace pod spec (Terraform/YAML):
  ```yaml
  - name: qdrant
    image: qdrant/qdrant:v1.12.0
    resources:
      requests: { memory: "256Mi", cpu: "100m" }
    volumeMounts:
    - name: workspace-data
      mountPath: /qdrant/storage
      subPath: .cortado/qdrant
  ```
- Collection name: `ws-{workspaceID}`. Vector dimensions: 768 (`text-embedding-004`) or 1024 (`voyage-code-3`).

**Key detail**: The Qdrant sidecar stores its data in `/workspace/.cortado/qdrant` — a subdirectory of the persistent volume. It persists across workspace stop/start automatically.

---

### Task 5.1.3 — Incremental index updates via Pub/Sub
- File change events (from WatchFiles) are published to a new Pub/Sub topic `cortado-file-changes`.
- A `cortado-indexer-updater` Cloud Run service (long-running, not a Job) subscribes and processes updates:
  - `MODIFIED`/`CREATED`: re-chunk file, delete old Qdrant vectors for that path, insert new vectors.
  - `DELETED`: delete all Qdrant vectors where `metadata.file == path`.
- Debounce: batch file changes for the same workspace over 5-second windows.

Terraform:
```hcl
resource "google_pubsub_topic" "file_changes" {
  name = "cortado-file-changes-${var.env}"
}
resource "google_pubsub_subscription" "indexer_updater" {
  name  = "cortado-indexer-updater-${var.env}"
  topic = google_pubsub_topic.file_changes.name
  ack_deadline_seconds = 60
  push_config {
    push_endpoint = "${google_cloud_run_v2_service.indexer_updater.uri}/ingest"
  }
}
```

---

## Feature 5.2 — Inline AI Completion
**Duration**: Weeks 21–22 (3 tasks, ~4 days)

### Task 5.2.1 — Completion context builder + AI endpoint
- Control plane: `POST /v1/workspaces/{id}/ai/complete`.
- Context: code prefix (4KB) + suffix (1KB) + RAG top-3 chunks (Qdrant search query = last 3–5 lines before cursor).
- Call AI model with streaming (Claude Haiku or Gemini Flash for latency).
- Stream response as SSE (`data: {"token": "..."}\n\n`).

**Key detail**: The AI API key must never leave the control plane — it's loaded from Secret Manager at startup. The Flutter client calls the control plane's `/ai/complete` endpoint, which proxies to the AI provider. Terraform manages the secret resource:
```hcl
resource "google_secret_manager_secret" "ai_api_key" {
  secret_id = "cortado-ai-api-key-${var.env}"
  replication { auto {} }
}
```

---

### Task 5.2.2 — Streaming completion in Dart
- `CortadoAIService.streamCompletion(context)` returns `Stream<String>` of tokens.
- Uses `http.Client().send()` with `response.stream` for SSE parsing.
- Cancel the stream immediately on any keydown event (before the debounce timer — stale completions are worse than no completions).

---

### Task 5.2.3 — Ghost text in CodeMirror
- `Decoration.widget` with a `GhostTextWidget` that renders a `<span>` with the accumulated tokens, styled `color: rgba(128,128,128,0.6); pointer-events: none`.
- Tab → accept (insert ghost text at cursor, clear decoration).
- Escape → dismiss.
- Any other key → cancel the in-flight request, clear decoration.

**Challenge**: Multi-line ghost text uses a single `<pre>` element in the widget. Test vim-mode editors and editors with custom keymaps — many intercept Tab before CodeMirror sees it. If Tab doesn't accept ghost text, debug the event handler priority (`EditorView.domEventHandlers` vs `keymap.of`).

---

## Feature 5.3 — AI Chat Panel
**Duration**: Weeks 23–24 (3 tasks, ~4 days)

### Task 5.3.1 — Chat API endpoint with RAG
- `POST /v1/workspaces/{id}/ai/chat`.
- RAG: Qdrant search with the user's message → top-5 chunks.
- System prompt includes relevant chunks + current file content.
- Summarize conversation history every 10 turns to manage context window.
- Stream SSE response.
- Store conversation history in Firestore per workspace (persists across sessions).

---

### Task 5.3.2 — Chat panel widget
- `CortadoChatPanel`: scrollable message list + text input.
- Render AI markdown with `flutter_markdown`. Code blocks use `flutter_highlight`.
- "Insert at cursor" button on code blocks.
- `ValueNotifier<String>` for in-progress message — only rebuilds the streaming bubble, not the whole list.

---

### Task 5.3.3 — @-mention context injection
- `@filename` and `@SymbolName` mentions auto-complete from file tree + symbol index.
- Parse mentions before sending, inject file contents / symbol definitions into context.
- Truncate large files: if a mentioned file > 100 lines, include only the semantic unit (function/class) nearest to the current cursor.

---

---

# RELEASE v0.6 — "Local Mirror & Sync"
### Weeks 25–30 | ~90 hours
**Exit criterion**: `cortado-daemon` syncs a local directory to the cloud workspace bidirectionally in near real-time.

---

## Feature 6.1 — Local Daemon
**Duration**: Weeks 25–27 (5 tasks, ~7 days)

### Task 6.1.1 — cortado-daemon architecture
- Standalone Go binary, separate from the workspace agent.
- Local WebSocket proxy on `ws://127.0.0.1:9731` (listen on `127.0.0.1` only — never `0.0.0.0`).
- Responds with `Access-Control-Allow-Origin: *` for the CORS preflight (browser requires this).
- Service definition files shipped with the binary (launchd plist for macOS, systemd user unit for Linux, NSSM config for Windows).
- Install script at `https://install.cortado.dev/daemon` (Shell script served from GCS, path managed via Terraform `google_storage_bucket_object`).

**Key detail**: Use `modernc.org/sqlite` (pure-Go SQLite, no CGO) for the local state database (`~/.cortado/state.db`). This allows `CGO_ENABLED=0` for the daemon binary, enabling simple cross-compilation:
```makefile
build-daemon:
    GOOS=darwin  GOARCH=arm64  CGO_ENABLED=0 go build -o dist/cortado-daemon-macos-arm64  ./cmd/daemon
    GOOS=linux   GOARCH=amd64  CGO_ENABLED=0 go build -o dist/cortado-daemon-linux-amd64  ./cmd/daemon
    GOOS=windows GOARCH=amd64  CGO_ENABLED=0 go build -o dist/cortado-daemon-windows.exe ./cmd/daemon
```

**Challenge**: macOS TCC permissions. Watching directories outside of `~/Documents`, `~/Desktop`, and `~/Downloads` requires Full Disk Access on macOS 13+. The install script must guide the user to grant this. Alternatively, use `NSOpenPanel` to have the user explicitly pick the workspace root — this grants access without Full Disk Access. Implement the `NSOpenPanel` approach via a thin macOS helper (SwiftUI or Objective-C) bundled with the daemon for macOS only.

---

### Task 6.1.2 — Filesystem watcher (cross-platform)
- `fsnotify` watcher on user-configured workspace roots.
- Local state index: SQLite table `file_state(path TEXT PK, checksum TEXT, mod_time INT, synced_clock INT)`.
- Process events: compute xxHash64 before and after 50ms debounce; only emit op if checksum changed.
- Default excludes: `node_modules`, `.git`, `.dart_tool`, `build`, `Pods`, `.gradle`, `*.pyc`, `__pycache__`.

**Challenge**: inotify limit (Linux, default 8192 watches). Warn at 80% capacity. Print instructions to increase: `echo fs.inotify.max_user_watches=524288 | sudo tee /etc/sysctl.d/40-inotify.conf && sudo sysctl -p`.

---

### Task 6.1.3 — FileSync gRPC proto + stream
- New service in `proto/filesync/v1/filesync.proto` (separate from agent proto):
  ```protobuf
  service FileSyncService {
    rpc Sync(stream SyncMessage) returns (stream SyncMessage);
  }
  ```
- `SyncMessage.FileOp` carries `op_id` (UUID for dedup), `path`, `OpType`, `content` (for files <256KB) or `patch` (bsdiff for larger files), `local_clock`, `checksum`.
- Control plane acts as relay: receives ops from daemon, forwards to workspace agent, forwards agent ops back to daemon.

**Challenge**: Initial sync (first connect) requires a Merkle-tree diff to determine what to sync. Implement a simplified version: daemon sends a `StateVector` message containing `Map<path, checksum>` for all local files. Control plane compares against workspace state and replies with a list of `{path, direction}` ("send local→cloud" or "send cloud→local") for differing files. Full Merkle tree is over-engineered for v0.6 — this flat-map comparison is O(n) in file count but correct.

---

### Task 6.1.4 — Conflict detection and resolution
- Vector clock per file: `{localClock, remoteClock, lastSyncedClock}`.
- Conflict: both sides modified since `lastSyncedClock`. For text files: attempt `diff3` 3-way merge (call system `diff3` binary or use a pure-Go implementation). On merge failure: emit `ConflictNotice` on mux channel `0x0600` over the local daemon WebSocket bridge at `ws://127.0.0.1:9731`.
- Binary files: last-write-wins by `modTime`.
- Log all merge operations to `~/.cortado/merge.log`.

---

### Task 6.1.5 — Flutter package: daemon bridge
- `CortadoLocalDaemonBridge` connects to `ws://127.0.0.1:9731`.
- Exposes `startSync(localPath, workspaceId)`, `stopSync`, `getSyncStatus`.
- If daemon not running: show "Install Cortado Daemon" banner with download link.
- File tree shows sync status (spinner on syncing files, conflict icon on conflicted files) via the `VfsNotifier` state.

---

---

# RELEASE v0.7 — "Port Forwarding"
### Weeks 31–34 | ~60 hours
**Exit criterion**: User code on a bound port is accessible in the browser via an authenticated URL. Flutter web preview works.

---

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
- Validates JWT (same middleware as control plane), resolves pod DNS, proxies via workspace agent's port-proxy endpoint.
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

---

# RELEASE v0.8 — "Public Beta"
### Weeks 35–39 | ~75 hours
**Exit criterion**: Multi-tenant. Stripe billing charges users. Package published on pub.dev.

---

## Feature 8.1 — Multi-Tenancy
**Duration**: Week 35 (3 tasks, ~4 days)

### Task 8.1.1 — Tenant namespace isolation (Terraform)
```hcl
# Tenant namespace is created dynamically by control plane on tenant registration
# But the RBAC template and NetworkPolicy are static Terraform resources
resource "null_resource" "tenant_base_rbac" {
  provisioner "local-exec" {
    command = "kubectl apply -f ${path.module}/k8s/tenant-rbac-template.yaml"
  }
}
```
NetworkPolicy blocks cross-tenant pod communication. Verify with a test pod.

---

### Task 8.1.2 — Tenant self-service API
- `POST /v1/tenants`, `GET /v1/tenants/me`, `POST /v1/tenants/me/api-keys`.
- API key generation, bcrypt storage, namespace creation on registration.

---

### Task 8.1.3 — Stripe billing integration
All Stripe resources via Terraform (using the `stripe` Terraform provider):
```hcl
# terraform/modules/billing/main.tf
resource "stripe_product" "cpu" { name = "Cortado CPU (vCPU-seconds)" }
resource "stripe_price" "cpu_per_second" {
  product  = stripe_product.cpu.id
  currency = "usd"
  recurring {
    interval       = "month"
    usage_type     = "metered"
    aggregate_usage = "sum"
  }
  unit_amount_decimal = "0.001"  # $0.00001 per unit = $0.00001/vCPU-sec
}
```
- OpenMeter for event collection → Stripe Meters API for invoicing.
- `POST /v1/tenants/me/billing/checkout` → Stripe Checkout session.
- Test full billing loop: start workspace for 60 seconds, verify Stripe usage event received.

---

## Feature 8.2 — Package Polish & pub.dev Publish
**Duration**: Weeks 36–37 (3 tasks, ~4 days)

### Task 8.2.1 — API surface review
- Audit every exported symbol. Add `@Deprecated` for anything uncertain.
- Run `dart doc .` and fix all missing `///` doc comments.
- Ensure `dart analyze` produces zero warnings.

---

### Task 8.2.2 — Example app and README
- `example/` contains a minimal but complete IDE demo (~200 lines of Dart).
- `README.md`: 5-minute quickstart, ASCII architecture diagram, API reference link, badge showing pub.dev score.
- `flutter pub publish --dry-run` passes with zero errors.

---

### Task 8.2.3 — Publish
- Run full integration tests against the production GCP environment.
- Test Chrome, macOS desktop, Windows desktop.
- `flutter pub publish`.
- Announce: pub.dev, Flutter Discord, Hacker News.

**Challenge**: pub.dev automated scoring penalizes: missing `example/`, no `LICENSE` file, `dart analyze` warnings, test coverage <40%. Target 130/130 pub points. The most commonly missed: the `LICENSE` file must be in the repo root, not just linked from `pubspec.yaml`.

---

## Feature 8.3 — Hardening & Observability
**Duration**: Weeks 38–39 (3 tasks, ~4 days)

### Task 8.3.1 — OpenTelemetry (Terraform-managed sink)
```hcl
resource "google_monitoring_dashboard" "cortado" {
  dashboard_json = file("${path.module}/dashboards/cortado-overview.json")
}
resource "google_monitoring_alert_policy" "terminal_latency" {
  display_name = "Terminal latency p99 > 300ms"
  conditions {
    display_name = "latency"
    condition_threshold {
      filter          = "metric.type=\"custom.googleapis.com/cortado/terminal_rtt_p99\""
      comparison      = "COMPARISON_GT"
      threshold_value = 300
      duration        = "120s"
    }
  }
  notification_channels = [var.alert_notification_channel]
}
```
Instrument control plane and agent with OTel spans. Key metrics: terminal RTT, cold start duration, billing event lag, LSP request latency.

---

### Task 8.3.2 — Error handling audit
- Replace all `log.Fatal` in Go with structured `zap.Logger` + graceful error returns.
- Global error handler in Flutter package: surfaces errors to consuming devs via `CortadoErrorHandler` callback rather than throwing unhandled exceptions.

---

### Task 8.3.3 — Load test
- `k6` script simulating 50 concurrent workspace sessions (WS open, Open PTY frame, sustained keystroke stream).
- Target: terminal latency p50 <80ms, p99 <300ms at 50 concurrent sessions.
- Fix first bottleneck found (likely: Cloud Run concurrency limit or gRPC connection pool exhaustion).

---

---

## Summary Table

| Week | Release | Feature | Est. Hours |
|------|---------|---------|------------|
| 1 | v0.1 | Repo, devcontainer, Terraform IAM | 12 |
| 1–3 | v0.1 | Workspace agent (PTY) | 18 |
| 2–3 | v0.1 | Control plane gateway + dev bypass | 15 |
| 4–5 | v0.1 | Flutter terminal widget | 15 |
| 6–7 | v0.2 | Workspace CRUD + scale-to-zero | 18 |
| 8 | v0.2 | Basic billing events (Pub/Sub + BQ) | 9 |
| 9 | v0.2 | JWT auth (last task of v0.2) | 12 |
| 10 | v0.3 | File API + agent FS operations | 12 |
| 11–12 | v0.3 | File tree + CodeMirror editor | 12 |
| 13 | v0.3 | PVC lifecycle + restic snapshots | 9 |
| 15 | v0.4 | LSP gateway + agent LSP manager | 12 |
| 16–17 | v0.4 | Completions, diagnostics, hover, go-to-def | 18 |
| 20 | v0.5 | Tree-sitter indexer + Qdrant sidecar | 12 |
| 21–22 | v0.5 | AI inline completion (streaming) | 12 |
| 23–24 | v0.5 | AI chat panel + @-mentions | 12 |
| 25–27 | v0.6 | Local daemon (Go binary, cross-platform) | 21 |
| 28–30 | v0.6 | Sync protocol + conflict resolution | 18 |
| 31–32 | v0.7 | Port forward gateway (Terraform + Cloud Run) | 12 |
| 33–34 | v0.7 | Flutter web preview (Xvfb / build+serve) | 9 |
| 35 | v0.8 | Multi-tenancy (Terraform namespaces + RBAC) | 12 |
| 36–37 | v0.8 | Package polish, example app, pub.dev publish | 12 |
| 38–39 | v0.8 | OTel dashboards, error audit, load test | 12 |

**Total: ~291 hours over 39 weeks (~7.5h/week effective)**

*The remaining ~7.5h/week covers debugging production issues, reading unfamiliar API docs, Terraform state conflicts, and reviewing your own code after a day away.*
