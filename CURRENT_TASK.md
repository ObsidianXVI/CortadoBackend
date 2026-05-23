# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.4 (Flutter Package — Terminal Widget) → Task 1.4.4

## Status
BLOCKED

## What was done last session
Completed Task 1.4.2 by finalizing the Dart mux codec as a reusable protocol surface: exported the terminal channel and message-type constants, retained the zero-copy decode payload view with an explicit safety comment, and expanded tests to cover round-trip encoding, hard-coded Go interoperability bytes, truncated frame rejection, and the documented view semantics. Re-ran Flutter package tests and analysis to keep the package green.

## What was done this session
Converted `demo_app` from the default counter scaffold into a real terminal smoke harness for Task 1.4.4: added a configurable Flutter Web UI for control-plane base URL, workspace ID, and shell selection; wired it to the local `cortado` package; added a draggable terminal resize surface plus manual smoke checklist; documented the run/query-parameter flow in `demo_app/README.md`; and replaced the demo app test with coverage for config parsing and the injected-client connect path. Verified `demo_app` with `flutter pub get`, `flutter analyze`, `flutter test`, and `flutter build web`, and re-ran `flutter analyze` plus `flutter test` in `flutter/`. Investigated the live dev environment and confirmed the intended smoke target is workspace ID `workspace-pod-test`, but the actual Cloud Run base URL is unavailable because Terraform state does not currently expose `control_plane_service_uri` and `terraform/envs/dev/terraform.tfvars` still has `control_plane_image_tag = "pending"`. Added the missing `control-plane/Dockerfile` and `build-control-plane` GitHub Actions workflow so a deployable control-plane image can now be built and pushed, and fixed the `workspace-pod-test` Kubernetes manifest to create the headless Service plus `cortado/workspace-id` label that the terminal bridge expects. Applied the updated smoke-workspace manifest via Terraform and verified the service and endpoint now exist in-cluster.

## Remaining work this session
- Resolve the Cloud Run-to-workspace-agent data-plane architecture gap captured in `DECISIONS_NEEDED.md`.
- Build and publish the control-plane container image, then deploy a real `cortado-control-plane-dev` service.
- Obtain or deploy a live dev control-plane image so Terraform state exposes `control_plane_service_uri`.
- Run the full Chrome → Cloud Run → GKE → PTY smoke flow against `workspace-pod-test`.
- Record the round-trip latency from Chrome DevTools WebSocket frames and note whether the v0.1 latency target is met.

## Definition of done
- [x] `demo_app` is a runnable Flutter Web smoke harness that accepts a control-plane base URL, workspace ID, and shell
- [x] `demo_app` exposes a draggable terminal surface so resize verification can be exercised from the browser UI
- [x] `demo_app/README.md` documents the manual smoke flow, query-parameter support, and checklist commands
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter pub get` passes in `demo_app/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter analyze` returns zero warnings in `demo_app/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter test` passes in `demo_app/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter build web` passes in `demo_app/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter test` passes in `flutter/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter analyze` returns zero warnings in `flutter/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `control-plane/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes in `control-plane/`
- [x] `docker build -f control-plane/Dockerfile -t cortado-control-plane:test .` passes
- [x] `terraform -chdir=terraform/envs/dev validate` passes
- [x] `terraform -chdir=terraform/envs/prod validate` passes
- [x] `workspace-pod-test` now has a matching headless Service and ready endpoint in `cortado-workspaces`
- [ ] Terraform state exposes a live `control_plane_service_uri` for the dev environment
- [ ] The full live smoke sequence succeeds against `workspace-pod-test`: `echo hello_v0_1`, `vim`, `python3`, and resize via `tput cols`
- [ ] Round-trip latency is measured and recorded from Chrome DevTools WebSocket frames

## Next task after this one
Task 2.1.1 — Workspace CRUD endpoints
See _dev/docs/release_timeline.md §Feature 2.1 Task 2.1.1 for full spec

## Blocked on / decisions needed
- External deployment state, not product code:
  - `cd terraform/envs/dev && terraform output -raw control_plane_service_uri` currently fails with `Output "control_plane_service_uri" not found`.
  - `terraform/envs/dev/terraform.tfvars` still sets `control_plane_image_tag = "pending"`, so the dev Cloud Run service has not been deployed from current state.
  - The smoke test should target workspace ID `workspace-pod-test` once the control plane base URL is available.
  - The current bridge resolves `*.cortado-workspaces.svc.cluster.local`, which Cloud Run cannot reach in the current infrastructure layout; see `DECISIONS_NEEDED.md`.
