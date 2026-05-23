package auth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestServiceCreateSessionIssuesTokens(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	now := time.Date(2026, time.May, 23, 13, 30, 0, 0, time.UTC)
	repository := &repositoryStub{
		apiKeys: []APIKeyRecord{
			{
				Hash:     string(hash),
				TenantID: "tenant-1",
			},
		},
	}

	service, err := NewService(ServiceConfig{
		Now:           func() time.Time { return now },
		PrivateKeyPEM: privateKeyPEM,
		Repository:    repository,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	tokens, err := service.CreateSession(context.Background(), "secret-api-key", "user-1")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if tokens.RefreshToken == "" || tokens.AccessToken == "" {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
	if len(repository.savedRefreshTokens) != 1 {
		t.Fatalf("unexpected saved refresh token count: %d", len(repository.savedRefreshTokens))
	}

	verifier, err := keyfunc.NewJWKSetJSON(json.RawMessage(service.JWKS()))
	if err != nil {
		t.Fatalf("new jwks verifier: %v", err)
	}
	token, err := jwt.ParseWithClaims(tokens.AccessToken, &AccessClaims{}, verifier.Keyfunc, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if got := token.Header["kid"]; got != service.keyID {
		t.Fatalf("unexpected token header kid: got %v want %q", got, service.keyID)
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok {
		t.Fatalf("unexpected claims type: %T", token.Claims)
	}
	if claims.Subject != "user-1" || claims.TenantID != "tenant-1" || claims.ID == "" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(now.Add(accessTokenTTL)) {
		t.Fatalf("unexpected expiry: %#v", claims.ExpiresAt)
	}
}

func TestServiceCreateSessionUsesValidationCache(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	cache := &cacheStub{entries: map[string]string{}}
	repository := &repositoryStub{
		apiKeys: []APIKeyRecord{
			{Hash: string(hash), TenantID: "tenant-1"},
		},
	}

	service, err := NewService(ServiceConfig{
		Cache:         cache,
		PrivateKeyPEM: privateKeyPEM,
		Repository:    repository,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.CreateSession(context.Background(), "secret-api-key", "user-1"); err != nil {
		t.Fatalf("first create session: %v", err)
	}
	repository.apiKeys = nil
	if _, err := service.CreateSession(context.Background(), "secret-api-key", "user-2"); err != nil {
		t.Fatalf("second create session: %v", err)
	}

	if repository.listCalls != 1 {
		t.Fatalf("unexpected repository list calls: got %d want 1", repository.listCalls)
	}
	if cache.putCalls != 1 || cache.getCalls < 2 {
		t.Fatalf("unexpected cache calls: gets=%d puts=%d", cache.getCalls, cache.putCalls)
	}
}

func TestServiceCreateSessionRejectsInvalidCredentials(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	service, err := NewService(ServiceConfig{
		PrivateKeyPEM: privateKeyPEM,
		Repository:    &repositoryStub{},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.CreateSession(context.Background(), "missing", "user-1")
	if err == nil || !strings.Contains(err.Error(), ErrInvalidCredentials.Error()) {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

type repositoryStub struct {
	apiKeys            []APIKeyRecord
	listCalls          int
	savedRefreshTokens []RefreshTokenRecord
}

func (r *repositoryStub) ListAPIKeys(_ context.Context) ([]APIKeyRecord, error) {
	r.listCalls++
	return append([]APIKeyRecord(nil), r.apiKeys...), nil
}

func (r *repositoryStub) SaveRefreshToken(_ context.Context, token RefreshTokenRecord) error {
	r.savedRefreshTokens = append(r.savedRefreshTokens, token)
	return nil
}

type cacheStub struct {
	entries  map[string]string
	getCalls int
	putCalls int
}

func (c *cacheStub) Close() error {
	return nil
}

func (c *cacheStub) GetTenantID(_ context.Context, apiKey string) (string, bool, error) {
	c.getCalls++
	tenantID, ok := c.entries[cacheKey(apiKey)]
	return tenantID, ok, nil
}

func (c *cacheStub) PutTenantID(_ context.Context, apiKey, tenantID string, _ time.Duration) error {
	c.putCalls++
	c.entries[cacheKey(apiKey)] = tenantID
	return nil
}
