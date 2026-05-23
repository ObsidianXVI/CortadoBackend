package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/cortado/control-plane/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestSessionRouteIssuesTokensWithoutDevBypass(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	router := NewRouter(RouterConfig{
		SessionSvc: sessionServiceStub{
			tokens: auth.SessionTokens{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", bytes.NewBufferString(`{"api_key":"secret","user_id":"user-1"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected session response body")
	}
}

func TestSessionRefreshRouteIssuesAccessTokenWithoutDevBypass(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	router := NewRouter(RouterConfig{
		SessionSvc: sessionServiceStub{
			refreshAccessToken: "new-access-token",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/refresh", bytes.NewBufferString(`{"refresh_token":"refresh-token"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected refresh response body")
	}
}

func TestSessionRefreshRouteRejectsInvalidRefreshToken(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	router := NewRouter(RouterConfig{
		SessionSvc: sessionServiceStub{
			refreshErr: auth.ErrInvalidRefreshToken,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/refresh", bytes.NewBufferString(`{"refresh_token":"invalid"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestJWKSRouteServesDocument(t *testing.T) {
	authService := mustAuthService(t)
	router := NewRouter(RouterConfig{
		JWKSProvider: authService,
	})

	req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if got := rec.Body.Bytes(); string(got) != string(authService.JWKS()) {
		t.Fatalf("unexpected body: %q", got)
	}
}

type sessionServiceStub struct {
	err                error
	refreshErr         error
	refreshAccessToken string
	tokens             auth.SessionTokens
}

func (s sessionServiceStub) CreateSession(_ context.Context, _, _ string) (auth.SessionTokens, error) {
	return s.tokens, s.err
}

func (s sessionServiceStub) RefreshSession(_ context.Context, _ string) (string, error) {
	return s.refreshAccessToken, s.refreshErr
}

func mustAuthService(t *testing.T) *auth.Service {
	t.Helper()

	privateKeyPEM, err := auth.GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	service, err := auth.NewService(auth.ServiceConfig{
		PrivateKeyPEM: privateKeyPEM,
		Repository: &authRepositoryStub{
			apiKeys: []auth.APIKeyRecord{{TenantID: "tenant-1", Hash: string(hash)}},
		},
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	return service
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

func (r *authRepositoryStub) GetRefreshToken(_ context.Context, _ string) (auth.RefreshTokenRecord, bool, error) {
	return auth.RefreshTokenRecord{}, false, nil
}
