# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.4 (Flutter Package — Terminal Widget) → Task 1.4.4

## Status
DONE

## What was done last session
Completed Task 1.4.2 by finalizing the Dart mux codec as a reusable protocol surface: exported the terminal channel and message-type constants, retained the zero-copy decode payload view with an explicit safety comment, and expanded tests to cover round-trip encoding, hard-coded Go interoperability bytes, truncated frame rejection, and the documented view semantics. Re-ran Flutter package tests and analysis to keep the package green.

## What was done this session
Closed the remaining infrastructure gap for the dev environment by importing the pre-existing Firestore `(default)` database into Terraform state and rerunning `terraform -chdir=terraform/envs/dev apply -auto-approve`, which left the dev stack fully managed and kept `control_plane_service_uri = https://cortado-control-plane-dev-dzozcgk63q-uc.a.run.app` live. Installed local Chrome plus `xvfb`, verified the DevTools automation path, and served the built Flutter smoke harness from `demo_app/build/web` for a real browser pass against the live Cloud Run control plane.

The Chrome validation exposed and fixed the last browser-specific control-plane issues for Task 1.4.4. First, browser WebSocket upgrades sent `Sec-WebSocket-Protocol: cortado-v1`, but the gateway upgrader did not advertise that subprotocol, so the browser handshake failed until the control plane explicitly negotiated `cortado-v1`. Second, long-lived browser PTY sessions were later torn down with `ENHANCE_YOUR_CALM` / `too_many_pings` because the control-plane gRPC client forced aggressive custom keepalive pings to the workspace agent; the bridge now relies on default gRPC client behavior instead. After redeploying image `20260523-110240-keepalivefix`, the live browser smoke passed for `echo hello_v0_1`, `python3` with `print('py_ok')`, `vim` full-screen redraw, and PTY resize propagation (`tput cols` changed from `100` to `130` after a browser-driven resize). Browser-observed round-trip latency was recorded at about `342 ms` using a Chrome DevTools-driven DOM echo timer on the live page because the MCP Network adapter did not expose WebSocket frame timings directly.

## Remaining work this session
None. Advance to Task 2.1.1.

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
- [x] The full live smoke sequence succeeds against `workspace-pod-test`: `echo hello_v0_1`, `vim`, `python3`, and resize via `tput cols`
- [x] Browser-observed round-trip latency is measured and recorded from the Chrome smoke session (`~342 ms` via DevTools-driven DOM timing on the live page)

## Next task after this one
Task 2.1.1 — Workspace CRUD endpoints
See _dev/docs/release_timeline.md §Feature 2.1 Task 2.1.1 for full spec

## Blocked on / decisions needed
None.
