variable "control_plane_sa_email" {
  description = "Email address of the control plane service account."
  type        = string
}

variable "env" {
  description = "Deployment environment name."
  type        = string
}

variable "labels" {
  description = "Common labels applied to supported resources."
  type        = map(string)
}

variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "region" {
  description = "Google Cloud region for regional resources."
  type        = string
}

variable "workspace_agent_sa_name" {
  description = "Fully-qualified name of the workspace agent service account."
  type        = string
}
