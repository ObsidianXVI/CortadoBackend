# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.4 (Real JWT Authentication) → Task 2.4.3

## Status
IN PROGRESS

## What was done last session
Completed Task 2.4.2 by replacing the control-plane dev-only auth gate with JWKS-backed JWT validation, keeping the dev-bypass fallback gated to development only, accepting browser WebSocket JWTs from `?token=...`, injecting `tenant_id` and `user_id` from claims, and extending the control-plane test suite so the protected HTTP and WebSocket entry points now verify real bearer tokens.

## What was done this session
Implemented the full code path for Task 2.4.3 by adding `POST /v1/sessions/refresh` on the control plane, reusing persisted opaque refresh tokens to mint fresh access JWTs, introducing a shared Flutter `CortadoAuthSession` that stores access/refresh tokens plus parsed JWT expiry, switching Flutter HTTP requests to bearer headers and browser WebSocket upgrades to `?token=...`, and re-running `cd control-plane && go test ./...`, `CGO_ENABLED=0 go build ./...`, `cd flutter && flutter test`, and `cd flutter && flutter analyze`.

## Remaining work this session
Decide and execute the final verification/release steps: either run the literal 9-hour real-JWT soak from the spec or a user-approved shortened expiry-based soak, then create and push the `v0.2.0` tag if that release verification is considered sufficient.

## Definition of done
- [x] Shared Flutter auth state stores the access JWT, refresh token, and parsed `exp` claim for `CortadoClient` and `WorkspaceManager`
- [x] The client refreshes the access JWT five minutes before expiry via `POST /v1/sessions/refresh`
- [x] The client also refreshes synchronously when a request or connection sees an expired or suspended-session JWT
- [x] HTTP requests use `Authorization: Bearer {jwt}` and browser WebSocket upgrades use `?token={jwt}`
- [x] Development fallback auth remains available only when `CORTADO_ENV=development`
- [x] Any new control-plane refresh flow is covered by tests and `cd control-plane && go test ./...` plus `CGO_ENABLED=0 go build ./...` pass
- [x] `cd flutter && flutter test` passes
- [x] `cd flutter && flutter analyze` passes
- [ ] End-to-end real-JWT refresh verification is completed with either the literal 9-hour soak or a user-approved shortened equivalent
- [ ] `git tag v0.2.0 && git push --tags` is executed

## Next task after this one
v0.3 → Feature 3.1 (File API) → Task 3.1.1 — Proto: filesystem operations
See _dev/docs/release_timeline.md §Feature 3.1 Task 3.1.1 for full spec

## Blocked on / decisions needed
Need user confirmation on whether to do a shortened expiry-based soak instead of a literal 9-hour verification, and whether to create/push the `v0.2.0` tag in this session.
