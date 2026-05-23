variable "env" {
  description = "Deployment environment name."
  type        = string
}

variable "image_tag" {
  description = "Artifact Registry image tag for the control-plane service."
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
