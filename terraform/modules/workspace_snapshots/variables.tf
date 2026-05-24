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
  description = "Google Cloud region for the bucket location."
  type        = string
}

variable "workspace_agent_service_account_email" {
  description = "Workspace agent service account email granted bucket access."
  type        = string
}
