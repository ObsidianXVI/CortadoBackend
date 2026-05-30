# CURRENT TASK

## Release · Feature · Task
v0.7 → Feature 7.1 (Port Forward Gateway) → Task 7.1.3

## Status
PENDING

## What was done last session
Completed Task 8.4.4 by introducing platform-tenant bootstrap plus platform API key management routes behind normal Cortado user sessions, extending `POST /v1/sessions` and JWT claims so platform API keys can mint platform-scoped Cortado sessions without a `user_id`, and documenting the split between personal first-party auth and SaaS backend platform auth.

## What was done this session
Integrated the new auth backend flows into `demo_app` by adding first-party Firebase-to-Cortado session exchange, platform-tenant bootstrap plus platform API key minting/listing, and platform-session bootstrap with an empty `user_id` so the demo can exercise the new auth model end to end before resuming the deferred preview work. Follow-up fixes switched personal API key issue/list actions to prefer the exchanged Cortado session instead of raw Firebase tokens, added clearer messaging when the dev-only tenant-claim route is not mounted, fixed the reusable Flutter Firebase auth client to avoid `Firebase.apps` pre-init crashes on web, added top-level control-plane CORS handling so Flutter web browser requests can reach the API with auth headers, corrected Firestore tenant upserts to pass map data with `firestore.MergeAll` so first-login personal-tenant provisioning no longer crashes, filtered reserved Firebase ID-token claims out of the dev bootstrap custom-claims write so `Assign Dev Tenant` no longer fails on keys like `auth_time`, made workspace stop/delete ignore unreachable-agent flush/snapshot calls so broken never-scheduled workspaces can still be cleaned up, corrected the demo/workspace readiness path so the demo now defaults to the real Cortado workspace-agent image while the control plane only marks a workspace `RUNNING` once the pod is actually `Ready`, fixed local control-plane agent routing by reusing the pod-backed resolver and preferring live pod IPs over in-cluster service DNS so terminal, file, and idle-inspection RPCs work from the VM-hosted server process, made idle monitoring tolerate older workspace-agent images that do not yet implement `GetIdleStatus` so mixed-version local dev setups no longer spam `Unimplemented` logs, and bumped the demo’s default workspace image to the newer `cortado-workspace:781d613` tag so newly provisioned demo workspaces expose the file RPCs used by `Load File` / `Save File`.

## Remaining work this session
Begin Task 7.1.3 for Flutter web preview: build/run preview from the workspace, detect the bound preview port, and embed the forwarded preview URL inside the demo UI.

## Definition of done
- [ ] "Run Preview" triggers `flutter build web` in the workspace and streams progress to the terminal
- [ ] the demo waits for the preview port to bind and exposes an "Open Preview" action
- [ ] the forwarded preview renders inside the Flutter web demo through an embedded frame
- [ ] docs and validation cover the preview-specific flow

## Next task after this one
TBD after Task 7.1.3 completes.

## Blocked on / decisions needed
No active blockers.
