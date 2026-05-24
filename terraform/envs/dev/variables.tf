variable "env" {
  description = "Deployment environment name."
  type        = string
}

variable "control_plane_image_tag" {
  description = "Artifact Registry image tag for the control-plane service."
  type        = string
}

variable "indexer_updater_image_tag" {
  description = "Artifact Registry image tag for the indexer-updater service."
  type        = string
}

variable "cluster_dns_domain" {
  description = "Additive VPC scope DNS domain used for headless Service discovery from Cloud Run."
  type        = string
}

variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "network_name" {
  description = "VPC network name used by the cluster and Cloud Run service."
  type        = string
}

variable "region" {
  description = "Google Cloud region for deployed resources."
  type        = string
}

variable "subnetwork_name" {
  description = "Subnetwork name used by the cluster and Cloud Run service."
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

variable "workspace_namespace" {
  description = "Kubernetes namespace that contains workspace Services."
  type        = string
  default     = "cortado-workspaces"
}
