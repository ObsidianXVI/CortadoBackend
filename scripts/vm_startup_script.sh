#!/usr/bin/env bash
# GCP VM Startup Script
# Runs as root on every boot and keeps the VM runtime aligned with the tailnet.

set -euo pipefail

info()  { echo "[INFO] $*"; }
warn()  { echo "[WARN] $*" >&2; }

PROJECT_DIR="${CORTADO_PROJECT_DIR:-$(pwd)}"
SSH_USER="${CORTADO_SSH_USER:-${SUDO_USER:-}}"
if [[ -z "${SSH_USER}" || "${SSH_USER}" == "root" ]]; then
  SSH_USER="$(stat -c %U "${PROJECT_DIR}" 2>/dev/null || true)"
fi
if [[ -z "${SSH_USER}" || "${SSH_USER}" == "root" ]]; then
  SSH_USER="$(getent passwd 1000 | cut -d: -f1)"
fi
if [[ -z "${SSH_USER}" ]]; then
  SSH_USER="${USER:-ubuntu}"
fi

SSH_HOME="$(getent passwd "${SSH_USER}" | cut -d: -f6)"
if [[ -z "${SSH_HOME}" ]]; then
  SSH_HOME="/home/${SSH_USER}"
fi

SSH_GROUP="$(id -gn "${SSH_USER}" 2>/dev/null || echo "${SSH_USER}")"
BIN_DIR="${SSH_HOME}/bin"
ENV_FILE="${PROJECT_DIR}/.env"

FLUTTER_VERSION="3.41.9"
DART_VERSION="3.11.5"
FLUTTER_BIN="${SSH_HOME}/tools/flutter/bin/flutter"
DART_BIN="${SSH_HOME}/tools/flutter/bin/dart"

CODEX_RELAY_VERSION="1.1.0"
CODEX_RELAY_WRAPPER="${BIN_DIR}/codex-relay"
CODEX_RELAY_STARTER="${BIN_DIR}/codex-relay-start"
CODEX_RELAY_STOPPER="${BIN_DIR}/codex-relay-stop"
RUNTIME_START="/usr/local/bin/cortado-vm-runtime-start"
RUNTIME_STOP="/usr/local/bin/cortado-vm-runtime-stop"
RUNTIME_UNIT="/etc/systemd/system/cortado-vm-runtime.service"
RELAY_UNIT="/etc/systemd/system/cortado-codex-relay.service"

install -d -m 0755 /usr/local/bin /etc/systemd/system
install -d -m 0755 "${BIN_DIR}"
install -d -o "${SSH_USER}" -g "${SSH_GROUP}" -m 0755 "${PROJECT_DIR}"

info "Writing Codex Relay wrapper for ${SSH_USER}..."
cat > "${CODEX_RELAY_WRAPPER}" <<EOF
#!/usr/bin/env bash
set -euo pipefail

if [[ -f "${ENV_FILE}" ]]; then
  set -o allexport
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +o allexport
fi

WORKSPACE_DIR="\${CODEX_RELAY_WORKSPACE_PATH:-\${CODEX_RELAY_WORKSPACE:-${PROJECT_DIR}}}"
export CODEX_RELAY_WORKSPACE_PATH="\$WORKSPACE_DIR"
export CODEX_HOME="\${CODEX_HOME:-${SSH_HOME}/.codex}"
export CODEX_BIN="\${CODEX_BIN:-\$(command -v codex || true)}"
export CODEX_RELAY_AUTH_DB_PATH="\${CODEX_RELAY_AUTH_DB_PATH:-\$WORKSPACE_DIR/.codex-relay/auth.db}"

cd "\$WORKSPACE_DIR"
exec npx --yes "codex-relay@${CODEX_RELAY_VERSION}" "\$@"
EOF
chmod 0755 "${CODEX_RELAY_WRAPPER}"
chown -R "${SSH_USER}:${SSH_GROUP}" "${BIN_DIR}"

info "Writing Codex Relay start helper for ${SSH_USER}..."
cat > "${CODEX_RELAY_STARTER}" <<EOF
#!/usr/bin/env bash
set -euo pipefail

if [[ -f "${ENV_FILE}" ]]; then
  set -o allexport
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +o allexport
fi

PROJECT_DIR="${PROJECT_DIR}"
RELAY_BIN="${BIN_DIR}/codex-relay"
WORKSPACE_DIR="\${CODEX_RELAY_WORKSPACE_PATH:-\${CODEX_RELAY_WORKSPACE:-\$PROJECT_DIR}}"
export CODEX_RELAY_WORKSPACE_PATH="\$WORKSPACE_DIR"
export CODEX_HOME="\${CODEX_HOME:-${SSH_HOME}/.codex}"
export CODEX_BIN="\${CODEX_BIN:-\$(command -v codex || true)}"
export CODEX_RELAY_AUTH_DB_PATH="\${CODEX_RELAY_AUTH_DB_PATH:-\$WORKSPACE_DIR/.codex-relay/auth.db}"
cd "\$WORKSPACE_DIR"

