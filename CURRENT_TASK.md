# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.1 (Repo & Dev Environment Bootstrap) → Task 1.1.5

## Status
IN PROGRESS

## What was done last session
Completed the Terraform bootstrap for the dev environment: created the GKE Autopilot cluster and same-region Artifact Registry repository, fixed the unsupported CRIU Terraform step, installed the GKE kubectl auth plugin, retrieved cluster credentials, and verified cluster API access plus Artifact Registry visibility from the VM.

## What was done this session
Verified that `kubectl get namespaces` works against `cortado-dev`, confirmed the cluster starts with zero nodes on Autopilot until a workload is scheduled, and identified the next infrastructure step: create the `cortado-workspaces` namespace plus the `workspace-sa` Kubernetes service account that is annotated to the existing `cortado-workspace-agent-dev` GSA.

## Remaining work this session
Create the `cortado-workspaces` namespace and `workspace-sa` Kubernetes service account in the dev cluster, annotate that KSA for Workload Identity with `cortado-workspace-agent-dev@cortado-ide.iam.gserviceaccount.com`, and verify the manifest state so later workspace pod deployments can use `serviceAccountName: workspace-sa`.

## Definition of done
- [ ] Namespace `cortado-workspaces` exists in the `cortado-dev` cluster
- [ ] Kubernetes service account `workspace-sa` exists in namespace `cortado-workspaces`
- [ ] `workspace-sa` is annotated with `iam.gke.io/gcp-service-account=cortado-workspace-agent-dev@cortado-ide.iam.gserviceaccount.com`
- [ ] Optional Workload Identity helper annotation `iam.gke.io/return-principal-id-as-email="true"` is applied
- [ ] `kubectl get serviceaccount workspace-sa -n cortado-workspaces -o yaml` shows the expected annotations
- [ ] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.2.1 — Proto definition: agent gRPC service
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.1 for full spec

## Blocked on / decisions needed
None.
