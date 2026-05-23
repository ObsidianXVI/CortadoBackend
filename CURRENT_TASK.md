# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.4 (Flutter Package — Terminal Widget) → Task 1.4.1

## Status
DONE

## What was done last session
Completed Task 1.3.4 by wiring the control-plane terminal bridge from the WebSocket mux to the workspace agent gRPC PTY stream, including cached per-workspace gRPC connections, default bridge-based connect-handler wiring, close-frame surfacing for setup failures, and bufconn-backed integration coverage for open/data/exit and connection reuse.

## What was done this session
Completed Task 1.4.1 by upgrading the placeholder Flutter package dependencies, adding a public `CortadoClient` with platform-aware WebSocket connection handling and browser-safe dev auth query parameters, and wiring broadcast frame/error streams for inbound mux traffic. Implemented the initial `MuxFrame` codec needed by the client, exported the public package API, and added unit coverage for the client URI/auth/send-receive path plus codec round-trips and the hard-coded Go frame bytes.

## Remaining work this session
None.

## Definition of done
- [x] `flutter/` contains the package dependencies needed for the WebSocket client (`web_socket_channel`, Riverpod, Freezed annotations/generators)
- [x] `CortadoClient` connects to `/v1/workspaces/{id}/connect` and uses `?dev_token=dev-bypass` on Flutter Web
- [x] The client awaits `ready`, listens for stream `onError`/`onDone`, and decodes inbound mux frames
- [x] The public package exports the client API cleanly from `flutter/lib/cortado.dart`
- [x] Package tests cover the client’s URI/auth behavior and mux-frame send/receive path
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter analyze` returns zero warnings in `flutter/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.4.2 — Mux frame codec in Dart
See _dev/docs/release_timeline.md §Feature 1.4 Task 1.4.2 for full spec

## Blocked on / decisions needed
None.
