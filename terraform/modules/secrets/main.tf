resource "google_secret_manager_secret" "jwt_private_key" {
  project   = var.project_id
  secret_id = "cortado-jwt-private-key-${var.env}"
  labels    = var.labels

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_iam_member" "control_plane_reader" {
  secret_id = google_secret_manager_secret.jwt_private_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = var.control_plane_service_account_member
}
