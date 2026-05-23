# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.4 (Flutter Package — Terminal Widget) → Task 1.4.3

## Status
DONE

## What was done last session
Completed Task 1.4.2 by finalizing the Dart mux codec as a reusable protocol surface: exported the terminal channel and message-type constants, retained the zero-copy decode payload view with an explicit safety comment, and expanded tests to cover round-trip encoding, hard-coded Go interoperability bytes, truncated frame rejection, and the documented view semantics. Re-ran Flutter package tests and analysis to keep the package green.

## What was done this session
Completed Task 1.4.3 by adding a web-only `CortadoTerminal` widget that opens terminal channels over the existing `CortadoClient`, bridges xterm.js input/output through `HtmlElementView` and `dart:js_interop`, and emits PTY resize events over a dedicated mux resize frame. Added the corresponding control-plane mux resize payload codec and gRPC bridge handling, exported the terminal widget from the package, wired the in-repo `demo_app` host HTML with xterm.js and the local JS bridge, documented the resolved asset-delivery and resize-protocol decisions, and kept both Go and Flutter verification green.

## Remaining work this session
None.

## Definition of done
- [x] The Flutter package exports a `CortadoTerminal` widget that renders an xterm.js-backed terminal on Flutter Web
- [x] Terminal input writes data frames through `CortadoClient`, and incoming mux data frames render into the terminal widget
- [x] Resize events emit a dedicated mux resize frame and the control plane maps that payload onto the agent PTY resize stream
- [x] The host-app HTML integration path is represented in `demo_app/web/index.html` with xterm.js and addon includes plus the local JS bridge
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...` passes in `control-plane/`
- [x] `CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` passes in `control-plane/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter test` passes in `flutter/`
- [x] `/home/OBSiDIAN/tools/flutter/bin/flutter analyze` returns zero warnings in `flutter/`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.4.4 — End-to-end smoke test
See _dev/docs/release_timeline.md §Feature 1.4 Task 1.4.4 for full spec

## Blocked on / decisions needed
None.
