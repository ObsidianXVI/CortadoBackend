# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.4 (Real JWT Authentication) → Task 2.4.3

## Status
IN PROGRESS

## What was done last session
Completed Task 2.4.2 by replacing the control-plane dev-only auth gate with JWKS-backed JWT validation, keeping the dev-bypass fallback gated to development only, accepting browser WebSocket JWTs from `?token=...`, injecting `tenant_id` and `user_id` from claims, and extending the control-plane test suite so the protected HTTP and WebSocket entry points now verify real bearer tokens.

## What was done this session
Loaded the Task 2.4.3 spec and audited the current Flutter client plus control-plane session surface; the client still authenticates with dev-bypass headers/query params and the control plane does not yet expose `/v1/sessions/refresh`, so the refresh work is now scoped against the actual code rather than the release outline alone.

## Remaining work this session
Add the refresh-token exchange on the control plane if needed by the client contract, teach `CortadoClient` to store JWT metadata and refresh tokens, schedule refresh five minutes before expiry while also refreshing synchronously when a request sees an expired token, replace dev-bypass auth with bearer auth for HTTP and WebSocket connections while preserving the development fallback, and cover the new behavior with Flutter plus control-plane verification before tagging `v0.2.0`.

## Definition of done
- [ ] `CortadoClient` stores the access JWT, refresh token, and parsed `exp` claim
- [ ] The client refreshes the access JWT five minutes before expiry via `POST /v1/sessions/refresh`
- [ ] The client also refreshes synchronously when a request or connection sees an expired or suspended-session JWT
- [ ] HTTP requests use `Authorization: Bearer {jwt}` and browser WebSocket upgrades use `?token={jwt}`
- [ ] Development fallback auth remains available only when `CORTADO_ENV=development`
- [ ] Any new control-plane refresh flow is covered by tests and `cd control-plane && go test ./...` plus `CGO_ENABLED=0 go build ./...` pass
- [ ] `cd flutter && flutter test` passes
- [ ] `cd flutter && flutter analyze` passes

## Next task after this one
v0.3 → Feature 3.1 (File API) → Task 3.1.1 — Proto: filesystem operations
See _dev/docs/release_timeline.md §Feature 3.1 Task 3.1.1 for full spec

## Blocked on / decisions needed
None.
