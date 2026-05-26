# CURRENT TASK

## Release · Feature · Task
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.1

## Status
COMPLETE

## What was done last session
Completed the now-superseded direct-OIDC Feature 8.4.1 slice by adding tenant auth-provider persistence and validation to the control plane.

## What was done this session
Implemented the Firebase browser-session exchange slice in the control plane by adding `POST /v1/sessions/exchange/firebase`, wiring Firebase token verification into the session issuer, auto-provisioning stable first-party Cortado user and personal-tenant records on first login, and extending Firestore auth storage plus test coverage for the new flow.

## Remaining work this session
Task 8.4.1 is complete. Resume with Task 8.4.2 for the Flutter package auth client and embedded auth surface.

## Definition of done
- [x] `POST /v1/sessions/exchange/firebase` exists for Cortado-managed browser auth
- [x] first successful login provisions a stable Cortado user profile plus personal tenant
- [x] browser apps can establish a normal Cortado session without bringing their own backend or API-key bootstrap
- [x] relevant Go tests/builds pass for the new Task 8.4.1 slice

## Next task after this one
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.2
After the Firebase exchange slice lands, move to the Flutter package auth client and embedded auth surface for the zero-backend browser path.

## Blocked on / decisions needed
No active blockers.
