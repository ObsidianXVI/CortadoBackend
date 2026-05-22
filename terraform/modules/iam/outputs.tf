output "control_plane_service_account_email" {
  description = "Email address of the control-plane service account."
  value       = google_service_account.control_plane.email
}

output "control_plane_service_account_name" {
  description = "Fully-qualified name of the control-plane service account."
  value       = google_service_account.control_plane.name
}

output "control_plane_service_account_member" {
  description = "IAM member string for the control-plane service account."
  value       = "serviceAccount:${google_service_account.control_plane.email}"
}

output "workspace_agent_service_account_email" {
  description = "Email address of the workspace-agent service account."
  value       = google_service_account.workspace_agent.email
}

output "workspace_agent_service_account_name" {
  description = "Fully-qualified name of the workspace-agent service account."
  value       = google_service_account.workspace_agent.name
}

output "workspace_agent_service_account_member" {
  description = "IAM member string for the workspace-agent service account."
  value       = "serviceAccount:${google_service_account.workspace_agent.email}"
}

output "control_plane_project_roles" {
  description = "Project IAM roles granted to the control-plane service account."
  value       = [for binding in google_project_iam_member.control_plane_project_roles : binding.role]
}
