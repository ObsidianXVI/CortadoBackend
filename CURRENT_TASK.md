# CURRENT TASK

## Release · Feature · Task
v0.7 → Feature 7.1 (Port Forward Gateway) → Task 7.1.3

## Status
IN PROGRESS

## What was done last session
Completed Task 7.1.2 by adding the dedicated `cortado-portforward` Cloud Run gateway binary and Dockerfile, validating workspace ownership plus detected port exposure before forwarding, splitting plain HTTP reverse-proxy handling from raw WebSocket hijack/tunnel handling, wiring a workspace agent `ListPorts` client into the control plane, and extending the Terraform Cloud Run module plus both env stacks with the port-forward service deployment path and URI outputs.

## What was done this session
Completed a user-directed auth maintenance slice in the control plane by adding Firebase-authenticated API key issuance/list/revoke endpoints, binding issued API keys to the Firebase UID, and extending cached API key validation to carry both tenant and user identity so session creation cannot impersonate a different user with a user-bound key.

Also upgraded `demo_app` so the localhost showcase can register or log into Firebase, mint a Cortado API key in-app, and feed that key back into the existing session/workspace flow while keeping the manual `.env` API key path as a fallback. The active feature pointer remains on Task 7.1.3; the Flutter preview work has not advanced during this interruption.

Extended that localhost demo bootstrap again so brand-new Firebase users can self-assign the development tenant claim through a dev-only control-plane route, then mint a Cortado API key without leaving the app.

Drafted the next production auth direction into the planning docs: browser-driven OIDC token exchange as the first no-server path, with tenant-backend server-to-server minting explicitly deferred. Added the new feature spec, release-timeline entry, and technical-report architecture notes without changing the active implementation milestone.

## Remaining work this session
Task 7.1.3:
- add a Flutter "Run Preview" flow that drives `flutter build web`
- detect the preview server port through the port-watch surface and expose an "Open Preview" action
- embed the preview inside an `IFrame` using the port-forward gateway URL and any gateway headers needed for framing

## Definition of done
- [ ] Flutter package can trigger a preview build/start flow for a workspace
- [ ] preview readiness is derived from the workspace port watch/list surface
- [ ] embedded preview uses the port-forward gateway URL shape expected by the backend
- [ ] relevant Flutter tests/analyze pass for the preview slice

## Next task after this one
Hold after Feature 7.1 completes; do not advance to v0.8 without explicit instruction.
See _dev/features/feat-7-1.md for the active Feature 7.1 spec

## Blocked on / decisions needed
No active blockers.
