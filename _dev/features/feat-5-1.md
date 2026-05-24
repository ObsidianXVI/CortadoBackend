## Feature 5.1 — Codebase Indexing Pipeline

### Task 5.1.1 — Tree-sitter chunker (Python microservice)
- Build `indexer/` Python service (Cloud Run Job, not a service).
- Tree-sitter chunking per language (Dart, Python, JS, Go). Fallback: 50-line windows with 10-line overlap.
- Terraform:
  ```hcl
  resource "google_cloud_run_v2_job" "indexer" {
    name     = "cortado-indexer-${var.env}"
    location = var.region
    template {
      template {
        service_account = var.indexer_sa_email
        containers {
          image = "${var.region}-docker.pkg.dev/${var.project_id}/cortado/indexer:${var.image_tag}"
        }
      }
    }
  }
  ```
- The control plane triggers indexer jobs via the Cloud Run Jobs API (on workspace create and on file change events from Pub/Sub).

**Challenge**: tree-sitter Python package requires compiled C extensions. Pin `tree-sitter==0.22.0` — the 0.23.x API is breaking. Build in Docker's build stage, not at runtime.

---

### Task 5.1.2 — Embedding pipeline + Qdrant sidecar
- Use `text-embedding-004` (Vertex AI) or `voyage-code-3` (Voyage AI). Batch in groups of 100 chunks.
- Deploy Qdrant as a sidecar container in the workspace pod spec (Terraform/YAML):
  ```yaml
  - name: qdrant
    image: qdrant/qdrant:v1.12.0
    resources:
      requests: { memory: "256Mi", cpu: "100m" }
    volumeMounts:
    - name: workspace-data
      mountPath: /qdrant/storage
      subPath: .cortado/qdrant
  ```
- Collection name: `ws-{workspaceID}`. Vector dimensions: 768 (`text-embedding-004`) or 1024 (`voyage-code-3`).

**Key detail**: The Qdrant sidecar stores its data in `/workspace/.cortado/qdrant` — a subdirectory of the persistent volume. It persists across workspace stop/start automatically.

---

### Task 5.1.3 — Incremental index updates via Pub/Sub
- File change events (from WatchFiles) are published to a new Pub/Sub topic `cortado-file-changes`.
- A `cortado-indexer-updater` Cloud Run service (long-running, not a Job) subscribes and processes updates:
  - `MODIFIED`/`CREATED`: re-chunk file, delete old Qdrant vectors for that path, insert new vectors.
  - `DELETED`: delete all Qdrant vectors where `metadata.file == path`.
- Debounce: batch file changes for the same workspace over 5-second windows.

Terraform:
```hcl
resource "google_pubsub_topic" "file_changes" {
  name = "cortado-file-changes-${var.env}"
}
resource "google_pubsub_subscription" "indexer_updater" {
  name  = "cortado-indexer-updater-${var.env}"
  topic = google_pubsub_topic.file_changes.name
  ack_deadline_seconds = 60
  push_config {
    push_endpoint = "${google_cloud_run_v2_service.indexer_updater.uri}/ingest"
  }
}
```
