# Decisions Needed

- Should the daemon-to-control-plane FileSync stream ship as authenticated TLS/gRPC immediately, or is an h2c/dev-auth transport acceptable until Task 6.1.5 wires the daemon client and host-app bridge?
  Context: Task 6.1.3 now exposes the control-plane FileSyncService and keeps the relay/proto slice moving, but the feature spec/technical report call out an auth'd TLS stream while the current repo only has HTTP auth patterns and the daemon client is not implemented yet.
