#!/usr/bin/env bash

set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/cortado_dev_smoke.sh"

usage() {
  cat <<'EOF'
Usage:
  scripts/dev_portforward_probe.sh [workspace-id] [port] [path]

Examples:
  scripts/dev_portforward_probe.sh
  scripts/dev_portforward_probe.sh ws-123 8080 /
  scripts/dev_portforward_probe.sh ws-123 8080 /index.html

Defaults:
  workspace-id  uses CORTADO_WORKSPACE_ID or .tmp/portforward-smoke/workspace-env.sh
  port          8080
  path          /
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || "${1:-}" == "help" ]]; then
  usage
  exit 0
fi

require_cmd curl
require_cmd terraform

workspace_id="${1:-$(saved_workspace_id)}"
port="${2:-8080}"
path_part="${3:-/}"
base_url="$(portforward_url)"
trimmed_path="${path_part#/}"
target_url="${base_url%/}/${workspace_id}/${port}"

if [[ -n "$trimmed_path" && "$trimmed_path" != "/" ]]; then
  target_url="${target_url}/${trimmed_path}"
else
  target_url="${target_url}/"
fi

printf 'Probing: %s\n\n' "$target_url"

curl --include --show-error --silent --fail-with-body \
  -H "X-Cortado-Dev-Token: dev-bypass" \
  "$target_url"
