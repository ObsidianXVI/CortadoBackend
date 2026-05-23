package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/cortado/control-plane/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthMiddlewareRejectsMissingCredentials(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	handler := NewAuthMiddleware(AuthConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareInjectsJWTClaims(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	jwksJSON, accessToken := mustIssueAccessToken(t, "tenant-1", "user-1")
	handler := NewAuthMiddleware(AuthConfig{JWKSJSON: jwksJSON})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantID(r.Context())
		if !ok || tenantID != "tenant-1" {
			t.Fatalf("unexpected tenant context: %q %t", tenantID, ok)
		}

		userID, ok := UserID(r.Context())
		if !ok || userID != "user-1" {
			t.Fatalf("unexpected user context: %q %t", userID, ok)
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestAuthMiddlewareFallsBackToDevBypassInDevelopment(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	handler := NewAuthMiddleware(AuthConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantID(r.Context())
		if !ok || tenantID != "dev-tenant" {
			t.Fatalf("unexpected tenant context: %q %t", tenantID, ok)
		}

		userID, ok := UserID(r.Context())
		if !ok || userID != "dev-user" {
			t.Fatalf("unexpected user context: %q %t", userID, ok)
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestAuthMiddlewareRejectsInvalidJWTBeforeDevBypassFallback(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	jwksJSON, _ := mustIssueAccessToken(t, "tenant-1", "user-1")
	handler := NewAuthMiddleware(AuthConfig{JWKSJSON: jwksJSON})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareAcceptsWebSocketQueryToken(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	jwksJSON, accessToken := mustIssueAccessToken(t, "tenant-1", "user-1")
	handler := NewAuthMiddleware(AuthConfig{JWKSJSON: jwksJSON})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/ws-123/connect?token="+accessToken, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNoContent)
	}
}

func mustIssueAccessToken(t *testing.T, tenantID, userID string) ([]byte, string) {
	t.Helper()

	privateKeyPEM, err := auth.GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	service, err := auth.NewService(auth.ServiceConfig{
		PrivateKeyPEM: privateKeyPEM,
		Repository: &authRepositoryStub{
			apiKeys: []auth.APIKeyRecord{{TenantID: tenantID, Hash: mustHashAPIKey(t, "secret-api-key")}},
		},
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	tokens, err := service.CreateSession(context.Background(), "secret-api-key", userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	return service.JWKS(), tokens.AccessToken
}

func mustHashAPIKey(t *testing.T, apiKey string) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}
	return string(hash)
}

type authRepositoryStub struct {
	apiKeys []auth.APIKeyRecord
}

func (r *authRepositoryStub) ListAPIKeys(_ context.Context) ([]auth.APIKeyRecord, error) {
	return append([]auth.APIKeyRecord(nil), r.apiKeys...), nil
}

func (r *authRepositoryStub) SaveRefreshToken(_ context.Context, token auth.RefreshTokenRecord) error {
	return nil
}
