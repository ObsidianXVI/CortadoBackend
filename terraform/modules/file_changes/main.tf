resource "google_pubsub_topic" "file_changes" {
  project = var.project_id
  name    = "cortado-file-changes-${var.env}"
  labels  = var.labels
}

resource "google_pubsub_subscription" "indexer_updater" {
  project              = var.project_id
  name                 = "cortado-indexer-updater-${var.env}"
  topic                = google_pubsub_topic.file_changes.id
  ack_deadline_seconds = var.ack_deadline_seconds
  labels               = var.labels

  push_config {
    push_endpoint = "${var.indexer_updater_service_uri}/ingest"
  }
}
