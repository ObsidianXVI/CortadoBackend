resource "google_storage_bucket" "daemon_install" {
  name     = "install.cortado.dev"
  location = var.region

  labels = local.common_labels

  project                     = var.project_id
  public_access_prevention    = "inherited"
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "daemon_install_public_read" {
  bucket = google_storage_bucket.daemon_install.name
  member = "allUsers"
  role   = "roles/storage.objectViewer"
}

resource "google_storage_bucket_object" "daemon_install_script" {
  bucket       = google_storage_bucket.daemon_install.name
  name         = "daemon"
  content_type = "text/x-shellscript"
  source       = "${path.module}/../../../scripts/install_cortado_daemon.sh"
}
