resource "google_container_cluster" "main" {
  name     = "cortado-${var.env}"
  location = var.region

  enable_autopilot = true

  release_channel {
    channel = "RAPID"
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

# This requires gcloud to be available where terraform apply runs.
resource "null_resource" "enable_criu" {
  triggers = {
    cluster = google_container_cluster.main.id
  }

  provisioner "local-exec" {
    command = <<-EOT
      gcloud container clusters update ${google_container_cluster.main.name} \
        --region ${var.region} \
        --enable-checkpoint-restore \
        --project ${var.project_id}
    EOT
  }
}
