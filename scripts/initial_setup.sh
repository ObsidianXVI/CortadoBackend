#!/usr/bin/env bash
# =============================================================================
# initial_setup.sh — Cortado Dev VM Bootstrap
# Run once after first SSH into the VM.
# VM: Ubuntu 26.04 minimal, e2-medium, us-central1-a, project: cortado-ide
# Run as: your SSH user (not root). Uses sudo internally where needed.
# =============================================================================
set -euo pipefail

# ── Colour output ─────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
die()     { echo -e "${RED}[FAIL]${NC} $*" >&2; exit 1; }

# ── Pinned versions (mirror devcontainer Dockerfile) ─────────────────────────
GO_VERSION=1.23.4
FLUTTER_VERSION=3.27.0
TERRAFORM_VERSION=1.9.8
BUF_VERSION=1.47.2
NODE_MAJOR=22
DOCKER_VERSION=26.0.0
DOCKER_CE_PKG_VERSION="5:${DOCKER_VERSION}-1~ubuntu.24.04~noble"
DOCKER_CLI_PKG_VERSION="5:${DOCKER_VERSION}-1~ubuntu.24.04~noble"
CONTAINERD_IO_PKG_VERSION=1.6.28-2
DOCKER_BUILDX_PLUGIN_PKG_VERSION=0.13.1-1~ubuntu.24.04~noble
DOCKER_COMPOSE_PLUGIN_PKG_VERSION=2.25.0-1~ubuntu.24.04~noble
TAILSCALE_VERSION=1.38.4
KUBECTL_VERSION=v1.36.1
CODEX_CLI_VERSION=0.133.0
CODEX_RELAY_VERSION=1.1.0
GOPLS_VERSION=v0.30.0
PROTOC_GEN_GO_VERSION=v1.36.11
PROTOC_GEN_GO_GRPC_VERSION=v1.70.0
STATICCHECK_VERSION=v0.6.1
GOLANGCI_LINT_VERSION=v1.64.8
WIRE_VERSION=v0.7.0
AIR_VERSION=v1.61.7
K9S_VERSION=0.32.7
HELM_VERSION=3.16.4

ARCH=$(dpkg --print-architecture)   # amd64
UNAME_ARCH=$(uname -m)               # x86_64

TOOLS_DIR="$HOME/tools"
mkdir -p "$TOOLS_DIR"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Source .env if present (copied via gcloud scp before this script runs) ───
ENV_FILE="$PROJECT_DIR/.env"
if [[ -f "$ENV_FILE" ]]; then
  info "Sourcing $ENV_FILE"
  set -o allexport
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +o allexport
else
  warn ".env not found at $ENV_FILE — some tools may need manual API key config."
fi

# Optional Tailscale provisioning inputs.
# Put these in ~/.env on the VM if you want setup to be fully hands-off.
TS_AUTHKEY="${TS_AUTHKEY:-}"
TS_HOSTNAME="${TS_HOSTNAME:-}"
TS_TAGS="${TS_TAGS:-}"

# =============================================================================
# 1. SYSTEM PACKAGES
# =============================================================================
info "Updating apt and installing base packages..."
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -qq
sudo apt-get upgrade -y -qq
sudo apt-get install -y -qq \
  git curl wget unzip zip xz-utils \
  tmux htop jq tree \
  build-essential pkg-config \
  ca-certificates gnupg lsb-release \
  software-properties-common apt-transport-https \
  python3 python3-pip \
  protobuf-compiler \
  openssh-client \
  netcat-openbsd \
  ripgrep fd-find \
  libssl-dev \
  socat

success "Base packages installed."


