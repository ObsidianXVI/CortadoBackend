
## Feature 1.2 — Workspace Agent (PTY Core)

### Task 1.2.1 — Proto definition: agent gRPC service
**What to do:**
- Write `proto/agent/v1/agent.proto`:
  ```protobuf
  syntax = "proto3";
  package agent.v1;
  option go_package = "github.com/your-org/cortado/agent/gen/agent/v1";

  service WorkspaceAgentService {
    rpc CreatePty(PtyRequest)            returns (PtyResponse);
    rpc StreamPty(stream PtyInput)       returns (stream PtyOutput);
    rpc Health(HealthRequest)            returns (HealthResponse);
  }

  message PtyRequest {
    uint32 cols  = 1;
    uint32 rows  = 2;
    string shell = 3;  // defaults to /bin/bash
    repeated string env = 4;
  }
  message PtyResponse { string pty_id = 1; }

  message PtyInput {
    string pty_id = 1;
    oneof payload {
      bytes      data   = 2;
      WindowSize resize = 3;
      int32      signal = 4;
    }
  }
  message PtyOutput {
    oneof payload {
      bytes data     = 1;
      int32 exit_code = 2;  // sent when process exits
    }
  }
  message WindowSize { uint32 cols = 1; uint32 rows = 2; }
  message HealthRequest {}
  message HealthResponse { string status = 1; }
  ```
- Run `buf lint` (must pass), then `buf generate`.

**Key detail**: `StreamPty` takes a stream of `PtyInput` rather than a single `PtyId` at open time. This means the first message on the stream is always a `PtyInput` with `pty_id` set and no `payload`. The Go server reads this first message to identify which PTY session to bind to. The alternative (a separate `OpenStream(pty_id)` RPC that returns a bidirectional stream) isn't expressible in proto3 — bidirectional streaming RPCs always start with the client's first message.

**Challenge**: Dart gRPC bidirectional streaming (`ClientCall`) has subtly different cancellation semantics from Go. When the Dart client calls `call.cancel()`, the Go server's `stream.Recv()` returns a non-EOF error (status `CANCELLED`). Your Go recv loop must handle this explicitly — don't treat it as an unexpected error worth logging loudly.

---

