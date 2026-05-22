variable "project_id" {
  description = "GCP project ID that owns the IAM resources."
  type        = string
}

variable "env" {
  description = "Environment name used for resource labels."
  type        = string
}

variable "control_plane_project_roles" {
  description = "Project IAM roles granted to the control-plane service account."
  type        = set(string)
  default     = ["roles/container.developer"]
}