# =============================================================================
# 3. DOCKER
# =============================================================================
install_pinned_docker() {
  sudo install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | sudo gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
  sudo chmod a+r /etc/apt/keyrings/docker.gpg

  # Ubuntu 26.04 may not yet have a Docker repo entry.
  # Try the release codename first; fall back to noble (24.04) if it fails.
  CODENAME=$(. /etc/os-release && echo "${VERSION_CODENAME}")
  DOCKER_REPO="https://download.docker.com/linux/ubuntu"

  echo \
    "deb [arch=${ARCH} signed-by=/etc/apt/keyrings/docker.gpg] ${DOCKER_REPO} \
    ${CODENAME} stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

  if ! sudo apt-get update -qq 2>/dev/null | grep -q "docker"; then
    warn "Docker repo for '${CODENAME}' not found. Falling back to noble (24.04)."
    echo \
      "deb [arch=${ARCH} signed-by=/etc/apt/keyrings/docker.gpg] ${DOCKER_REPO} \
      noble stable" \
      | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt-get update -qq
  fi

  sudo rm -f /usr/local/bin/docker /usr/local/bin/dockerd \
    /usr/local/bin/containerd /usr/local/bin/docker-compose \
    /usr/local/bin/docker-buildx
  sudo apt-get install -y -qq \
    --allow-downgrades \
    --allow-change-held-packages \
    docker-ce="${DOCKER_CE_PKG_VERSION}" \
    docker-ce-cli="${DOCKER_CLI_PKG_VERSION}" \
    containerd.io="${CONTAINERD_IO_PKG_VERSION}" \
    docker-buildx-plugin="${DOCKER_BUILDX_PLUGIN_PKG_VERSION}" \
    docker-compose-plugin="${DOCKER_COMPOSE_PLUGIN_PKG_VERSION}"
}

DOCKER_INSTALLED_VERSION=""
if command -v docker &>/dev/null; then
  DOCKER_INSTALLED_VERSION="$(docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//' || true)"
fi
if [[ "$DOCKER_INSTALLED_VERSION" != "$DOCKER_VERSION" ]]; then
  info "Installing Docker CE..."
  install_pinned_docker

  # Allow current user to run Docker without sudo
  sudo usermod -aG docker "$USER"
  success "Docker ${DOCKER_VERSION} installed. NOTE: Log out and back in (or run 'newgrp docker') for group to take effect."
else
  success "Docker ${DOCKER_INSTALLED_VERSION} already installed."
fi

# Docker Hub is the registry for this setup.
warn "Docker images should be tagged as obsidianxvi/cortado:v1."

# =============================================================================
# 4. GOOGLE CLOUD SDK
# =============================================================================
if ! command -v gcloud &>/dev/null; then
  info "Installing Google Cloud SDK..."
  curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    | sudo gpg --dearmor -o /etc/apt/keyrings/cloud.google.gpg
  echo "deb [signed-by=/etc/apt/keyrings/cloud.google.gpg] \
    https://packages.cloud.google.com/apt cloud-sdk main" \
    | sudo tee /etc/apt/sources.list.d/google-cloud-sdk.list
  sudo apt-get update -qq
  sudo apt-get install -y -qq google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin
  success "gcloud installed."
else
  info "gcloud already present — updating components..."
  gcloud components update --quiet 2>/dev/null || true
  success "gcloud up to date."
fi

# Configure gcloud defaults (uses the VM's service account ADC automatically)
gcloud config set project cortado-ide --quiet
gcloud config set compute/zone us-central1-a --quiet
gcloud config set compute/region us-central1 --quiet

# =============================================================================
# 5. GO
# =============================================================================
GO_TARBALL="go${GO_VERSION}.linux-${ARCH}.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"
GO_INSTALL_DIR="/usr/local/go"

if [[ ! -d "$GO_INSTALL_DIR" ]] || \
   [[ "$(/usr/local/go/bin/go version 2>/dev/null | awk '{print $3}')" != "go${GO_VERSION}" ]]; then
  info "Installing Go ${GO_VERSION}..."
  wget -q "$GO_URL" -O "/tmp/${GO_TARBALL}"
  sudo rm -rf "$GO_INSTALL_DIR"
  sudo tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
  rm "/tmp/${GO_TARBALL}"
  success "Go ${GO_VERSION} installed."
else
  success "Go ${GO_VERSION} already installed."
fi

