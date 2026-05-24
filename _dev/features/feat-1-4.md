## Feature 1.4 — Flutter Package: Terminal Widget

### Task 1.4.1 — Package scaffold and WebSocket client
**What to do:**
- `flutter create --template=package cortado` inside `flutter/`
- Add to `pubspec.yaml`:
  ```yaml
  dependencies:
    web_socket_channel: ^3.0.0
    riverpod: ^2.5.0
    freezed_annotation: ^2.4.0
  dev_dependencies:
    build_runner: ^2.4.0
    freezed: ^2.4.0
    riverpod_generator: ^2.4.0
  ```
- Implement `CortadoClient`:
  ```dart
  class CortadoClient {
    final String baseUrl;
    // In v0.1: always sends X-Cortado-Dev-Token: dev-bypass
    // In v0.2: sends Authorization: Bearer {jwt}
    final String _devToken = 'dev-bypass';

    late WebSocketChannel _ws;
    final _frames = StreamController<MuxFrame>.broadcast();

    Future<void> connect(String workspaceId) async {
      final uri = Uri.parse('$baseUrl/v1/workspaces/$workspaceId/connect')
          .replace(scheme: 'wss');
      _ws = WebSocketChannel.connect(uri,
          protocols: ['cortado-v1'],
          headers: {'X-Cortado-Dev-Token': _devToken});
      await _ws.ready;
      _ws.stream.listen(_onFrame, onError: _onError, onDone: _onDone);
    }

    void _onFrame(dynamic raw) {
      final bytes = raw as Uint8List;
      final frame = MuxFrame.decode(bytes);
      _frames.add(frame);
    }

    void sendFrame(int channelId, int msgType, Uint8List payload) {
      _ws.sink.add(MuxFrame(channelId, msgType, payload).encode());
    }

    Stream<MuxFrame> framesForChannel(int channelId) =>
        _frames.stream.where((f) => f.channelId == channelId);
  }
  ```

**Key detail**: `WebSocketChannel.connect` on Flutter Web uses the browser's native WebSocket API, which does not support custom HTTP headers (the `headers` parameter is silently ignored on web). Passing `X-Cortado-Dev-Token` as a header will not work in the browser. For v0.1 in dev, pass the token as a query parameter instead: `?dev_token=dev-bypass`. The control plane middleware checks for either the header (for non-browser clients) or the query param (for browser clients).

**Challenge**: The `web_socket_channel` package throws `WebSocketChannelException` (not standard Dart `Exception`) on connection failure. Both `_ws.ready` (a Future) and `_ws.stream` can produce errors independently. Wrap `await _ws.ready` in try/catch AND add `onError` to the stream listener — missing either path gives silent connection failures that appear as a frozen terminal with no error message.

---

### Task 1.4.2 — Mux frame codec in Dart
**What to do:**
- Implement `MuxFrame`:
  ```dart
  class MuxFrame {
    final int channelId;  // uint16
    final int msgType;    // uint8
    final Uint8List payload;

    const MuxFrame(this.channelId, this.msgType, this.payload);

    Uint8List encode() {
      final bd = ByteData(7 + payload.length);
      bd.setUint16(0, channelId, Endian.big);
      bd.setUint8(2, msgType);
      bd.setUint32(3, payload.length, Endian.big);
      final out = bd.buffer.asUint8List();
      out.setRange(7, 7 + payload.length, payload);
      return out;
    }

    static MuxFrame decode(Uint8List bytes) {
      assert(bytes.length >= 7, 'Frame too short');
      final bd = ByteData.sublistView(bytes);
      final channelId = bd.getUint16(0, Endian.big);
      final msgType = bd.getUint8(2);
      final payloadLen = bd.getUint32(3, Endian.big);
      assert(bytes.length == 7 + payloadLen, 'Frame length mismatch');
      return MuxFrame(channelId, msgType,
          Uint8List.sublistView(bytes, 7, 7 + payloadLen));
    }
  }
  ```
