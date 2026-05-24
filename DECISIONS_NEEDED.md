# Decisions Needed

- Should unresolved local-sync `ConflictNotice` messages on mux channel `0x0600` be emitted on the local daemon WebSocket (`ws://127.0.0.1:9731`) or on the control-plane workspace mux (`/v1/workspaces/{id}/connect`)?
  Context: Task 6.1.4 now has daemon-side conflict detection, merge attempts, and merge logging, but the feature spec only says "emit on WS mux channel `0x0600`" without clarifying which WebSocket transport owns that channel before Task 6.1.5 lands the daemon bridge.
