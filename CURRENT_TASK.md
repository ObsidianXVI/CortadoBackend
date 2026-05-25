# CURRENT TASK

## Release · Feature · Task
v0.8 → Feature 8.4 (Direct OIDC Session Exchange) → Task 8.4.1

## Status
COMPLETE

## What was done last session
Drafted the first production no-server auth path as Feature 8.4 by adding the OIDC browser token-exchange architecture to the technical report, inserting the work into the release timeline, and creating the dedicated feature spec at `_dev/features/feat-8-4.md`.

## What was done this session
Completed Task 8.4.1 by adding a Firestore-backed tenant metadata layer for auth-provider configuration, validating OIDC discovery or explicit issuer/JWKS metadata plus audiences, signing algorithms, user-claim mapping, and claim requirements on write, and exposing Firebase-protected tenant self-service CRUD endpoints at `/v1/tenant/auth-provider` for the follow-on session-exchange work.

## Remaining work this session
None. Task 8.4.1 is complete and the next queued task remains the deferred v0.7 Flutter web preview slice.

## Definition of done
- [x] tenant auth-provider configuration exists with strict validation for discovery/JWKS and audience settings
- [x] control plane exposes tenant-scoped CRUD endpoints for provider config
- [x] provider metadata is persisted in the tenant configuration layer and reused by follow-on exchange work
- [x] relevant Go tests/builds pass for the Task 8.4.1 slice

## Next task after this one
v0.7 → Feature 7.1 (Port Forward Gateway) → Task 7.1.3
Resume the deferred Flutter web preview work captured in `_dev/features/feat-7-1.md` immediately after the Feature 8.4 auth slice unless redirected again.

## Blocked on / decisions needed
No active blockers.
