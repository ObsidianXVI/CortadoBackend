variable "env" {
  description = "Deployment environment name."
  type        = string
}

variable "cluster_dns_domain" {
  description = "Additive VPC scope DNS domain used to resolve workspace Services from Cloud Run."
  type        = string
}

variable "cluster_name" {
  description = "GKE cluster name used by the control plane for workspace API operations."
  type        = string
}

variable "auth_cache_addr" {
  description = "Redis-compatible cache address used for auth API-key validation."
  type        = string
}

variable "image_tag" {
  description = "Artifact Registry image tag for the control-plane service."
  type        = string
}

variable "jwt_private_key_secret_id" {
  description = "Secret Manager secret ID containing the JWT private key PEM."
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

variable "network_name" {
  description = "VPC network name used for Direct VPC egress."
  type        = string
}

variable "region" {
  description = "Google Cloud region."
  type        = string
}

variable "repository_id" {
  description = "Artifact Registry repository ID for Cortado images."
  type        = string
}

variable "service_account_email" {
  description = "Service account email the Cloud Run service runs as."
  type        = string
}

variable "subnetwork_name" {
  description = "Subnetwork name used for Direct VPC egress."
  type        = string
}

variable "workspace_namespace" {
  description = "Kubernetes namespace that contains workspace Services."
  type        = string
}

variable "usage_events_topic_name" {
  description = "Pub/Sub topic name that receives usage events from workspace agents."
  type        = string
}