export PATH="/usr/local/go/bin:$PATH"
export GOPATH="$HOME/go"
export GOBIN="$HOME/go/bin"
mkdir -p "$GOBIN"

# =============================================================================
# 6. GO TOOLS (gopls, protoc-gen-go, staticcheck, etc.)
# =============================================================================
info "Installing Go tools..."
GO_TOOLS=(
  "golang.org/x/tools/gopls@${GOPLS_VERSION}"
  "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
  "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"
  "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"
  "github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
  "github.com/google/wire/cmd/wire@${WIRE_VERSION}"
  "github.com/air-verse/air@${AIR_VERSION}"       # hot reload for Go services
)
for tool in "${GO_TOOLS[@]}"; do
  go install "$tool" 2>/dev/null && success "  go install $tool" || warn "  failed: $tool"
done

# =============================================================================
# 7. NODE.JS (for Codex CLI and Codex Relay)
# =============================================================================
if ! command -v node &>/dev/null || \
   [[ "$(node --version | cut -d. -f1 | tr -d v)" -lt "$NODE_MAJOR" ]]; then
  info "Installing Node.js ${NODE_MAJOR} LTS..."
  curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | sudo -E bash -
  sudo apt-get install -y -qq nodejs
  success "Node.js $(node --version) installed."
else
  success "Node.js $(node --version) already present."
fi

# =============================================================================
# TAILSCALE + SSH
# =============================================================================
install_pinned_tailscale() {
  local codename repo_codename
  codename=$(. /etc/os-release && echo "${VERSION_CODENAME}")
  repo_codename="$codename"
  sudo install -m 0755 -d /usr/share/keyrings
  if ! curl -fsSL "https://pkgs.tailscale.com/stable/ubuntu/${repo_codename}.tailscale-keyring.list" \
      -o /tmp/tailscale.list; then
    warn "Tailscale repo does not have ${repo_codename}; falling back to noble."
    repo_codename="noble"
    curl -fsSL "https://pkgs.tailscale.com/stable/ubuntu/${repo_codename}.tailscale-keyring.list" \
      -o /tmp/tailscale.list
  fi
  curl -fsSL "https://pkgs.tailscale.com/stable/ubuntu/${repo_codename}.noarmor.gpg" \
    | sudo tee /usr/share/keyrings/tailscale-archive-keyring.gpg >/dev/null
  sudo cp /tmp/tailscale.list /etc/apt/sources.list.d/tailscale.list
  sudo rm -f /usr/local/bin/tailscale /usr/local/bin/tailscaled
  sudo apt-get update -qq
  sudo apt-get install -y -qq \
    --allow-downgrades \
    --allow-change-held-packages \
    "tailscale=${TAILSCALE_VERSION}"
}

if ! command -v tailscale &>/dev/null; then
  info "Installing Tailscale ${TAILSCALE_VERSION}..."
  install_pinned_tailscale
  success "Tailscale ${TAILSCALE_VERSION} installed."
else
  TAILSCALE_INSTALLED_VERSION="$(tailscale version 2>/dev/null | awk 'NR==1 {print $1}' || true)"
  if [[ "$TAILSCALE_INSTALLED_VERSION" != "$TAILSCALE_VERSION" ]]; then
    info "Updating Tailscale to ${TAILSCALE_VERSION}..."
    install_pinned_tailscale
    success "Tailscale ${TAILSCALE_VERSION} installed."
  else
    success "Tailscale ${TAILSCALE_INSTALLED_VERSION} already installed."
  fi
fi

