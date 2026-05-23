package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
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
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRequest     = errors.New("api_key and user_id are required")
)

type Repository interface {
	ListAPIKeys(ctx context.Context) ([]APIKeyRecord, error)
	SaveRefreshToken(ctx context.Context, token RefreshTokenRecord) error
}

type ValidationCache interface {
	Close() error
	GetTenantID(ctx context.Context, apiKey string) (string, bool, error)
	PutTenantID(ctx context.Context, apiKey, tenantID string, ttl time.Duration) error
}

type ServiceConfig struct {
	Cache         ValidationCache
	Now           func() time.Time
	PrivateKeyPEM string
	Repository    Repository
}

type Service struct {
	cache      ValidationCache
	jwksJSON   []byte
	keyID      string
	now        func() time.Time
	privateKey *rsa.PrivateKey
	repository Repository
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
		cache:      cfg.Cache,
		jwksJSON:   jwksJSON,
		keyID:      keyID,
		now:        cfg.Now,
		privateKey: privateKey,
		repository: cfg.Repository,
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

	tenantID, err := s.resolveTenantID(ctx, apiKey)
	if err != nil {
		return SessionTokens{}, err
	}

	now := s.now().UTC()
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
		return SessionTokens{}, fmt.Errorf("sign jwt access token: %w", err)
	}

	refreshToken := uuid.NewString()
	if err := s.repository.SaveRefreshToken(ctx, RefreshTokenRecord{
		CreatedAt:    now,
		ExpiresAt:    now.Add(refreshTokenTTL),
		JTI:          claims.ID,
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

func (s *Service) JWKS() []byte {
	return append([]byte(nil), s.jwksJSON...)
}

func (s *Service) resolveTenantID(ctx context.Context, apiKey string) (string, error) {
	if s.cache != nil {
		tenantID, ok, err := s.cache.GetTenantID(ctx, apiKey)
		if err != nil {
			return "", fmt.Errorf("get validation cache entry: %w", err)
		}
		if ok {
			return tenantID, nil
		}
	}

	apiKeys, err := s.repository.ListAPIKeys(ctx)
	if err != nil {
		return "", fmt.Errorf("list api keys: %w", err)
	}

	for _, record := range apiKeys {
		if record.Revoked || strings.TrimSpace(record.Hash) == "" || strings.TrimSpace(record.TenantID) == "" {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(record.Hash), []byte(apiKey)); err != nil {
			continue
		}
		if s.cache != nil {
			if err := s.cache.PutTenantID(ctx, apiKey, record.TenantID, validationCacheTTL); err != nil {
				return "", fmt.Errorf("write validation cache entry: %w", err)
			}
		}
		return record.TenantID, nil
	}

	return "", ErrInvalidCredentials
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

func (c *redisValidationCache) GetTenantID(ctx context.Context, apiKey string) (string, bool, error) {
	tenantID, err := c.client.Get(ctx, cacheKey(apiKey)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return tenantID, true, nil
}

func (c *redisValidationCache) PutTenantID(ctx context.Context, apiKey, tenantID string, ttl time.Duration) error {
	return c.client.Set(ctx, cacheKey(apiKey), tenantID, ttl).Err()
}

type memoryValidationCache struct {
	mu      sync.Mutex
	entries map[string]memoryValidationCacheEntry
}

type memoryValidationCacheEntry struct {
	expiresAt time.Time
	tenantID  string
}

func newMemoryValidationCache() *memoryValidationCache {
	return &memoryValidationCache{
		entries: map[string]memoryValidationCacheEntry{},
	}
}

func (c *memoryValidationCache) Close() error {
	return nil
}

func (c *memoryValidationCache) GetTenantID(_ context.Context, apiKey string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[cacheKey(apiKey)]
	if !ok {
		return "", false, nil
	}
	if time.Now().UTC().After(entry.expiresAt) {
		delete(c.entries, cacheKey(apiKey))
		return "", false, nil
	}
	return entry.tenantID, true, nil
}

func (c *memoryValidationCache) PutTenantID(_ context.Context, apiKey, tenantID string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[cacheKey(apiKey)] = memoryValidationCacheEntry{
		expiresAt: time.Now().UTC().Add(ttl),
		tenantID:  tenantID,
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
