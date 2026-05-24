## Feature 1.1 — Repository & Dev Environment Bootstrap

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
    repository_id = "cortado-${var.env}"
    format        = "DOCKER"
  }
  resource "google_artifact_registry_repository_iam_member" "control_plane_writer" {
    location   = google_artifact_registry_repository.cortado.location
    repository = google_artifact_registry_repository.cortado.repository_id
    role       = "roles/artifactregistry.writer"
    member     = "serviceAccount:${var.control_plane_sa_email}"
  }
  ```
- Keep the Artifact Registry repository in the same region as the GKE cluster by setting `location = var.region`. For dev this means `us-central1`, which keeps image pulls in-region and avoids unnecessary cross-region transfer.
- Run `terraform apply` and verify: cluster appears in GCP console and the regional registry is accessible.

**Key detail**: GKE Autopilot clusters provision slowly (5–15 minutes for first `terraform apply`). The `null_resource` for CRIU runs after cluster creation; it will fail silently if the `gcloud` binary isn't in `$PATH` at `terraform apply` time. Since you're running Terraform from inside the devcontainer (which has `gcloud` installed), this is fine. Add a comment in the `null_resource` to make this dependency explicit.

**Challenge**: The Workload Identity binding references a Kubernetes namespace and service account (`cortado-workspaces/workspace-sa`) that don't exist yet — they'll be created when you deploy the first workspace pod. Terraform will apply the IAM binding regardless (GCP accepts it even if the KSA doesn't exist yet), but `terraform plan` will show no drift, which is correct. Create the namespace and KSA in a Kubernetes manifest applied via `kubectl apply` (or via the `kubernetes` Terraform provider if you want to keep everything in Terraform — acceptable but adds provider complexity for v0.1).

---

### Task 1.1.5 — GKE namespace + KSA bootstrap for workspace pods
**What to do:**
- Create a Kubernetes manifest for the initial workspace namespace and service account:
  ```yaml
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
      iam.gke.io/gcp-service-account: cortado-workspace-agent-dev@cortado-ide.iam.gserviceaccount.com
      iam.gke.io/return-principal-id-as-email: "true"
  ```
- Apply it to the `cortado-dev` cluster with `kubectl apply -f ...`.
- Verify:
  ```bash
  kubectl get namespace cortado-workspaces
  kubectl get serviceaccount workspace-sa -n cortado-workspaces -o yaml
  ```
- Use `serviceAccountName: workspace-sa` in future workspace pod specs.

**Key detail**: Terraform already created the Google service account and the IAM binding that allows the Kubernetes service account principal `cortado-workspaces/workspace-sa` to act as that GSA. This task creates the Kubernetes-side objects that complete the Workload Identity link.

**Challenge**: A brand-new Autopilot cluster often shows `kubectl get nodes` as empty until you schedule a workload. That is expected and doesn't block namespace or service-account creation. The first actual workspace pod will trigger node provisioning.

---