find_relay_pid() {
  local pid line
  if [[ -f "\$PROJECT_DIR/.codex-relay/server.pid" ]]; then
    pid="\$(cat "\$PROJECT_DIR/.codex-relay/server.pid" 2>/dev/null || true)"
    if [[ "\$pid" =~ ^[0-9]+$ ]] && kill -0 "\$pid" 2>/dev/null; then
      echo "\$pid"
      return 0
    fi
  fi

  if command -v ss >/dev/null 2>&1; then
    line="\$(ss -lptn 'sport = :8787' 2>/dev/null | awk 'NR>1 {print; exit}' || true)"
    if [[ "\$line" =~ pid=([0-9]+) ]]; then
      pid="\${BASH_REMATCH[1]}"
      if kill -0 "\$pid" 2>/dev/null; then
        echo "\$pid"
        return 0
      fi
    fi
  fi
}

use_existing_relay() {
  local pid cwd
  pid="\$(find_relay_pid)"
  if [[ -z "\$pid" ]]; then
    return 1
  fi

  cwd="\$(readlink "/proc/\$pid/cwd" 2>/dev/null || true)"
  if [[ -n "\$cwd" && -d "\$cwd" ]]; then
    cd "\$cwd"
  fi
  return 0
}

if command -v tailscale >/dev/null 2>&1; then
  TAILSCALE_IP="\$(tailscale ip -4 2>/dev/null | head -1 || true)"
  if [[ -n "\$TAILSCALE_IP" ]]; then
    export CODEX_RELAY_PUBLIC_URL="http://\${TAILSCALE_IP}:8787"
  fi
fi

if use_existing_relay; then
  for _ in {1..20}; do
    if "\$RELAY_BIN" qr; then
      exit 0
    fi
    sleep 1
  done
  echo "Codex Relay is running, but the QR is not ready yet. Try: \$RELAY_BIN qr" >&2
  exit 1
fi

"\$RELAY_BIN" --bg
for _ in {1..20}; do
  if "\$RELAY_BIN" qr; then
    exit 0
  fi
  sleep 1
done

echo "Codex Relay started, but the QR is not ready yet. Try: \$RELAY_BIN qr" >&2
exit 1
EOF
chmod 0755 "${CODEX_RELAY_STARTER}"
chown "${SSH_USER}:${SSH_GROUP}" "${CODEX_RELAY_STARTER}"

info "Writing Codex Relay stop helper for ${SSH_USER}..."
cat > "${CODEX_RELAY_STOPPER}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${CODEX_RELAY_WORKSPACE_PATH:-${CODEX_RELAY_WORKSPACE:-$HOME/CortadoBackend}}"
PID_FILE="$PROJECT_DIR/.codex-relay/server.pid"

find_relay_pid() {
  local pid line
  if [[ -f "$PID_FILE" ]]; then
    pid="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
      echo "$pid"
      return 0
    fi
  fi

  if command -v ss >/dev/null 2>&1; then
    line="$(ss -lptn 'sport = :8787' 2>/dev/null | awk 'NR>1 {print; exit}' || true)"
    if [[ "$line" =~ pid=([0-9]+) ]]; then
      pid="${BASH_REMATCH[1]}"
      if kill -0 "$pid" 2>/dev/null; then
        echo "$pid"
        return 0
      fi
    fi
  fi
}

PID="$(find_relay_pid || true)"
if [[ -n "$PID" ]]; then
  kill -TERM "$PID"
  for _ in {1..30}; do
    if ! kill -0 "$PID" 2>/dev/null; then
      break
    fi
    sleep 1
  done
fi
EOF
chmod 0755 "${CODEX_RELAY_STOPPER}"
chown "${SSH_USER}:${SSH_GROUP}" "${CODEX_RELAY_STOPPER}"

info "Writing runtime hooks..."
cat > "${RUNTIME_START}" <<EOF
#!/usr/bin/env bash
set -euo pipefail

SSH_USER="${SSH_USER}"
SSH_HOME="${SSH_HOME}"
ENV_FILE="${ENV_FILE}"
FLUTTER_BIN="${FLUTTER_BIN}"
DART_BIN="${DART_BIN}"
EXPECTED_FLUTTER_VERSION="${FLUTTER_VERSION}"
EXPECTED_DART_VERSION="${DART_VERSION}"

export PATH="${SSH_HOME}/tools/flutter/bin:${SSH_HOME}/go/bin:${SSH_HOME}/bin:/usr/local/go/bin:${PATH}"

