#!/usr/bin/env bash

set -euo pipefail

cortado_repo_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd
}

cortado_state_dir() {
  printf '%s\n' "$(cortado_repo_root)/.tmp/portforward-smoke"
}

ensure_cortado_state_dir() {
  mkdir -p "$(cortado_state_dir)"
}

dev_tfvars_path() {
  printf '%s\n' "$(cortado_repo_root)/terraform/envs/dev/terraform.tfvars"
}

dev_env_file() {
  printf '%s\n' "$(cortado_state_dir)/dev-env.sh"
}

workspace_env_file() {
  printf '%s\n' "$(cortado_state_dir)/workspace-env.sh"
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    printf 'missing required command: %s\n' "$cmd" >&2
    exit 1
  fi
}

tf_output_raw() {
  local name="$1"
  terraform -chdir="$(cortado_repo_root)/terraform/envs/dev" output -raw "$name"
}

tfvars_value() {
  local key="$1"
  awk -F= -v key="$key" '
    $1 ~ "^[[:space:]]*" key "[[:space:]]*$" {
      value = $2
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      gsub(/^"/, "", value)
      gsub(/"$/, "", value)
      print value
      exit
    }
  ' "$(dev_tfvars_path)"
}

dev_project_id() {
  printf '%s\n' "${CORTADO_PROJECT_ID:-$(tfvars_value project_id)}"
}

dev_region() {
  printf '%s\n' "${CORTADO_REGION:-$(tfvars_value region)}"
}

artifact_repository_id() {
  printf '%s\n' "${CORTADO_ARTIFACT_REPOSITORY_ID:-$(tf_output_raw artifact_registry_repository_id)}"
}

control_plane_url() {
  printf '%s\n' "${CORTADO_CONTROL_PLANE_URL:-$(tf_output_raw control_plane_service_uri)}"
}

portforward_url() {
  printf '%s\n' "${CORTADO_PORTFORWARD_URL:-$(tf_output_raw portforward_service_uri)}"
}

default_workspace_image() {
  local image_name image_tag
  image_name="${CORTADO_WORKSPACE_IMAGE_NAME:-cortado-workspace}"
  image_tag="${CORTADO_WORKSPACE_IMAGE_TAG:-$(tfvars_value workspace_image_tag)}"
  printf '%s-docker.pkg.dev/%s/%s/%s:%s\n' \
    "$(dev_region)" \
    "$(dev_project_id)" \
    "$(artifact_repository_id)" \
    "$image_name" \
    "$image_tag"
}

saved_workspace_id() {
  if [[ -n "${CORTADO_WORKSPACE_ID:-}" ]]; then
    printf '%s\n' "$CORTADO_WORKSPACE_ID"
    return 0
  fi

  if [[ -f "$(workspace_env_file)" ]]; then
    # shellcheck disable=SC1090
    source "$(workspace_env_file)"
    if [[ -n "${CORTADO_WORKSPACE_ID:-}" ]]; then
      printf '%s\n' "$CORTADO_WORKSPACE_ID"
      return 0
    fi
  fi

  printf '\n' >&2
  printf 'no workspace id found. Set CORTADO_WORKSPACE_ID or run scripts/dev_workspace.sh create first.\n' >&2
  exit 1
}

workspace_api_url() {
  local workspace_id="${1:-}"
  local action="${2:-}"
  local base

  base="$(control_plane_url)"
  if [[ -z "$workspace_id" ]]; then
    printf '%s/v1/workspaces\n' "${base%/}"
    return 0
  fi

  if [[ -z "$action" ]]; then
    printf '%s/v1/workspaces/%s\n' "${base%/}" "$workspace_id"
    return 0
  fi

  printf '%s/v1/workspaces/%s/%s\n' "${base%/}" "$workspace_id" "$action"
}

write_dev_env_file() {
  local image="$1"
  local tag="$2"

  ensure_cortado_state_dir
  cat >"$(dev_env_file)" <<EOF
export CORTADO_CONTROL_PLANE_URL='$(control_plane_url)'
export CORTADO_PORTFORWARD_URL='$(portforward_url)'
export CORTADO_DEV_TOKEN='dev-bypass'
export CORTADO_PORTFORWARD_IMAGE='${image}'
export CORTADO_PORTFORWARD_IMAGE_TAG='${tag}'
EOF
}

write_workspace_env_file() {
  local workspace_id="$1"
  local workspace_image="$2"

  ensure_cortado_state_dir
  cat >"$(workspace_env_file)" <<EOF
export CORTADO_WORKSPACE_ID='${workspace_id}'
export CORTADO_WORKSPACE_IMAGE='${workspace_image}'
export CORTADO_CONTROL_PLANE_URL='$(control_plane_url)'
export CORTADO_PORTFORWARD_URL='$(portforward_url)'
export CORTADO_DEV_TOKEN='dev-bypass'
EOF
}

curl_json() {
  curl --silent --show-error \
    -H "X-Cortado-Dev-Token: dev-bypass" \
    -H "Content-Type: application/json" \
    "$@"
}
