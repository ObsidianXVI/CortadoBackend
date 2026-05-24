# CURRENT TASK

## Release · Feature · Task
v0.3 → Feature 3.3 (Persistent Volume and Snapshots) → Task 3.3.2

## Status
PENDING

## What was done last session
Completed Task 3.3.1 by keeping the explicit workspace PVC lifecycle in `PodManager`, adding a bounded wait/retry loop before recreating a pod against a terminating `ReadWriteOnce` volume, and extending the control-plane tests to cover PVC spec details, restart wait behavior, timeout cleanup, and the required `go test` / `go build` verification.

## What was done this session
Task 3.3.2 has not been started yet.

## Remaining work this session
Implement workspace snapshot support end-to-end:
- Terraform snapshot bucket + IAM
- restic in workspace image
- snapshot RPC in proto + generated code
- agent snapshot implementation
- control-plane stop-flow snapshot trigger

## Definition of done
- [ ] Terraform provisions the workspace snapshot GCS bucket with 30-day lifecycle cleanup in dev and prod
- [ ] Terraform grants the workspace agent service account object-creator access to the snapshot bucket
- [ ] The workspace image includes `restic`
- [ ] Agent proto defines `CreateSnapshot`
- [ ] Agent implements snapshot creation with the configured bucket/repository target
- [ ] Control plane triggers `CreateSnapshot` during stop with a 30-second timeout and does not fail the stop path if snapshotting times out
- [ ] Relevant proto / agent / control-plane tests pass
- [ ] `cd proto && buf lint` passes
- [ ] `cd agent && GOTOOLCHAIN=local go test ./...` passes
- [ ] `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...` passes
- [ ] `cd control-plane && GOTOOLCHAIN=local go test ./...` passes
- [ ] `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...` passes
- [ ] `terraform -chdir=terraform/envs/dev validate` passes
- [ ] `terraform -chdir=terraform/envs/prod validate` passes

## Next task after this one
Feature 4.1 → Task 4.1.1 — Proto: LSP service
See _dev/features/feat-4-1.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the standing file-API and CodeMirror follow-ups; no new blockers are recorded for Task 3.3.2 yet.