if [[ -f "\${ENV_FILE}" ]]; then
  set -o allexport
  # shellcheck disable=SC1090
  source "\${ENV_FILE}"
  set +o allexport
  printenv OPENAI_API_KEY | codex login --with-api-key
fi

if [[ -x "\${FLUTTER_BIN}" ]]; then
  FLUTTER_INSTALLED_VERSION="$("\${FLUTTER_BIN}" --version 2>/dev/null | head -1 | awk '{print \$2}')"
  if [[ "\${FLUTTER_INSTALLED_VERSION}" != "\${EXPECTED_FLUTTER_VERSION}" ]]; then
    echo "[WARN] Expected Flutter \${EXPECTED_FLUTTER_VERSION}, found \${FLUTTER_INSTALLED_VERSION:-missing} at \${FLUTTER_BIN}." >&2
  fi
else
  echo "[WARN] Flutter is not installed at \${FLUTTER_BIN}. Run scripts/initial_setup.sh to install Flutter \${EXPECTED_FLUTTER_VERSION}." >&2
fi

if [[ -x "\${DART_BIN}" ]]; then
  DART_INSTALLED_VERSION="$("\${DART_BIN}" --version 2>&1 | sed -E 's/^Dart SDK version: ([0-9.]+).*/\1/' | head -1)"
  if [[ "\${DART_INSTALLED_VERSION}" != "\${EXPECTED_DART_VERSION}" ]]; then
    echo "[WARN] Expected Dart \${EXPECTED_DART_VERSION}, found \${DART_INSTALLED_VERSION:-missing} at \${DART_BIN}." >&2
  fi
else
  echo "[WARN] Dart is not installed at \${DART_BIN}. Run scripts/initial_setup.sh to install bundled Dart \${EXPECTED_DART_VERSION}." >&2
fi

if command -v tailscale >/dev/null 2>&1; then
  systemctl enable --now tailscaled >/dev/null 2>&1 || true

  if tailscale status --json 2>/dev/null | jq -e '.BackendState == "Running"' >/dev/null 2>&1; then
    tailscale set --ssh >/dev/null 2>&1 || true
  elif [[ -n "\${TS_AUTHKEY:-}" ]]; then
    UP_ARGS=(--auth-key="\${TS_AUTHKEY}" --ssh)
    [[ -n "\${TS_HOSTNAME:-}" ]] && UP_ARGS+=(--hostname="\${TS_HOSTNAME}")
    [[ -n "\${TS_TAGS:-}" ]] && UP_ARGS+=(--advertise-tags="\${TS_TAGS}")
    tailscale up "\${UP_ARGS[@]}"
  else
    echo "[WARN] TS_AUTHKEY is not set; skipping tailscale up." >&2
  fi
fi

systemctl start cortado-codex-relay.service >/dev/null 2>&1 || true
EOF
chmod 0755 "${RUNTIME_START}"

cat > "${RUNTIME_STOP}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

systemctl stop cortado-codex-relay.service >/dev/null 2>&1 || true

if command -v tailscale >/dev/null 2>&1; then
  tailscale down >/dev/null 2>&1 || true
fi

systemctl stop tailscaled >/dev/null 2>&1 || true
EOF
chmod 0755 "${RUNTIME_STOP}"

info "Writing systemd units..."
cat > "${RELAY_UNIT}" <<EOF
[Unit]
Description=Cortado Codex Relay
After=network-online.target tailscaled.service
Wants=network-online.target tailscaled.service

[Service]
Type=oneshot
RemainAfterExit=yes
User=${SSH_USER}
Group=${SSH_GROUP}
WorkingDirectory=${PROJECT_DIR}
Environment=HOME=${SSH_HOME}
Environment=CODEX_RELAY_WORKSPACE_PATH=${PROJECT_DIR}
ExecStart=${CODEX_RELAY_STARTER}
ExecStop=${CODEX_RELAY_STOPPER}
TimeoutStartSec=120
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
EOF

cat > "${RUNTIME_UNIT}" <<EOF
[Unit]
Description=Cortado VM runtime coordinator
After=network-online.target tailscaled.service cortado-codex-relay.service
Wants=network-online.target tailscaled.service cortado-codex-relay.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=${RUNTIME_START}
ExecStop=${RUNTIME_STOP}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
if command -v tailscale >/dev/null 2>&1; then
  systemctl enable --now tailscaled >/dev/null 2>&1 || true
fi
systemctl enable --now cortado-codex-relay.service >/dev/null 2>&1 || true
systemctl enable --now cortado-vm-runtime.service >/dev/null 2>&1 || true

info "Tailnet runtime is now managed by systemd."