### Task 1.2.2 — PTY management in Go
**What to do:**
- Initialize `agent/` Go module: `go mod init github.com/your-org/cortado/agent`
- Add dependency: `github.com/creack/pty` (Go's canonical PTY library — this is the core reason the agent is Go).
- Implement `internal/pty/manager.go`:
  ```go
  package pty

  import (
      "errors"
      "os"
      "os/exec"
      "sync"
      "syscall"

      "github.com/creack/pty"
      "github.com/google/uuid"
  )

  type Session struct {
      ID  string
      ptm *os.File    // PTY master fd
      cmd *exec.Cmd
      mu  sync.Mutex
  }

  type Manager struct {
      sessions sync.Map
  }

  func (m *Manager) Create(shell string, cols, rows uint16, env []string) (*Session, error) {
      if shell == "" { shell = "/bin/bash" }
      cmd := exec.Command(shell)
      cmd.Env = append(os.Environ(), env...)
      cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

      ptm, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
      if err != nil { return nil, err }

      s := &Session{ID: uuid.NewString(), ptm: ptm, cmd: cmd}
      m.sessions.Store(s.ID, s)
      return s, nil
  }

  func (m *Manager) Write(id string, data []byte) error {
      s, ok := m.sessions.Load(id)
      if !ok { return errors.New("session not found") }
      _, err := s.(*Session).ptm.Write(data)
      return err
  }

  func (m *Manager) Read(id string, buf []byte) (int, error) {
      s, ok := m.sessions.Load(id)
      if !ok { return 0, errors.New("session not found") }
      return s.(*Session).ptm.Read(buf)
  }

  func (m *Manager) Resize(id string, cols, rows uint16) error {
      s, ok := m.sessions.Load(id)
      if !ok { return errors.New("session not found") }
      return pty.Setsize(s.(*Session).ptm, &pty.Winsize{Cols: cols, Rows: rows})
  }

  func (m *Manager) Kill(id string) {
      if s, ok := m.sessions.LoadAndDelete(id); ok {
          sess := s.(*Session)
          sess.cmd.Process.Kill()
          sess.ptm.Close()
      }
  }
  ```
- Write a unit test that spawns `bash`, sends `echo hello_cortado\n`, reads until it finds `hello_cortado` in the output, then kills the session.

**Key detail**: `EIO` from `ptm.Read()` is not an error — it's the normal signal that the slave PTY has no more writers (the shell exited). Your read loop in the gRPC handler must treat `syscall.EIO` as clean termination:
```go
n, err := sess.ptm.Read(buf)
if err != nil {
    if errors.Is(err, syscall.EIO) { return nil } // shell exited cleanly
    return err
}
```
This is the single most important correctness detail in the PTY code. Missing it causes the read goroutine to spin on errors after every shell exit, logging noise and leaking goroutines.

**Challenge**: `pty.StartWithSize` requires the shell binary to exist in the image. During local devcontainer testing, `/bin/bash` exists. In the workspace Docker image, verify the shell is present before attempting to start it — return a descriptive error (`"shell /bin/bash not found in image"`) rather than a raw `exec` error.

---

### Task 1.2.3 — gRPC server and StreamPty implementation
**What to do:**
- Implement `internal/server/agent_server.go` satisfying the generated `WorkspaceAgentServiceServer` interface.
- `StreamPty` implementation:
  ```go
  func (s *AgentServer) StreamPty(stream pb.WorkspaceAgentService_StreamPtyServer) error {
      // First message must identify the session
      first, err := stream.Recv()
      if err != nil { return err }
      sessionID := first.PtyId

      ctx := stream.Context()
      g, ctx := errgroup.WithContext(ctx)

      // PTY → gRPC: read output and send downstream
      g.Go(func() error {
          buf := make([]byte, 4096)
          for {
              n, err := s.ptyMgr.Read(sessionID, buf)
              if err != nil {
                  if errors.Is(err, syscall.EIO) { return nil }
                  return err
              }
              if err := stream.Send(&pb.PtyOutput{
                  Payload: &pb.PtyOutput_Data{Data: buf[:n]},
              }); err != nil { return err }
          }
      })

      // gRPC → PTY: receive input and write to PTY
      g.Go(func() error {
          for {
              msg, err := stream.Recv()
              if err != nil { return err }
              switch p := msg.Payload.(type) {
              case *pb.PtyInput_Data:
                  s.ptyMgr.Write(sessionID, p.Data)
              case *pb.PtyInput_Resize:
                  s.ptyMgr.Resize(sessionID, uint16(p.Resize.Cols), uint16(p.Resize.Rows))
              case *pb.PtyInput_Signal:
                  // Handle SIGINT (2), SIGTERM (15), etc.
                  // Find the shell's process group and signal it
              }
          }
      })

      return g.Wait()
  }
  ```
- Start the gRPC server on `:9090`. Add `CORTADO_AGENT_GRPC_PORT` env var.

**Key detail**: The two goroutines in `StreamPty` share a lifetime — if one exits (PTY read returns EIO after shell exit), the other should also stop. `errgroup` + a derived context handles this: when the PTY-read goroutine returns `nil` (EIO), `errgroup` cancels the context but doesn't propagate the nil return. The gRPC-receive goroutine, however, is blocked on `stream.Recv()` which doesn't respect context cancellation. Unblock it by calling `stream.Context().Done()` — but the stream's context is already the parent context, not the errgroup context. The cleanest fix: use a separate `context.CancelFunc` that closes a channel, and select between `stream.Recv()` (via a wrapper goroutine that sends to a channel) and the cancel channel.

**Challenge**: gRPC stream `Send()` is not goroutine-safe. Only one goroutine may call `Send()` at a time. In the `StreamPty` design above, only the PTY-read goroutine calls `Send()` (the other goroutine only calls `Recv()`), so this is safe. If you later add a separate "send exit code" path, ensure it goes through the same goroutine that owns `Send()`.

---

### Task 1.2.4 — Dockerfile for workspace agent
**What to do:**
- Write `agent/Dockerfile` as a multi-stage build:
  ```dockerfile
  FROM golang:1.23-alpine AS builder
  WORKDIR /build
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -ldflags="-s -w" -o cortado-agent ./cmd/agent

  FROM ubuntu:22.04
  RUN apt-get update && apt-get install -y \
      bash zsh git curl ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && useradd -m -u 1000 workspace
  COPY --from=builder /build/cortado-agent /usr/local/bin/cortado-agent
  EXPOSE 9090
  ENV CORTADO_ENV=production
  ENTRYPOINT ["/usr/local/bin/cortado-agent"]
  ```
- The Docker image build and push is defined as a GitHub Actions job (not a Terraform resource):
  ```yaml
  # .github/workflows/build-agent.yml
  - name: Build and push workspace agent
    run: |
      IMAGE="us-central1-docker.pkg.dev/${{ vars.GCP_PROJECT }}/cortado-dev/cortado-workspace"
      docker build -t $IMAGE:${{ github.sha }} agent/
      docker push $IMAGE:${{ github.sha }}
  ```
- `CGO_ENABLED=0` is required for the static binary. Verify: `docker run --rm $IMAGE file /usr/local/bin/cortado-agent` should print `statically linked`.

**Key detail**: The `CORTADO_ENV=production` in the Dockerfile means any code guarded by `if os.Getenv("CORTADO_ENV") == "development"` will not run in production containers. The dev-bypass auth check is one such guard. This is belt-and-suspenders — the bypass should also check that the incoming token is literally `dev-bypass` and reject anything else, but the env var ensures it can never be reached in a production-built image.

**Challenge**: The Ubuntu base image adds ~80MB. Consider `debian:bookworm-slim` (~30MB) if you don't need Ubuntu-specific packages. The main constraint is that `bash`, `git`, and `curl` are needed for user shell sessions. `debian:bookworm-slim` has all of these available via `apt`. Switch before v0.3 when the image starts growing with language runtimes — the difference compounds.

---
