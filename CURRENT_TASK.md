# CURRENT TASK

## Release · Feature · Task
v0.1 → Feature 2.4 (Real JWT Authentication) → Task 2.4.1

## Status
IN PROGRESS

## What was done last session
Completed Task 2.3.2 by adding workspace-agent usage-event emission with WAL persistence and replay, wiring a `FlushUsageWAL` gRPC path plus control-plane pre-delete flushing, and extending Terraform so workspace agents receive usage topic config and Pub/Sub publisher IAM.

## What was done this session
Loaded the Task 2.4.1 spec and advanced the release/task pointers so JWT issuance, JWKS exposure, Secret Manager wiring, and API-key validation can be implemented against the documented acceptance criteria instead of guessed requirements.

## Remaining work this session
Add Terraform-managed Secret Manager resources and IAM for the JWT signing key, implement `POST /v1/sessions` with Firestore-backed hashed API-key validation and Redis/Dragonfly caching, issue 8-hour JWTs plus 30-day refresh tokens, expose `GET /.well-known/jwks.json`, and cover the new auth surface with tests and required Go/Terraform verification.

## Definition of done
- [ ] Terraform creates the per-environment JWT private-key Secret Manager secret and grants the control-plane service account `roles/secretmanager.secretAccessor`
- [ ] `POST /v1/sessions` accepts `{api_key, user_id}` and validates the hashed API key against Firestore
- [ ] The session endpoint returns an 8-hour JWT access token and a 30-day opaque UUID refresh token
- [ ] Issued JWTs include `sub`, `tid`, `exp`, and `jti` claims
- [ ] `GET /.well-known/jwks.json` exposes the public key for JWT verification
- [ ] API-key validation uses a 5-minute Redis/Dragonfly cache keyed by a hash of the raw API key to avoid repeated bcrypt work
- [ ] `cd proto && buf lint` passes
- [ ] `cd control-plane && CGO_ENABLED=0 go build ./...` passes
- [ ] `cd control-plane && go test ./...` passes
- [ ] `terraform validate` passes for `terraform/envs/dev`
- [ ] `terraform validate` passes for `terraform/envs/prod`

## Next task after this one
Task 2.4.2 — JWT validation middleware
See _dev/docs/release_timeline.md §Feature 2.4 Task 2.4.2 for full spec

## Blocked on / decisions needed
None.
