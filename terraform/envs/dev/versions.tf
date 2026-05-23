terraform {
  required_version = ">= 1.9.8, < 2.0.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }

  backend "gcs" {
    bucket = "cortado-tf-state-dev"
    prefix = "terraform/state"
  }
}
