locals {
  pubsub_service_agent = format(
    "serviceAccount:service-%s@gcp-sa-pubsub.iam.gserviceaccount.com",
    var.project_number,
  )
}

resource "google_pubsub_topic" "usage_events" {
  project = var.project_id
  name    = "cortado-usage-events-${var.env}"
  labels  = var.labels
}

resource "google_pubsub_topic" "usage_events_dlq" {
  project = var.project_id
  name    = "cortado-usage-events-dlq-${var.env}"
  labels  = var.labels
}

resource "google_bigquery_dataset" "billing" {
  project    = var.project_id
  dataset_id = "cortado_billing_${var.env}"
  location   = var.region
  labels     = var.labels
}

resource "google_bigquery_table" "usage_events" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.billing.dataset_id
  table_id   = "usage_events"
  labels     = var.labels
  schema     = file("${path.module}/schemas/usage_events.json")

  time_partitioning {
    field = "event_time"
    type  = "DAY"
  }
}

resource "google_bigquery_dataset_iam_member" "pubsub_data_editor" {
  dataset_id = google_bigquery_dataset.billing.dataset_id
  role       = "roles/bigquery.dataEditor"
  member     = local.pubsub_service_agent
}

resource "google_bigquery_dataset_iam_member" "pubsub_metadata_viewer" {
  dataset_id = google_bigquery_dataset.billing.dataset_id
  role       = "roles/bigquery.metadataViewer"
  member     = local.pubsub_service_agent
}

resource "google_pubsub_subscription" "usage_to_bigquery" {
  project = var.project_id
  name    = "cortado-usage-to-bq-${var.env}"
  topic   = google_pubsub_topic.usage_events.id
  labels  = var.labels

  bigquery_config {
    table            = "${google_bigquery_table.usage_events.project}.${google_bigquery_table.usage_events.dataset_id}.${google_bigquery_table.usage_events.table_id}"
    use_table_schema = true
    write_metadata   = true
  }

  depends_on = [
    google_bigquery_dataset_iam_member.pubsub_data_editor,
    google_bigquery_dataset_iam_member.pubsub_metadata_viewer,
  ]
}
