## Feature 8.4 — Direct OIDC Session Exchange
**Duration**: Week 40 (3 tasks, ~4 days)

*This feature adds the first production-grade no-server authentication path. Tenant-backend server-to-server session minting remains deferred and is not part of this scope.*

### Goal
- Let a tenant-managed browser app authenticate users with its own OIDC provider and exchange that external user token directly for a Cortado session.
- Keep Cortado out of tenant identity databases and admin consoles.
- Preserve a future upgrade path to server-to-server minting without throwing away the exchange implementation.

### Non-goals
- Do not implement tenant-backend server-to-server session minting yet.
- Do not support opaque OAuth access tokens that require provider-specific introspection APIs.
- Do not make Cortado a full identity broker or generic social-login gateway.

### Task 8.4.1 — Tenant auth-provider configuration
- Add tenant-scoped auth provider config for:
  - OIDC discovery URL or explicit `issuer` + `jwks_uri`
  - allowed audience / client IDs
  - accepted signing algorithms
  - `user_id` claim mapping (default `sub`)
  - optional claim requirements for org/team membership
- Expose control-plane CRUD endpoints for this config under the tenant self-service API surface.
- Persist provider config alongside tenant metadata and validate it on write.

**Challenge**: provider metadata can be subtly malformed or incomplete. Validate discovery responses up front and fail tenant config writes early instead of surfacing confusing auth failures at session-exchange time.

---

### Task 8.4.2 — `POST /v1/sessions/exchange`
- Implement a new session exchange endpoint that accepts a tenant-scoped external JWT and returns Cortado `{access_token, refresh_token}`.
- Resolve the tenant's configured discovery/JWKS metadata, cache signing keys, and validate:
  - `iss`
  - `aud`
  - `exp` / `nbf`
  - signature
  - allowed signing algorithms
- Map the configured user claim into Cortado's internal `user_id`.
- Mint Cortado JWTs using the existing internal session machinery so the rest of the control plane continues to consume Cortado-native tokens only.

**Challenge**: many providers return both ID tokens and access tokens, but only some access tokens are JWTs with stable claims. Keep the first implementation strict: accept JWT-like tokens only, with a clear error when a tenant tries to send an opaque token.

---

### Task 8.4.3 — Flutter exchange client + example integration
- Extend the Flutter package auth session to support exchanging an externally obtained OIDC token for a Cortado session.
- Add a package example showing:
  - browser sign-in handled by the consuming app
  - Cortado session exchange call
  - normal workspace create/connect flow after exchange
- Document the provider requirements and the difference between:
  - direct browser exchange
  - development-only Firebase bootstrap
  - future server-to-server minting

**Challenge**: browser auth SDKs vary a lot. Keep the Flutter package surface narrow: Cortado should accept a token string and not own the upstream sign-in UI or provider SDK lifecycle.

---

### Definition of done
- [ ] Tenant auth-provider config exists with strict validation for discovery/JWKS and audience settings
- [ ] `POST /v1/sessions/exchange` validates tenant-issued external JWTs and returns Cortado session tokens
- [ ] Exchange path reuses Cortado's internal JWT/refresh-token machinery after claim mapping
- [ ] Flutter package exposes a small exchange client surface without embedding provider-specific auth logic
- [ ] Docs clearly position browser exchange as the first no-server path and server-to-server minting as deferred follow-up
