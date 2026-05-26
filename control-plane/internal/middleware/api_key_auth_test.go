package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/your-org/cortado/control-plane/internal/auth"
)

func TestAPIKeyAuthMiddlewareInjectsJWTClaims(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	jwksJSON, accessToken := mustIssueAccessToken(t, "tenant-1", "user-1")
	handler := NewAPIKeyAuthMiddleware(APIKeyAuthConfig{
		JWKSJSON: jwksJSON,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, tenantOK := TenantID(r.Context())
		userID, userOK := UserID(r.Context())
		if !tenantOK || !userOK || tenantID != "tenant-1" || userID != "user-1" {
			t.Fatalf("unexpected context values tenant=%q user=%q", tenantID, userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAPIKeyAuthMiddlewareFallsBackToFirebaseToken(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	handler := NewAPIKeyAuthMiddleware(APIKeyAuthConfig{
		TenantClaim: "tenant_id",
		Verifier: firebaseVerifierStub{
			token: &auth.VerifiedFirebaseToken{
				UID:    "firebase-user-1",
				Claims: map[string]any{"tenant_id": "tenant-1"},
			},
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, tenantOK := TenantID(r.Context())
		userID, userOK := UserID(r.Context())
		if !tenantOK || !userOK || tenantID != "tenant-1" || userID != "firebase-user-1" {
			t.Fatalf("unexpected context values tenant=%q user=%q", tenantID, userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	req.Header.Set("Authorization", "Bearer firebase-id-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAPIKeyAuthMiddlewarePreservesMissingFirebaseTenantClaimError(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	handler := NewAPIKeyAuthMiddleware(APIKeyAuthConfig{
		TenantClaim: "tenant_id",
		Verifier: firebaseVerifierStub{
			token: &auth.VerifiedFirebaseToken{
				UID:    "firebase-user-1",
				Claims: map[string]any{},
			},
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected middleware to reject request")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	req.Header.Set("Authorization", "Bearer firebase-id-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusForbidden)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != auth.ErrTenantClaimMissing.Error() {
		t.Fatalf("unexpected body: got %q want %q", body, auth.ErrTenantClaimMissing.Error())
	}
}
