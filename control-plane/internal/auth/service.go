package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL     = 8 * time.Hour
	refreshTokenTTL    = 30 * 24 * time.Hour
	validationCacheTTL = 5 * time.Minute
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRequest      = errors.New("api_key and user_id are required")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidRefreshInput = errors.New("refresh_token is required")
)

type Repository interface {
	EnsurePersonalTenant(ctx context.Context, tenant PersonalTenantRecord) error
	GetFirstPartyAccount(ctx context.Context, firebaseUID string) (FirstPartyAccount, bool, error)
	ListAPIKeys(ctx context.Context) ([]APIKeyRecord, error)
	GetRefreshToken(ctx context.Context, refreshToken string) (RefreshTokenRecord, bool, error)
	SaveRefreshToken(ctx context.Context, token RefreshTokenRecord) error
	SaveFirstPartyAccount(ctx context.Context, account FirstPartyAccount) error
}

type ValidationCache interface {
	Close() error
	GetAPIKeyIdentity(ctx context.Context, apiKey string) (APIKeyIdentity, bool, error)
	PutAPIKeyIdentity(ctx context.Context, apiKey string, identity APIKeyIdentity, ttl time.Duration) error
}

type ServiceConfig struct {
	Cache            ValidationCache
	FirebaseVerifier FirebaseTokenVerifier
	Now              func() time.Time
	PrivateKeyPEM    string
	Repository       Repository
}

type Service struct {
	cache            ValidationCache
	firebaseVerifier FirebaseTokenVerifier
	jwksJSON         []byte
	keyID            string
	now              func() time.Time
	privateKey       *rsa.PrivateKey
	repository       Repository
}

type AccessClaims struct {
	TenantID string `json:"tid"`
	jwt.RegisteredClaims
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Repository == nil {
		return nil, errors.New("repository is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if strings.TrimSpace(cfg.PrivateKeyPEM) == "" {
		return nil, errors.New("jwt private key is required")
	}
	if cfg.Cache == nil {
		cfg.Cache = newMemoryValidationCache()
	}

	privateKey, err := parsePrivateKey(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse jwt private key: %w", err)
	}
	keyID, err := publicKeyID(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("derive jwt key id: %w", err)
	}
	jwksJSON, err := jwksForPublicKey(&privateKey.PublicKey, keyID)
	if err != nil {
		return nil, fmt.Errorf("encode jwks: %w", err)
	}

	return &Service{
		cache:            cfg.Cache,
		firebaseVerifier: cfg.FirebaseVerifier,
		jwksJSON:         jwksJSON,
		keyID:            keyID,
		now:              cfg.Now,
		privateKey:       privateKey,
		repository:       cfg.Repository,
	}, nil
}

func NewValidationCacheFromEnv() ValidationCache {
	addr := strings.TrimSpace(os.Getenv("CORTADO_AUTH_CACHE_ADDR"))
	if addr == "" {
		return newMemoryValidationCache()
	}

	return &redisValidationCache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: os.Getenv("CORTADO_AUTH_CACHE_PASSWORD"),
		}),
	}
}

func (s *Service) Close() error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Close()
}

func (s *Service) CreateSession(ctx context.Context, apiKey, userID string) (SessionTokens, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(userID) == "" {
		return SessionTokens{}, ErrInvalidRequest
	}

	identity, err := s.resolveAPIKeyIdentity(ctx, apiKey)
	if err != nil {
		return SessionTokens{}, err
	}
	if identity.UserID != "" && identity.UserID != strings.TrimSpace(userID) {
		return SessionTokens{}, ErrInvalidCredentials
	}

	return s.createSessionTokens(ctx, identity.TenantID, userID)
}

func (s *Service) ExchangeFirebaseSession(ctx context.Context, idToken string) (SessionTokens, error) {
	if s.firebaseVerifier == nil {
		return SessionTokens{}, errors.New("firebase verifier is not configured")
	}

	verified, err := s.firebaseVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return SessionTokens{}, err
	}

	account, err := s.resolveFirstPartyAccount(ctx, verified)
	if err != nil {
		return SessionTokens{}, err
	}

	return s.createSessionTokens(ctx, account.PersonalTenantID, account.UserID)
}

func (s *Service) RefreshSession(ctx context.Context, refreshToken string) (string, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return "", ErrInvalidRefreshInput
	}

	record, found, err := s.repository.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", fmt.Errorf("get refresh token: %w", err)
	}
	if !found {
		return "", ErrInvalidRefreshToken
	}

	now := s.now().UTC()
	if !now.Before(record.ExpiresAt.UTC()) || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.UserID) == "" {
		return "", ErrInvalidRefreshToken
	}

	accessToken, _, err := s.issueAccessToken(now, record.TenantID, record.UserID)
	if err != nil {
		return "", err
	}
	return accessToken, nil
}