if command -v tailscale &>/dev/null; then
  sudo systemctl enable --now tailscaled >/dev/null 2>&1 || true
  TAILSCALE_BACKEND_STATE="$(tailscale status --json 2>/dev/null | jq -r '.BackendState // empty' || true)"
  if [[ "$TAILSCALE_BACKEND_STATE" == "Running" ]]; then
    info "Tailscale is already connected."
    sudo tailscale set --ssh >/dev/null 2>&1 || true
    success "Tailscale SSH enabled."
  elif [[ -n "$TS_AUTHKEY" ]]; then
    info "Connecting VM to Tailscale with TS_AUTHKEY..."
    TAILSCALE_UP_ARGS=(--auth-key="$TS_AUTHKEY")
    [[ -n "$TS_HOSTNAME" ]] && TAILSCALE_UP_ARGS+=(--hostname="$TS_HOSTNAME")
    [[ -n "$TS_TAGS" ]] && TAILSCALE_UP_ARGS+=(--advertise-tags="$TS_TAGS")
    sudo tailscale up "${TAILSCALE_UP_ARGS[@]}" --ssh
    success "Tailscale connected and SSH enabled."
  else
    warn "Tailscale is installed but not connected yet."
    warn "Run: sudo tailscale up --ssh"
    warn "After the node appears in your tailnet, verify SSH is enabled with: sudo tailscale set --ssh"
  fi
fi

# =============================================================================
# 8. FLUTTER + DART
# =============================================================================
FLUTTER_DIR="$TOOLS_DIR/flutter"

if [[ ! -d "$FLUTTER_DIR" ]] || \
   [[ "$("$FLUTTER_DIR/bin/flutter" --version 2>/dev/null | head -1 | awk '{print $2}')" != "$FLUTTER_VERSION" ]]; then
  info "Installing Flutter ${FLUTTER_VERSION}..."
  rm -rf "$FLUTTER_DIR"
  git clone --depth 1 \
    --branch "$FLUTTER_VERSION" \
    https://github.com/flutter/flutter.git \
    "$FLUTTER_DIR" \
    --quiet
  export PATH="$FLUTTER_DIR/bin:$PATH"
  # Pre-cache web artifacts so `flutter run -d web-server` starts cleanly.
  flutter precache --web --quiet
  flutter config --no-analytics --quiet
  flutter doctor --suppress-analytics 2>/dev/null || true
  success "Flutter ${FLUTTER_VERSION} installed."
else
  success "Flutter ${FLUTTER_VERSION} already installed."
  export PATH="$FLUTTER_DIR/bin:$PATH"
fi

# Dart pub global tools
info "Installing Dart pub global tools..."
DART_TOOLS=(
  "build_runner"
  "freezed"
  "riverpod_generator"
  "dart_style"
)
for tool in "${DART_TOOLS[@]}"; do
  dart pub global activate "$tool" --quiet 2>/dev/null && \
    success "  dart pub global: $tool" || warn "  failed: $tool"
done

# =============================================================================
# 9. TERRAFORM
# =============================================================================
TF_ZIP="terraform_${TERRAFORM_VERSION}_linux_${ARCH}.zip"
TF_URL="https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/${TF_ZIP}"

if ! command -v terraform &>/dev/null || \
   [[ "$(terraform version -json 2>/dev/null | jq -r '.terraform_version')" != "$TERRAFORM_VERSION" ]]; then
  info "Installing Terraform ${TERRAFORM_VERSION}..."
  wget -q "$TF_URL" -O "/tmp/${TF_ZIP}"
  sudo unzip -o -q "/tmp/${TF_ZIP}" -d /usr/local/bin
  rm "/tmp/${TF_ZIP}"
  success "Terraform ${TERRAFORM_VERSION} installed."
else
  success "Terraform ${TERRAFORM_VERSION} already installed."
fi

# =============================================================================
# 10. BUF CLI
# =============================================================================
BUF_BIN="/usr/local/bin/buf"
BUF_URL="https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-Linux-x86_64"

if ! command -v buf &>/dev/null || \
   [[ "$(buf --version 2>/dev/null)" != "$BUF_VERSION" ]]; then
  info "Installing Buf ${BUF_VERSION}..."
  sudo curl -fsSL "$BUF_URL" -o "$BUF_BIN"
  sudo chmod +x "$BUF_BIN"
  success "Buf ${BUF_VERSION} installed."
else
  success "Buf ${BUF_VERSION} already installed."
fi

# =============================================================================
# 11. KUBECTL
# =============================================================================
if ! command -v kubectl &>/dev/null; then
  KUBECTL_INSTALLED_VERSION=""
