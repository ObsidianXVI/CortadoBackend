# Decisions Needed

## 23/05/26

### Task 1.4.3 — Web terminal asset delivery
- Question: how should the `cortado` Flutter package deliver the required `xterm.js` web assets to consuming apps?
- Context: the feature spec says to add scripts to `flutter/web/index.html`, but `flutter/` is a package, not an app, so that file is not used by downstream consumers. The repo does have `demo_app/web/index.html` for local testing, but changing only the demo app would not solve package-consumer integration.
- Options:
  - Require consuming apps to add the `<script>` and `<link>` tags to their own `web/index.html`, and document that setup in the package README/example.
  - Vendor the assets in the package and load them dynamically at runtime from package-served assets.
  - Another package-level integration mechanism specified by the user.

### Task 1.4.3 — Terminal resize transport over the mux
- Question: what mux message type and payload format should the Flutter terminal use to send PTY resize events to the control plane?
- Context: the current control-plane bridge only accepts `Open`, `Data`, and `Close` terminal messages, while the task spec requires `ResizeObserver callback -> Dart sends resize MuxFrame -> server resizes PTY`. The agent gRPC API already supports `WindowSize resize`, but the WebSocket mux protocol does not yet define how resize travels from browser to control plane.
- Impact: `CortadoTerminal` can be scaffolded without this, but full Task 1.4.3 completion and the resize smoke test depend on a concrete mux protocol decision.
