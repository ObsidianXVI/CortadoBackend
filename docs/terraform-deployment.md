# Terraform and Deployment

The infrastructure lives under [`terraform/`](../terraform). It is split into reusable modules and per-environment roots.

## Layout

- `terraform/modules/` contains reusable modules for GKE, Cloud Run, IAM, Redis, Secret Manager, and billing events.
- `terraform/envs/dev/` contains the development environment root.
- `terraform/envs/prod/` contains the production environment root.
- `terraform/k8s/` contains Kubernetes manifests that are applied during bootstrap.

## Environment Topology

Each environment provisions:

- a GKE Autopilot cluster
- a Cloud Run control-plane service
- Firestore support for workspace and auth state
- Redis for auth validation cache
- Pub/Sub for usage events
- Secret Manager for the JWT private key
- Artifact Registry for Docker images

The Terraform code labels all resources with:

```hcl
{ env = var.env, project = "cortado" }
```

## GKE

The GKE module creates:

- an Autopilot cluster named `cortado-<env>`
- Workload Identity enabled for the project
- a Google Artifact Registry repository in the same region
- an IAM binding that lets the workspace agent service account impersonate the Kubernetes service account used in `cortado-workspaces`

### Why this matters

The workspace pod needs cloud access without embedding static credentials. Workload Identity gives the pod access to GCP APIs using the Kubernetes service account mapping created by Terraform.

## Cloud Run Control Plane

The Cloud Run module packages the control plane image and wires in:

- Redis address for auth caching
- GKE cluster metadata for Kubernetes API access
- Secret Manager reference for the JWT private key
- Pub/Sub topic for usage events
- workspace namespace and cluster DNS domain

The control plane can then:

- authenticate users,
- create and stop workspaces,
- resolve workspace DNS,
- and bridge WebSocket traffic to the agent.

## Bootstrap Steps

There is a one-time bootstrap step for the Terraform state bucket:

```bash
./scripts/bootstrap.sh
```

That creates the dev state bucket used by Terraform backend initialization.

After the bucket exists, initialize Terraform from the environment root.

## Kubernetes Bootstrap

The environment roots use `null_resource` local-exec steps to apply Kubernetes manifests after the cluster exists. Those steps:

- fetch cluster credentials with `gcloud container clusters get-credentials`
- apply the workspace namespace and service-account manifest
- apply the workspace storage class
- optionally apply a test workspace pod

This is bootstrap glue, not application logic.

## Workspace Runtime Objects

The Kubernetes manifests and module defaults converge on these names:

- namespace: `cortado-workspaces`
- service account: `workspace-sa`
- storage class: `cortado-workspace`
- mount path: `/workspace`
- agent port: `9090`

The control plane and agent code assume those defaults unless overridden through environment variables.

## Image Flow

The docs and code intentionally use Artifact Registry in `us-central1` so workspace and control-plane image pulls remain in-region.

Example image name:

```text
us-central1-docker.pkg.dev/cortado-ide/cortado-dev/workspace:2026-05-23
```

## Relevant Files

- Environment roots: [`terraform/envs/dev/main.tf`](../terraform/envs/dev/main.tf), [`terraform/envs/prod/main.tf`](../terraform/envs/prod/main.tf)
- GKE module: [`terraform/modules/gke/main.tf`](../terraform/modules/gke/main.tf)
- Cloud Run module: [`terraform/modules/cloudrun/main.tf`](../terraform/modules/cloudrun/main.tf)
- IAM module: [`terraform/modules/iam/main.tf`](../terraform/modules/iam/main.tf)
- README: [`terraform/README.md`](../terraform/README.md)
