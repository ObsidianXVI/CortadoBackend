# DECISIONS

## 23/05/26

- v0.1 uses Google Artifact Registry for container image distribution.
  Rationale: keep the image registry colocated with GKE in `us-central1` so workspace and control-plane image pulls stay in-region, reducing cross-region transfer risk and keeping GCP IAM-based access control for deploys. Terraform and deployment specs should reference `us-central1-docker.pkg.dev/...` image names.
