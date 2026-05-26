package auth

import (
	"context"
	"encoding/json"
	"errors"
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
	token, err := jwt.ParseWithClaims(
		tokens.AccessToken,
		&AccessClaims{},
		verifier.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithTimeFunc(func() time.Time { return now }),
		jwt.WithoutClaimsValidation(),
	)
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

	cache := &cacheStub{entries: map[string]APIKeyIdentity{}}
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

func TestServiceCreateSessionRejectsMismatchedBoundUser(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	service, err := NewService(ServiceConfig{
		PrivateKeyPEM: privateKeyPEM,
		Repository: &repositoryStub{
			apiKeys: []APIKeyRecord{
				{
					Hash:     string(hash),
					TenantID: "tenant-1",
					UserID:   "firebase-user-1",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.CreateSession(context.Background(), "secret-api-key", "user-2"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

func TestServiceCreateSessionRejectsMissingUserForPersonalKey(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	service, err := NewService(ServiceConfig{
		PrivateKeyPEM: privateKeyPEM,
		Repository: &repositoryStub{
			apiKeys: []APIKeyRecord{
				{
					Hash:     string(hash),
					TenantID: "tenant-1",
					UserID:   "user-1",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.CreateSession(context.Background(), "secret-api-key", ""); !errors.Is(err, ErrUserIDRequired) {
		t.Fatalf("expected user id required error, got %v", err)
	}
}

func TestServiceCreateSessionIssuesPlatformTokensWithoutUserID(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("platform-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	now := time.Date(2026, time.May, 26, 3, 0, 0, 0, time.UTC)
	repository := &repositoryStub{
		apiKeys: []APIKeyRecord{
			{
				Hash:            string(hash),
				Kind:            APIKeyKindPlatform,
				TenantID:        "platform-tenant-1",
				CreatedByUserID: "user-1",
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

	tokens, err := service.CreateSession(context.Background(), "platform-api-key", "")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	claims := parseAccessClaims(t, service, tokens.AccessToken, now)
	if claims.ActorType != ActorTypePlatform {
		t.Fatalf("unexpected actor type: %#v", claims)
	}
	if claims.Subject != "platform:platform-tenant-1" {
		t.Fatalf("unexpected subject: %#v", claims)
	}
	if claims.TenantID != "platform-tenant-1" {
		t.Fatalf("unexpected tenant: %#v", claims)
	}
	if len(repository.savedRefreshTokens) != 1 || repository.savedRefreshTokens[0].ActorType != ActorTypePlatform {
		t.Fatalf("unexpected refresh tokens: %#v", repository.savedRefreshTokens)
	}
}

func TestServiceCreateSessionRejectsUserIDForPlatformKey(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("platform-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	service, err := NewService(ServiceConfig{
		PrivateKeyPEM: privateKeyPEM,
		Repository: &repositoryStub{
			apiKeys: []APIKeyRecord{
				{
					Hash:     string(hash),
					Kind:     APIKeyKindPlatform,
					TenantID: "platform-tenant-1",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.CreateSession(context.Background(), "platform-api-key", "user-1"); !errors.Is(err, ErrPlatformUserID) {
		t.Fatalf("expected platform user id error, got %v", err)
	}
}

func TestServiceCreateSessionRejectsMismatchedCachedBoundUser(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	cache := &cacheStub{
		entries: map[string]APIKeyIdentity{
			cacheKey("secret-api-key"): {
				TenantID: "tenant-1",
				UserID:   "firebase-user-1",
			},
		},
	}

	service, err := NewService(ServiceConfig{
		Cache:         cache,
		PrivateKeyPEM: privateKeyPEM,
		Repository:    &repositoryStub{},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.CreateSession(context.Background(), "secret-api-key", "user-2"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials error, got %v", err)
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

func TestServiceExchangeFirebaseSessionProvisionsFirstPartyAccountAndTenant(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	now := time.Date(2026, time.May, 26, 2, 15, 0, 0, time.UTC)
	repository := &repositoryStub{}
	service, err := NewService(ServiceConfig{
		FirebaseVerifier: firebaseVerifierStub{
			token: &VerifiedFirebaseToken{
				UID: "firebase-user-1",
				Claims: map[string]any{
					"email": "user@example.com",
					"name":  "User One",
				},
			},
		},
		Now:           func() time.Time { return now },
		PrivateKeyPEM: privateKeyPEM,
		Repository:    repository,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	tokens, err := service.ExchangeFirebaseSession(context.Background(), "firebase-id-token")
	if err != nil {
		t.Fatalf("exchange firebase session: %v", err)
	}

	claims := parseAccessClaims(t, service, tokens.AccessToken, now)
	expectedUserID := firstPartyUserID("firebase-user-1")
	expectedTenantID := firstPartyTenantID("firebase-user-1")
	if claims.Subject != expectedUserID || claims.TenantID != expectedTenantID {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if len(repository.savedAccounts) != 1 {
		t.Fatalf("unexpected saved account count: got %d want 1", len(repository.savedAccounts))
	}
	account := repository.savedAccounts[0]
	if account.FirebaseUID != "firebase-user-1" || account.UserID != expectedUserID || account.PersonalTenantID != expectedTenantID {
		t.Fatalf("unexpected first-party account: %#v", account)
	}
	if account.Email != "user@example.com" || account.DisplayName != "User One" {
		t.Fatalf("unexpected first-party account profile fields: %#v", account)
	}
	if len(repository.ensuredTenants) != 1 {
		t.Fatalf("unexpected ensured tenant count: got %d want 1", len(repository.ensuredTenants))
	}
	tenant := repository.ensuredTenants[0]
	if tenant.TenantID != expectedTenantID || tenant.OwnerUserID != expectedUserID || tenant.Kind != "personal" {
		t.Fatalf("unexpected personal tenant record: %#v", tenant)
	}
}

func TestServiceExchangeFirebaseSessionReusesExistingFirstPartyAccount(t *testing.T) {
	t.Parallel()

	privateKeyPEM, err := GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	now := time.Date(2026, time.May, 26, 2, 15, 0, 0, time.UTC)
	repository := &repositoryStub{
		firstPartyAccounts: map[string]FirstPartyAccount{
			"firebase-user-1": {
				CreatedAt:        now.Add(-time.Hour),
				FirebaseUID:      "firebase-user-1",
				PersonalTenantID: "tenant-existing",
				UpdatedAt:        now.Add(-time.Hour),
				UserID:           "user-existing",
			},
		},
	}
	service, err := NewService(ServiceConfig{
		FirebaseVerifier: firebaseVerifierStub{
			token: &VerifiedFirebaseToken{
				UID: "firebase-user-1",
			},
		},
		Now:           func() time.Time { return now },
		PrivateKeyPEM: privateKeyPEM,
		Repository:    repository,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	firstTokens, err := service.ExchangeFirebaseSession(context.Background(), "firebase-id-token")
	if err != nil {
		t.Fatalf("first exchange firebase session: %v", err)
	}
	secondTokens, err := service.ExchangeFirebaseSession(context.Background(), "firebase-id-token")
	if err != nil {
		t.Fatalf("second exchange firebase session: %v", err)
	}

	firstClaims := parseAccessClaims(t, service, firstTokens.AccessToken, now)
	secondClaims := parseAccessClaims(t, service, secondTokens.AccessToken, now)
	if firstClaims.Subject != "user-existing" || firstClaims.TenantID != "tenant-existing" {
		t.Fatalf("unexpected first claims: %#v", firstClaims)
	}
	if secondClaims.Subject != "user-existing" || secondClaims.TenantID != "tenant-existing" {
		t.Fatalf("unexpected second claims: %#v", secondClaims)
	}
	if len(repository.savedAccounts) != 0 {
		t.Fatalf("expected existing account to be reused without rewrite, saved %d accounts", len(repository.savedAccounts))
	}
	if len(repository.ensuredTenants) != 2 {
		t.Fatalf("unexpected ensured tenant count: got %d want 2", len(repository.ensuredTenants))
	}
}

func TestServiceRefreshSessionIssuesNewAccessToken(t *testing.T) {
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

	issued, err := service.CreateSession(context.Background(), "secret-api-key", "user-1")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	initialClaims := parseAccessClaims(t, service, issued.AccessToken, now)

	now = now.Add(2 * time.Hour)
	refreshedToken, err := service.RefreshSession(context.Background(), issued.RefreshToken)
	if err != nil {
		t.Fatalf("refresh session: %v", err)
	}
	refreshedClaims := parseAccessClaims(t, service, refreshedToken, now)

	if refreshedClaims.Subject != "user-1" {
		t.Fatalf("unexpected subject: got %q want %q", refreshedClaims.Subject, "user-1")
	}
	if refreshedClaims.TenantID != "tenant-1" {
		t.Fatalf("unexpected tenant: got %q want %q", refreshedClaims.TenantID, "tenant-1")
	}
	if refreshedClaims.ID == initialClaims.ID {
		t.Fatalf("expected new jti, both were %q", refreshedClaims.ID)
	}
	if refreshedClaims.ExpiresAt == nil || !refreshedClaims.ExpiresAt.Time.Equal(now.Add(accessTokenTTL)) {
		t.Fatalf("unexpected expiry: %#v", refreshedClaims.ExpiresAt)
	}
}

func TestServiceRefreshSessionRejectsInvalidOrExpiredToken(t *testing.T) {
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

	issued, err := service.CreateSession(context.Background(), "secret-api-key", "user-1")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := service.RefreshSession(context.Background(), "missing-refresh-token"); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected invalid refresh token error, got %v", err)
	}

	now = now.Add(refreshTokenTTL + time.Minute)
	if _, err := service.RefreshSession(context.Background(), issued.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected invalid refresh token error for expired token, got %v", err)
	}
}

func parseAccessClaims(t *testing.T, service *Service, accessToken string, now time.Time) *AccessClaims {
	t.Helper()

	verifier, err := keyfunc.NewJWKSetJSON(json.RawMessage(service.JWKS()))
	if err != nil {
		t.Fatalf("new jwks verifier: %v", err)
	}
	token, err := jwt.ParseWithClaims(
		accessToken,
		&AccessClaims{},
		verifier.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithTimeFunc(func() time.Time { return now }),
		jwt.WithoutClaimsValidation(),
	)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok {
		t.Fatalf("unexpected claims type: %T", token.Claims)
	}
	return claims
}

type repositoryStub struct {
	apiKeys            []APIKeyRecord
	ensuredTenants     []PersonalTenantRecord
	firstPartyAccounts map[string]FirstPartyAccount
	refreshTokens      map[string]RefreshTokenRecord
	listCalls          int
	savedAccounts      []FirstPartyAccount
	savedRefreshTokens []RefreshTokenRecord
}

func (r *repositoryStub) ListAPIKeys(_ context.Context) ([]APIKeyRecord, error) {
	r.listCalls++
	return append([]APIKeyRecord(nil), r.apiKeys...), nil
}

func (r *repositoryStub) EnsurePersonalTenant(_ context.Context, tenant PersonalTenantRecord) error {
	r.ensuredTenants = append(r.ensuredTenants, tenant)
	return nil
}

func (r *repositoryStub) GetFirstPartyAccount(_ context.Context, firebaseUID string) (FirstPartyAccount, bool, error) {
	if r.firstPartyAccounts == nil {
		return FirstPartyAccount{}, false, nil
	}
	account, ok := r.firstPartyAccounts[firebaseUID]
	return account, ok, nil
}

func (r *repositoryStub) SaveRefreshToken(_ context.Context, token RefreshTokenRecord) error {
	if r.refreshTokens == nil {
		r.refreshTokens = map[string]RefreshTokenRecord{}
	}
	r.refreshTokens[token.RefreshToken] = token
	r.savedRefreshTokens = append(r.savedRefreshTokens, token)
	return nil
}

func (r *repositoryStub) SaveFirstPartyAccount(_ context.Context, account FirstPartyAccount) error {
	if r.firstPartyAccounts == nil {
		r.firstPartyAccounts = map[string]FirstPartyAccount{}
	}
	r.firstPartyAccounts[account.FirebaseUID] = account
	r.savedAccounts = append(r.savedAccounts, account)
	return nil
}

func (r *repositoryStub) GetRefreshToken(_ context.Context, refreshToken string) (RefreshTokenRecord, bool, error) {
	token, ok := r.refreshTokens[refreshToken]
	return token, ok, nil
}

type firebaseVerifierStub struct {
	err   error
	token *VerifiedFirebaseToken
}

func (s firebaseVerifierStub) VerifyIDToken(_ context.Context, _ string) (*VerifiedFirebaseToken, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.token, nil
}

type cacheStub struct {
	entries  map[string]APIKeyIdentity
	getCalls int
	putCalls int
}

func (c *cacheStub) Close() error {
	return nil
}

func (c *cacheStub) GetAPIKeyIdentity(_ context.Context, apiKey string) (APIKeyIdentity, bool, error) {
	c.getCalls++
	identity, ok := c.entries[cacheKey(apiKey)]
	if !ok {
		return APIKeyIdentity{}, false, nil
	}
	return identity, true, nil
}

func (c *cacheStub) PutAPIKeyIdentity(_ context.Context, apiKey string, identity APIKeyIdentity, _ time.Duration) error {
	c.putCalls++
	c.entries[cacheKey(apiKey)] = identity
	return nil
}
