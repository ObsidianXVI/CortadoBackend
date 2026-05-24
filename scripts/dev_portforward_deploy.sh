#!/usr/bin/env bash

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/cortado_dev_smoke.sh"

usage() {
  cat <<'EOF'
Usage:
  scripts/dev_portforward_deploy.sh [image-tag]

Builds the portforward Cloud Run image, pushes it to Artifact Registry, applies
the dev Terraform stack with the new image tag, and writes:

  .tmp/portforward-smoke/dev-env.sh

If no image tag is supplied, a UTC timestamp tag is generated.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

require_cmd docker
require_cmd gcloud
require_cmd terraform

repo_root="$(cortado_repo_root)"
tag="${1:-$(date -u +%Y%m%d-%H%M%S-portforward)}"
region="$(dev_region)"
project_id="$(dev_project_id)"
repository_id="$(artifact_repository_id)"
image="${region}-docker.pkg.dev/${project_id}/${repository_id}/cortado-portforward:${tag}"

printf 'Using image tag: %s\n' "$tag"
printf 'Artifact Registry image: %s\n' "$image"

gcloud auth configure-docker "${region}-docker.pkg.dev" --quiet
docker build -f "${repo_root}/control-plane/Dockerfile.portforward" -t "$image" "$repo_root"
docker push "$image"

terraform -chdir="${repo_root}/terraform/envs/dev" apply -auto-approve -var="portforward_image_tag=${tag}"

write_dev_env_file "$image" "$tag"

printf '\nDeploy complete.\n'
printf 'Control plane URL : %s\n' "$(control_plane_url)"
printf 'Portforward URL   : %s\n' "$(portforward_url)"
printf 'Saved env file    : %s\n' "$(dev_env_file)"
printf '\nNext:\n'
printf '  source %s\n' "$(dev_env_file)"
printf '  ./scripts/dev_workspace.sh create\n'
