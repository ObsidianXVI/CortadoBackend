# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.4 (Real JWT Authentication) → Task 2.4.2

## Status
IN PROGRESS

## What was done last session
Completed Task 2.4.1 by adding JWT session issuance and JWKS exposure in the control plane, wiring Firestore-backed API-key and refresh-token persistence, adding Redis-compatible API-key validation caching, and extending Terraform with Secret Manager and Memorystore resources plus Cloud Run runtime injection for the signing key and cache address.

## What was done this session
Loaded the Task 2.4.2 spec and advanced the release/task pointers so the JWT-first middleware chain, dev-bypass fallback, and WebSocket token handling can be implemented against the documented acceptance criteria instead of guessed requirements.

## Remaining work this session
Replace the dev-only middleware with JWT validation backed by JWKS, preserve the dev-bypass path only in development, accept JWTs from the WebSocket query string for browser upgrades, and cover the new auth chain with tests plus the required Go verification.

## Definition of done
- [ ] The control plane validates `Authorization: Bearer {jwt}` using JWKS-backed key resolution
- [ ] The middleware falls back to `X-Cortado-Dev-Token: dev-bypass` only when `CORTADO_ENV=development`
- [ ] Valid JWTs inject `tenant_id` and `user_id` request context from the `tid` and `sub` claims
- [ ] Browser WebSocket upgrades can authenticate with `?token={jwt}` on `/v1/workspaces/{id}/connect`
- [ ] Invalid or missing JWTs return `401 Unauthorized` on protected HTTP and WebSocket entry points
- [ ] `cd control-plane && CGO_ENABLED=0 go build ./...` passes
- [ ] `cd control-plane && go test ./...` passes

## Next task after this one
Task 2.4.3 — JWT refresh in Flutter client + tag v0.2
See _dev/docs/release_timeline.md §Feature 2.4 Task 2.4.3 for full spec

## Blocked on / decisions needed
None.
