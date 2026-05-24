## Feature 2.4 — Real JWT Authentication (End of v0.2)

*Auth is implemented last in v0.2 so that all prior features were debugged without auth complexity. After these tasks, the dev-bypass is still available in dev environments but the production path requires real JWTs.*

### Task 2.4.1 — JWT issuance and JWKS endpoint
**What to do:**
- Generate an RSA-2048 keypair. Store the private key in GCP Secret Manager (Terraform manages the *resource*, not the value):
  ```hcl
  resource "google_secret_manager_secret" "jwt_private_key" {
    secret_id = "cortado-jwt-private-key-${var.env}"
    replication { auto {} }
  }
  resource "google_secret_manager_secret_iam_member" "control_plane_reader" {
    secret_id = google_secret_manager_secret.jwt_private_key.id
    role      = "roles/secretmanager.secretAccessor"
    member    = "serviceAccount:${var.control_plane_sa_email}"
  }
  ```
  The key *value* is added manually: `gcloud secrets versions add cortado-jwt-private-key-dev --data-file=private_key.pem`.
- Implement `POST /v1/sessions`: accepts `{api_key, user_id}`, validates API key against Firestore (bcrypt hash), returns `{access_token (JWT, 8h), refresh_token (opaque UUID, 30d)}`.
- JWT claims: `{sub: user_id, tid: tenant_id, exp, jti}`. Workspaces are authorized per-request (not embedded in the JWT).
- Expose `GET /.well-known/jwks.json` with the public key.

**Key detail**: API keys are stored hashed (`bcrypt.GenerateFromPassword(key, 12)`). The raw key is shown to the tenant once at creation time (via the dashboard, implemented in v0.8). For dev, generate a test API key manually and insert its bcrypt hash into Firestore directly. The control plane never stores or logs the raw key.

**Challenge**: bcrypt hash comparison takes ~100ms (by design, to resist brute force). This makes `POST /v1/sessions` slow. Cache the validation result in Dragonfly/Redis (key: hash of the API key, value: tenant_id, TTL 5 minutes). On cache hit, skip bcrypt. On cache miss, bcrypt and store in cache. This reduces p99 from 100ms to <5ms for repeat sessions.

---

### Task 2.4.2 — JWT validation middleware
**What to do:**
- Replace `DevBypassAuth` middleware with a chain: try JWT first, fall back to dev-bypass *only if* `CORTADO_ENV=development`.
  ```go
  func AuthMiddleware(jwks *keyfunc.JWKS) func(http.Handler) http.Handler {
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              // Dev bypass (compile-time guarded)
              if os.Getenv("CORTADO_ENV") == "development" {
                  if r.Header.Get("X-Cortado-Dev-Token") == "dev-bypass" {
                      next.ServeHTTP(w, injectDevContext(r))
                      return
                  }
              }
              // Real JWT validation
              tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
              token, err := jwt.Parse(tokenStr, jwks.Keyfunc)
              if err != nil || !token.Valid {
                  http.Error(w, "unauthorized", 401)
                  return
              }
              claims := token.Claims.(jwt.MapClaims)
              ctx := context.WithValue(r.Context(), ctxKeyTenantID, claims["tid"])
              ctx = context.WithValue(ctx, ctxKeyUserID, claims["sub"])
              next.ServeHTTP(w, r.WithContext(ctx))
          })
      }
  }
  ```
- Use `github.com/MicahParks/keyfunc` for automatic JWKS key rotation.

**Challenge**: The WebSocket upgrade (`/v1/workspaces/{id}/connect`) can't set `Authorization` headers from browsers (browser WebSocket API doesn't support custom headers). Pass the JWT as a query parameter: `?token={jwt}` and extract it in the middleware specifically for WebSocket upgrade requests. Validate the token the same way. Note: query parameters appear in server access logs — consider the JWT expiry window (8h is long for a URL-embedded token; use a short-lived connection token (5 min TTL) for WebSocket URLs specifically.

---

### Task 2.4.3 — JWT refresh in Flutter client + tag v0.2
**What to do:**
- `CortadoClient` stores the JWT and its `exp` claim.
- A background timer fires 5 minutes before expiry: calls `POST /v1/sessions/refresh` with the refresh token, receives a new JWT, stores it.
- Replace `X-Cortado-Dev-Token` with `Authorization: Bearer {jwt}` in all requests (the dev bypass remains as a fallback in `CORTADO_ENV=development`).
- Verify end-to-end with real JWTs: create workspace, open terminal, let it run for 9 hours, confirm the refresh fires and the session continues without interruption.
- Tag the release: `git tag v0.2.0 && git push --tags`.

**Challenge**: The refresh timer must survive Flutter app backgrounding (on mobile targets) and browser tab suspension (on web). Flutter Web's timer continues to run if the tab is active but is throttled by the browser when the tab is in the background. Handle this by also checking the JWT expiry on every request — if expired (tab was suspended), refresh synchronously before proceeding.

---

---
