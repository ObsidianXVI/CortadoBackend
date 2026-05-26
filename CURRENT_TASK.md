# CURRENT TASK

## Release · Feature · Task
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.3

## Status
COMPLETE

## What was done last session
Completed Task 8.4.2 by adding the Flutter-side first-party auth client, embedded auth widget, and automatic Firebase-to-Cortado session exchange support.

## What was done this session
Completed Task 8.4.3 by allowing `/v1/api-keys` to authenticate with a normal Cortado session first and fall back to Firebase token verification for compatibility, adding Flutter-side personal API key issuance/list/revoke helpers for the headless reuse path, and updating the auth/docs coverage around personal API key management after one-time first-party sign-in.

## Remaining work this session
Task 8.4.3 is complete. Resume with Task 8.4.4 for platform API keys for SaaS backends.

## Definition of done
- [x] authenticated Cortado users can mint, list, and revoke long-lived personal API keys after one-time sign-in
- [x] API keys are returned raw only at issuance time while the backend continues storing only hashed key material
- [x] the Flutter package exposes personal API key management for CLI/local-tooling bootstrap after first-party auth
- [x] docs and tests position personal API keys as a headless reuse path rather than the default browser credential

## Next task after this one
v0.8 → Feature 8.4 (Cortado-Managed Auth + API Key Modes) → Task 8.4.4
After personal API keys are in place, add platform API keys for SaaS backends that authenticate as one Cortado entity without headed user sign-in flows.

## Blocked on / decisions needed
No active blockers.
