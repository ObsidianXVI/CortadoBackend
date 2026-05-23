variable "env" {
  description = "Deployment environment name."
  type        = string
}

variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "region" {
  description = "Google Cloud region for deployed resources."
  type        = string
}

variable "workspace_image_name" {
  description = "Artifact Registry image name for the workspace agent."
  type        = string
  default     = "cortado-workspace"
}

variable "workspace_image_tag" {
  description = "Artifact Registry image tag for the workspace agent test pod."
  type        = string
}

variable "workspace_test_pod_enabled" {
  description = "Whether to apply the one-off workspace agent test pod manifest."
  type        = bool
  default     = false
}
