# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.4 (Flutter Package — Terminal Widget) → Task 1.4.2

## Status
DONE

## What was done last session
Completed Task 1.4.1 by upgrading the placeholder Flutter package dependencies, adding a public `CortadoClient` with platform-aware WebSocket connection handling and browser-safe dev auth query parameters, and wiring broadcast frame/error streams for inbound mux traffic. Implemented the initial `MuxFrame` codec needed by the client, exported the public package API, and added unit coverage for the client URI/auth/send-receive path plus codec round-trips and the hard-coded Go frame bytes.

## What was done this session
Completed Task 1.4.2 by finalizing the Dart mux codec as a reusable protocol surface: exported the terminal channel and message-type constants, retained the zero-copy decode payload view with an explicit safety comment, and expanded tests to cover round-trip encoding, hard-coded Go interoperability bytes, truncated frame rejection, and the documented view semantics. Re-ran Flutter package tests and analysis to keep the package green.

## Remaining work this session
None.

## Definition of done
- [x] `MuxFrame` encodes channel id, message type, and payload length using big-endian layout matching the Go gateway
- [x] `MuxFrame.decode` validates header/payload size and returns a zero-copy payload view with the mutation caveat documented inline
- [x] Dart tests cover a round-trip encode/decode path and the hard-coded Go interoperability frame bytes
- [x] Dart tests reject invalid/truncated frames
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter test` passes in `flutter/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter analyze` returns zero warnings in `flutter/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.4.3 — Terminal widget (xterm.js via HtmlElementView)
See _dev/docs/release_timeline.md §Feature 1.4 Task 1.4.3 for full spec

## Blocked on / decisions needed
None.
