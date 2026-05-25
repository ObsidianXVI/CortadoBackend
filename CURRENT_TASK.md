# CURRENT TASK

## Release · Feature · Task
v0.8 → Feature 8.4 (Direct OIDC Session Exchange) → Task 8.4.1

## Status
IN PROGRESS

## What was done last session
Drafted the first production no-server auth path as Feature 8.4 by adding the OIDC browser token-exchange architecture to the technical report, inserting the work into the release timeline, and creating the dedicated feature spec at `_dev/features/feat-8-4.md`.

## What was done this session
Repointed the active task tracker to the newly approved OIDC exchange work. Preserved the previously active v0.7 port-forward preview task as the explicit next task so it can resume immediately after this auth slice without getting skipped.

## Remaining work this session
Task 8.4.1:
- add tenant-scoped auth-provider configuration for OIDC discovery or explicit issuer/JWKS metadata
- validate allowed audiences, signing algorithms, user-claim mapping, and optional claim requirements on write
- expose tenant self-service CRUD endpoints for auth-provider configuration
- persist the provider config alongside tenant metadata for later session exchange use

## Definition of done
- [ ] tenant auth-provider configuration exists with strict validation for discovery/JWKS and audience settings
- [ ] control plane exposes tenant-scoped CRUD endpoints for provider config
- [ ] provider metadata is persisted in the tenant configuration layer and reused by follow-on exchange work
- [ ] relevant Go tests/builds pass for the Task 8.4.1 slice

## Next task after this one
v0.7 → Feature 7.1 (Port Forward Gateway) → Task 7.1.3
Resume the deferred Flutter web preview work captured in `_dev/features/feat-7-1.md` immediately after the Feature 8.4 auth slice unless redirected again.

## Blocked on / decisions needed
No active blockers.
