# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.2 (Workspace Agent — PTY Core) → Task 1.2.5

## Status
DONE

## What was done last session
Completed Task 1.2.4 by adding the multi-stage agent Dockerfile, a GitHub Actions build/push workflow, and verifying the built container contains a statically linked `cortado-agent` binary.

## What was done this session
Added Terraform-managed Kubernetes bootstrap wiring under `terraform/k8s/` and both env roots. The base namespace/service account manifest is now re-applied by `null_resource.k8s_bootstrap`, and the one-off workspace pod test manifest is gated by `workspace_test_pod_enabled` and parameterized by an Artifact Registry image tag. Pushed the current workspace-agent image to `us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:781d613`, applied the dev Terraform changes, and verified the live cluster state: `workspace-sa` still has the expected Workload Identity annotations and `workspace-pod-test` reached `Ready` after Autopilot scaled up a node.

## Remaining work this session
None.

## Definition of done
- [x] `terraform/k8s/workspace-namespace.yaml` exists and is applied from Terraform
- [x] `terraform/k8s/workspace-pod-test.yaml` exists for validating agent deployment
- [x] Dev/prod env roots contain the Kubernetes bootstrap `null_resource`
- [x] `terraform validate` passes in `terraform/envs/dev`
- [x] `terraform validate` passes in `terraform/envs/prod`
- [x] `terraform apply` succeeds in `terraform/envs/dev`
- [x] Namespace/service account bootstrap remains correct in the dev cluster
- [x] The workspace test pod image exists in Artifact Registry and the pod reaches `Ready`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.3.1 — Control plane Go app skeleton + dev bypass
See _dev/docs/release_timeline.md §Feature 1.3 Task 1.3.1 for full spec

## Blocked on / decisions needed
None.
