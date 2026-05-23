variable "control_plane_service_account_member" {
  description = "IAM member string for the control-plane service account."
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
