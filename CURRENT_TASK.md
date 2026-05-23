# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.4 (Real JWT Authentication) → Task 2.4.3

## Status
BLOCKED

## What was done last session
Implemented the repo-side code for Task 2.4.3 by adding the control-plane refresh endpoint and Firestore-backed refresh-token lookup path, introducing a shared Flutter auth-session object that stores JWT expiry and refresh metadata, switching Flutter HTTP and WebSocket auth to bearer-token usage with development-only fallback behavior retained, and verifying the integrated Go and Flutter test/analyze/build matrix.

## What was done this session
Ran a shortened expiry-based Flutter refresh smoke that proves the timer-driven JWT rotation feeds both HTTP bearer auth and browser WebSocket `?token=` auth paths, re-ran `cd flutter && flutter test` plus `flutter analyze`, created the local release tag `v0.2.0`, and then hit a GitHub authentication failure when attempting `git push --tags` over the configured HTTPS remote.

## Remaining work this session
Push the already-created local tag `v0.2.0` once GitHub credentials are available for the `origin` remote, then advance the active task pointer to v0.3 Feature 3.1 Task 3.1.1.

## Definition of done
- [x] Shared Flutter auth state stores the access JWT, refresh token, and parsed `exp` claim for `CortadoClient` and `WorkspaceManager`
- [x] The client refreshes the access JWT five minutes before expiry via `POST /v1/sessions/refresh`
- [x] The client also refreshes synchronously when a request or connection sees an expired or suspended-session JWT
- [x] HTTP requests use `Authorization: Bearer {jwt}` and browser WebSocket upgrades use `?token={jwt}`
- [x] Development fallback auth remains available only when `CORTADO_ENV=development`
- [x] Any new control-plane refresh flow is covered by tests and `cd control-plane && go test ./...` plus `CGO_ENABLED=0 go build ./...` pass
- [x] `cd flutter && flutter test` passes
- [x] `cd flutter && flutter analyze` passes
- [x] End-to-end shortened expiry-based refresh verification is completed
- [ ] `git push --tags` succeeds for `v0.2.0`

## Next task after this one
v0.3 → Feature 3.1 (File API) → Task 3.1.1 — Proto: filesystem operations
See _dev/features/feat-3-1.md for full spec

## Blocked on / decisions needed
GitHub authentication is not configured in this environment for the HTTPS `origin` remote, so `git push --tags` fails before `v0.2.0` can be published.
