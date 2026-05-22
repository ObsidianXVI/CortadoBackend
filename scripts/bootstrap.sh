#!/usr/bin/env bash
# Run once, never again.

gcloud storage buckets create gs://cortado-tf-state-dev \
  --location=us-central1 \
  --uniform-bucket-level-access
