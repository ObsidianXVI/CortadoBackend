## Feature 3.3 — Persistent Volume and Snapshots
**Duration**: Week 13 (2 tasks, ~3 days)

### Task 3.3.1 — PVC lifecycle in control plane
**What to do:**
- `WorkspacePodManager.Create` now also creates a PVC:
  ```go
  pvc := &corev1.PersistentVolumeClaim{
      ObjectMeta: metav1.ObjectMeta{
          Name: "ws-" + workspaceID,
          Namespace: "cortado-workspaces",
      },
      Spec: corev1.PersistentVolumeClaimSpec{
          StorageClassName: ptr("premium-rwo"),
          AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
          Resources: corev1.ResourceRequirements{
              Requests: corev1.ResourceList{
                  corev1.ResourceStorage: resource.MustParse("10Gi"),
              },
          },
      },
  }
  ```
- On `start`, use `VolumeClaimTemplates` → reuse the existing PVC by name.
- On permanent `delete`, delete the PVC explicitly (PVCs are not garbage-collected with pods by default in GKE).

**Challenge**: `ReadWriteOnce` means only one pod can mount the PVC at a time. The stop flow must confirm pod deletion before the start flow creates a new pod. Add a 30-second wait + retry loop in the start path: if PVC is `Bound` to a terminating pod, wait for it to release before scheduling the new pod.

---

### Task 3.3.2 — Workspace snapshots (restic to GCS via Terraform)
**What to do:**
- Terraform: create a GCS snapshot bucket:
  ```hcl
  resource "google_storage_bucket" "workspace_snapshots" {
    name          = "cortado-snapshots-${var.project_id}-${var.env}"
    location      = var.region
    storage_class = "NEARLINE"
    lifecycle_rule {
      condition { age = 30 }
      action    { type = "Delete" }
    }
  }
  resource "google_storage_bucket_iam_member" "agent_writer" {
    bucket = google_storage_bucket.workspace_snapshots.name
    role   = "roles/storage.objectCreator"
    member = "serviceAccount:${var.workspace_agent_sa_email}"
  }
  ```
- Add `restic` to the workspace Dockerfile.
- Add `CreateSnapshot(context.Context, *pb.SnapshotRequest) returns (*pb.SnapshotResponse)` to the agent gRPC service.
- Control plane calls `CreateSnapshot` in the stop flow with a 30-second timeout (fire-and-forget if timeout exceeded).

---

---
