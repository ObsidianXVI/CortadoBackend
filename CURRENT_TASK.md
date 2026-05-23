# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.4 (Flutter Package — Terminal Widget) → Task 1.4.4

## Status
BLOCKED

## What was done last session
Completed Task 1.4.2 by finalizing the Dart mux codec as a reusable protocol surface: exported the terminal channel and message-type constants, retained the zero-copy decode payload view with an explicit safety comment, and expanded tests to cover round-trip encoding, hard-coded Go interoperability bytes, truncated frame rejection, and the documented view semantics. Re-ran Flutter package tests and analysis to keep the package green.

## What was done this session
Resolved the infrastructure path for the current spec by keeping the control plane on Cloud Run and updating Terraform plus runtime wiring for Direct VPC egress and GKE Cloud DNS additive VPC scope: the GKE module now provisions `cortado-dev.internal`, the Cloud Run module injects the workspace namespace/DNS domain and private egress network settings, and the control-plane resolver/runtime now targets `*.svc.cortado-dev.internal` instead of hard-coding `cluster.local`. Built and pushed real dev images for both the control plane (`20260523-103219-stalefix`) and workspace agent (`20260523-102947-workspace-tools`), updated `terraform/envs/dev/terraform.tfvars` to those tags, replaced the dev GKE cluster to pick up the DNS mode, recreated the bootstrap/test workspace resources, and confirmed Terraform now exposes `control_plane_service_uri = https://cortado-control-plane-dev-dzozcgk63q-uc.a.run.app`.

The live smoke uncovered and fixed two product issues needed for Task 1.4.4. First, the workspace image was missing `python3` and `vim`, and PTY sessions started without `TERM`, so the documented smoke commands were not actually runnable; the workspace image now installs `python3` and `vim`, and the PTY manager defaults `TERM=xterm-256color` when the session env does not provide one. Second, recycling `workspace-pod-test` exposed that the control plane permanently cached stale gRPC connections per workspace; `CreatePty` now evicts/redials the cached workspace connection on `DeadlineExceeded`/`Unavailable`, and a regression test covers that path. After redeploying both images, an authenticated live WebSocket smoke through the Cloud Run service succeeded for `echo hello_v0_1`, resize verification (`COLUMNS=132`, `stty size -> 43 132`, `tput cols -> 132`), interactive `python3`, and `vim` open/quit traffic, with observed echo round-trip around 4-5 ms through the local auth proxy.

## Remaining work this session
- Apply the remaining Terraform resources that still fail for caller-permission reasons: `google_firestore_database.default` and `module.cloudrun.google_cloud_run_v2_service_iam_member.public`.
- Run the browser-side `demo_app` smoke against the live service in Chrome once a usable Chrome binary/tooling path is available and direct unauthenticated access is in place.
- Record browser-observed WebSocket latency from Chrome DevTools rather than the lower-level proxy smoke probe.

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
- [x] Terraform state exposes a live `control_plane_service_uri` for the dev environment
- [ ] The full live smoke sequence succeeds against `workspace-pod-test`: `echo hello_v0_1`, `vim`, `python3`, and resize via `tput cols`
- [ ] Round-trip latency is measured and recorded from Chrome DevTools WebSocket frames

## Next task after this one
Task 2.1.1 — Workspace CRUD endpoints
See _dev/docs/release_timeline.md §Feature 2.1 Task 2.1.1 for full spec

## Blocked on / decisions needed
- External permissions/tooling, not product architecture:
  - Full `terraform -chdir=terraform/envs/dev apply -auto-approve` still fails on `google_firestore_database.default` with `Error 403: The caller does not have permission`.
  - Full `terraform -chdir=terraform/envs/dev apply -auto-approve` still fails on `module.cloudrun.google_cloud_run_v2_service_iam_member.public` with `Permission 'run.services.setIamPolicy' denied`, so the live Cloud Run URL is still IAM-protected unless accessed through an authenticated local proxy.
  - Browser-specific smoke automation is currently unavailable on this workstation because no Chrome executable is installed at `/opt/google/chrome/chrome`, so the final Chrome/DevTools validation remains manual or requires local browser tooling to be installed.