- Endianness must match the Go side exactly (`binary.BigEndian`). Write a cross-language test: produce a known frame in Go (`channel=0x0001, type=0x01, payload=[0x41,0x42,0x43]`), hardcode its bytes, and assert that Dart's `decode` produces the same values.

**Challenge**: `Uint8List.sublistView(bytes, 7)` creates a *view* (no copy). Operations on this view affect the original buffer. If the caller modifies `bytes` after calling `decode`, the payload inside the returned `MuxFrame` is silently corrupted. For the terminal hot path (high-frequency frames), views are worth the risk. Add a comment: `// View, not copy — do not mutate source bytes after decode.`

---

### Task 1.4.3 — Terminal widget (xterm.js via HtmlElementView)
**What to do:**
- Download `xterm.js` v5.x and `xterm-addon-fit.js`, place in `flutter/web/js/`.
- Add to `flutter/web/index.html`:
  ```html
  <link rel="stylesheet" href="js/xterm.css"/>
  <script src="js/xterm.js"></script>
  <script src="js/xterm-addon-fit.js"></script>
  <script src="js/cortado_xterm.js"></script>
  ```
- Write `flutter/web/js/cortado_xterm.js`:
  ```javascript
  window.CortadoXterm = {
    _terms: {},
    init(container, id, onDataCallback) {
      const term = new Terminal({
        fontFamily: '"JetBrains Mono", "Fira Code", monospace',
        fontSize: 14,
        cursorBlink: true,
      });
      const fit = new FitAddon.FitAddon();
      term.loadAddon(fit);
      term.open(container);
      fit.fit();
      this._terms[id] = { term, fit };
      term.onData(data => onDataCallback(id, data));
      new ResizeObserver(() => fit.fit()).observe(container);
    },
    write(id, data) { this._terms[id]?.term.write(data); },
    getSize(id) {
      const t = this._terms[id]?.term;
      return t ? { cols: t.cols, rows: t.rows } : null;
    },
    dispose(id) {
      this._terms[id]?.term.dispose();
      delete this._terms[id];
    }
  };
  ```
- Implement `CortadoTerminal` widget using `HtmlElementView` and `dart:js_interop` to call the JS bridge.

**Key detail**: `ResizeObserver` on the container element automatically calls `fit.fit()` when the widget is resized by Flutter layout. The fit addon calculates the correct `cols` and `rows` from the container's pixel dimensions, then you must send a resize frame to the server. Wire: `ResizeObserver callback → JS calls Dart → Dart sends resize MuxFrame → server resizes PTY`.

**Challenge**: Flutter Web's CanvasKit renderer renders Flutter widgets on a canvas, and `HtmlElementView` creates a "platform view hole" in that canvas. In Flutter 3.19+ this generally works, but the hole's z-ordering can conflict with Flutter dialogs and overlays that render *above* the canvas. If you render a Flutter dropdown or tooltip over the terminal, it disappears behind the xterm canvas. Workaround: use Flutter's `Overlay` system for all IDE overlays (file tree popups, completion dropdowns) rather than regular `Stack` children.

---

### Task 1.4.4 — End-to-end smoke test
**What to do:**
- Manually verify the full chain: Flutter Web (Chrome) → WebSocket → Cloud Run control plane → gRPC → GKE workspace pod → PTY → shell.
- Type `echo hello_v0_1` in the terminal widget, see the output.
- Type `vim` — verify that vim's TUI renders correctly (tests proper PTY/VT100 handling).
- Type `python3` — verify an interactive REPL works (tests that the PTY handles input prompts correctly without echoing back incorrectly).
- Test resize: drag the terminal widget to a different size, run `tput cols` in the shell — should reflect the new width.
- Record the round-trip latency: type a character, measure time until it appears back on screen. Target: <150ms from `us-central1` with a Singapore client.

**Challenge**: The latency target may not be met from Singapore to `us-central1` (~200ms RTT). If it's consistently above 200ms, note it as a known issue for v0.1 and plan regional deployment (`asia-southeast1`) for v0.2. The latency measurement itself is done with Chrome DevTools Network tab (WebSocket frames tab shows send/receive timestamps per frame).

---

---
