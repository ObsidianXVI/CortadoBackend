variable "control_plane_sa_email" {
  description = "Email address of the control plane service account."
  type        = string
}

variable "cluster_dns_domain" {
  description = "Additive VPC scope DNS domain used by non-GKE clients to resolve headless Services."
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

variable "network_name" {
  description = "VPC network name used by the GKE cluster and Cloud Run egress."
  type        = string
}

variable "region" {
  description = "Google Cloud region for regional resources."
  type        = string
}

variable "subnetwork_name" {
  description = "Subnetwork name used by the GKE cluster and Cloud Run egress."
  type        = string
}

variable "workspace_agent_sa_name" {
  description = "Fully-qualified name of the workspace agent service account."
  type        = string
}
