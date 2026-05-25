# demo_app

This app is the Flutter Web showcase harness for Cortado's embedded editor and
terminal story. It provisions one real Ubuntu workspace through the Cortado
control plane, opens the same workspace file through multiple editor packages,
and reuses one shared Cortado terminal for the shell/bootstrap flow.

## What It Demonstrates

- real session creation through `POST /v1/sessions`
- real workspace provisioning/start/stop/delete against the current backend
- one shared Ubuntu workspace image: `ubuntu:24.04`
- manual bootstrap of Flutter inside the workspace terminal
- the same `lib/main.dart` file edited through:
  - `flutter_monaco`
  - `flutter_code_editor`
  - `code_forge_web` for the CodeForge web page
  - `lite_code_editor`

## Local Env Setup

Create a local `.env` file in `demo_app/` using `.env.example` as the base:

```dotenv
CORTADO_BASE_URL=http://localhost:8080
CORTADO_DEMO_API_KEY=your-local-demo-key
CORTADO_DEMO_USER_ID=demo-user
CORTADO_WORKSPACE_IMAGE=ubuntu:24.04
CORTADO_WORKSPACE_CPU=1
CORTADO_WORKSPACE_MEMORY_GB=2
CORTADO_FILE_PATH=lib/main.dart
CORTADO_SHELL=/bin/bash
```

Notes:

- `.env` is intentionally ignored by git.
- this is only safe for the agreed localhost-only recording workflow
- the API key is still bundled client-side at runtime and must remain a narrow,
  low-scope demo credential

## Run

```bash
cd demo_app
/home/OBSiDIAN/tools/flutter/bin/flutter run -d chrome
```

Optional query params can override the env-backed defaults:

```text
?baseUrl=http://localhost:8080&workspaceId=ws-123&filePath=lib/main.dart
```

## Demo Flow

1. Press `Authenticate`.
2. Press `Provision Workspace` to create a real workspace using `ubuntu:24.04`.
3. Wait for the workspace to reach `RUNNING`.
4. Use the shared terminal to run the bootstrap commands shown in the UI:

   ```bash
   apt-get update
   apt-get install -y curl git unzip xz-utils zip libglu1-mesa
   git clone https://github.com/flutter/flutter.git -b stable /opt/flutter
   export PATH="/opt/flutter/bin:$PATH"
   flutter doctor
   flutter create --platforms=web .
   ```

5. Press `Load File` to open `lib/main.dart`.
6. Switch between the editor package pages and edit the same file.
7. Press `Save File` to write the current draft back to the workspace.

## Package Notes

### `flutter_monaco`

- Best "familiar IDE" feel in the demo.
- Good page for showing Monaco-backed syntax editing with Cortado doing file
  persistence and terminal transport.

### `flutter_code_editor`

- Pure-Flutter editor with highlighting, folding, and gutter support.
- Good middle ground between simple text editing and richer IDE behavior.

### `code_forge_web`

- Used instead of `code_forge` because the upstream `code_forge` package does
  not support Flutter Web.
- This page demonstrates the CodeForge family in the required web-only demo app.

### `lite_code_editor`

- Lowest-overhead page in the demo.
- Good for showing the minimal "edit-save-verify in terminal" path.

## Technical Notes

- `demo_app` uses a local `dependency_overrides` entry for
  `freezed_annotation: ^3.1.0` so current `flutter_monaco` can coexist with the
  current local Cortado Flutter package.
- The Cortado Flutter package does not currently expose public workspace
  `get/list/delete` helpers, so this app calls those control-plane endpoints
  directly where needed.
- `demo_app/web/index.html` still carries the xterm.js assets used by
  `CortadoTerminal`.

## Validation

```bash
cd demo_app
/home/OBSiDIAN/tools/flutter/bin/flutter analyze
/home/OBSiDIAN/tools/flutter/bin/flutter test
```
