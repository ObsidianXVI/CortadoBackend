# Default Firestore database (Native mode) in us-central1.
resource "google_firestore_database" "default" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"

  depends_on = [
    google_project_service.api["firestore.googleapis.com"],
  ]
}
