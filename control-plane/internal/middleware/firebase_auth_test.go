package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/cortado/control-plane/internal/auth"
)

func TestFirebaseAuthMiddlewareInjectsTenantAndUser(t *testing.T) {
	t.Parallel()

	handler := NewFirebaseAuthMiddleware(FirebaseAuthConfig{
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

func TestFirebaseAuthMiddlewareRejectsMissingTenantClaim(t *testing.T) {
	t.Parallel()

	handler := NewFirebaseAuthMiddleware(FirebaseAuthConfig{
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
}

type firebaseVerifierStub struct {
	err   error
	token *auth.VerifiedFirebaseToken
}

func (s firebaseVerifierStub) VerifyIDToken(_ context.Context, _ string) (*auth.VerifiedFirebaseToken, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.token, nil
}