else
  KUBECTL_INSTALLED_VERSION="$(kubectl version --client --short 2>/dev/null | awk 'NR==1 {print $3}' || true)"
fi
if [[ "$KUBECTL_INSTALLED_VERSION" != "$KUBECTL_VERSION" ]]; then
  info "Installing kubectl ${KUBECTL_VERSION}..."
  sudo curl -fsSL \
    "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl" \
    -o /usr/local/bin/kubectl
  sudo chmod +x /usr/local/bin/kubectl
  # GKE auth plugin (required for GKE clusters since k8s 1.26)
  sudo apt-get install -y -qq google-cloud-cli-gke-gcloud-auth-plugin 2>/dev/null || true
  success "kubectl ${KUBECTL_VERSION} installed."
else
  success "kubectl ${KUBECTL_INSTALLED_VERSION} already installed."
fi

# =============================================================================
# 12. HELM
# =============================================================================
HELM_INSTALLED_VERSION=""
if command -v helm &>/dev/null; then
  HELM_INSTALLED_VERSION="$(helm version --short 2>/dev/null | sed -E 's/^v//; s/\+.*$//' | head -1 || true)"
fi
if [[ "$HELM_INSTALLED_VERSION" != "$HELM_VERSION" ]]; then
  info "Installing Helm v${HELM_VERSION}..."
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 \
    | DESIRED_VERSION="v${HELM_VERSION}" bash
  success "Helm installed."
else
  success "Helm v${HELM_INSTALLED_VERSION} already installed."
fi

# =============================================================================
# 13. K9S
# =============================================================================
K9S_TAR="k9s_Linux_${ARCH}.tar.gz"
K9S_URL="https://github.com/derailed/k9s/releases/download/v${K9S_VERSION}/${K9S_TAR}"

K9S_INSTALLED_VERSION=""
if command -v k9s &>/dev/null; then
  K9S_INSTALLED_VERSION="$(k9s version --short 2>/dev/null | awk 'NR==1 {print $1}' | sed 's/^v//' || true)"
fi
if [[ "$K9S_INSTALLED_VERSION" != "$K9S_VERSION" ]]; then
  info "Installing k9s ${K9S_VERSION}..."
  curl -fsSL "$K9S_URL" | sudo tar -C /usr/local/bin -xz k9s
  success "k9s installed."
else
  success "k9s ${K9S_INSTALLED_VERSION} already installed."
fi

# =============================================================================
# 14. CODEX CLI
# =============================================================================
if ! command -v codex &>/dev/null; then
  info "Installing Codex CLI..."
  sudo npm install -g "@openai/codex@${CODEX_CLI_VERSION}" --quiet
  success "Codex CLI installed."
else
  CODEX_INSTALLED_VERSION="$(codex --version 2>/dev/null || true)"
  if [[ "$CODEX_INSTALLED_VERSION" != "$CODEX_CLI_VERSION" ]]; then
    info "Updating Codex CLI to ${CODEX_CLI_VERSION}..."
    sudo npm install -g "@openai/codex@${CODEX_CLI_VERSION}" --quiet
    success "Codex CLI updated."
  else
    success "Codex CLI ${CODEX_INSTALLED_VERSION} already installed."
  fi
fi

# =============================================================================
# 15. CODEX RELAY HELPER
# =============================================================================
mkdir -p "$HOME/bin"
CODEX_RELAY_WRAPPER="$HOME/bin/codex-relay"
if [[ ! -x "$CODEX_RELAY_WRAPPER" ]]; then
  info "Writing Codex Relay helper..."
  cat > "$CODEX_RELAY_WRAPPER" << 'RELAYEROF'
#!/usr/bin/env bash
set -euo pipefail

if [[ -f "$HOME/CortadoBackend/.env" ]]; then
  set -o allexport
  # shellcheck disable=SC1090
  source "$HOME/CortadoBackend/.env"
  set +o allexport
fi

