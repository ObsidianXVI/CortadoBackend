# DECISIONS

## 23/05/26

- v0.1 uses Docker Hub for container image distribution instead of Google Artifact Registry.
  Rationale: reduce early infrastructure surface area and avoid managing a separate Google-hosted registry before demand is validated. Terraform and deployment specs should reference Docker Hub image names rather than `*.pkg.dev`.
