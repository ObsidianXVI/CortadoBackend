## Feature 8.4 — Cortado-Managed Auth + API Key Modes
**Duration**: Week 40 (4 tasks, ~6 days)

*This feature makes Cortado usable with zero backend work for embedded Flutter web IDEs while still preserving a long-lived API-key path for headless personal usage and SaaS/server integrations.*

### Goal
- Let frontend-package developers embed Cortado without building custom auth middleware or a Cortado-specific backend.
- Make Cortado-managed Firebase auth the default end-user identity path for browser IDEs.
- Preserve long-lived API keys for both personal headless usage and server-side SaaS platforms that want Cortado-to-platform trust only.

### Non-goals
- Do not require tenant-managed OIDC or other BYO-auth integration for the default product path.
- Do not make long-lived API keys the normal browser credential for embedded apps.
- Do not implement enterprise SAML/OIDC federation in this slice.
- Do not require Cortado to understand or verify a SaaS platform's downstream end-user identities in the platform API-key mode.

### Task 8.4.1 — First-party Firebase session exchange
- Add a new `POST /v1/sessions/exchange/firebase` endpoint that accepts a Firebase ID token issued by Cortado's own Firebase project.
- Verify the token, auto-provision a Cortado user profile on first login, and create or resolve that user's default personal tenant namespace.
- Return Cortado `{access_token, refresh_token}` so the Flutter package can proceed with normal workspace calls without a separate API-key bootstrap step.
- Persist a stable mapping between Firebase UID, Cortado user ID, and the user's default personal tenant.

**Challenge**: first-login provisioning must be idempotent and stable under retries so package consumers never have to think about internal tenant creation or partial bootstrap state.

---

### Task 8.4.2 — Flutter first-party auth client + embedded auth surface
- Add package-level auth helpers for email/password and Google login against Cortado-managed Firebase Auth.
- Expose a low-friction embedded auth surface that host Flutter web apps can drop in without adding backend middleware.
- After Firebase sign-in, exchange the Firebase ID token for a Cortado session automatically and hand the result to the existing Cortado client/workspace layers.
- Document both the turnkey embedded-auth path and the lower-level path for hosts that want custom UI but still rely on Cortado-managed auth.

**Challenge**: keep the package embeddable and low-opinion while still delivering a genuinely zero-backend adoption path.

---

### Task 8.4.3 — Personal API key issuance + management
- Let an authenticated Cortado user mint, list, and revoke long-lived personal API keys after a one-time Firebase sign-in.
- Return the raw key only once, store only a hash, and allow that key to create Cortado sessions later without another headed auth flow.
- Position personal API keys as a headless/power-user path for CLI, local tooling, and non-interactive developer workflows rather than the default browser path.
- Ensure personal API keys remain bound to the owning Cortado user and personal tenant so they cannot impersonate another Cortado user.

**Challenge**: keep the browser-first path on refreshable session tokens while still making headless personal access practical and easy to reason about.

---

### Task 8.4.4 — Platform API keys for SaaS backends
- Introduce a platform tenant entity that can hold long-lived server-side API keys independently of first-party Cortado end-user accounts.
- Let a SaaS platform authenticate to Cortado as one external entity and call Cortado APIs from its own backend without interactive login flows.
- Treat the platform's internal end users as opaque to Cortado unless the platform chooses to send user metadata for its own bookkeeping.
- Document this as the path for full SaaS products that already run their own identity systems and only need Cortado-to-platform trust.

**Challenge**: keep platform API-key auth clearly separate from the first-party end-user auth model so permissions, billing, and attribution do not get blurred.

---

### Definition of done
- [ ] Cortado-managed Firebase session exchange exists and auto-provisions first-party users plus personal tenants
- [ ] Flutter package supports zero-backend login/register flows against Cortado-managed auth
- [ ] Personal API keys can be minted after one-time auth and reused headlessly
- [ ] Platform API keys exist for SaaS/server integrations that authenticate as one Cortado entity
- [ ] Docs clearly position first-party auth as the default path and platform API keys as the backend integration path