WORKSPACE_DIR="${CODEX_RELAY_WORKSPACE_PATH:-${CODEX_RELAY_WORKSPACE:-$HOME/CortadoBackend}}"
export CODEX_RELAY_WORKSPACE_PATH="$WORKSPACE_DIR"
export CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
export CODEX_BIN="${CODEX_BIN:-$(command -v codex || true)}"
export CODEX_RELAY_AUTH_DB_PATH="${CODEX_RELAY_AUTH_DB_PATH:-$WORKSPACE_DIR/.codex-relay/auth.db}"

cd "$WORKSPACE_DIR"
exec npx --yes "codex-relay@1.1.0" "$@"
RELAYEROF
  chmod +x "$CODEX_RELAY_WRAPPER"
  success "Codex Relay helper written."
fi

CODEX_RELAY_STARTER="$HOME/bin/codex-relay-start"
if [[ ! -x "$CODEX_RELAY_STARTER" ]]; then
  info "Writing Codex Relay start helper..."
  cat > "$CODEX_RELAY_STARTER" << 'RELAYSTARTEOF'
#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${CODEX_RELAY_WORKSPACE_PATH:-$HOME/CortadoBackend}"
RELAY_BIN="$HOME/bin/codex-relay"
WORKSPACE_DIR="${CODEX_RELAY_WORKSPACE_PATH:-${CODEX_RELAY_WORKSPACE:-$PROJECT_DIR}}"
export CODEX_RELAY_WORKSPACE_PATH="$WORKSPACE_DIR"
export CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
export CODEX_BIN="${CODEX_BIN:-$(command -v codex || true)}"
export CODEX_RELAY_AUTH_DB_PATH="${CODEX_RELAY_AUTH_DB_PATH:-$WORKSPACE_DIR/.codex-relay/auth.db}"
cd "$WORKSPACE_DIR"

