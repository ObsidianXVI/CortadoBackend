# demo_app

This app is the Flutter Web showcase harness for Cortado's embedded editor and
terminal story. It provisions one real Cortado workspace-agent image through the Cortado
control plane, opens the same workspace file through multiple editor packages,
and reuses one shared Cortado terminal for the shell/bootstrap flow. It now
also supports Firebase email/password sign-up and sign-in inside the app so you
can exchange directly into a Cortado session, mint personal API keys, and try
the platform-tenant / platform-key backend flow without leaving the demo UI.

## What It Demonstrates

- real session creation through `POST /v1/sessions`
- first-party Firebase session exchange through `POST /v1/sessions/exchange/firebase`
- Firebase-backed API key minting through `POST /v1/api-keys`
- platform-tenant bootstrap plus platform API key minting through `/v1/platform-tenants/...`
- real workspace provisioning/start/stop/delete against the current backend
- one shared Cortado workspace image:
  `us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:20260523-102947-workspace-tools`
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
CORTADO_FIREBASE_API_KEY=your-firebase-web-api-key
CORTADO_FIREBASE_AUTH_DOMAIN=your-project.firebaseapp.com
CORTADO_FIREBASE_PROJECT_ID=your-firebase-project
CORTADO_FIREBASE_APP_ID=1:1234567890:web:abcdef
CORTADO_FIREBASE_MESSAGING_SENDER_ID=1234567890
CORTADO_FIREBASE_STORAGE_BUCKET=your-project.firebasestorage.app
CORTADO_FIREBASE_MEASUREMENT_ID=
CORTADO_FIREBASE_EMAIL=demo@example.com
CORTADO_FIREBASE_PASSWORD=change-me
CORTADO_FIREBASE_DEV_TENANT_ID=demo-tenant
CORTADO_WORKSPACE_IMAGE=us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:20260523-102947-workspace-tools
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
- In `CORTADO_ENV=development`, the app can call the dev-only
  `/v1/dev/firebase/tenant-claim` route to self-assign the Firebase `tenant_id`
  custom claim before minting a Cortado API key.
- `CORTADO_FIREBASE_DEV_TENANT_ID` controls which dev tenant the app asks the
  control plane to assign. If omitted, the backend falls back to `demo-tenant`.

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

1. Use `Register User` or `Login` in the `Identity Bootstrap` panel.
2. For the browser-first path, press `Exchange Session` to turn the current Firebase sign-in into a normal Cortado session immediately.
3. For the headless personal-key path, press `Assign Dev Tenant` if needed, then `Mint Personal Key`, then `New Session`.
4. For the SaaS/backend path, use `Create Platform Tenant`, then `Mint Platform Key`. The session form is auto-filled with the new key; leave `Demo User ID` empty before pressing `New Session`.
5. Press `Provision Workspace` to create a real workspace using the Cortado
   workspace image.
6. Wait for the workspace to reach `RUNNING`.
7. Use the shared terminal to run the bootstrap commands shown in the UI:

   ```bash
   apt-get update
   apt-get install -y curl git unzip xz-utils zip libglu1-mesa
   git clone https://github.com/flutter/flutter.git -b stable /opt/flutter
   export PATH="/opt/flutter/bin:$PATH"
   flutter doctor
   flutter create --platforms=web .
   ```

8. Press `Load File` to open `lib/main.dart`.
9. Switch between the editor package pages and edit the same file.
10. Press `Save File` to write the current draft back to the workspace.

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

- `demo_app` now relies entirely on the Cortado Flutter package for workspace
  create/get/start/stop/delete plus file load/save behavior.
- Firebase sign-up/sign-in stays inside the demo app, but the tenant binding
  still comes from the Firebase `tenant_id` custom claim required by the
  control-plane API-key routes.
- The demo now exposes all three auth shapes the backend supports:
  - direct first-party Firebase session exchange,
  - personal API keys for headless user flows,
  - platform tenants plus platform API keys for SaaS backend flows.
- In development, the demo can assign that claim itself through the
  control-plane dev bootstrap route before retrying API-key minting.
- The local Cortado Flutter package now aligns on
  `freezed_annotation/freezed: ^3.1.0`, so the Monaco integration no longer
  needs a demo-local dependency override.
- `demo_app/web/index.html` still carries the xterm.js assets used by
  `CortadoTerminal`.

## Validation

```bash
cd demo_app
/home/OBSiDIAN/tools/flutter/bin/flutter analyze
/home/OBSiDIAN/tools/flutter/bin/flutter test
```
