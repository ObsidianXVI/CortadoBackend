# CURRENT TASK

## Release · Feature · Task
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.2

## Status
COMPLETE

## What was done last session
Completed Task 8.4.1 by adding the control-plane Firebase session exchange endpoint plus stable first-party user and personal-tenant provisioning.

## What was done this session
Implemented the Flutter-side first-party auth slice by adding `CortadoFirebaseAuthClient`, a drop-in `CortadoEmbeddedAuth` widget, Firebase-to-Cortado session exchange support inside `CortadoAuthSession`, package README/docs updates, and package test coverage for the zero-backend browser auth path.

## Remaining work this session
Task 8.4.2 is complete. Resume with Task 8.4.3 for personal API key issuance and management after one-time Firebase auth.

## Definition of done
- [x] package-level auth helpers exist for Firebase email/password and Google sign-in
- [x] package exposes a low-friction embedded auth surface for host Flutter web apps
- [x] Firebase sign-in automatically exchanges into a reusable Cortado session for existing workspace/client layers
- [x] Flutter package docs and tests cover the new zero-backend auth path

## Next task after this one
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.3
After the Flutter auth surface lands, add personal API key issuance and management for headless reuse after one-time first-party auth.

## Blocked on / decisions needed
No active blockers.
