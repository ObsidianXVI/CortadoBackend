locals {
  bucket_name = "cortado-snapshots-${var.project_id}-${var.env}"
}

resource "google_storage_bucket" "workspace_snapshots" {
  name                        = local.bucket_name
  project                     = var.project_id
  location                    = var.region
  storage_class               = "NEARLINE"
  uniform_bucket_level_access = true
  labels                      = var.labels

  lifecycle_rule {
    action {
      type = "Delete"
    }

    condition {
      age = 30
    }
  }
}

resource "google_storage_bucket_iam_member" "workspace_agent_object_creator" {
  bucket = google_storage_bucket.workspace_snapshots.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${var.workspace_agent_service_account_email}"
}
