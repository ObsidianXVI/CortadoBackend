resource "google_service_account" "control_plane" {
  account_id   = "cortado-control-plane-${var.env}"
  display_name = "Cortado Control Plane"
  project      = var.project_id
  description  = "Cortado control plane service account for ${var.env}."
}

resource "google_service_account" "workspace_agent" {
  account_id   = "cortado-workspace-agent-${var.env}"
  display_name = "Cortado Workspace Agent"
  project      = var.project_id
  description  = "Cortado workspace agent service account for ${var.env}."
}

resource "google_project_iam_member" "control_plane_project_roles" {
  for_each = var.control_plane_project_roles

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.control_plane.email}"
}
