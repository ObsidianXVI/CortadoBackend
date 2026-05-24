#!/usr/bin/env bash
set -euo pipefail

binary_path="${CORTADO_DAEMON_BINARY_PATH:-$(command -v cortado-daemon || true)}"
os_name="$(uname -s)"
user_home="${HOME:-}"

if [[ -z "${binary_path}" ]]; then
  cat <<'EOF' >&2
error: `cortado-daemon` was not found on PATH.

This installer currently wires the local service definitions around an existing
daemon binary. Export CORTADO_DAEMON_BINARY_PATH=/path/to/cortado-daemon or put
the binary on PATH before running the installer.
EOF
  exit 1
fi

if [[ -z "${user_home}" ]]; then
  echo "error: HOME must be set for daemon installation" >&2
  exit 1
fi

case "${os_name}" in
  Linux)
    install -d -m 0755 "${user_home}/.config/systemd/user"
    cat >"${user_home}/.config/systemd/user/cortado-daemon.service" <<EOF
[Unit]
Description=Cortado Local Daemon
After=network.target

[Service]
Type=simple
ExecStart=${binary_path}
Restart=on-failure
RestartSec=2
Environment=CORTADO_DAEMON_LISTEN_ADDR=127.0.0.1:9731

[Install]
WantedBy=default.target
EOF
    systemctl --user daemon-reload
    systemctl --user enable --now cortado-daemon.service
    ;;
  Darwin)
    install -d -m 0755 "${user_home}/Library/LaunchAgents"
    cat >"${user_home}/Library/LaunchAgents/com.cortado.daemon.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.cortado.daemon</string>
  <key>ProgramArguments</key>
  <array>
    <string>${binary_path}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
EOF
    launchctl unload "${user_home}/Library/LaunchAgents/com.cortado.daemon.plist" >/dev/null 2>&1 || true
    launchctl load "${user_home}/Library/LaunchAgents/com.cortado.daemon.plist"
    cat <<'EOF'
note: macOS may require explicit filesystem permission for directories outside
~/Documents, ~/Desktop, and ~/Downloads. If the daemon cannot watch your
workspace root, grant Full Disk Access or re-run after selecting the workspace
root with the future NSOpenPanel helper.
EOF
    ;;
  *)
    echo "error: unsupported platform ${os_name}. Use the Windows NSSM asset in daemon/packaging/windows/ for Windows installs." >&2
    exit 1
    ;;
esac

echo "cortado-daemon service installed using ${binary_path}"
