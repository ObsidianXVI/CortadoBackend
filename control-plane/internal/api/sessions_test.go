package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/cortado/control-plane/internal/auth"
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

func TestJWKSRouteServesDocument(t *testing.T) {
	router := NewRouter(RouterConfig{
		JWKSProvider: jwksProviderStub{payload: []byte(`{"keys":[{"kid":"kid-1"}]}`)},
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
	if got := rec.Body.String(); got != `{"keys":[{"kid":"kid-1"}]}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

type sessionServiceStub struct {
	err    error
	tokens auth.SessionTokens
}

func (s sessionServiceStub) CreateSession(_ context.Context, _, _ string) (auth.SessionTokens, error) {
	return s.tokens, s.err
}

type jwksProviderStub struct {
	payload []byte
}

func (j jwksProviderStub) JWKS() []byte {
	return append([]byte(nil), j.payload...)
}
