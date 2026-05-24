# CURRENT TASK

## Release · Feature · Task
v0.7 → Feature 7.1 (Port Forward Gateway) → Task 7.1.2

## Status
IN PROGRESS

## What was done last session
Completed Task 5.2.3 by wiring `CortadoAIService` into `CortadoCodeEditor`, adding CodeMirror ghost-text decorations plus inline completion debounce/cancel key handling in the web bridge, exposing inline completion interop through the editor platform adapters, trimming echoed prefix overlap before ghost rendering, and covering the integration with Flutter widget tests plus JS-side helper tests/build verification.

## What was done this session
Completed Task 7.1.1 by extending the agent proto with `ListPorts` and `WatchPorts`, adding a procfs-backed port monitor that parses `/proc/net/tcp` and `/proc/net/tcp6`, filtering out reserved and privileged ports, wiring the new list/watch RPCs into the agent server with polling-based add/remove diffs, and covering the slice with parser tests plus bufconn agent server tests.

## Remaining work this session
Task 7.1.2:
- add the dedicated port-forward HTTP and WebSocket gateway service
- proxy validated workspace traffic onto detected workspace ports
- wire the gateway deployment/runtime path into Terraform and the Cloud Run/GKE topology

## Definition of done
- [ ] dedicated gateway service can proxy HTTP traffic to workspace ports after validating workspace/port requests
- [ ] WebSocket upgrade traffic is tunneled separately from plain HTTP forwarding
- [ ] Terraform/runtime wiring exists for the port-forward service deployment path
- [ ] relevant control-plane or gateway tests/build pass for the new slice

## Next task after this one
v0.7 → Feature 7.1 → Task 7.1.3 — Flutter web preview
See _dev/features/feat-7-1.md for the active Feature 7.1 spec

## Blocked on / decisions needed
No active blockers.
