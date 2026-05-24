## Feature 4.1 — LSP Gateway

### Task 4.1.1 — Proto: LSP service
- Add to `agent.proto`:
  ```protobuf
  rpc OpenLSP(OpenLSPRequest)           returns (OpenLSPResponse);
  rpc StreamLSP(stream LSPMessage)      returns (stream LSPMessage);
  ```
- `OpenLSPRequest`: `{language: string}`.
- `LSPMessage`: `{data: bytes}` — raw JSON-RPC frames (Content-Length framing is stripped by the agent; the gateway sees raw JSON).

**Challenge**: The Dart language server is invoked as `dart language-server --protocol=lsp`. Add the Dart SDK to the workspace Docker image — this adds ~300MB but is required for Dart LSP. Use a build arg so non-Dart workspaces can skip this layer:
```dockerfile
ARG INCLUDE_DART_SDK=false
RUN if [ "$INCLUDE_DART_SDK" = "true" ]; then \
    wget https://storage.googleapis.com/.../dart-sdk.zip && ...; fi
```

---

### Task 4.1.2 — Agent-side LSP process manager
- Implement `LSPManager` in the Go agent: spawns `dart language-server --protocol=lsp` as a subprocess, wraps its stdin/stdout in a Content-Length frame parser/unparser, bridges to the gRPC `StreamLSP` bidirectional stream.
- Lazy start: spawn the language server process on first `OpenLSP` call, not at agent startup.
- Restart on crash: watch the process with `cmd.Wait()`, restart up to 3 times, emit a close event to the gRPC stream if all retries exhausted.

**Key detail**: The Content-Length framing parser must handle `\r\n` (CRLF) line endings — `bufio.Scanner` splits on `\n` and leaves a trailing `\r` in the parsed Content-Length value. Use `strings.TrimSpace` after scanning the header line.

---

### Task 4.1.3 — LSP routing in control plane + WS mux channel
- WebSocket mux channel range `0x0100–0x01FF` for LSP (one channel per language).
- On `Open` frame for an LSP channel: call `OpenLSP(language)` on the agent, establish `StreamLSP` gRPC stream, bind to WS mux channel.
- The gateway is a transparent proxy — it does not parse LSP JSON content.
- Increase mux max frame size for LSP channels to 4MB (completion lists for large projects can exceed the 16KB default).

---
