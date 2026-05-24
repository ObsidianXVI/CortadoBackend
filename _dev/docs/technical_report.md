# Cortado: Technical Architecture Report
## A Flutter/Dart SaaS Package for Cloud IDE Backends

**Version 1.0 — April 2026**

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [System Architecture Overview](#2-system-architecture-overview)
3. [Flutter/Dart Package Architecture](#3-flutterdart-package-architecture)
4. [Terminal & Shell Provisioning](#4-terminal--shell-provisioning)
5. [Virtual Workspace Architecture](#5-virtual-workspace-architecture)
6. [Resource Scaling & Billing](#6-resource-scaling--billing)
7. [Port Forwarding & Chrome Support](#7-port-forwarding--chrome-support)
8. [Language Server Protocol (LSP) & Extensions](#8-language-server-protocol-lsp--extensions)
9. [Codebase Indexing](#9-codebase-indexing)
10. [AI Features](#10-ai-features)
11. [Security Architecture](#11-security-architecture)
12. [Data Layer & Persistence](#12-data-layer--persistence)
13. [Observability & Metering](#13-observability--metering)
14. [Multi-Tenancy & Isolation](#14-multi-tenancy--isolation)
15. [Bottlenecks, Risks & Mitigation](#15-bottlenecks-risks--mitigation)
16. [Alternatives Matrix](#16-alternatives-matrix)
17. [Development Roadmap](#17-development-roadmap)

---

## 1. Executive Summary

Cortado is a Flutter Web package providing a complete cloud IDE backend-as-a-service layer. Developers building IDE experiences in Flutter embed the package and gain managed provisioning of compute, terminals, file systems, LSP servers, AI tooling, and billing — without building any of that infrastructure themselves.

The product operates as two interacting layers:

- **The Cortado Flutter Package**: A Dart library providing typed APIs, WebSocket-backed widgets (terminal view, file tree, editor integration), and state management for workspace lifecycle.
- **The Cortado Control Plane (GCP)**: A set of managed backend services handling provisioning, scaling, billing, filesystem sync, LSP brokering, and AI inference.

The central thesis is that a cloud IDE backend is **enormously** complex to build correctly (security boundaries, autoscaling, billing accuracy, LSP latency), and Cortado abstracts all of it behind a clean package API. The package author (you) operates the infrastructure; consuming developers pay for workspace-seconds.

**Core technical decisions up front:**

- Workspace containers run on **GKE Autopilot**; scale-to-zero is native.
- Container images are distributed from **Google Artifact Registry** in the same region as the GKE cluster so image pulls stay in-region and fit GCP IAM-based deployment workflows.
- Filesystem sync uses **CRDT-based operational transforms** over gRPC streaming, with a companion local daemon (`cortado-agent`) the host IDE installs.
- Terminals are **PTY-over-WebSocket** bridged through an in-container `ttyd`/custom agent.
- LSP proxying routes through a **central LSP gateway** that multiplexes client connections to per-workspace language server processes.
- AI features run on **Vertex AI** (Gemini) or via the Anthropic/OpenAI APIs, with a RAG pipeline backed by **Vertex AI Vector Search** or **Weaviate**.
- Billing uses **OpenMeter** (or Stripe Meters) for per-second resource metering, with a GCP Cloud Billing export pipeline.

---

## 2. System Architecture Overview

### 2.1 Topology

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Host IDE App                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              Cortado Flutter Package                        │   │
│  │  CortadoClient  WorkspaceManager  TerminalWidget  EditorSync│   │
│  └──────────────────────┬──────────────────────────────────────┘   │
│                         │ HTTPS / WSS / gRPC-Web                   │
└─────────────────────────┼───────────────────────────────────────────┘
                          │
          ┌───────────────▼───────────────┐
          │    Cortado Control Plane      │
          │  (Cloud Run / GKE)            │
          │                               │
          │  ┌──────────┐ ┌────────────┐  │
          │  │ Auth     │ │ Workspace  │  │
          │  │ Service  │ │ Orchestrat.│  │
          │  └──────────┘ └────┬───────┘  │
          │  ┌──────────┐      │          │
          │  │ Billing  │ ┌────▼───────┐  │
          │  │ Metering │ │ Proxy/     │  │
          │  └──────────┘ │ Gateway   │  │
          │  ┌──────────┐ └────┬───────┘  │
          │  │ AI       │      │          │
          │  │ Service  │      │          │
          │  └──────────┘      │          │
          └───────────────────┼───────────┘
                              │
          ┌───────────────────▼───────────────┐
          │      Workspace Pods (GKE)          │
          │  ┌────────────────────────────┐   │
          │  │  cortado-workspace-agent   │   │
          │  │  - PTY/shell management    │   │
          │  │  - Filesystem watcher      │   │
          │  │  - LSP process mgmt        │   │
          │  │  - Port forward proxy      │   │
          │  │  - Chrome/Xvfb             │   │
          │  └────────────────────────────┘   │
          │  User code, tools, runtimes        │
          └────────────────────────────────────┘
```

### 2.2 Request Paths

**Terminal keystroke → PTY:**
```
Flutter Widget (keydown) 
  → WebSocket frame (binary, utf8 encoded)
  → Cortado WS Gateway (Cloud Run) 
  → mTLS tunnel to workspace pod 
  → cortado-agent (Unix domain socket) 
  → pty.write() → kernel PTY → shell process
  → PTY read → reverse path → widget render
```

**File save (local mirror mode):**
```
Local editor saves file
  → cortado-local-daemon (file system watcher, inotify/FSEvents)
  → gRPC FileSync stream to Control Plane
  → operation logged + forwarded to workspace pod
  → workspace agent applies op to container filesystem
  → optional: git commit / snapshot
```

**LSP completion request:**
```
Flutter editor widget (textChanged)
  → Cortado LSP Client (JSON-RPC over WebSocket)
  → LSP Gateway (Control Plane)
  → routes to workspace-specific LSP server process
  → language server responds (textDocument/completion)
  → Gateway forwards response back
  → Flutter widget renders completions
```

### 2.3 Key Design Principles

**Multiplexed single connection**: Each workspace uses a single long-lived WebSocket/HTTP2 connection from the client, multiplexed into logical channels (terminal, filesystem events, LSP, metrics). This avoids TCP connection storms and simplifies NAT traversal.

**Agent-not-sidecar**: The `cortado-workspace-agent` is a single compiled Go binary baked into the workspace image. It handles all intra-container concerns without a sidecar, reducing pod overhead.

**Push-pull file consistency**: Filesystem ops use a vector-clock model. Each node (local daemon, cloud agent) maintains a logical clock. Conflicts are resolved by "last writer wins at op level" with a merge strategy configurable per file type (binary = fail-open, text = line-level merge).

**Lazy LSP startup**: Language servers are spawned on first `textDocument/didOpen` event, not at workspace creation. This saves RAM and startup latency for polyglot workspaces.

---

## 3. Flutter/Dart Package Architecture

### 3.1 Package Surface Area

The consuming developer's interaction with Cortado looks like this:

```dart
// pubspec.yaml
dependencies:
  cortado: ^1.0.0

// main.dart
import 'package:cortado/cortado.dart';

void main() {
  Cortado.initialize(
    apiKey: 'crt_live_xxxx',
    tenantId: 'my-ide-app',
  );
  runApp(MyIDEApp());
}

// Anywhere in the widget tree:
class WorkspaceScreen extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return CortadoWorkspaceProvider(
      workspaceId: 'ws_abc123',
      child: Column(children: [
        Expanded(child: CortadoTerminal()),
        Expanded(child: CortadoFileTree()),
      ]),
    );
  }
}
```

### 3.2 Internal Package Structure

```
cortado/
├── lib/
│   ├── cortado.dart                  # Public barrel export
│   ├── src/
│   │   ├── client/
│   │   │   ├── cortado_client.dart   # HTTP + WebSocket client
│   │   │   ├── auth_interceptor.dart # JWT refresh, API key handling
│   │   │   ├── retry_policy.dart     # Exponential backoff
│   │   │   └── channel_mux.dart      # Logical channel multiplexer
│   │   ├── workspace/
│   │   │   ├── workspace_manager.dart
│   │   │   ├── workspace_model.dart  # Freezed data classes
│   │   │   ├── workspace_state.dart  # Riverpod/Bloc state
│   │   │   └── provisioning_service.dart
│   │   ├── terminal/
│   │   │   ├── terminal_controller.dart
│   │   │   ├── pty_codec.dart        # Binary PTY encoding/decoding
│   │   │   └── terminal_widget.dart  # Actual Flutter widget
│   │   ├── filesystem/
│   │   │   ├── vfs_node.dart         # Virtual FS tree model
│   │   │   ├── file_sync_service.dart
│   │   │   ├── crdt_ops.dart         # File operation CRDTs
│   │   │   └── local_daemon_bridge.dart  # talks to local agent
│   │   ├── lsp/
│   │   │   ├── lsp_client.dart       # JSON-RPC 2.0 over WS
│   │   │   ├── lsp_models.dart       # LSP spec types
│   │   │   └── lsp_provider.dart     # Exposes completions/diagnostics
│   │   ├── ai/
│   │   │   ├── ai_service.dart
│   │   │   ├── context_window.dart
│   │   │   └── inline_completion.dart
│   │   ├── billing/
│   │   │   ├── usage_tracker.dart    # Client-side usage display
│   │   │   └── billing_model.dart
│   │   └── ports/
│   │       ├── port_forwarder.dart
│   │       └── chrome_viewer.dart    # iframe + auth proxy
```

### 3.3 State Management Strategy

Cortado internally uses **Riverpod** (specifically `riverpod_annotation` for code generation). Consumers can use any state management they want — the package exposes both imperative APIs and reactive streams.

```dart
// Reactive (Riverpod consumers can watch these directly)
final workspaceProvider = StateNotifierProvider<WorkspaceNotifier, WorkspaceState>(...);
final terminalOutputProvider = StreamProvider.family<String, String>((ref, termId) {
  return ref.watch(terminalControllerProvider(termId)).outputStream;
});

// Imperative (for non-Riverpod consumers)
final ws = await Cortado.workspaces.create(
  image: 'cortado/ubuntu-dev:22.04',
  resources: WorkspaceResources(cpu: 2, memoryGb: 4),
);
final term = await ws.createTerminal();
term.write('ls -la\n');
await for (final output in term.output) {
  print(output);
}
```

### 3.4 WebSocket Multiplexing Protocol

Rather than opening separate WebSocket connections for terminal, LSP, and file sync, Cortado uses a single WebSocket with a lightweight framing layer:

```
Frame format (binary):
┌────────────┬────────────┬──────────────┬─────────────────┐
│ Channel ID │ Msg Type   │ Payload Len  │ Payload         │
│ 2 bytes    │ 1 byte     │ 4 bytes      │ variable        │
└────────────┴────────────┴──────────────┴─────────────────┘

Channel IDs:
  0x0001-0x00FF: Terminal sessions (1 per PTY)
  0x0100-0x01FF: LSP instances (1 per language)
  0x0200       : File sync channel
  0x0300       : Metrics/heartbeat
  0x0400       : Port forward tunnels

Msg Types:
  0x01: Data
  0x02: Open channel
  0x03: Close channel
  0x04: Flow control (credit-based backpressure)
  0x05: Error
  0xFF: Ping/pong
```

This is essentially a stripped-down version of the QUIC stream model. An alternative is using **QUIC directly** (via the `quic_go` library server-side and Dart's `http3` package), which gives native multiplexing + better mobile performance, but browser support for raw QUIC without HTTP/3 semantics requires WebTransport.

**WebTransport** (Chrome 97+, Firefox 114+) is worth evaluating as the underlying transport: it provides datagrams and streams over QUIC natively in browser, which would eliminate the need for the custom framing layer. The Dart `web_transport` package is immature (as of 2025), but this is a viable medium-term path.

### 3.5 Terminal Widget Implementation

The terminal widget is the most latency-sensitive piece. Options:

**Option A: Embed xterm.js via HtmlElementView** (recommended for web)

```dart
// In Flutter Web, render xterm.js in a platform view
class CortadoTerminal extends StatefulWidget { ... }

class _CortadoTerminalState extends State<CortadoTerminal> {
  late final String _viewId;

  @override
  void initState() {
    super.initState();
    _viewId = 'cortado-terminal-${widget.terminalId}';
    // Register the platform view
    ui.platformViewRegistry.registerViewFactory(_viewId, (int viewId) {
      final element = html.DivElement()
        ..id = 'xterm-container-$viewId'
        ..style.width = '100%'
        ..style.height = '100%';
      _initXterm(element);
      return element;
    });
  }

  void _initXterm(html.DivElement container) {
    // Call JS interop to initialize xterm.js
    _XtermInterop.init(container, widget.terminalId);
    // Wire WebSocket data → xterm
    widget.controller.outputStream.listen((data) {
      _XtermInterop.write(widget.terminalId, data);
    });
    // Wire xterm input → WebSocket
    _XtermInterop.onData(widget.terminalId, (data) {
      widget.controller.write(data);
    });
  }

  @override
  Widget build(BuildContext context) {
    return HtmlElementView(viewType: _viewId);
  }
}
```

The JS interop for xterm.js uses `dart:js_interop`:

```dart
@JS('CortadoXterm.init')
external void _xtermInit(html.Element container, String id);

@JS('CortadoXterm.write')
external void _xtermWrite(String id, String data);
```

And the companion `cortado_xterm.js`:
```javascript
window.CortadoXterm = {
  terminals: {},
  init(container, id) {
    const term = new Terminal({ fontFamily: 'JetBrains Mono', fontSize: 14 });
    const fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);
    term.open(container);
    fitAddon.fit();
    this.terminals[id] = { term, fitAddon };
    term.onData(data => window._cortadoTermInput?.(id, data));
  },
  write(id, data) { this.terminals[id]?.term.write(data); },
  resize(id, cols, rows) { this.terminals[id]?.term.resize(cols, rows); }
};
```

**Option B: Pure Dart terminal renderer** — packages like `xterm` (pub.dev) provide a pure-Dart VT100/VT220 parser and canvas renderer. Performance is generally adequate for typical usage but can lag under high-volume output (e.g., `cat large_file.txt`). The `xterm` package uses a `CanvasElement` for rendering in web, which is efficient. This is the better choice for non-web Flutter targets (macOS, Windows, Linux).

**Recommended strategy**: Use `xterm` (pure Dart) for desktop targets; use xterm.js via HtmlElementView for web (better font rendering, addon ecosystem).

---

## 4. Terminal & Shell Provisioning

### 4.1 PTY Lifecycle on GCP

When a user requests a terminal, the flow is:

1. Client calls `POST /v1/workspaces/{id}/terminals`
2. Control plane checks workspace pod is running (scale from zero if needed)
3. Control plane calls workspace agent's gRPC endpoint: `CreatePty(PtyRequest{cols, rows, env, shell})`
4. Agent forks a shell under a dedicated Unix user (per-workspace), opens a PTY pair
5. Agent returns `pty_id`
6. Control plane creates a WebSocket channel and binds it to the PTY via a streaming gRPC call
7. Client receives `{ terminal_id, ws_channel_id }`
8. Client opens WebSocket and sends frames on that channel ID

**The workspace agent's PTY management in Go:**

```go
// agent/pty_manager.go
type PtySession struct {
    ID     string
    ptm    *os.File   // PTY master
    pts    *os.File   // PTY slave
    cmd    *exec.Cmd
    mu     sync.Mutex
}

func (pm *PtyManager) Create(req *pb.PtyRequest) (*PtySession, error) {
    ptm, pts, err := pty.Open() // golang.org/x/sys/unix
    if err != nil { return nil, err }

    cmd := exec.Command(req.Shell)
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid:  true,
        Setctty: true,
        Ctty:    3, // pts fd
    }
    cmd.Stdin = pts
    cmd.Stdout = pts
    cmd.Stderr = pts
    cmd.Env = append(os.Environ(), req.Env...)
    // Drop privileges to workspace user
    cmd.SysProcAttr.Credential = &syscall.Credential{
        Uid: pm.workspaceUID,
        Gid: pm.workspaceGID,
    }

    if err := cmd.Start(); err != nil { return nil, err }
    pts.Close()  // Close slave in parent after fork

    sess := &PtySession{
        ID: uuid.New().String(),
        ptm: ptm,
        cmd: cmd,
    }
    pm.sessions.Store(sess.ID, sess)
    return sess, nil
}

func (pm *PtyManager) StreamPty(id string, stream pb.WorkspaceAgent_StreamPtyServer) error {
    sess := pm.sessions.Load(id).(*PtySession)
    // Goroutine: PTY → gRPC stream
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := sess.ptm.Read(buf)
            if err != nil { return }
            stream.Send(&pb.PtyOutput{Data: buf[:n]})
        }
    }()
    // Main: gRPC stream → PTY
    for {
        in, err := stream.Recv()
        if err != nil { return err }
        switch v := in.Payload.(type) {
        case *pb.PtyInput_Data:
            sess.ptm.Write(v.Data.Data)
        case *pb.PtyInput_Resize:
            setWinSize(sess.ptm, v.Resize.Cols, v.Resize.Rows)
        }
    }
}
```

### 4.2 Shell Environment Setup

Each workspace pod runs with a base image that includes the shell environment. The image layering strategy is critical for cold-start performance:

```dockerfile
# cortado-base:ubuntu-22.04
FROM ubuntu:22.04

# Install common tools in a single RUN to minimize layers
RUN apt-get update && apt-get install -y \
    bash zsh fish \
    git curl wget \
    build-essential \
    python3 python3-pip \
    nodejs npm \
    && rm -rf /var/lib/apt/lists/*

# Cortado workspace agent (pre-compiled Go binary)
COPY --from=cortado-agent-builder /agent /usr/local/bin/cortado-agent

# Per-workspace user setup script
COPY scripts/setup-workspace-user.sh /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/cortado-agent", "--serve"]
```

**Alternatives to baking images:**

- **Nix-based environments**: Define workspace environments as Nix flakes. Users specify `{ packages = [ pkgs.nodejs pkgs.rustup ]; }` and the workspace container runs a Nix environment. This enables reproducible environments and very fast layer caching via Nix's content-addressed store. The Cortado agent can call `nix develop` to activate the environment before shell spawn. Tools: `devenv.sh` (Nix-based dev environments), `nixpacks` (buildpack-style).

- **Devcontainer spec**: Support the VS Code `devcontainer.json` spec. Parse the spec, build/pull the specified image, apply `postCreateCommand`, etc. This gives instant compatibility with existing VS Code devcontainer configs.

- **Buildpacks (Cloud Native Buildpacks)**: Detect the language from the repo and auto-install runtimes. Tools: `pack` CLI, Google's `buildpacks`. Slower first-build but zero config for users.

### 4.3 Container Runtimes on GCP

**Option A: GKE Autopilot** (recommended primary)

GKE Autopilot handles node provisioning automatically. Workspaces run as Pods with resource requests. Autopilot supports both AMD64 and ARM64 node pools, and has Spot node support for cost savings.

```yaml
# workspace pod spec
apiVersion: v1
kind: Pod
metadata:
  name: ws-abc123
  namespace: cortado-workspaces
  labels:
    cortado/workspace-id: abc123
    cortado/tenant-id: tenant456
spec:
  serviceAccountName: workspace-sa
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: workspace
    image: us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace-ubuntu:22.04
    resources:
      requests:
        cpu: "1"
        memory: "2Gi"
      limits:
        cpu: "4"
        memory: "8Gi"
    volumeMounts:
    - name: workspace-data
      mountPath: /workspace
    - name: cortado-agent-socket
      mountPath: /run/cortado
  volumes:
  - name: workspace-data
    persistentVolumeClaim:
      claimName: ws-abc123-pvc
  - name: cortado-agent-socket
    emptyDir: {}
```

**Option B: Cloud Run Jobs** (for ephemeral workspaces)

For workspaces that don't need persistent storage between sessions, Cloud Run Jobs offer better scale-to-zero semantics and per-second billing. The limitation is the 60-minute request timeout for Cloud Run services (though Jobs can run longer). Use Cloud Run for short-lived tasks; GKE for long-running sessions.

**Option C: Firecracker MicroVMs via Fly.io or Hetzner**

Fly.io uses Firecracker and has a Machines API that's well-suited for this use case. Each workspace is a Firecracker VM (not a container), providing stronger isolation. Cold starts are ~100-300ms for Firecracker (vs ~50ms for container). Fly.io has persistent volumes (`fly volumes create`), built-in anycast networking, and WireGuard-based private networking between VMs. This is a compelling alternative to GKE for a less Google-locked architecture.

**Option D: Modal Labs**

Modal has a Python-first API for spinning up container functions with GPU support. Not suitable as a primary compute backend (no persistent shell sessions), but worth considering for AI inference workloads.

### 4.4 Scale-to-Zero Mechanics

The challenge with scale-to-zero for interactive IDEs is **cold start latency**. A terminal reconnection that triggers a cold start feels terrible at 10+ seconds.

**Strategies:**

**Warm pool**: Maintain N pre-warmed blank pods per workspace image. When a workspace starts, a pre-warmed pod is assigned and the user's data is mounted. After session end, the pod is cleaned and returned to the pool. This adds ~N * pod_cost in overhead but eliminates cold starts. Typically N=2-3 per active region is sufficient.

**Snapshot-and-restore (CRIU)**: Checkpoint running container processes with CRIU (Checkpoint/Restore in Userspace) and restore them on next access. GKE supports CRIU via the `--enable-checkpoint-restore` flag. This captures the full process state including open file descriptors, in-flight network connections, etc. Cold start from CRIU snapshot: ~200ms vs ~5-10s from scratch. The challenge is that CRIU snapshots can be large (GBs for complex dev environments).

**Volume-first approach**: Rather than snapshotting the container, snapshot only the persistent volume. Use GCP's `VolumeSnapshot` API to create instant snapshots. Pod startup still takes the full container start time, but the volume is immediately available.

For GPU workspaces specifically, cold starts are expensive because CUDA initialization takes 5-20 seconds. CRIU-based snapshotting of GPU contexts is partially supported via NVIDIA's CUDA checkpoint support (experimental in CUDA 12.x).

**Implementation of idle detection:**

```go
// In cortado-agent, track terminal activity
type IdleTracker struct {
    lastActivity atomic.Value // time.Time
    threshold    time.Duration
}

func (t *IdleTracker) RecordActivity() {
    t.lastActivity.Store(time.Now())
}

func (t *IdleTracker) IsIdle() bool {
    last, ok := t.lastActivity.Load().(time.Time)
    if !ok { return true }
    return time.Since(last) > t.threshold
}

// Control plane polls via gRPC health check
// After idleThreshold, sends SIGTERM to workspace, takes CRIU checkpoint
// Stores checkpoint to GCS, deletes pod
```

---

## 5. Virtual Workspace Architecture

### 5.1 The Two Modes

**Mode A: Local Mirror** — The canonical source of truth is the local filesystem. Cloud is a mirror. Useful when the developer's IDE runs locally but needs cloud compute for builds/tests.

**Mode B: Cloud-Canonical** — The canonical source of truth is the cloud filesystem. Useful for pure browser IDEs where there is no local filesystem to sync from.

Both modes can coexist in the same workspace (different paths can have different modes). The architecture uses the same CRDT-based sync protocol; the difference is only in conflict resolution policy and which side "wins" on first sync.

### 5.2 Local Mirror: The Cortado Local Daemon

The local daemon is a small Go binary (`cortado-daemon`) installed by the consuming IDE app. It runs as a background process and exposes a Unix socket (or named pipe on Windows) for the Flutter app to talk to.

```
Architecture:
  Flutter App (web browser)
    ↕ WebSocket (localhost:9731)
  cortado-daemon (Go, runs on user machine)
    ↕ gRPC stream (TLS, auth'd)
  Cortado File Sync Service (GCP)
    ↕ gRPC stream
  Workspace Agent (container)
```

**Daemon responsibilities:**
1. Watch local filesystem for changes (using FSEvents/inotify/ReadDirectoryChangesW)
2. Translate changes into canonical ops (CREATE, MODIFY, DELETE, RENAME, CHMOD)
3. Stream ops to cloud over gRPC
4. Receive ops from cloud, apply to local filesystem
5. Handle conflicts using vector clock comparison

**Change detection implementation:**

```go
// daemon/watcher.go
import "github.com/fsnotify/fsnotify"

type Watcher struct {
    fsw     *fsnotify.Watcher
    ops     chan FileOp
    ignore  *gitignore.GitIgnore
}

func (w *Watcher) Watch(root string) error {
    // Load .gitignore patterns (also .cortadoignore)
    w.ignore, _ = gitignore.CompileIgnoreFile(filepath.Join(root, ".gitignore"))

    // Walk and add all directories
    return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if d.IsDir() && !w.ignore.MatchesPath(path) {
            w.fsw.Add(path)
        }
        return nil
    })
}

func (w *Watcher) processEvents() {
    // Debounce rapid successive writes (editor saving temp files)
    debounce := make(map[string]*time.Timer)

    for event := range w.fsw.Events {
        if w.ignore.MatchesPath(event.Name) { continue }

        // Debounce: wait 50ms after last event for same file
        if t, ok := debounce[event.Name]; ok { t.Stop() }
        path := event.Name
        debounce[path] = time.AfterFunc(50*time.Millisecond, func() {
            w.emitOp(path, event.Op)
        })
    }
}
```

**The sync protocol (gRPC):**

```protobuf
// filesync.proto
service FileSync {
  rpc Sync(stream FileSyncMessage) returns (stream FileSyncMessage);
}

message FileSyncMessage {
  oneof payload {
    FileOp       op          = 1;
    Ack          ack         = 2;
    Conflict     conflict    = 3;
    StateVector  state_vec   = 4;  // For initial sync
    FileContent  content     = 5;  // For full file transfer
  }
}

message FileOp {
  string   op_id    = 1;  // UUID
  string   path     = 2;
  OpType   type     = 3;
  bytes    content  = 4;  // For WRITE ops, compressed with zstd
  bytes    patch    = 5;  // For text files: binary diff (bsdiff format)
  int64    clock    = 6;  // Logical Lamport clock
  string   node_id  = 7;  // Which node originated this op
  bytes    checksum = 8;  // xxHash64 of content
}
```

**Binary delta compression:**

For large files, sending the full content on every change is wasteful. Use **bsdiff** (or the faster **zstd-based rsync algorithm** implemented in the `librsync` Go binding) to send binary patches:

```go
func computePatch(oldContent, newContent []byte) ([]byte, error) {
    // Use bsdiff for general binary files
    var patch bytes.Buffer
    if err := bsdiff.Diff(bytes.NewReader(oldContent),
                           bytes.NewReader(newContent),
                           &patch); err != nil {
        return nil, err
    }
    // Compress the patch itself
    return zstd.Compress(nil, patch.Bytes())
}
```

For **text files** (detected by MIME type), use character-level or line-level OT (Operational Transform) instead:

```go
// Text OT using the Automerge CRDT library (Go binding)
// Or implement Yjs-compatible ops for interop with CodeMirror/Monaco
```

**Yjs integration**: Yjs (and its Go port `yjs-go`, or the reference implementation via WASM) is the industry standard for real-time collaborative text editing. If Cortado wants to support multiplayer editing (two developers in the same workspace), Yjs is the right primitive. A single Yjs document can represent a file; ops are `Insert(pos, char)` and `Delete(pos, len)`. Yjs ops commute, so there are no conflicts.

### 5.3 Cloud-Canonical Mode

In this mode, the workspace filesystem lives entirely on a GCP persistent disk (or GCS with a FUSE mount). The Flutter client interacts with files via the Cortado File API:

```
GET  /v1/workspaces/{id}/files/{path}    → file content
PUT  /v1/workspaces/{id}/files/{path}    → write file
DELETE /v1/workspaces/{id}/files/{path}  → delete
GET  /v1/workspaces/{id}/files/{path}?list=true  → directory listing
WebSocket /v1/workspaces/{id}/files/watch → real-time change events
```

**Storage backend options:**

**Option A: GCP Persistent Disk (SSD)** — best I/O performance (~100K IOPS for PD-SSD), lowest latency. Cost: ~$0.17/GB/month. Min size: 10GB. Works well for GKE.

**Option B: GCS + GCS FUSE** — mount a GCS bucket as a filesystem in the container. Read latency is higher (~10ms first read vs ~0.1ms for PD), but storage is cheaper (~$0.02/GB/month) and unlimited. Good for large files and infrequently-accessed data. GCS FUSE v2 (the "experimental" kernel module mode) significantly improves performance.

**Option C: Filestore (NFS)** — GCP's managed NFS. Enables multiple pods to share the same filesystem (useful for multi-container workspaces). Minimum capacity: 1TB (~$200/month), which makes it cost-prohibitive for per-workspace storage. Use only for shared artifact caches.

**Option D: Tigris (S3-compatible, global)** — Tigris provides S3-compatible object storage with global replication and strong consistency. Accessible via a FUSE mount or the S3 API. Good for teams that need cross-region workspace access.

**Recommended**: GCP PD-SSD for active workspaces (attached to pod), with automatic migration to GCS when workspace is hibernated (using a `rclone sync` snapshot).

### 5.4 Workspace Filesystem Snapshot & Versioning

Workspaces need history. Options:

**Git-based**: Run `git` inside the workspace, commit on save (or on session end). This is what Gitpod does. The filesystem IS a git repo. Provides diff, restore, blame for free. Downside: git doesn't handle binary files well, and `.git` directory overhead can be significant.

**GCP Disk Snapshots**: Incremental block-level snapshots. Very fast and storage-efficient. But restoring requires detaching/reattaching a disk, which has multi-second latency.

**Restic/Borg**: Application-level backup tools. Restic in particular supports deduplication across snapshots and can write to GCS. A `restic backup /workspace` command creates an incremental snapshot in seconds. Restore is also fast. This is the recommended approach for user-facing "workspace history."

```bash
# In cortado-agent, triggered on session end or timer
restic --repo gs:cortado-snapshots/ws-abc123 backup /workspace \
  --tag "session-end" \
  --exclude "node_modules" \
  --exclude ".git"
```

---

## 6. Resource Scaling & Billing

### 6.1 Resource Dimensions

Each workspace has independently scalable dimensions:

| Resource | Unit | Min | Max | Granularity |
|----------|------|-----|-----|-------------|
| CPU | vCPU | 0.25 | 96 | 0.25 vCPU |
| Memory | GB | 0.5 | 624 | 0.5 GB |
| Ephemeral storage | GB | 10 | 2000 | 1 GB |
| GPU | GPU (fractional) | 0 | 8 | 1/7 GPU (MIG) |
| Persistent storage | GB | 1 | unlimited | 1 GB |

### 6.2 Dynamic Resource Adjustment

GKE supports **Vertical Pod Autoscaler (VPA)** for in-place resource adjustment. The workspace agent monitors its own CPU/memory usage and can request upscaling:

```go
// agent/resource_monitor.go
func (m *ResourceMonitor) CheckAndRequestUpscale() {
    metrics := m.collectMetrics()

    // CPU threshold: >80% sustained for 30 seconds
    if metrics.CPUPercent > 80 && m.cpuHighDuration > 30*time.Second {
        m.requestUpscale(ResourceDelta{CPU: 2.0}) // Add 2 vCPUs
    }

    // Memory threshold: >85% usage
    if metrics.MemoryPercent > 85 {
        m.requestUpscale(ResourceDelta{MemoryGB: 4.0}) // Add 4GB
    }
}

func (m *ResourceMonitor) requestUpscale(delta ResourceDelta) {
    // Call control plane API to update pod resource limits
    // Control plane will patch the Pod spec and trigger VPA
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    m.controlPlaneClient.UpdateResources(ctx, &pb.UpdateResourcesRequest{
        WorkspaceId: m.workspaceID,
        Delta:       delta.ToProto(),
    })
}
```

VPA in GKE supports `UpdateMode: InPlace` (as of Kubernetes 1.27 with the `InPlacePodVerticalScaling` feature gate), which resizes pod CPU/memory without restart.

**GPU attachment**: GPU workloads require a node with a GPU. The VPA can't add a GPU to a running pod — the pod must be scheduled on a GPU node. The right model is: workspaces declare at creation time whether they need GPU (triggering scheduling on GPU nodepool), or they perform a "GPU attach" operation which is implemented as pod deletion + recreation on a GPU node. To reduce GPU cost, use **NVIDIA MIG (Multi-Instance GPU)** to slice A100s into fractional GPUs (1/7th of an A100 = ~10GB VRAM). In GKE: `nvidia.com/mig-1g.10gb: 1`.

### 6.3 Billing Architecture

**Per-second billing requires metering at sub-second granularity.** The agent reports resource usage events; the billing service aggregates them.

**Event-driven metering architecture:**

```
workspace-agent → Pub/Sub topic: cortado-usage-events
  ↓
Cloud Dataflow (streaming pipeline) or BigQuery Subscription
  ↓
BigQuery: cortado_billing.usage_events table
  ↓
dbt models → cortado_billing.invoice_line_items
  ↓
Stripe Billing (via Stripe Meters API) or manual invoicing
```

**Usage event schema (published every 10 seconds):**

```json
{
  "workspace_id": "ws_abc123",
  "tenant_id": "tenant_456",
  "user_id": "user_789",
  "timestamp": "2026-04-30T10:00:00.000Z",
  "duration_seconds": 10,
  "resources": {
    "cpu_vcpu_seconds": 20.0,
    "memory_gb_seconds": 40.0,
    "gpu_seconds": 0,
    "storage_gb_hours": 0.0028
  },
  "region": "us-central1"
}
```

**OpenMeter** (open-source usage metering, https://openmeter.io) provides a purpose-built solution for exactly this:

```go
// In cortado-agent or billing service
client := openmeter.NewClient(os.Getenv("OPENMETER_API_KEY"))

client.IngestEvent(ctx, &openmeter.Event{
    ID:          uuid.NewString(),
    Type:        "workspace.resource.usage",
    Source:      "cortado-agent",
    Subject:     fmt.Sprintf("workspace/%s", workspaceID),
    Time:        time.Now(),
    Data: map[string]any{
        "cpu_vcpu_seconds":   cpuVcpuSeconds,
        "memory_gb_seconds":  memGbSeconds,
        "workspace_id":       workspaceID,
    },
})
```

OpenMeter handles deduplication, windowing, and aggregation. It integrates with Stripe Billing's Meter API for automatic invoice generation.

**Stripe Meters API** (launched 2024) provides:
```javascript
// Create a meter for CPU usage
const meter = await stripe.billing.meters.create({
  display_name: 'CPU vCPU-Seconds',
  event_name: 'cpu_vcpu_seconds',
  default_aggregation: { formula: 'sum' },
});

// Report usage
await stripe.billing.meterEvents.create({
  event_name: 'cpu_vcpu_seconds',
  payload: {
    value: String(cpuVcpuSeconds),
    stripe_customer_id: customerId,
  },
});
```

**Pricing model design:**

| Resource | Price | Notes |
|----------|-------|-------|
| CPU | $0.000010/vCPU-second | ~$0.036/vCPU-hour |
| Memory | $0.0000013/GB-second | ~$0.0047/GB-hour |
| Storage | $0.000000056/GB-second | ~$0.20/GB-month |
| GPU (T4) | $0.000123/GPU-second | ~$0.44/GPU-hour |
| GPU (A100 MIG 1/7) | $0.000389/GPU-second | ~$1.40/GPU-hour |

Scale-to-zero means zero charges during idle. Free tier: 20 CPU-hours/month, 40 GB-hours memory/month (attracts individual developers).

### 6.4 Cost Passthrough Model

Cortado's own infrastructure costs on GCP need to be marked up appropriately. The primary cost drivers:

- **GKE Autopilot node cost**: ~$0.0085/vCPU-second (standard VMs). Cortado charges $0.000010/vCPU-second = $0.036/vCPU-hour. GCP costs ~$0.031/vCPU-hour on standard nodes, so margin is ~16% at list price. Spot VMs reduce GCP cost to ~$0.009/vCPU-hour, increasing margin to ~75%.

- **Spot VM risk**: Spot VMs can be preempted. The workspace agent must handle SIGTERM gracefully, checkpoint state (filesystem snapshot + CRIU if possible), and trigger pod rescheduling. This requires careful handling but is worth it for the cost savings.

---

## 7. Port Forwarding & Chrome Support

### 7.1 Port Forwarding Architecture

When user code inside a workspace binds to a port (e.g., `flutter run -d web-server --web-port 8080`), Cortado needs to expose that port to the developer's browser. This is more complex than it sounds because:

- The workspace pod has no public IP
- The client is a Flutter Web app (can't open arbitrary TCP connections)
- We need authentication (don't expose workspace ports publicly)

**Architecture:**

```
Browser (Flutter app)
  ↕ HTTPS / WebSocket (authenticated)
Cortado Port Forward Gateway (Cloud Run)
  ↕ mTLS to workspace pod
Workspace agent (port 9732)
  ↕ TCP connection to localhost:8080
User application (port 8080)
```

The Port Forward Gateway acts as an authenticated HTTP reverse proxy. It validates the JWT, resolves the workspace pod address, and proxies the request.

**Implementation (Go, using `net/http` reverse proxy):**

```go
// gateway/port_forwarder.go
func (gw *PortForwardGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // URL format: /pf/{workspaceId}/{port}/{path...}
    parts := strings.SplitN(r.URL.Path, "/", 5)
    workspaceID, port := parts[2], parts[3]

    // Auth check
    if err := gw.auth.Validate(r.Header.Get("Authorization")); err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }

    // Resolve workspace pod internal address
    podAddr, err := gw.resolver.GetPodAddress(r.Context(), workspaceID)
    if err != nil {
        http.Error(w, "Workspace not found", 404)
        return
    }

    // Build target URL for the workspace agent's port-forward proxy
    // The agent proxies to localhost:{port} inside the container
    target, _ := url.Parse(fmt.Sprintf("https://%s/proxy/%s", podAddr, port))

    proxy := &httputil.ReverseProxy{
        Director: func(req *http.Request) {
            req.URL.Scheme = target.Scheme
            req.URL.Host = target.Host
            req.URL.Path = "/" + strings.Join(parts[4:], "/")
            req.Header.Set("X-Forwarded-Host", r.Host)
        },
        Transport: gw.mtlsTransport, // mTLS to workspace agent
    }

    // WebSocket upgrade handling
    if websocket.IsWebSocketUpgrade(r) {
        gw.proxyWebSocket(w, r, podAddr, port)
        return
    }

    proxy.ServeHTTP(w, r)
}
```

In the workspace agent:
```go
// agent/port_proxy.go — exposes local ports via a reverse proxy endpoint
func (pp *PortProxy) ServeProxy(w http.ResponseWriter, r *http.Request) {
    // Extract target port from path
    portStr := chi.URLParam(r, "port")
    port, _ := strconv.Atoi(portStr)

    // Security: only allow ports in the allowed range (1024-65535)
    // and only ports that are actually bound (check /proc/net/tcp6)
    if !pp.isPortBound(port) {
        http.Error(w, "Port not bound", 404)
        return
    }

    target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", port))
    proxy := httputil.NewSingleHostReverseProxy(target)
    proxy.ServeHTTP(w, r)
}
```

**Alternative: Tunnel-based approach (Cloudflare Tunnel / ngrok model)**

Instead of a gateway-mediated proxy, use a tunnel:
1. Workspace agent establishes outbound tunnel to Cortado tunnel server (or Cloudflare Tunnel)
2. Each port gets a unique subdomain: `{workspaceId}-{port}.preview.cortado.dev`
3. Requests to that subdomain route through the tunnel to the workspace

This is simpler but has higher latency (extra hop) and requires wildcard DNS + TLS cert for `*.preview.cortado.dev`. Cloudflare Tunnel (`cloudflared`) can be embedded in the workspace agent for this exact purpose.

### 7.2 Chrome in the Cloud (Flutter Web Preview)

For Flutter web development, the developer needs to see their app running in a real browser. The approach:

1. Inside the workspace, Xvfb provides a virtual framebuffer
2. Google Chrome runs in `--headless=new` mode (or with Xvfb for full non-headless)
3. Chrome's DevTools Protocol (CDP) is exposed on port 9222
4. A VNC or WebRTC-based screen capture is streamed to the Flutter client

**Architecture options:**

**Option A: noVNC (VNC over WebSocket)**

```bash
# In workspace, on user request:
export DISPLAY=:99
Xvfb :99 -screen 0 1920x1080x24 &
google-chrome --display=:99 http://localhost:8080 &
x11vnc -display :99 -forever -nopw -listen localhost -rfbport 5900 &
websockify 6080 localhost:5900 &
# noVNC JavaScript client connects to ws://localhost:6080
```

This is the simplest approach. noVNC renders the remote desktop in HTML5 Canvas. Frame rate is typically 15-30 FPS. Latency depends on network (~50-200ms).

**Option B: WebRTC screen capture (lower latency)**

Use `ffmpeg` + a WebRTC server (`pion/webrtc`) to capture the Xvfb display and stream it over WebRTC (with SRTP encoding). WebRTC typically achieves 30-60 FPS at lower latency than VNC.

```go
// agent/chrome_streamer.go
func (cs *ChromeStreamer) Start(displayNum int) error {
    // Start ffmpeg capturing Xvfb display, pipe to WebRTC
    cs.cmd = exec.Command("ffmpeg",
        "-f", "x11grab",
        "-r", "30",
        "-s", "1920x1080",
        "-i", fmt.Sprintf(":%d", displayNum),
        "-vcodec", "libvpx-vp9",
        "-b:v", "2M",
        "-f", "rawvideo",
        "-pix_fmt", "yuv420p",
        "pipe:1",
    )
    // Pipe ffmpeg output to WebRTC track...
    return cs.startWebRTCStream()
}
```

In the Flutter widget, display the WebRTC stream:
```dart
// Use flutter_webrtc package
final renderer = RTCVideoRenderer();
await renderer.initialize();
final pc = await createPeerConnection({...});
pc.onTrack = (event) {
  renderer.srcObject = event.streams[0];
};
// Widget
RTCVideoView(renderer)
```

**Option C: Chrome DevTools Protocol (CDP) replay**

For simpler preview needs (checking layout, not interaction), stream CDP screenshot events:
```javascript
// CDP screenshot streaming
const browser = await puppeteer.launch({ headless: true });
const page = await browser.newPage();
await page.goto('http://localhost:8080');

// Stream screenshots via WebSocket
setInterval(async () => {
  const screenshot = await page.screenshot({ type: 'webp', quality: 80 });
  ws.send(screenshot);
}, 1000/30);
```

This approach gives a "remote display" at lower complexity than Xvfb/WebRTC, but doesn't support interaction (mouse/keyboard events aren't forwarded to the DOM).

**Option D: Flipping the model with WASM**

Alternatively, rather than running Chrome in the cloud, run the Flutter app's compiled WASM/JS in the *client's* browser. The Cortado workspace handles the build (`flutter build web`), outputs the artifacts to a URL, and the client's browser runs it directly. No need for cloud Chrome at all. This is the most practical approach for "Flutter web preview" specifically.

---

## 8. Language Server Protocol (LSP) & Extensions

### 8.1 LSP Architecture

The Language Server Protocol defines JSON-RPC communication between an editor (client) and a language intelligence server. In Cortado's context:

```
Flutter Editor Widget (LSP client)
  ↕ JSON-RPC over WebSocket (via Cortado LSP bridge)
Cortado LSP Gateway (in control plane)
  ↕ JSON-RPC over stdin/stdout (forwarded)
Language Server (in workspace pod)
  - dart analysis_server
  - rust-analyzer
  - typescript-language-server
  - pyright / pylsp
  - clangd
  etc.
```

**LSP Gateway responsibilities:**
- Route LSP messages to the correct workspace's language server instance
- Handle reconnections (client disconnects, server restarts)
- Log LSP messages for debugging
- Rate-limit expensive operations (e.g., `workspace/symbol` search)
- Aggregate diagnostics for the billing/analytics system

**Workspace agent LSP management:**

```go
// agent/lsp_manager.go
type LSPManager struct {
    servers  map[string]*lspServer  // language → server
    mu       sync.RWMutex
}

type lspServer struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    msgs   chan []byte  // outgoing messages to client
}

func (m *LSPManager) GetOrStart(language string) (*lspServer, error) {
    m.mu.RLock()
    if srv, ok := m.servers[language]; ok {
        m.mu.RUnlock()
        return srv, nil
    }
    m.mu.RUnlock()

    return m.startServer(language)
}

func (m *LSPManager) startServer(language string) (*lspServer, error) {
    var cmd *exec.Cmd
    switch language {
    case "dart":
        dartSdk := os.Getenv("DART_SDK_PATH")
        cmd = exec.Command(filepath.Join(dartSdk, "bin/dart"),
            "language-server", "--protocol=lsp")
    case "typescript", "javascript":
        cmd = exec.Command("typescript-language-server", "--stdio")
    case "rust":
        cmd = exec.Command("rust-analyzer")
    case "python":
        cmd = exec.Command("pyright-langserver", "--stdio")
    case "go":
        cmd = exec.Command("gopls")
    default:
        return nil, fmt.Errorf("unsupported language: %s", language)
    }

    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()

    srv := &lspServer{cmd: cmd, stdin: stdin, stdout: stdout, msgs: make(chan []byte, 100)}
    m.servers[language] = srv

    // Start reading LSP responses
    go srv.readLoop()

    // Send initialize request
    m.sendInitialize(srv)

    return srv, nil
}
```

**LSP message framing (Content-Length protocol):**

```go
// LSP uses HTTP-like header framing over stdio
func (s *lspServer) Send(method string, params interface{}) error {
    msg := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      atomic.AddInt64(&s.seq, 1),
        "method":  method,
        "params":  params,
    }
    body, _ := json.Marshal(msg)
    header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
    s.mu.Lock()
    defer s.mu.Unlock()
    _, err := s.stdin.Write(append([]byte(header), body...))
    return err
}

func (s *lspServer) readLoop() {
    reader := bufio.NewReader(s.stdout)
    for {
        // Read Content-Length header
        var contentLength int
        for {
            line, _ := reader.ReadString('\n')
            line = strings.TrimSpace(line)
            if line == "" { break }
            if strings.HasPrefix(line, "Content-Length: ") {
                fmt.Sscanf(line, "Content-Length: %d", &contentLength)
            }
        }
        body := make([]byte, contentLength)
        io.ReadFull(reader, body)
        s.msgs <- body
    }
}
```

### 8.2 LSP in the Flutter Package

The Flutter package implements an LSP client:

```dart
// lsp/lsp_client.dart
class CortadoLSPClient {
  final _pending = <int, Completer<Map<String, dynamic>>>{};
  int _nextId = 1;
  late final WebSocketChannel _channel;

  Future<Map<String, dynamic>> sendRequest(
    String method,
    Map<String, dynamic> params,
  ) async {
    final id = _nextId++;
    final completer = Completer<Map<String, dynamic>>();
    _pending[id] = completer;

    _channel.sink.add(jsonEncode({
      'jsonrpc': '2.0',
      'id': id,
      'method': method,
      'params': params,
    }));

    return completer.future.timeout(const Duration(seconds: 10));
  }

  void _handleMessage(dynamic raw) {
    final msg = jsonDecode(raw as String) as Map<String, dynamic>;

    if (msg.containsKey('id') && _pending.containsKey(msg['id'])) {
      // Response to a request
      _pending.remove(msg['id'])!.complete(msg['result']);
    } else if (msg.containsKey('method')) {
      // Notification or request from server
      _handleNotification(msg['method'], msg['params']);
    }
  }

  // High-level API for editor integration
  Future<List<CompletionItem>> getCompletions(
    String uri, Position position,
  ) async {
    final result = await sendRequest('textDocument/completion', {
      'textDocument': {'uri': uri},
      'position': position.toJson(),
    });
    final list = result['items'] as List? ?? [];
    return list.map(CompletionItem.fromJson).toList();
  }

  Future<List<Diagnostic>> getDiagnostics(String uri) async {
    // Note: diagnostics are push (server sends publishDiagnostics notification)
    // This method returns cached diagnostics
    return _diagnosticsCache[uri] ?? [];
  }

  Stream<List<Diagnostic>> watchDiagnostics(String uri) {
    return _diagnosticsStream
        .where((event) => event.uri == uri)
        .map((event) => event.diagnostics);
  }
}
```

### 8.3 Extension Environments

VS Code extensions are Node.js processes communicating via VS Code's extension host API. Cortado can support a compatible extension environment, but this requires significant investment.

**Option A: VS Code Server (Code-OSS)**

Run `code-server` (the open-source VS Code server) inside the workspace container. The user accesses VS Code's full UI via the browser, and extensions run in the container. The Cortado Flutter package would embed this as an iframe/webview rather than providing its own editor widget. This sacrifices UI customization but provides instant compatibility with the VS Code extension ecosystem.

```dockerfile
# In workspace image
RUN curl -fsSL https://code-server.dev/install.sh | sh
```

**Option B: OpenVSX + Custom Extension Host**

Implement a subset of the VS Code extension API. Extensions from the OpenVSX registry (the open-source VS Code marketplace) can run in a custom extension host. This is what Gitpod and Eclipse Theia do. It requires implementing `vscode.d.ts` APIs. Key APIs to implement:

- `vscode.workspace.fs` — filesystem access
- `vscode.languages.registerCompletionItemProvider` — LSP-like completions
- `vscode.window.showTextDocument` — editor integration
- `vscode.debug` — debugger protocol

The extension host runs as a Node.js worker process in the workspace container. Extensions communicate with it via message passing (same pattern as VS Code's extension host).

**Option C: Zed extensions** (newer standard)

Zed's extension API is simpler and WASM-based. Extensions are compiled to WASM and run in a sandboxed host. This is a cleaner design than VS Code's extension host but has a much smaller extension ecosystem (as of 2026).

**Option D: Language-specific tool integration**

Rather than a general extension system, integrate specific tools directly: linters, formatters, debuggers. Dart/Flutter tools (dart fix, dart format, flutter analyze), prettier, eslint, gofmt, etc. Invoke them as CLI tools from the workspace agent and stream results to the Flutter client.

### 8.4 Debug Adapter Protocol (DAP)

The Debug Adapter Protocol (from Microsoft, like LSP) standardizes debugger integration. The architecture mirrors LSP:

```
Flutter Debug UI (DAP client)
  ↕ JSON over WebSocket
Cortado DAP Gateway
  ↕ JSON over stdin/stdout
Debug Adapter (in workspace)
  - dart-debug-adapter (part of Dart SDK)
  - debugpy (Python)
  - dlv (Go)
  - node --inspect (Node.js/TypeScript)
```

The Dart SDK ships with a debug adapter at `dart debug_adapter`. Cortado can route DAP requests to it for in-container Flutter/Dart debugging.

---

## 9. Codebase Indexing

### 9.1 Traditional (Non-AI) Indexing

Fast code navigation requires a proper index. The approaches:

**ctags / Universal Ctags**: Fastest to build, produces a simple tag index (symbol → file:line). Supports 150+ languages. Used by vim, emacs, and many editors.

```bash
# Build index in workspace
ctags --recurse --fields=+l --languages=Dart,Python,JavaScript,Go .
```

**Tree-sitter**: A parser generator that builds concrete syntax trees incrementally. Much richer than ctags — understands scope, types, references. Used by Neovim, GitHub's semantic code navigation, and many modern tools. Has bindings for most languages. The Rust crate `tree-sitter` is the reference implementation; Go and Python bindings exist.

**Kythe**: Google's cross-language code indexing system. Generates a facts graph (nodes = code entities, edges = relationships). Supports Dart (via the `dart_kythe` analyzer), Java, C++, Python, TypeScript. Extremely thorough but complex to set up.

**Sourcegraph's `scip` (SCIP: SCIP Code Intelligence Protocol)**: A newer standard from Sourcegraph, designed as a successor to LSIF (Language Server Index Format). Language-specific indexers (scip-typescript, scip-python, scip-java) generate `.scip` files. Cortado can run these indexers and store the output in a database (Sourcegraph uses PostgreSQL with a custom schema, but SCIP output can also be stored in any column store).

**Recommended**: Tree-sitter for in-editor navigation (incremental, fast), SCIP for cross-file/cross-repo indexing (richer, used for AI context).

### 9.2 AI-Based Codebase Indexing (RAG Architecture)

The goal: given any natural language query or a code snippet, find the most relevant code in the codebase. This requires a vector embedding index.

**Architecture:**

```
Codebase files
  ↓ Chunking strategy (AST-aware)
Code chunks (with metadata)
  ↓ Embedding model
  ↓ (text-embedding-gecko, voyage-code-2, cohere-embed-v3)
Vector embeddings (768 or 1536 dim)
  ↓ Upsert
Vector database
  (Vertex AI Vector Search / Weaviate / Qdrant / pgvector)
  ↓ ANN search
Top-K relevant chunks
  ↓ Reranking (optional, Cohere Rerank)
Final context for LLM
```

**Chunking strategy**: This is the most important architectural decision for RAG quality. Naive line-based chunking loses semantic structure. AST-aware chunking is much better:

```python
# indexer/chunker.py
import tree_sitter_dart as tsdart
from tree_sitter import Language, Parser

DART_LANGUAGE = Language(tsdart.language())

def chunk_dart_file(source: str, file_path: str) -> list[dict]:
    parser = Parser(DART_LANGUAGE)
    tree = parser.parse(source.encode())
    chunks = []

    def visit(node, parent_name=None):
        # Chunk at function/method/class level
        if node.type in ('function_declaration', 'method_declaration',
                          'class_declaration', 'constructor_declaration'):
            name_node = node.child_by_field_name('name')
            name = name_node.text.decode() if name_node else 'anonymous'
            full_name = f"{parent_name}.{name}" if parent_name else name

            chunk_text = source[node.start_byte:node.end_byte]

            # Add context: parent class, imports, docstring
            context = extract_context(tree, node, source)

            chunks.append({
                'text': f"{context}\n\n{chunk_text}",
                'metadata': {
                    'file': file_path,
                    'name': full_name,
                    'type': node.type,
                    'start_line': node.start_point[0],
                    'end_line': node.end_point[0],
                }
            })

            # Recurse into class members
            for child in node.children:
                visit(child, full_name)
        else:
            for child in node.children:
                visit(child, parent_name)

    visit(tree.root_node)
    return chunks
```

**Embedding models for code:**

- `voyage-code-3` (Voyage AI): Best-in-class for code, 1024 dimensions, 16K context window. ~$0.00012/1K tokens.
- `text-embedding-004` (Google/Vertex AI): Good for multilingual code, 768 dimensions. Free tier available.
- `cohere-embed-v3` (Cohere): Strong for code+English mixed queries. 1024 dimensions.
- `nomic-embed-code` (Nomic AI): Open-source, can be self-hosted in workspace for offline use.

**Vector databases:**

- **Vertex AI Vector Search**: GCP-native, fully managed, high performance at scale. Batch upsert + streaming updates. No self-hosting needed. Query latency ~10-20ms at 1M vectors.
- **Weaviate** (self-hosted or Weaviate Cloud): Open-source, rich filtering, hybrid search (BM25 + vector). Can be deployed as a GKE sidecar or as a managed service. Good for sub-100K vector collections.
- **Qdrant**: Rust-based, very fast, excellent filtering support. Self-hosted or Qdrant Cloud. Strong candidate for per-workspace lightweight indices.
- **pgvector** (PostgreSQL extension): If Cortado already uses PostgreSQL, pgvector adds ANN search with `ivfflat` or `hnsw` indexes. Simple to operate but not as fast as dedicated vector DBs at scale.

**Recommended per-workspace architecture**: Each workspace has a small Qdrant collection (running as a sidecar, ~200MB RAM) for fast in-workspace search. Cross-workspace search (across all repos for a tenant) uses Vertex AI Vector Search.

**Incremental indexing**: Don't re-index the whole codebase on every change. Use the file change events from the sync system:

```python
# indexer/incremental.py
async def handle_file_change(event: FileChangeEvent):
    if event.type in ('CREATE', 'MODIFY'):
        content = await read_file(event.path)
        chunks = chunk_file(content, event.path)

        # Delete old vectors for this file
        await qdrant.delete(
            collection_name=event.workspace_id,
            points_selector=qdrant.FilterSelector(
                filter=qdrant.Filter(
                    must=[qdrant.FieldCondition(
                        key="file",
                        match=qdrant.MatchValue(value=event.path)
                    )]
                )
            )
        )

        # Embed new chunks
        embeddings = await embed(chunks)
        await qdrant.upsert(
            collection_name=event.workspace_id,
            points=[
                qdrant.PointStruct(
                    id=uuid4().hex,
                    vector=emb,
                    payload=chunk['metadata']
                )
                for chunk, emb in zip(chunks, embeddings)
            ]
        )

    elif event.type == 'DELETE':
        await qdrant.delete(...)  # similar to above
```

---

## 10. AI Features

### 10.1 Feature Taxonomy

**Tier 1 (core, must-have):**
- Inline code completion (Copilot-style, next-token or multi-line)
- Chat-based code assistant (contextual Q&A about the codebase)
- Natural language to code generation

**Tier 2 (differentiated):**
- Codebase-aware chat (RAG-backed, "explain this function", "how does auth work here")
- Automated code review
- Intelligent rename/refactor
- Test generation

**Tier 3 (advanced):**
- Autonomous coding agents (agentic loops, tool use)
- Semantic search across repos
- Automatic documentation generation
- Anomaly detection in runtime behavior

### 10.2 Inline Code Completion

The technical challenge: completions must be low-latency (<300ms perceived) and high quality.

**Architecture:**

```
User types in editor
  → Debounce (150ms)
  → Collect context:
      - Current file content (with cursor position)
      - Related files (via RAG lookup)
      - Recent edits (rolling window)
  → Streaming inference request
  → Stream tokens back to editor
  → Render ghost text
```

**Model selection:**
- **Gemini 2.5 Flash** (via Vertex AI): Very fast, strong at code, streaming support. Good cost/quality tradeoff.
- **Claude 3.5 Haiku** (via Anthropic API): Strong code completions, 200K context window. ~$0.80/M input tokens.
- **DeepSeek Coder V2 (self-hosted on Vertex/GKE)**: Open-source model, can be deployed on GKE GPU nodes. Eliminates per-token API costs at scale.
- **Qwen2.5-Coder** (self-hosted): Another strong open-source option.

For **FIM (Fill-In-Middle) completions** — the user has code above and below the cursor and needs the middle filled in — use models that support FIM tokens:

```python
# FIM completion request (DeepSeek/CodeLlama format)
prompt = f"<|fim_prefix|>{prefix}<|fim_suffix|>{suffix}<|fim_middle|>"
```

Claude and Gemini support FIM via system prompt engineering even without native FIM tokens.

**Streaming implementation in Dart:**

```dart
// ai/inline_completion.dart
Stream<String> getInlineCompletion(CompletionContext ctx) async* {
  final request = await http.Client().send(
    http.Request('POST', Uri.parse('$baseUrl/v1/completions'))
      ..headers['Authorization'] = 'Bearer $apiKey'
      ..headers['Content-Type'] = 'application/json'
      ..body = jsonEncode({
        'model': 'gemini-2.5-flash',
        'stream': true,
        'max_tokens': 200,
        'messages': [
          {'role': 'system', 'content': _buildSystemPrompt(ctx)},
          {'role': 'user', 'content': _buildCompletionPrompt(ctx)},
        ],
      }),
  );

  final stream = request.stream.transform(utf8.decoder);
  await for (final chunk in stream) {
    for (final line in chunk.split('\n')) {
      if (line.startsWith('data: ') && line != 'data: [DONE]') {
        final data = jsonDecode(line.substring(6));
        final token = data['choices'][0]['delta']['content'] as String?;
        if (token != null) yield token;
      }
    }
  }
}
```

### 10.3 Agentic Coding (Tool Use)

The most powerful AI feature: a coding agent that can autonomously execute multi-step tasks.

**Architecture:**

```
User: "Fix all the failing tests"
  ↓
Agent loop (in Cortado backend):
  1. Read test output (tool: run_command("flutter test"))
  2. Identify failing tests (parse output)
  3. Read relevant source files (tool: read_file)
  4. Generate fix (LLM inference with full context)
  5. Write fix (tool: write_file)
  6. Verify (tool: run_command("flutter test"))
  7. Repeat if still failing
  ↓
Stream progress events to Flutter client
  ↓
Render changes in file tree + diff view
```

**Tool definitions (for Claude's tool use API):**

```python
tools = [
    {
        "name": "run_command",
        "description": "Run a shell command in the workspace and return stdout/stderr",
        "input_schema": {
            "type": "object",
            "properties": {
                "command": {"type": "string"},
                "working_dir": {"type": "string", "default": "/workspace"},
                "timeout_seconds": {"type": "integer", "default": 30}
            },
            "required": ["command"]
        }
    },
    {
        "name": "read_file",
        "description": "Read the contents of a file in the workspace",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string"}
            },
            "required": ["path"]
        }
    },
    {
        "name": "write_file",
        "description": "Write content to a file (overwrites existing content)",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string"},
                "content": {"type": "string"}
            },
            "required": ["path", "content"]
        }
    },
    {
        "name": "search_codebase",
        "description": "Semantic search across the codebase using natural language",
        "input_schema": {
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "top_k": {"type": "integer", "default": 5}
            }
        }
    },
    {
        "name": "list_directory",
        "description": "List files and directories at a path",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "default": "/workspace"}
            }
        }
    }
]
```

**The agent loop:**

```python
# ai/agent_loop.py
async def run_agent(task: str, workspace_id: str, stream_callback: Callable):
    messages = [{"role": "user", "content": task}]

    while True:
        response = await anthropic_client.messages.create(
            model="claude-opus-4-5",
            max_tokens=4096,
            tools=tools,
            messages=messages,
            system=CODING_AGENT_SYSTEM_PROMPT,
        )

        # Stream text chunks to client
        for block in response.content:
            if block.type == "text":
                await stream_callback(AgentEvent(type="thinking", text=block.text))

        # Execute tool calls
        tool_results = []
        for block in response.content:
            if block.type == "tool_use":
                await stream_callback(AgentEvent(
                    type="tool_call",
                    tool=block.name,
                    input=block.input
                ))
                result = await execute_tool(block.name, block.input, workspace_id)
                await stream_callback(AgentEvent(
                    type="tool_result",
                    tool=block.name,
                    result=str(result)[:2000]  # Truncate for display
                ))
                tool_results.append({
                    "type": "tool_result",
                    "tool_use_id": block.id,
                    "content": str(result),
                })

        if response.stop_reason == "end_turn":
            break

        # Continue loop with tool results
        messages.append({"role": "assistant", "content": response.content})
        messages.append({"role": "user", "content": tool_results})
```

**Agent safety**: Agents can do destructive things. Safety measures:
- Dry-run mode: show the diff before applying changes
- Undo stack: every tool call that modifies files creates a snapshot
- Permission scoping: the agent runs as the workspace user (not root)
- Budget limits: max N tool calls, max execution time, max tokens

### 10.4 AI Context Window Management

Models have finite context windows. For large codebases (100K+ lines), you can't fit everything in context. Strategy:

**Contextual retrieval**: Use the RAG system to fetch the top-K most relevant code chunks for the current task. Include: the current file, directly imported files, recently edited files, and semantically related files.

**Context budget allocation** (for a 200K token context window):
- System prompt: 2K tokens
- Current file: up to 20K tokens
- Related files (RAG top-5): up to 40K tokens
- Recent conversation history: up to 30K tokens
- Tool results: up to 50K tokens
- Reserve for response: 20K tokens

```python
def build_context(task, workspace_id, current_file, cursor_pos, conversation_history):
    budget = ContextBudget(total=180_000)  # leave room for response

    # Always include current file
    current_content = read_file(current_file)
    budget.allocate("current_file", tokens(current_content), priority=1)

    # RAG: find related code
    query = f"{task}\n\nCurrent file: {current_file}"
    related = vector_search(workspace_id, query, top_k=10)

    # Greedy allocation by relevance score
    for chunk in related:
        t = tokens(chunk.text)
        if budget.can_allocate("related", t):
            budget.allocate("related", t, content=chunk.text)

    return budget.build_messages()
```

---

## 11. Security Architecture

### 11.1 Workspace Isolation

Each workspace pod must be isolated from other tenants. Security layers:

**Container-level isolation (gVisor)**:
GKE supports gVisor (runsc) as a container runtime. gVisor intercepts all Linux syscalls in userspace, preventing kernel exploits. Enable for workspace pods:

```yaml
# RuntimeClass for gVisor
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: gvisor
handler: runsc
---
# In pod spec:
spec:
  runtimeClassName: gvisor
```

gVisor adds ~5-10% CPU overhead and ~20% memory overhead. Worth it for tenant isolation.

**Alternatives to gVisor:**
- **Kata Containers**: Each pod runs in a lightweight VM. Stronger isolation than gVisor, higher overhead (~30% CPU). Better for security-sensitive workloads.
- **seccomp profiles**: Restrict syscalls without a kernel sandbox. Less protection than gVisor but near-zero overhead.
- **Firecracker** (via Fly.io or direct): True VM isolation, near-container startup times. Best security posture.

**Network isolation:**

```yaml
# NetworkPolicy: workspace pods can only talk to Cortado's agent
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: workspace-isolation
  namespace: cortado-workspaces
spec:
  podSelector:
    matchLabels:
      cortado/role: workspace
  policyTypes:
  - Ingress
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: cortado-system  # Only to Cortado's own services
    ports:
    - port: 443
  - to:  # Allow DNS
    - namespaceSelector: {}
    ports:
    - port: 53
      protocol: UDP
  # Allow internet egress (user code needs to download packages etc.)
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
        except:
        - 10.0.0.0/8    # Block access to GKE internal network
        - 172.16.0.0/12
        - 192.168.0.0/16
```

### 11.2 Authentication & Authorization

**JWT-based auth flow:**

```
1. Consuming IDE app authenticates its users (any auth provider)
2. IDE app calls Cortado API with its own API key + user identifier
3. Cortado issues a scoped JWT for that user's workspace session
4. Flutter package uses this JWT for all WebSocket/API calls
5. JWT encodes: {tenant_id, user_id, workspace_ids[], permissions[], exp}
6. Control plane and workspace agent both validate JWTs using shared JWKS
```

**mTLS for pod-to-control-plane communication:**

```go
// In workspace agent, load certificates from Kubernetes secret
cert, _ := tls.LoadX509KeyPair("/etc/cortado/tls.crt", "/etc/cortado/tls.key")
caCert, _ := os.ReadFile("/etc/cortado/ca.crt")
caPool := x509.NewCertPool()
caPool.AppendCertsFromPEM(caCert)

tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    RootCAs:      caPool,
    ServerName:   "cortado-gateway.cortado-system.svc",
}
```

Certificates are rotated by **cert-manager** (running in the control plane cluster) using Let's Encrypt for external-facing services and a private CA for pod-to-pod communication.

### 11.3 Secret Management

User secrets (API keys, tokens, SSH keys) needed inside workspaces:

**GCP Secret Manager**: Secrets are stored in Secret Manager. The workspace pod's service account has IAM permission to access specific secrets. The agent exposes secrets to shell sessions via environment variables or mounted files.

```bash
# User defines secrets in Cortado UI
# They're stored as: projects/{project}/secrets/ws-{id}-{name}

# Agent injects them at shell start:
export GITHUB_TOKEN=$(gcloud secrets versions access latest \
  --secret=ws-abc123-github-token 2>/dev/null)
```

**Alternative**: **Vault** (HashiCorp) with Kubernetes auth. The workspace agent authenticates to Vault using its Kubernetes service account token and retrieves secrets. More flexible than GCP Secret Manager but another service to operate.

---

## 12. Data Layer & Persistence

### 12.1 Primary Database

**Cloud Spanner** (recommended for Cortado's control plane data):

Workspaces, billing records, and usage metrics need:
- Global consistency (billing must be accurate across regions)
- High availability (99.999% SLA)
- Horizontal scalability

Cloud Spanner provides all three. Key tables:

```sql
CREATE TABLE tenants (
  tenant_id STRING(36) NOT NULL,
  name STRING(256),
  stripe_customer_id STRING(64),
  created_at TIMESTAMP,
  plan STRING(32),
) PRIMARY KEY (tenant_id);

CREATE TABLE workspaces (
  workspace_id STRING(36) NOT NULL,
  tenant_id STRING(36) NOT NULL,
  user_id STRING(36),
  status STRING(32),  -- STOPPED, STARTING, RUNNING, HIBERNATING
  image STRING(256),
  region STRING(64),
  pod_name STRING(128),
  created_at TIMESTAMP,
  last_active_at TIMESTAMP,
  cpu_vcpu FLOAT64,
  memory_gb FLOAT64,
  storage_gb INT64,
  gpu_type STRING(64),
  gpu_count INT64,
) PRIMARY KEY (tenant_id, workspace_id),
  INTERLEAVE IN PARENT tenants ON DELETE CASCADE;

CREATE TABLE usage_events (
  event_id STRING(36) NOT NULL,
  workspace_id STRING(36) NOT NULL,
  tenant_id STRING(36) NOT NULL,
  event_time TIMESTAMP NOT NULL,
  duration_seconds INT64,
  cpu_vcpu_seconds FLOAT64,
  memory_gb_seconds FLOAT64,
  gpu_seconds FLOAT64,
  storage_gb_seconds FLOAT64,
) PRIMARY KEY (tenant_id, workspace_id, event_time DESC);
```

**Alternative**: **CockroachDB** — PostgreSQL-compatible, distributed. Easier to migrate from existing PostgreSQL codebases. Lower operational complexity than Spanner. Good if the team is more comfortable with SQL.

**Alternative for simpler early stages**: **Cloud SQL** (PostgreSQL on GCP) with read replicas. Not globally distributed but much simpler. Switch to Spanner/CockroachDB when you need multi-region.

### 12.2 Caching Layer

**Dragonfly DB** (Redis-compatible, much higher throughput): Cache workspace metadata, JWT validation results, LSP response caches. Dragonfly uses a multi-threaded architecture (vs Redis's single thread) and is 10-25x more efficient per node.

**Alternative**: Memorystore for Redis (GCP managed Redis).

Key caching strategies:
- **Workspace state cache**: Cache `workspace.status` for 1 second. Avoids Spanner reads on every WebSocket ping.
- **JWT cache**: Cache valid JWTs (keyed by `jti` claim) to avoid JWKS fetch on every request.
- **LSP response cache**: Cache `textDocument/hover` responses keyed by `{file_hash}:{position}`. These are deterministic for unchanged files.

---

## 13. Observability & Metering

### 13.1 Metrics Architecture

**OpenTelemetry** is the standard for instrumentation. Both the Go agent and Dart package should emit OTel spans and metrics.

```go
// agent/otel.go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

func initTracing(ctx context.Context) {
    exporter, _ := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("cortado-otel-collector.cortado-system.svc:4317"),
        otlptracegrpc.WithInsecure(),
    )
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName("cortado-workspace-agent"),
            attribute.String("workspace.id", workspaceID),
        )),
    )
    otel.SetTracerProvider(tp)
}
```

OTel collector → **Google Cloud Monitoring** (for GCP-native alerting) and **Grafana Cloud** (for dashboards).

**Key metrics to instrument:**
- Terminal session latency (keystroke-to-render round trip)
- Workspace cold start duration (by image, region, resource class)
- LSP request latency (by language, request type)
- File sync lag (op created timestamp vs applied timestamp)
- AI completion acceptance rate (user accepted vs dismissed ghost text)
- Billing event ingestion lag

### 13.2 Logging

Structured logging using **zap** (Go) in the agent and `package:logging` (Dart) in the client. All logs written to stdout → collected by GKE's Fluent Bit → Google Cloud Logging.

Log correlation via trace IDs (from OTel context) enables end-to-end request tracing from client to container.

---

## 14. Multi-Tenancy & Isolation

### 14.1 Tenant Hierarchy

```
Cortado (you, the platform operator)
  └── Tenant (the IDE developer using Cortado's package)
       └── User (the IDE's end users)
            └── Workspace
```

The Cortado API key is per-tenant. Tenants configure their workspace images, resource limits, and billing plans. End users are authenticated by the tenant's IDE app; Cortado only needs a stable user identifier.

### 14.2 Namespace Isolation in GKE

Each tenant gets a dedicated Kubernetes namespace:

```
cortado-tenant-{tenantId}
```

With RBAC, ResourceQuota, and NetworkPolicy scoped per namespace. This gives billing isolation (easy to aggregate by namespace), resource limits (can enforce per-tenant maximums), and permission boundaries.

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-quota
  namespace: cortado-tenant-abc
spec:
  hard:
    pods: "100"
    requests.cpu: "200"
    requests.memory: "400Gi"
    persistentvolumeclaims: "100"
    requests.storage: "10Ti"
```

---

## 15. Bottlenecks, Risks & Mitigation

### 15.1 Terminal Latency

**Problem**: Round-trip latency for terminal keystrokes. 100ms+ is noticeably sluggish for developers.

**Latency budget (optimistic):**
- Flutter event handler: 2ms
- WebSocket framing + TLS: 1ms
- Network (client → Cloud Run, Singapore→US): 50-200ms (!)
- Cloud Run → GKE pod (mTLS): 5ms
- PTY write + shell + PTY read: 1ms
- Reverse path: 50-200ms

Total (bad case): 500ms for a 200ms-RTT client. This is terrible.

**Mitigations:**
1. **Regional deployment**: Deploy control plane and workspace pods in the same region as the user. Use GCP's multi-region architecture. A Singapore-based user should have workspaces in `asia-southeast1`. With 10ms client-to-region RTT, total terminal latency drops to ~30ms.
2. **Local echo**: Like SSH clients do — immediately echo printable ASCII characters without waiting for the server. Non-printable (arrow keys, ctrl sequences) wait for server confirmation. Flutter's xterm widget can enable this.
3. **WebSocket compression**: Enable `permessage-deflate` to reduce bandwidth for verbose output.
4. **Edge caching/routing**: Use Cloudflare or GCP Cloud CDN for TLS termination close to the user, then forward via QUIC or HTTP/2 multiplexing to the workspace region.

### 15.2 File Sync Consistency

**Problem**: The CRDT-based sync can't handle all conflict types. Particularly: binary files, database files (SQLite), and compile artifacts create noisy change events.

**Mitigation**: `.cortadoignore` patterns. Default ignores: `*.o`, `*.class`, `*.pyc`, `node_modules/`, `.dart_tool/`, `build/`, `.gradle/`, `Pods/`. Allow users to customize. Also, rate-limit change events: at most 100 ops/second per workspace (excess ops are buffered, not dropped).

### 15.3 Cold Start Latency

**Problem**: Scale-to-zero is great for cost, but cold starts take 5-15 seconds for complex workspace images.

**Mitigation**: CRIU checkpointing (described in section 4.4) reduces cold start to <500ms for most workspaces. Requires GKE's `--enable-checkpoint-restore` feature (now stable in GKE 1.30+). Additionally, pre-pull images to all nodes via `DaemonSet` image-puller.

### 15.4 LSP Memory Consumption

**Problem**: Language servers can use significant RAM. `rust-analyzer` typically uses 1-4GB for a medium Rust project. Multiple LSPs (polyglot workspace) can exhaust workspace memory.

**Mitigation**: Lazy LSP startup (start only when a file of that type is opened). Language server memory limits (enforce with cgroup limits). Remote LSP option: run heavy LSPs (rust-analyzer) as separate pods with shared access via Cortado's LSP gateway, rather than inside each workspace pod.

### 15.5 Billing Accuracy

**Problem**: Usage events can be lost (pod crash, network partition). Underbilling is a revenue loss; overbilling destroys trust.

**Mitigation**: Usage events are persisted to a Write-Ahead Log (WAL) inside the workspace agent before being published to Pub/Sub. The agent replays the WAL on startup after a crash. Pub/Sub messages have at-least-once delivery; idempotency keys (`event_id`) on the metering side prevent double-counting.

### 15.6 API Key Security

**Problem**: The `cortado-daemon` (local component) needs to communicate with GCP. If the API key is embedded in the package, it can be extracted.

**Mitigation**: The Flutter package contains NO secrets. It communicates with the consuming IDE's backend, which holds the Cortado API key server-side. The package talks to the *tenant's* backend (or directly to Cortado after the tenant's backend issues a scoped session token).

### 15.7 GPU Availability

**Problem**: GCP GPU availability is notoriously constrained. Workspace users requesting GPUs may not get them for minutes.

**Mitigation**: Maintain a small warm GPU pool (1-2 GPU nodes always running). Offer reservation API: users can pre-reserve GPU windows. Also support alternative GPU providers: **Lambda Labs** (A100/H100 on-demand), **Vast.ai** (marketplace), **RunPod** (consumer GPUs at lower cost). Build a GPU provider abstraction layer.

---

## 16. Alternatives Matrix

### 16.1 Compute Alternatives

| Platform | Pros | Cons | Best For |
|----------|------|------|----------|
| GKE Autopilot | Managed, GCP-integrated, VPA/CRIU | GCP lock-in, cost | Primary choice |
| Fly.io Machines | Firecracker isolation, anycast, fast | Smaller scale, less ML tooling | European users, security-sensitive |
| Modal Labs | Excellent for GPU/ML, simple API | No persistent terminals | AI inference only |
| Hetzner + k3s | Very cheap, EU data residency | Self-managed | Cost-optimized tier |
| AWS EKS + Fargate | Existing AWS customers | No CRIU, Fargate limitations | Multi-cloud |
| Azure ACI | Simple container instances | No GPU, scale-to-zero limitations | Simple workspaces |

### 16.2 Storage Alternatives

| Solution | Latency | Cost | Best For |
|----------|---------|------|----------|
| GCP PD-SSD | 0.1ms | $0.17/GB-mo | Active workspaces |
| GCS + FUSE v2 | 5-20ms | $0.02/GB-mo | Hibernated workspaces |
| Tigris | 10-50ms (global) | $0.02/GB-mo | Cross-region |
| Filestore Basic | 1ms (NFS) | $0.20/GB-mo | Multi-pod shared |
| Warp (Cloudflare R2 + FUSE) | 20-100ms | $0.015/GB-mo | Cost-sensitive |
| Litestream (SQLite replication) | N/A | Free | SQLite-heavy workloads |

### 16.3 Real-time Communication Alternatives

| Transport | Multiplexing | Browser Support | Latency |
|-----------|-------------|-----------------|---------|
| WebSocket (current) | Manual (custom framing) | Universal | Good |
| WebTransport (QUIC) | Native streams+datagrams | Chrome 97+, FF 114+ | Best |
| gRPC-Web | Streams (server→client) | Via proxy | Good |
| SSE + HTTP POST | One-way events only | Universal | Adequate |
| HTTP/2 (fetch API) | Multiplexed requests | Modern browsers | Good |

### 16.4 AI Model Alternatives

| Model | Context | Speed | Code Quality | Cost |
|-------|---------|-------|--------------|------|
| Claude Opus 4 | 200K | Slow | Excellent | High |
| Claude Sonnet 4.6 | 200K | Fast | Very good | Medium |
| Gemini 2.5 Pro | 1M | Fast | Very good | Medium |
| GPT-4o | 128K | Fast | Good | Medium |
| DeepSeek V3 (hosted) | 64K | Medium | Very good | Low |
| Qwen2.5-Coder (self) | 32K | Fastest | Good | COGS only |

### 16.5 Sync Protocol Alternatives

| Protocol | Conflict Handling | Complexity | Use Case |
|----------|------------------|------------|----------|
| Custom OT (current) | Per-file strategy | High | Optimized for code |
| Yjs (CRDT) | Automatic merge | Medium | Real-time collab |
| rsync protocol | None (overwrite) | Low | Simple one-way sync |
| Git (CRFS) | Merge commits | High | Version history |
| Litestream | N/A (SQLite only) | Low | DB files |
| Automerge v2 | Automatic merge | Medium | Documents |

---

## 17. Development Roadmap

### Phase 0: Foundation (Months 1-3)

**Goal**: Single-tenant, single-region workspace with terminal and basic file sync.

Deliverables:
- GKE Autopilot cluster in `us-central1`
- Workspace and control plane images published to Artifact Registry in `us-central1`
- `cortado-workspace-agent` Go binary (PTY management, gRPC server)
- Control plane: workspace CRUD API (Cloud Run, Go)
- Flutter package: `CortadoClient`, `CortadoTerminal` widget, basic `WorkspaceManager`
- WebSocket multiplexing protocol v1
- Basic billing event pipeline (Pub/Sub → BigQuery)
- JWT auth flow
- `cortado-daemon` (local daemon, macOS + Linux)

**Key technical risks at this stage**: PTY latency (validate <100ms round trip in target regions), scale-to-zero cold start (implement warm pool immediately).

### Phase 1: Filesystem & LSP (Months 4-6)

Deliverables:
- File sync protocol (tree-sitter chunking, bsdiff patches, vector clock conflict resolution)
- `CortadoFileTree` widget, file open/save APIs
- LSP gateway + workspace agent LSP manager
- First-class Dart/Flutter LSP support
- `CortadoEditor` widget (integrating LSP completions, diagnostics)
- Workspace hibernation + CRIU checkpointing

### Phase 2: AI & Indexing (Months 7-10)

Deliverables:
- Codebase indexing pipeline (tree-sitter → embeddings → Qdrant)
- Inline completion widget (streaming ghost text)
- AI chat panel widget
- RAG pipeline for contextual code Q&A
- Agent loop (tool use, file editing, command execution)

### Phase 3: Scale, GPU & Extensions (Months 11-15)

Deliverables:
- Multi-region deployment (US, EU, Asia-Pacific)
- GPU workspace support (NVIDIA T4/A100 MIG)
- Port forwarding gateway
- Chrome/Xvfb support
- VS Code extension compatibility layer (OpenVSX subset)
- Multi-tenant billing and Stripe integration
- Flutter package v1.0 public release

### Phase 4: Enterprise & Ecosystem (Months 16+)

Deliverables:
- SSO (SAML/OIDC) for enterprise tenants
- Private networking (VPN, private clusters)
- On-premise workspace support (Cortado behind customer's firewall)
- Compliance certifications (SOC 2 Type II)
- Cortado CLI tools
- Public documentation + developer portal

---

## Appendix: Key Technology References

**Compute & Orchestration**
- GKE Autopilot: https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-overview
- CRIU (Checkpoint/Restore): https://criu.org
- gVisor: https://gvisor.dev
- Fly.io Machines API: https://fly.io/docs/machines/

**File Sync & Storage**
- Yjs CRDT: https://yjs.dev
- GCS FUSE v2: https://cloud.google.com/storage/docs/cloud-storage-fuse
- restic: https://restic.net

**Protocols & Standards**
- Language Server Protocol spec: https://microsoft.github.io/language-server-protocol
- Debug Adapter Protocol: https://microsoft.github.io/debug-adapter-protocol
- WebTransport: https://w3c.github.io/webtransport
- SCIP (Sourcegraph): https://github.com/sourcegraph/scip

**AI & Vector Search**
- Vertex AI Vector Search: https://cloud.google.com/vertex-ai/docs/vector-search
- Qdrant: https://qdrant.tech
- OpenMeter: https://openmeter.io
- Voyage AI (code embeddings): https://docs.voyageai.com/docs/embeddings

**Flutter/Dart**
- xterm (pure Dart terminal): https://pub.dev/packages/xterm
- flutter_webrtc: https://pub.dev/packages/flutter_webrtc
- Riverpod: https://riverpod.dev

**Billing**
- Stripe Meters API: https://stripe.com/docs/billing/subscriptions/usage-based
- OpenMeter: https://openmeter.cloud