find_relay_pid() {
  local pid line
  if [[ -f "$PROJECT_DIR/.codex-relay/server.pid" ]]; then
    pid="$(cat "$PROJECT_DIR/.codex-relay/server.pid" 2>/dev/null || true)"
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

use_existing_relay() {
  local pid cwd
  pid="$(find_relay_pid)"
  if [[ -z "$pid" ]]; then
    return 1
  fi

  cwd="$(readlink "/proc/$pid/cwd" 2>/dev/null || true)"
  if [[ -n "$cwd" && -d "$cwd" ]]; then
    cd "$cwd"
  fi
  return 0
}

if command -v tailscale >/dev/null 2>&1; then
  TAILSCALE_IP="$(tailscale ip -4 2>/dev/null | head -1 || true)"
  if [[ -n "$TAILSCALE_IP" ]]; then
    export CODEX_RELAY_PUBLIC_URL="http://${TAILSCALE_IP}:8787"
  fi
fi

if use_existing_relay; then
  for _ in {1..20}; do
    if "$RELAY_BIN" qr; then
      exit 0
    fi
    sleep 1
  done
  echo "Codex Relay is running, but the QR is not ready yet. Try: $RELAY_BIN qr" >&2
  exit 1
fi

if find_relay_pid >/dev/null 2>&1; then
  for _ in {1..20}; do
    if "$RELAY_BIN" qr; then
      exit 0
    fi
    sleep 1
  done
  echo "Codex Relay is running, but the QR is not ready yet. Try: $RELAY_BIN qr" >&2
  exit 1
fi

"$RELAY_BIN" --bg
for _ in {1..20}; do
  if "$RELAY_BIN" qr; then
    exit 0
  fi
  sleep 1
done

echo "Codex Relay started, but the QR is not ready yet. Try: $RELAY_BIN qr" >&2
exit 1
RELAYSTARTEOF
  chmod +x "$CODEX_RELAY_STARTER"
  success "Codex Relay start helper written."
fi

CODEX_RELAY_STOPPER="$HOME/bin/codex-relay-stop"
if [[ ! -x "$CODEX_RELAY_STOPPER" ]]; then
  info "Writing Codex Relay stop helper..."
  cat > "$CODEX_RELAY_STOPPER" << 'RELAYSTOPEOF'
#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${CODEX_RELAY_WORKSPACE_PATH:-${CODEX_RELAY_WORKSPACE:-$PROJECT_DIR}}"
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
RELAYSTOPEOF
  chmod +x "$CODEX_RELAY_STOPPER"
  success "Codex Relay stop helper written."
fi

# =============================================================================
# 16. TMUX CONFIG
# =============================================================================
TMUX_CONF="$HOME/.tmux.conf"
if [[ ! -f "$TMUX_CONF" ]]; then
  info "Writing tmux config..."
  cat > "$TMUX_CONF" << 'TMUXEOF'
# Cortado dev tmux config
set -g prefix C-a
unbind C-b
bind C-a send-prefix

# Mouse support
set -g mouse on

# 256 colours
set -g default-terminal "screen-256color"
set -ga terminal-overrides ",*256col*:Tc"

# Bigger scrollback
set -g history-limit 50000

# Status bar
set -g status-bg colour235
set -g status-fg colour250
set -g status-left '#[fg=colour33,bold][#S] '
set -g status-right '#[fg=colour250]%H:%M %d-%b '
set -g status-right-length 30

# Start windows and panes at 1, not 0
set -g base-index 1
setw -g pane-base-index 1
set -g renumber-windows on

# Split panes with | and -
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"

# Pane navigation (vim-like)
bind h select-pane -L
bind j select-pane -D
bind k select-pane -U
bind l select-pane -R

# Reload config
bind r source-file ~/.tmux.conf \; display-message "Config reloaded"
TMUXEOF
  success "tmux config written."
fi

# =============================================================================
# 17. PERMANENT PATH + ENVIRONMENT in ~/.bashrc
# =============================================================================
info "Configuring PATH in ~/.bashrc..."

BASHRC="$HOME/.bashrc"
PATH_BLOCK_START="# ── Cortado Dev Environment ──────────────────────────────────"
PATH_BLOCK_END="# ── End Cortado Dev Environment ──────────────────────────────"

# Idempotent: only add if block not already present
if ! grep -q "$PATH_BLOCK_START" "$BASHRC" 2>/dev/null; then
  cat >> "$BASHRC" << ENVEOF

${PATH_BLOCK_START}
export GOPATH="\$HOME/go"
export GOBIN="\$HOME/go/bin"
export FLUTTER_ROOT="\$HOME/tools/flutter"
export PATH="\$HOME/bin:/usr/local/go/bin:\$GOBIN:\$FLUTTER_ROOT/bin:\$HOME/.pub-cache/bin:\$PATH"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True
export DOCKER_BUILDKIT=1

# GCP project defaults
export CLOUDSDK_CORE_PROJECT=cortado-ide
export CLOUDSDK_COMPUTE_ZONE=us-central1-a
export CLOUDSDK_COMPUTE_REGION=us-central1

# Source .env if present (API keys etc.)
[[ -f "\$HOME/CortadoBackend/.env" ]] && set -o allexport && source "\$HOME/CortadoBackend/.env" && set +o allexport

# Aliases
alias k='kubectl'
alias kns='kubectl config set-context --current --namespace'
alias tf='terraform'
alias ll='ls -lah'
alias gs='git status'
alias gd='git diff'
${PATH_BLOCK_END}
ENVEOF
  success "PATH block added to ~/.bashrc."
else
  success "~/.bashrc PATH block already present."
fi

# Also add to ~/.profile for non-interactive login shells
if ! grep -q "Cortado Dev Environment" "$HOME/.profile" 2>/dev/null; then
  echo "source \$HOME/.bashrc" >> "$HOME/.profile"
fi

# Apply in current session
export GOPATH="$HOME/go"
export GOBIN="$HOME/go/bin"
export FLUTTER_ROOT="$TOOLS_DIR/flutter"
export PATH="$HOME/bin:/usr/local/go/bin:$GOBIN:$FLUTTER_ROOT/bin:$HOME/.pub-cache/bin:$PATH"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True

# =============================================================================
# 18. GIT CONFIG
# =============================================================================
info "Configuring git..."
# Only set if not already configured
git config --global user.email "${GIT_EMAIL:-$(gcloud config get-value account 2>/dev/null || echo 'dev@cortado.dev')}" 2>/dev/null || true
git config --global user.name "${GIT_NAME:-Cortado Dev}" 2>/dev/null || true
git config --global init.defaultBranch main
git config --global pull.rebase true
git config --global core.editor "nano"
git config --global push.autoSetupRemote true
success "Git configured."

# =============================================================================
# 19. CORTADO PROJECT DIRS
# =============================================================================
info "Creating project directory structure..."
PROJECT_DIR="$HOME/CortadoBackend"
mkdir -p \
  "$PROJECT_DIR" \
  "$HOME/.config/cortado" \
  "$HOME/.kube"

# If not already a git repo, initialise (dev starts here)
if [[ ! -d "$PROJECT_DIR/.git" ]]; then
  warn "No git repo at $PROJECT_DIR. If you're cloning from GitHub:"
  warn "  git clone git@github.com:your-org/cortado.git $PROJECT_DIR"
fi
success "Project dirs ready."

# =============================================================================
# 20. CODEX CONFIG DIRS
# =============================================================================
mkdir -p "$HOME/.codex" "$HOME/.agents"
# .codex/config.toml should already be present from gcloud scp
if [[ ! -f "$HOME/.codex/config.toml" ]]; then
  warn "~/.codex/config.toml not found — copy it via:"
  warn "  gcloud compute scp ~/.codex/config.toml cortado-dev-vm1:~/.codex/config.toml \\"
  warn "    --zone=us-central1-a --project=cortado-ide"
fi

# =============================================================================
# 21. FIREWALL RULE — allow SSH from current IP (idempotent hint)
# =============================================================================
# (already handled by GCE default-allow-ssh rule, just noting it)

# =============================================================================
# 23. FINAL CHECKS
# =============================================================================
echo ""
echo "══════════════════════════════════════════════════════"
echo " Cortado Dev VM — Setup Summary"
echo "══════════════════════════════════════════════════════"
echo ""

check_cmd() {
  local name="$1" cmd="$2"
  if command -v "$cmd" &>/dev/null; then
    printf "  %-22s ${GREEN}✓${NC}  %s\n" "$name" "$($cmd $3 2>/dev/null | head -1 || echo '(installed)')"
  else
    printf "  %-22s ${RED}✗${NC}  not found\n" "$name"
  fi
}

check_cmd "go"          go         "version"
check_cmd "gopls"       gopls      "version"
check_cmd "flutter"     flutter    "--version"
check_cmd "dart"        dart       "--version"
check_cmd "terraform"   terraform  "version"
check_cmd "buf"         buf        "--version"
check_cmd "kubectl"     kubectl    "version --client --short"
check_cmd "helm"        helm       "version --short"
check_cmd "k9s"         k9s        "version --short"
check_cmd "docker"      docker     "--version"
check_cmd "gcloud"      gcloud     "version"
check_cmd "node"        node       "--version"
check_cmd "npm"         npm        "--version"
check_cmd "codex"       codex      "--version"
check_cmd "tailscale"   tailscale  "version"
check_cmd "protoc"      protoc     "--version"

echo ""
echo "  Disk usage:"
df -h / | awk 'NR==2 {printf "    /  used: %s / %s (%s)\n", $3, $2, $5}'
echo ""
echo "  Swap:"
swapon --show 2>/dev/null | tail -1 | awk '{printf "    %s  %s\n", $1, $3}' || echo "    (none active)"
echo ""
echo "══════════════════════════════════════════════════════"
echo ""

warn "IMPORTANT: Run 'newgrp docker' or log out/in for Docker group to take effect."
warn "If you did not set TS_AUTHKEY in ~/CortadoBackend/.env, finish Tailscale with: sudo tailscale up --ssh"
warn "Start Codex Relay with: codex-relay-start"
success "Setup complete. Happy building."
