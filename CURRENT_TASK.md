# CURRENT TASK

## Release · Feature · Task
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.1

## Status
IN PROGRESS

## What was done last session
Completed the now-superseded direct-OIDC Feature 8.4.1 slice by adding tenant auth-provider persistence and validation to the control plane.

## What was done this session
Replanned Feature 8.4 around first-party Cortado-managed auth plus personal and platform API-key modes. Updated the dedicated feature spec, technical report, release timeline, release pointer, and architecture decisions so the old tenant-managed OIDC exchange path is no longer the active roadmap.

## Remaining work this session
Task 8.4.1:
- add `POST /v1/sessions/exchange/firebase` for Firebase ID tokens issued by Cortado's own Firebase project
- auto-provision a stable Cortado user profile and default personal tenant on first login
- return Cortado `{access_token, refresh_token}` without requiring API-key bootstrap for browser apps
- remove tenant-managed OIDC assumptions from the active browser auth path

## Definition of done
- [ ] `POST /v1/sessions/exchange/firebase` exists for Cortado-managed browser auth
- [ ] first successful login provisions a stable Cortado user profile plus personal tenant
- [ ] browser apps can establish a normal Cortado session without bringing their own backend or API-key bootstrap
- [ ] relevant Go tests/builds pass for the new Task 8.4.1 slice

## Next task after this one
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.2
After the Firebase exchange slice lands, move to the Flutter package auth client and embedded auth surface for the zero-backend browser path.

## Blocked on / decisions needed
No active blockers.
