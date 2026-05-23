# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.1 (Repo & Dev Environment Bootstrap) → Task 1.1.5

## Status
DONE

## What was done last session
Completed the Terraform bootstrap for the dev environment: created the GKE Autopilot cluster and same-region Artifact Registry repository, fixed the unsupported CRIU Terraform step, installed the GKE kubectl auth plugin, retrieved cluster credentials, and verified cluster API access plus Artifact Registry visibility from the VM.

## What was done this session
Created `scripts/k8s/workspace-bootstrap.yaml` with the `cortado-workspaces` namespace and `workspace-sa` service account annotated for Workload Identity. Applied it to the `cortado-dev` cluster using `kubectl apply -f scripts/k8s/workspace-bootstrap.yaml`, then re-applied to confirm idempotent `unchanged` state. Verified with `kubectl get namespace cortado-workspaces` and `kubectl get serviceaccount workspace-sa -n cortado-workspaces -o yaml`; both required annotations are present:
- `iam.gke.io/gcp-service-account=cortado-workspace-agent-dev@cortado-ide.iam.gserviceaccount.com`
- `iam.gke.io/return-principal-id-as-email="true"`

## Remaining work this session
None.

## Definition of done
- [x] Namespace `cortado-workspaces` exists in the `cortado-dev` cluster
- [x] Kubernetes service account `workspace-sa` exists in namespace `cortado-workspaces`
- [x] `workspace-sa` is annotated with `iam.gke.io/gcp-service-account=cortado-workspace-agent-dev@cortado-ide.iam.gserviceaccount.com`
- [x] Optional Workload Identity helper annotation `iam.gke.io/return-principal-id-as-email="true"` is applied
- [x] `kubectl get serviceaccount workspace-sa -n cortado-workspaces -o yaml` shows the expected annotations
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.2.1 — Proto definition: agent gRPC service
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.1 for full spec

## Blocked on / decisions needed
None.
