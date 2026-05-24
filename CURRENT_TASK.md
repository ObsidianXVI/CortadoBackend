# CURRENT TASK

## Release · Feature · Task
v0.4 → Feature 4.1 (LSP Gateway) → Task 4.1.1

## Status
PENDING

## What was done last session
Completed Task 3.3.2 by adding the snapshot RPC to the agent proto, wiring a restic-backed snapshot path into the workspace agent with env-driven repository configuration, triggering best-effort snapshots from the control-plane stop flow, provisioning the snapshot bucket plus Cloud Run/Secret Manager wiring in Terraform, and verifying the proto, Go, and Terraform gates.

## What was done this session
Task 4.1.1 has not been started yet.

## Remaining work this session
Implement the initial LSP gateway contract:
- add `OpenLSP` and `StreamLSP` RPCs to `agent.proto`
- define `OpenLSPRequest`, `OpenLSPResponse`, and `LSPMessage`
- update generated bindings after the proto change
- add the Dart SDK Docker build-arg path for Dart LSP support

## Definition of done
- [ ] `agent.proto` defines `OpenLSP(OpenLSPRequest) returns (OpenLSPResponse)` and `StreamLSP(stream LSPMessage) returns (stream LSPMessage)`
- [ ] `OpenLSPRequest` carries `language`
- [ ] `LSPMessage` carries raw JSON-RPC `data`
- [ ] Generated bindings are refreshed after the proto change
- [ ] The workspace Docker image supports an `INCLUDE_DART_SDK` build arg for optionally layering in the Dart SDK
- [ ] Relevant proto / agent verification passes
- [ ] `cd proto && buf lint` passes
- [ ] `cd agent && GOTOOLCHAIN=local go test ./...` passes
- [ ] `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...` passes

## Next task after this one
Feature 4.1 → Task 4.1.2 — Agent-side LSP process manager
See _dev/features/feat-4-1.md for full spec

## Blocked on / decisions needed
See `DECISIONS_NEEDED.md` for the standing file-API/CodeMirror follow-ups plus the Task 3.3.2 snapshot-bucket IAM-role confirmation for restic.
