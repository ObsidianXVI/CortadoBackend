# demo_app

This app is the manual smoke harness for the Flutter terminal package work in
Feature 1.4.

## Run

1. Start the app in Chrome:

   ```bash
   /home/OBSiDIAN/tools/flutter/bin/flutter run -d chrome
   ```

2. Provide a control-plane base URL and workspace ID in the form fields, or
   launch with query parameters:

   ```text
   ?baseUrl=https://control-plane.example.run.app&workspaceId=ws-123&shell=/bin/bash
   ```

3. Press `Connect` to open the terminal widget.

## Smoke checklist

- Run `echo hello_v0_1` and verify the echoed line renders in the terminal.
- Run `vim` and verify the full-screen TUI redraw works.
- Run `python3` and verify the REPL prompt accepts input without broken echo.
- Drag the resize handle below the terminal, then run `tput cols` to confirm
  the shell sees the new width.
- Measure keystroke round-trip timing in Chrome DevTools via the WebSocket
  frames inspector.

## Notes

- `demo_app/web/index.html` supplies the xterm.js CSS/JS includes required by
  the package on Flutter Web.
- The package still requires a live workspace ID from the broader Cortado
  environment; this app only provides the browser-side smoke harness.

### Editor bridge bundle (CodeMirror 6)

- The host page now also loads `cortado_editor.js`, a locally bundled
  CodeMirror 6 bridge used by the Flutter editor widget integration.
- Build/update the bundle when developing:

  ```bash
  cd demo_app/web
  npm install   # or: npm ci
  npm run build # outputs cortado_editor.js
  ```

- Included language modes: JavaScript/TypeScript, JSON, Python, Go, YAML.
  Dart currently falls back to plain text until we add a stable CM6 Dart mode.
- The bundle exposes a global `window.CortadoEditor` object with methods used
  by the Flutter `HtmlElementView` side:
  - `init(container, id, languageOrOptions, onChangeHash?, onSave?)`
  - `setContent(id, text, preserveSelection?)`
  - `getContent(id)` / `setLanguage(id, lang)` / `dispose(id)`

#### LSP completion bridge (Task 4.2.2)

- Dart registers a global completion request handler: window._cortadoLSPRequest
- JS calls it from a CodeMirror completion source after a 150ms debounce and resolves results when Dart calls window._cortadoLSPResult(requestId, items).
- Stale results are ignored if the cursor moved before the async result arrives.
- Request params shape:
  _cortadoLSPRequest({ editorId: string, requestId: number, position: { line: number; character: number }, lineText?: string, prefix?: string })
- Dart returns via: _cortadoLSPResult(requestId: number, items: Completion[])
  where Completion is a CodeMirror 6 completion entry.