func (s *Service) JWKS() []byte {
	return append([]byte(nil), s.jwksJSON...)
}

func (s *Service) resolveAPIKeyIdentity(ctx context.Context, apiKey string) (APIKeyIdentity, error) {
	if s.cache != nil {
		identity, ok, err := s.cache.GetAPIKeyIdentity(ctx, apiKey)
		if err != nil {
			return APIKeyIdentity{}, fmt.Errorf("get validation cache entry: %w", err)
		}
		if ok {
			return identity, nil
		}
	}

	apiKeys, err := s.repository.ListAPIKeys(ctx)
	if err != nil {
		return APIKeyIdentity{}, fmt.Errorf("list api keys: %w", err)
	}

	for _, record := range apiKeys {
		if record.Revoked || strings.TrimSpace(record.Hash) == "" || strings.TrimSpace(record.TenantID) == "" {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(record.Hash), []byte(apiKey)); err != nil {
			continue
		}
		identity := APIKeyIdentity{
			TenantID: record.TenantID,
			UserID:   strings.TrimSpace(record.UserID),
		}
		if s.cache != nil {
			if err := s.cache.PutAPIKeyIdentity(ctx, apiKey, identity, validationCacheTTL); err != nil {
				return APIKeyIdentity{}, fmt.Errorf("write validation cache entry: %w", err)
			}
		}
		return identity, nil
	}

	return APIKeyIdentity{}, ErrInvalidCredentials
}

func (s *Service) resolveFirstPartyAccount(ctx context.Context, token *VerifiedFirebaseToken) (FirstPartyAccount, error) {
	if token == nil || strings.TrimSpace(token.UID) == "" {
		return FirstPartyAccount{}, ErrFirebaseTokenInvalid
	}

	now := s.now().UTC()
	firebaseUID := strings.TrimSpace(token.UID)
	account, found, err := s.repository.GetFirstPartyAccount(ctx, firebaseUID)
	if err != nil {
		return FirstPartyAccount{}, fmt.Errorf("get first-party account: %w", err)
	}

	if found {
		updated := false
		if strings.TrimSpace(account.FirebaseUID) == "" {
			account.FirebaseUID = firebaseUID
			updated = true
		}
		if strings.TrimSpace(account.UserID) == "" {
			account.UserID = firstPartyUserID(firebaseUID)
			updated = true
		}
		if strings.TrimSpace(account.PersonalTenantID) == "" {
			account.PersonalTenantID = firstPartyTenantID(firebaseUID)
			updated = true
		}
		if account.CreatedAt.IsZero() {
			account.CreatedAt = now
			updated = true
		}
		if account.Email == "" {
			if email := firebaseStringClaim(token.Claims, "email"); email != "" {
				account.Email = email
				updated = true
			}
		}
		if account.DisplayName == "" {
			if displayName := firebaseDisplayName(token.Claims); displayName != "" {
				account.DisplayName = displayName
				updated = true
			}
		}
		if updated {
			account.UpdatedAt = now
			if err := s.repository.SaveFirstPartyAccount(ctx, account); err != nil {
				return FirstPartyAccount{}, fmt.Errorf("save first-party account: %w", err)
			}
		}
	} else {
		account = FirstPartyAccount{
			CreatedAt:        now,
			DisplayName:      firebaseDisplayName(token.Claims),
			Email:            firebaseStringClaim(token.Claims, "email"),
			FirebaseUID:      firebaseUID,
			PersonalTenantID: firstPartyTenantID(firebaseUID),
			UpdatedAt:        now,
			UserID:           firstPartyUserID(firebaseUID),
		}
		if err := s.repository.SaveFirstPartyAccount(ctx, account); err != nil {
			return FirstPartyAccount{}, fmt.Errorf("save first-party account: %w", err)
		}
	}

	if err := s.repository.EnsurePersonalTenant(ctx, personalTenantRecord(account, now)); err != nil {
		return FirstPartyAccount{}, fmt.Errorf("ensure personal tenant: %w", err)
	}

	return account, nil
}

func (s *Service) createSessionTokens(ctx context.Context, tenantID, userID string) (SessionTokens, error) {
	now := s.now().UTC()
	accessToken, jti, err := s.issueAccessToken(now, tenantID, userID)
	if err != nil {
		return SessionTokens{}, err
	}

	refreshToken := uuid.NewString()
	if err := s.repository.SaveRefreshToken(ctx, RefreshTokenRecord{
		CreatedAt:    now,
		ExpiresAt:    now.Add(refreshTokenTTL),
		JTI:          jti,
		RefreshToken: refreshToken,
		TenantID:     tenantID,
		UserID:       userID,
	}); err != nil {
		return SessionTokens{}, fmt.Errorf("save refresh token: %w", err)
	}

	return SessionTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func personalTenantRecord(account FirstPartyAccount, now time.Time) PersonalTenantRecord {
	createdAt := account.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}

	return PersonalTenantRecord{
		CreatedAt:   createdAt,
		DisplayName: account.DisplayName,
		Kind:        "personal",
		OwnerUserID: account.UserID,
		TenantID:    account.PersonalTenantID,
		UpdatedAt:   now,
	}
}

