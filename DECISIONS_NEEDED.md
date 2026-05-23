# Decisions Needed

## 23/05/26

- How should the v0.1 control plane reach workspace agents from Cloud Run?
  Context:
  - The current bridge dials in-cluster DNS names like `<workspace>.cortado-workspaces.svc.cluster.local:9090`.
  - Cloud Run is not currently configured with private VPC egress or GKE DNS visibility, so this data-plane path cannot work as implemented.
  - Task 1.4.4 smoke testing is blocked until we choose one of:
    1. Keep Cloud Run for the control plane and add the required private networking / DNS support so Cloud Run can reach the workspace headless Service.
    2. Change v0.1 deployment so the control plane runs inside GKE instead of Cloud Run, where the existing in-cluster DNS model already works.
  Recommendation:
  - Prefer option 1 if we want to stay aligned with the current v0.1 spec and Terraform module layout.
