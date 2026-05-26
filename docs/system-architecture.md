# System Architecture

## High-Level Structure

Cortado is not a standalone app. It is a backend package that IDE vendors embed into their own application. The runtime is split into three main processes:

- The **Flutter package** runs in the consumer app and talks to the control plane over HTTP and WebSocket.
- The **control plane** is a Go HTTP server that owns auth, workspace lifecycle, file proxying, and the browser-facing WebSocket mux.
- The **workspace agent** is a Go gRPC server that runs inside each workspace pod and owns PTYs, file I/O, and file watching.

The repository root mirrors that split:

- [`flutter/`](../flutter)
- [`control-plane/`](../control-plane)
- [`agent/`](../agent)
- [`proto/`](../proto)
- [`terraform/`](../terraform)

## End-to-End Request Path

1. A consumer app signs the user into Cortado-managed Firebase Auth and exchanges that token through `POST /v1/sessions/exchange/firebase`.
2. The Flutter package stores the access token and refresh token in `CortadoAuthSession`.
3. If the user wants a headless credential later, the authenticated app can mint a personal Cortado API key through `POST /v1/api-keys`.
4. The app creates or lists workspaces through `WorkspaceManager`.
5. The control plane stores workspace metadata in Firestore and asks the Kubernetes provisioner to create or stop the pod.
6. When the app opens a workspace, the control plane upgrades a WebSocket on `/v1/workspaces/{id}/connect`.
7. The browser/client and the control plane exchange binary mux frames.
8. For terminal traffic, the control plane opens a gRPC stream to the workspace agent and bridges data to and from the PTY.
9. For file sync traffic, the control plane opens a gRPC `WatchFiles` stream and forwards file events to the client.

## Workspace Topology

The Terraform modules create one GKE Autopilot cluster and one Cloud Run control-plane service per environment. Each workspace becomes:

- a Kubernetes Pod running the workspace image,
- a headless Service named after the workspace ID,
- a PVC mounted at `/workspace`,
- and a workload-identity-bound service account for GCP access.

The control plane resolves workspace DNS using the pattern:

`<workspace-id>.<workspace-namespace>.svc.<cluster-dns-domain>`

That DNS name is used for gRPC connections from Cloud Run to the agent pod.

## Where Data Lives

- **Firestore** stores workspace metadata, hashed API keys, and refresh-token/session data.
- **Redis** is used as the auth validation cache.
- **Pub/Sub** receives usage events from the workspace agent.
- **GKE** runs the workspaces and the agents.
- **Artifact Registry** stores the workspace and control-plane images.
- **Secret Manager** stores control-plane secrets such as the JWT private key.

## Example Workspace Record

```json
{
  "id": "ws-8f3d2b6d",
  "tenantId": "tenant-acme",
  "userId": "user-42",
  "image": "us-central1-docker.pkg.dev/cortado-ide/cortado-dev/workspace:2026-05-23",
  "resources": {
    "cpu": 1,
    "memoryGb": 2
  },
  "status": "RUNNING",
  "createdAt": "2026-05-23T22:00:00Z",
  "updatedAt": "2026-05-23T22:15:00Z",
  "lastActiveAt": "2026-05-23T22:14:21Z"
}
```

## Design Constraint

The control plane must be able to run outside the cluster. That is why it resolves Kubernetes access via kubeconfig when available, then falls back to in-cluster config, then falls back to GKE API endpoint discovery plus OAuth transport wrapping when deployed on Cloud Run.