func firebaseDisplayName(claims map[string]any) string {
	for _, key := range []string{"name", "display_name"} {
		if value := firebaseStringClaim(claims, key); value != "" {
			return value
		}
	}
	return ""
}

func firebaseStringClaim(claims map[string]any, key string) string {
	if claims == nil {
		return ""
	}
	value, ok := claims[key]
	if !ok {
		return ""
	}
	asString, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(asString)
}

func firstPartyUserID(firebaseUID string) string {
	return "user-" + stableIdentitySuffix("user:"+firebaseUID)
}

func firstPartyTenantID(firebaseUID string) string {
	return "tenant-" + stableIdentitySuffix("tenant:"+firebaseUID)
}

func stableIdentitySuffix(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:10])
}

func (s *Service) issueAccessToken(now time.Time, tenantID, userID string) (string, string, error) {
	claims := AccessClaims{
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID

	accessToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("sign jwt access token: %w", err)
	}
	return accessToken, claims.ID, nil
}

func parsePrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("decode pem block")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("expected RSA private key")
	}
	return privateKey, nil
}

func publicKeyID(publicKey *rsa.PublicKey) (string, error) {
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(publicKeyDER)
	return base64.RawURLEncoding.EncodeToString(sum[:8]), nil
}

func jwksForPublicKey(publicKey *rsa.PublicKey, keyID string) ([]byte, error) {
	type jwkKey struct {
		Alg string `json:"alg"`
		E   string `json:"e"`
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		N   string `json:"n"`
		Use string `json:"use"`
	}

	payload := struct {
		Keys []jwkKey `json:"keys"`
	}{
		Keys: []jwkKey{
			{
				Alg: "RS256",
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
				Kid: keyID,
				Kty: "RSA",
				N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				Use: "sig",
			},
		},
	}

	return json.Marshal(payload)
}

type redisValidationCache struct {
	client *redis.Client
}

func (c *redisValidationCache) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *redisValidationCache) GetAPIKeyIdentity(ctx context.Context, apiKey string) (APIKeyIdentity, bool, error) {
	payload, err := c.client.Get(ctx, cacheKey(apiKey)).Result()
	if errors.Is(err, redis.Nil) {
		return APIKeyIdentity{}, false, nil
	}
	if err != nil {
		return APIKeyIdentity{}, false, err
	}
	var identity APIKeyIdentity
	if err := json.Unmarshal([]byte(payload), &identity); err != nil {
		return APIKeyIdentity{}, false, err
	}
	return identity, true, nil
}

func (c *redisValidationCache) PutAPIKeyIdentity(ctx context.Context, apiKey string, identity APIKeyIdentity, ttl time.Duration) error {
	payload, err := json.Marshal(identity)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, cacheKey(apiKey), payload, ttl).Err()
}

type memoryValidationCache struct {
	mu      sync.Mutex
	entries map[string]memoryValidationCacheEntry
}

type memoryValidationCacheEntry struct {
	expiresAt time.Time
	identity  APIKeyIdentity
}

func newMemoryValidationCache() *memoryValidationCache {
	return &memoryValidationCache{
		entries: map[string]memoryValidationCacheEntry{},
	}
}

func (c *memoryValidationCache) Close() error {
	return nil
}

func (c *memoryValidationCache) GetAPIKeyIdentity(_ context.Context, apiKey string) (APIKeyIdentity, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[cacheKey(apiKey)]
	if !ok {
		return APIKeyIdentity{}, false, nil
	}
	if time.Now().UTC().After(entry.expiresAt) {
		delete(c.entries, cacheKey(apiKey))
		return APIKeyIdentity{}, false, nil
	}
	return entry.identity, true, nil
}

func (c *memoryValidationCache) PutAPIKeyIdentity(_ context.Context, apiKey string, identity APIKeyIdentity, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[cacheKey(apiKey)] = memoryValidationCacheEntry{
		expiresAt: time.Now().UTC().Add(ttl),
		identity:  identity,
	}
	return nil
}

func cacheKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return "auth_api_key:" + base64.RawURLEncoding.EncodeToString(sum[:])
}

func GenerateRSAKeyPEM() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	bytes := x509.MarshalPKCS1PrivateKey(privateKey)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: bytes})), nil
}
