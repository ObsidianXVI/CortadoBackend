# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.2 (Flutter Workspace Manager) → Task 2.2.1

## Status
DONE

## What was done last session
Completed Task 2.1.1 by adding the Firestore-backed workspace CRUD API to the control plane, provisioning PVC-backed workspace pods/services, wiring Cloud Run-compatible Kubernetes client bootstrap, and bootstrapping the shared workspace StorageClass in Terraform.

## What was done this session
Implemented Task 2.2.1 in the Flutter package by adding a REST-backed `WorkspaceManager`, freezed workspace/status models with JSON serialization, and a `CortadoWorkspaceProvider` that scopes both `workspaceId` and a `WorkspaceManager` via Riverpod while exposing `CortadoWorkspaceProvider.of(context).workspaceId`. `watchStatus()` now polls `/v1/workspaces/{id}` at the required 3-second transitional and 30-second running cadences using a cancellation-aware timer-backed stream, and the package exports/tests were expanded to cover create/start/stop requests, polling cadence transitions, terminal-state completion, and provider disposal cleanup.

## Remaining work this session
None. Advance to Task 2.2.2.

## Definition of done
- [x] `WorkspaceManager.create()` creates workspaces through `POST /v1/workspaces`
- [x] `WorkspaceManager.start()` and `WorkspaceManager.stop()` target the correct control-plane transition endpoints
- [x] `watchStatus()` polls every 3 seconds while a workspace is `CREATING`, `STARTING`, or `STOPPING`
- [x] `watchStatus()` drops to a 30-second polling cadence when a workspace is `RUNNING`
- [x] `watchStatus()` stops polling when the caller cancels the stream subscription
- [x] `Workspace` and `WorkspaceStatus` are implemented as freezed data classes with JSON serialization
- [x] `CortadoWorkspaceProvider` exposes `workspaceId` through `CortadoWorkspaceProvider.of(context)`
- [x] The Riverpod-backed status providers cancel their `watchStatus()` subscriptions on dispose
- [x] `cd flutter && flutter pub get` passes
- [x] `cd flutter && dart run build_runner build --delete-conflicting-outputs` passes
- [x] `cd flutter && flutter test` passes
- [x] `cd flutter && flutter analyze` passes

## Next task after this one
Task 2.2.2 — Reconnection after cold start
See _dev/docs/release_timeline.md §Feature 2.2 Task 2.2.2 for full spec

## Blocked on / decisions needed
None.
