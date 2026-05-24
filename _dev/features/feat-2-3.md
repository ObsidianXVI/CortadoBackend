## Feature 2.3 — Basic Billing Events

### Task 2.3.1 — Pub/Sub + BigQuery via Terraform
**What to do:**
- All infrastructure as Terraform:
  ```hcl
  resource "google_pubsub_topic" "usage_events" {
    name = "cortado-usage-events-${var.env}"
  }

  resource "google_bigquery_dataset" "billing" {
    dataset_id = "cortado_billing_${var.env}"
    location   = var.region
  }

  resource "google_bigquery_table" "usage_events" {
    dataset_id = google_bigquery_dataset.billing.dataset_id
    table_id   = "usage_events"
    schema     = file("${path.module}/schemas/usage_events.json")
    time_partitioning {
      type  = "DAY"
      field = "event_time"
    }
  }

  resource "google_pubsub_subscription" "usage_to_bq" {
    name  = "cortado-usage-to-bq-${var.env}"
    topic = google_pubsub_topic.usage_events.name
    bigquery_config {
      table            = "${var.project_id}.${google_bigquery_table.usage_events.dataset_id}.${google_bigquery_table.usage_events.table_id}"
      write_metadata   = true
    }
  }

  # Dead-letter topic for failed deliveries
  resource "google_pubsub_topic" "usage_events_dlq" {
    name = "cortado-usage-events-dlq-${var.env}"
  }
  ```
- Grant Pub/Sub service account BigQuery write access:
  ```hcl
  resource "google_bigquery_dataset_iam_member" "pubsub_writer" {
    dataset_id = google_bigquery_dataset.billing.dataset_id
    role       = "roles/bigquery.dataEditor"
    member     = "serviceAccount:service-${var.project_number}@gcp-sa-pubsub.iam.gserviceaccount.com"
  }
  ```

**Key detail**: `var.project_number` (not `project_id`) is needed for the Pub/Sub service account email. Add it as a Terraform variable: `data "google_project" "current" {}` and use `data.google_project.current.number`. This is not the same as `project_id` — forgetting this causes silent BigQuery write failures that are very difficult to diagnose.

---

### Task 2.3.2 — Usage event emission from agent + WAL
**What to do:**
- The workspace agent publishes a usage event to the `cortado-usage-events` Pub/Sub topic every 10 seconds while a PTY session is active.
- Event WAL: before publishing, append the event to `/workspace/.cortado/usage.wal` (a simple newline-delimited JSON file). After successful Pub/Sub acknowledgement, mark the event with `"published": true`. On agent startup, replay any unpublished events from the WAL.
- Grant workspace agent SA `pubsub.publisher` role:
  ```hcl
  resource "google_pubsub_topic_iam_member" "agent_publisher" {
    topic  = google_pubsub_topic.usage_events.name
    role   = "roles/pubsub.publisher"
    member = "serviceAccount:${google_service_account.workspace_agent.email}"
  }
  ```

**Challenge**: The WAL file at `/workspace/.cortado/` is on the persistent volume — it survives pod restarts. But if the workspace is permanently deleted (PVC deleted), the WAL is lost without replay. Add a final "flush" step in the delete flow: the control plane calls a `FlushUsageWAL` gRPC method on the agent before deleting the pod, which publishes all pending WAL events synchronously with a 10-second timeout.

---
