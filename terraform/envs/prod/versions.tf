terraform {
  required_version = ">= 1.9.8, < 2.0.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  backend "gcs" {
    bucket = "cortado-tf-state-prod"
    prefix = "terraform/state"
  }
}
