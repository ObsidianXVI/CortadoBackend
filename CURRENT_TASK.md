# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.1 (Repo & Dev Environment Bootstrap) → Task 1.1.3 + 1.1.4

## Status
IN PROGRESS

## What was done last session
Added Terraform env roots for `dev` and `prod`, IAM and GKE modules, bootstrap documentation, and the one-time dev state bucket bootstrap script. Verified `terraform init -backend=false` and `terraform validate` for both envs, created the dev state bucket, reinitialized the dev backend against GCS, and produced a successful dev `terraform plan`.

## What was done this session
Updated the feature specs and technical report to switch image distribution from Artifact Registry to Docker Hub, and recorded that decision in `DECISIONS.md` so upcoming infra and deployment work stays aligned.
Removed Artifact Registry API enablement, resources, and outputs from the Terraform configuration so the GKE stack matches the Docker Hub-based image flow.

## Remaining work this session
Run live `terraform apply` for the dev environment and verify the GKE cluster in GCP. Wire upcoming deployment work to Docker Hub image references instead of Artifact Registry. Bootstrap the prod backend bucket when the prod environment is ready.

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
- [ ] `terraform apply` provisions the dev APIs, IAM resources, and GKE cluster
- [ ] Dev GKE cluster appears in GCP

## Next task after this one
Task 1.2.1 — Proto definition: agent gRPC service
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.1 for full spec

## Blocked on / decisions needed
Explicit approval to run live `terraform apply` for the dev environment, since it will create billable GKE resources in GCP.
