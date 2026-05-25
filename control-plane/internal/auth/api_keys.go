package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAPIKeyEntropyBytes = 32
	defaultAPIKeyPrefix       = "cortado_"
)

var (
	ErrAPIKeyNotFound       = errors.New("api key not found")
	ErrTenantClaimMissing   = errors.New("firebase tenant claim is required")
	ErrFirebaseTokenInvalid = errors.New("invalid firebase id token")
	ErrFirebaseTokenMissing = errors.New("firebase id token is required")
)

type APIKeyRepository interface {
	ListAPIKeys(ctx context.Context) ([]APIKeyRecord, error)
	SaveAPIKey(ctx context.Context, record APIKeyRecord) error
}

type APIKeyServiceConfig struct {
	BcryptCost   int
	EntropyBytes int
	Now          func() time.Time
	Prefix       string
	Repository   APIKeyRepository
}

type APIKeyService struct {
	bcryptCost   int
	entropyBytes int
	now          func() time.Time
	prefix       string
	repository   APIKeyRepository
}

func NewAPIKeyService(cfg APIKeyServiceConfig) (*APIKeyService, error) {
	if cfg.Repository == nil {
		return nil, errors.New("repository is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.BcryptCost == 0 {
		cfg.BcryptCost = 12
	}
	if cfg.EntropyBytes == 0 {
		cfg.EntropyBytes = defaultAPIKeyEntropyBytes
	}
	if cfg.EntropyBytes < 16 {
		return nil, errors.New("entropy bytes must be at least 16")
	}
	if cfg.BcryptCost < bcrypt.MinCost || cfg.BcryptCost > bcrypt.MaxCost {
		return nil, errors.New("bcrypt cost is out of range")
	}
	if strings.TrimSpace(cfg.Prefix) == "" {
		cfg.Prefix = defaultAPIKeyPrefix
	}

	return &APIKeyService{
		bcryptCost:   cfg.BcryptCost,
		entropyBytes: cfg.EntropyBytes,
		now:          cfg.Now,
		prefix:       cfg.Prefix,
		repository:   cfg.Repository,
	}, nil
}

func (s *APIKeyService) IssueAPIKey(ctx context.Context, tenantID, userID string) (IssuedAPIKey, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(userID) == "" {
		return IssuedAPIKey{}, ErrInvalidRequest
	}

	rawKey, err := s.generateRawKey()
	if err != nil {
		return IssuedAPIKey{}, fmt.Errorf("generate api key: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), s.bcryptCost)
	if err != nil {
		return IssuedAPIKey{}, fmt.Errorf("hash api key: %w", err)
	}

	now := s.now().UTC()
	record := APIKeyRecord{
		ID:        uuid.NewString(),
		Hash:      string(hash),
		Revoked:   false,
		TenantID:  strings.TrimSpace(tenantID),
		UserID:    strings.TrimSpace(userID),
		CreatedAt: now,
	}
	if err := s.repository.SaveAPIKey(ctx, record); err != nil {
		return IssuedAPIKey{}, fmt.Errorf("save api key: %w", err)
	}

	return IssuedAPIKey{
		APIKey: rawKey,
		Record: record.Metadata(),
	}, nil
}

func (s *APIKeyService) ListAPIKeys(ctx context.Context, tenantID, userID string) ([]APIKey, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidRequest
	}

	records, err := s.repository.ListAPIKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	filtered := make([]APIKey, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.TenantID) != strings.TrimSpace(tenantID) {
			continue
		}
		if strings.TrimSpace(record.UserID) != strings.TrimSpace(userID) {
			continue
		}
		filtered = append(filtered, record.Metadata())
	}

	slices.SortFunc(filtered, func(left, right APIKey) int {
		return right.CreatedAt.Compare(left.CreatedAt)
	})
	return filtered, nil
}

func (s *APIKeyService) RevokeAPIKey(ctx context.Context, tenantID, userID, keyID string) (APIKey, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(keyID) == "" {
		return APIKey{}, ErrInvalidRequest
	}

	records, err := s.repository.ListAPIKeys(ctx)
	if err != nil {
		return APIKey{}, fmt.Errorf("list api keys: %w", err)
	}

	for _, record := range records {
		if record.ID != strings.TrimSpace(keyID) {
			continue
		}
		if strings.TrimSpace(record.TenantID) != strings.TrimSpace(tenantID) || strings.TrimSpace(record.UserID) != strings.TrimSpace(userID) {
			return APIKey{}, ErrAPIKeyNotFound
		}
		record.Revoked = true
		if err := s.repository.SaveAPIKey(ctx, record); err != nil {
			return APIKey{}, fmt.Errorf("save api key: %w", err)
		}
		return record.Metadata(), nil
	}

	return APIKey{}, ErrAPIKeyNotFound
}

func (s *APIKeyService) generateRawKey() (string, error) {
	bytes := make([]byte, s.entropyBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return s.prefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}
