# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.1 (Repo & Dev Environment Bootstrap) → Task 1.1.3 + 1.1.4

## Status
IN PROGRESS

## What was done last session
Added Terraform env roots for `dev` and `prod`, IAM and GKE modules, bootstrap documentation, and the one-time dev state bucket bootstrap script. Verified `terraform init -backend=false` and `terraform validate` for both envs, created the dev state bucket, reinitialized the dev backend against GCS, and produced a successful dev `terraform plan`.

## What was done this session
Reverted the temporary Docker Hub decision back to Artifact Registry across the tracked context files and restored the Terraform configuration to provision a same-region Artifact Registry repository alongside GKE.
Removed the unsupported CRIU post-create `gcloud` step from Terraform after live apply showed that the current `gcloud container clusters update` command does not accept `--enable-checkpoint-restore`.
Ran live `terraform apply` for the dev environment successfully. The remaining dev verification step is installing the GKE kubectl auth plugin on the VM and confirming cluster access via `kubectl`.

## Remaining work this session
Install `gke-gcloud-auth-plugin` on the dev VM, verify `kubectl` access to `cortado-dev`, and confirm the same-region Artifact Registry is usable. Bootstrap the prod backend bucket when the prod environment is ready.

## Definition of done
- [x] Terraform env roots exist for `dev` and `prod`
- [x] IAM and GKE modules are wired into the env roots
- [x] `terraform/README.md` documents the one-time backend bootstrap step
- [x] `scripts/bootstrap.sh` creates the dev Terraform state bucket
- [x] Dev state bucket `gs://cortado-tf-state-dev` exists
- [x] `terraform init -backend=false` succeeds in `terraform/envs/dev`
- [x] `terraform init -backend=false` succeeds in `terraform/envs/prod`
- [x] `terraform validate` succeeds in `terraform/envs/dev`
- [x] `terraform validate` succeeds in `terraform/envs/prod`
- [x] Backend-backed `terraform init -reconfigure` succeeds in `terraform/envs/dev`
- [x] `terraform plan` succeeds in `terraform/envs/dev`
- [x] `terraform apply` provisions the dev APIs, IAM resources, GKE cluster, and Artifact Registry
- [ ] Dev GKE cluster appears in GCP and Artifact Registry is accessible

## Next task after this one
Task 1.2.1 — Proto definition: agent gRPC service
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.1 for full spec

## Blocked on / decisions needed
None.
