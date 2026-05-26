# CURRENT TASK

## Release · Feature · Task
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.4

## Status
COMPLETE

## What was done last session
Completed Task 8.4.3 by allowing `/v1/api-keys` to authenticate with a normal Cortado session first and adding personal API key management for headless reuse after first-party sign-in.

## What was done this session
Completed Task 8.4.4 by introducing platform-tenant bootstrap plus platform API key management routes behind normal Cortado user sessions, extending `POST /v1/sessions` and JWT claims so platform API keys can mint platform-scoped Cortado sessions without a `user_id`, and documenting the split between personal first-party auth and SaaS backend platform auth.

## Remaining work this session
Feature 8.4 is complete. Resume the previously deferred v0.7 preview slice at Feature 7.1 Task 7.1.3.

## Definition of done
- [x] platform tenant records exist separately from first-party personal tenants
- [x] authenticated Cortado users can bootstrap, list, and manage platform API keys for owned platform tenants
- [x] platform API keys can mint Cortado sessions without a headed login flow or a `user_id`
- [x] docs and tests position platform API keys as the SaaS/backend integration path while first-party Firebase auth remains the default browser path

## Next task after this one
v0.7 → Feature 7.1 (Port Forward Gateway) → Task 7.1.3
Resume the deferred Flutter web preview slice now that the v0.8 auth feature is complete.

## Blocked on / decisions needed
No active blockers.
