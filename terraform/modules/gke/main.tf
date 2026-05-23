resource "google_container_cluster" "main" {
  name     = "cortado-${var.env}"
  location = var.region

  deletion_protection = false
  enable_autopilot    = true
  network             = var.network_name
  subnetwork          = var.subnetwork_name

  release_channel {
    channel = "RAPID"
  }

  dns_config {
    additive_vpc_scope_dns_domain = var.cluster_dns_domain
    cluster_dns                   = "CLOUD_DNS"
    cluster_dns_scope             = "CLUSTER_SCOPE"
  }

  resource_labels = var.labels

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
}

resource "google_service_account_iam_member" "workspace_agent_wi" {
  service_account_id = var.workspace_agent_sa_name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[cortado-workspaces/workspace-sa]"
}

resource "google_artifact_registry_repository" "main" {
  location      = var.region
  repository_id = "cortado-${var.env}"
  format        = "DOCKER"
  labels        = var.labels
}

resource "google_artifact_registry_repository_iam_member" "control_plane_writer" {
  location   = google_artifact_registry_repository.main.location
  repository = google_artifact_registry_repository.main.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${var.control_plane_sa_email}"
}
