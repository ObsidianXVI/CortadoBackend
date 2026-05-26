# Cortado Developer Docs

This directory documents the repository as it exists in this codebase, not as an abstract product idea. The goal is to help a developer who has low context on the stack understand how the pieces connect and where to look next.

## Repository Map

- `agent/` implements the per-workspace gRPC server that runs inside each workspace pod.
- `control-plane/` exposes the HTTP API, auth/session flow, workspace orchestration, and the WebSocket mux endpoint.
- `flutter/` is the package that IDE authors embed into their own app.
- `proto/` contains the gRPC contract shared by the Go and Dart sides.
- `terraform/` provisions the GCP runtime: GKE, Cloud Run, Firestore, Redis, Secret Manager, Pub/Sub, Artifact Registry, and the bootstrap manifests.
- `scripts/` contains repo bootstrap and utility scripts.

## Recommended Reading Order

1. [System architecture](system-architecture.md)
2. [Control plane](control-plane.md)
3. [Workspace agent](agent.md)
4. [Flutter package](flutter-package.md)
5. [Protocols and data shapes](protocols.md)
6. [Terraform and deployment](terraform-deployment.md)

## Quick Mental Model

The runtime path is:

`Flutter package` -> `control-plane HTTP API` -> `workspace record in Firestore` -> `GKE pod/service` -> `workspace agent gRPC` -> `PTY/filesystem` -> back to `control-plane` -> back to the Flutter client.

For production-style auth, the Flutter package can now either exchange a Cortado-managed Firebase sign-in directly into a Cortado session or bootstrap through Firebase-authenticated API key issuance. Those paths are documented in [flutter-package.md](flutter-package.md) and [control-plane.md](control-plane.md).

The control plane never talks to the local filesystem directly. File operations are proxied to the agent in the workspace pod. Terminal traffic is multiplexed over a single WebSocket using a binary frame format. The agent is the only process that touches the workspace mount at `/workspace`.

## Data Shapes You Will See Often

- Workspace records are JSON objects with `id`, `tenantId`, `userId`, `image`, `resources`, `status`, `createdAt`, `updatedAt`, and optionally `lastActiveAt`.
- Directory listings are arrays of entries with `name`, `size`, `isDir`, `modTime`, and `permissions`.
- File sync events are protobuf messages carrying a relative workspace path, an event type, and sometimes a checksum.
- Terminal frames are binary mux frames with a 7-byte header and a payload specific to the channel and message type.

## Open Decisions

Only unresolved items are tracked in `DECISIONS_NEEDED.md` now; the filesystem semantics for recursive delete, parent directory handling on write, and CodeMirror's Dart fallback are settled in `DECISIONS.md`.
