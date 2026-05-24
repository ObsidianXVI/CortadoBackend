#!/usr/bin/env bash

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/cortado_dev_smoke.sh"

usage() {
  cat <<'EOF'
Usage:
  scripts/dev_workspace.sh create [workspace-image]
  scripts/dev_workspace.sh status [workspace-id]
  scripts/dev_workspace.sh wait-running [workspace-id]
  scripts/dev_workspace.sh start [workspace-id]
  scripts/dev_workspace.sh stop [workspace-id]
  scripts/dev_workspace.sh delete [workspace-id]

Defaults:
  workspace-image  defaults to the dev workspace image from terraform/envs/dev
  workspace-id     defaults to CORTADO_WORKSPACE_ID or .tmp/portforward-smoke/workspace-env.sh
EOF
}

workspace_id_arg() {
  if [[ -n "${1:-}" ]]; then
    printf '%s\n' "$1"
    return 0
  fi
  saved_workspace_id
}

status_request() {
  local workspace_id="$1"
  curl_json "$(workspace_api_url "$workspace_id")"
}

print_status() {
  local workspace_id="$1"
  status_request "$workspace_id" | jq
}

wait_running() {
  local workspace_id="$1"
  local timeout_seconds="${CORTADO_WORKSPACE_READY_TIMEOUT_SECONDS:-240}"
  local poll_seconds="${CORTADO_WORKSPACE_POLL_SECONDS:-3}"
  local deadline status

  deadline=$((SECONDS + timeout_seconds))
  while (( SECONDS < deadline )); do
    status="$(status_request "$workspace_id" | jq -r '.workspace.status')"
    printf 'workspace %s status: %s\n' "$workspace_id" "$status"
    case "$status" in
      RUNNING)
        return 0
        ;;
      CREATING|STARTING)
        sleep "$poll_seconds"
        ;;
      *)
        printf 'workspace entered unexpected status: %s\n' "$status" >&2
        return 1
        ;;
    esac
  done

  printf 'timed out waiting for workspace %s to become RUNNING\n' "$workspace_id" >&2
  return 1
}

transition_workspace() {
  local action="$1"
  local workspace_id="$2"
  local http_code body_file

  body_file="$(mktemp)"
  http_code="$(
    curl --silent --show-error \
      --output "$body_file" \
      --write-out '%{http_code}' \
      -X POST \
      -H "X-Cortado-Dev-Token: dev-bypass" \
      "$(workspace_api_url "$workspace_id" "$action")"
  )"

  if [[ "$http_code" != "202" ]]; then
    printf '%s failed with status %s\n' "$action" "$http_code" >&2
    cat "$body_file" >&2
    rm -f "$body_file"
    exit 1
  fi

  jq . <"$body_file"
  rm -f "$body_file"
}

delete_workspace() {
  local workspace_id="$1"
  local http_code body_file

  body_file="$(mktemp)"
  http_code="$(
    curl --silent --show-error \
      --output "$body_file" \
      --write-out '%{http_code}' \
      -X DELETE \
      -H "X-Cortado-Dev-Token: dev-bypass" \
      "$(workspace_api_url "$workspace_id")"
  )"

  if [[ "$http_code" != "202" ]]; then
    printf 'delete failed with status %s\n' "$http_code" >&2
    cat "$body_file" >&2
    rm -f "$body_file"
    exit 1
  fi

  jq . <"$body_file"
  rm -f "$body_file"
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

require_cmd curl
require_cmd jq
require_cmd terraform

command="$1"
shift

case "$command" in
  create)
    workspace_image="${1:-$(default_workspace_image)}"
    body_file="$(mktemp)"
    http_code="$(
      curl --silent --show-error \
        --output "$body_file" \
        --write-out '%{http_code}' \
        -X POST \
        -H "X-Cortado-Dev-Token: dev-bypass" \
        -H "Content-Type: application/json" \
        -d "$(jq -cn --arg image "$workspace_image" '{image: $image}')" \
        "$(workspace_api_url)"
    )"

    if [[ "$http_code" != "202" ]]; then
      printf 'create failed with status %s\n' "$http_code" >&2
      cat "$body_file" >&2
      rm -f "$body_file"
      exit 1
    fi

    workspace_id="$(jq -r '.workspace.id' <"$body_file")"
    if [[ -z "$workspace_id" || "$workspace_id" == "null" ]]; then
      printf 'workspace create response did not include an id\n' >&2
      cat "$body_file" >&2
      rm -f "$body_file"
      exit 1
    fi

    write_workspace_env_file "$workspace_id" "$workspace_image"
    jq . <"$body_file"
    rm -f "$body_file"

    printf '\nSaved workspace env file: %s\n' "$(workspace_env_file)"
    wait_running "$workspace_id"
    printf '\nWorkspace ready.\n'
    printf 'source %s\n' "$(workspace_env_file)"
    ;;
  status)
    print_status "$(workspace_id_arg "${1:-}")"
    ;;
  wait-running)
    wait_running "$(workspace_id_arg "${1:-}")"
    ;;
  start)
    transition_workspace start "$(workspace_id_arg "${1:-}")"
    ;;
  stop)
    transition_workspace stop "$(workspace_id_arg "${1:-}")"
    ;;
  delete)
    delete_workspace "$(workspace_id_arg "${1:-}")"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    printf 'unknown command: %s\n\n' "$command" >&2
    usage >&2
    exit 1
    ;;
esac
