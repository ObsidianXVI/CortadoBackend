package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrPlatformTenantNotFound = errors.New("platform tenant not found")

type PlatformTenantRepository interface {
	GetPlatformTenant(ctx context.Context, tenantID string) (PlatformTenantRecord, bool, error)
	ListPlatformTenants(ctx context.Context, ownerUserID string) ([]PlatformTenantRecord, error)
	SavePlatformTenant(ctx context.Context, tenant PlatformTenantRecord) error
}

type PlatformTenantServiceConfig struct {
	APIKeys     *APIKeyService
	IDGenerator func() string
	Now         func() time.Time
	Repository  PlatformTenantRepository
}

type PlatformTenantService struct {
	apiKeys     *APIKeyService
	idGenerator func() string
	now         func() time.Time
	repository  PlatformTenantRepository
}

func NewPlatformTenantService(cfg PlatformTenantServiceConfig) (*PlatformTenantService, error) {
	if cfg.APIKeys == nil {
		return nil, errors.New("api key service is required")
	}
	if cfg.Repository == nil {
		return nil, errors.New("repository is required")
	}
	if cfg.IDGenerator == nil {
		cfg.IDGenerator = func() string {
			return "platform-" + uuid.NewString()
		}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	return &PlatformTenantService{
		apiKeys:     cfg.APIKeys,
		idGenerator: cfg.IDGenerator,
		now:         cfg.Now,
		repository:  cfg.Repository,
	}, nil
}

func (s *PlatformTenantService) CreatePlatformTenant(
	ctx context.Context,
	ownerUserID, displayName string,
) (PlatformTenant, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return PlatformTenant{}, ErrInvalidRequest
	}

	now := s.now().UTC()
	record := PlatformTenantRecord{
		CreatedAt:   now,
		DisplayName: strings.TrimSpace(displayName),
		Kind:        APIKeyKindPlatform,
		OwnerUserID: ownerUserID,
		TenantID:    s.idGenerator(),
		UpdatedAt:   now,
	}
	if err := s.repository.SavePlatformTenant(ctx, record); err != nil {
		return PlatformTenant{}, fmt.Errorf("save platform tenant: %w", err)
	}
	return record.Metadata(), nil
}

func (s *PlatformTenantService) ListPlatformTenants(
	ctx context.Context,
	ownerUserID string,
) ([]PlatformTenant, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, ErrInvalidRequest
	}

	records, err := s.repository.ListPlatformTenants(ctx, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list platform tenants: %w", err)
	}

	tenants := make([]PlatformTenant, 0, len(records))
	for _, record := range records {
		tenants = append(tenants, record.Metadata())
	}
	return tenants, nil
}

func (s *PlatformTenantService) IssuePlatformAPIKey(
	ctx context.Context,
	ownerUserID, tenantID string,
) (IssuedAPIKey, error) {
	if _, err := s.ownedTenant(ctx, ownerUserID, tenantID); err != nil {
		return IssuedAPIKey{}, err
	}
	issued, err := s.apiKeys.IssuePlatformAPIKey(ctx, tenantID, ownerUserID)
	if err != nil {
		return IssuedAPIKey{}, fmt.Errorf("issue platform api key: %w", err)
	}
	return issued, nil
}

func (s *PlatformTenantService) ListPlatformAPIKeys(
	ctx context.Context,
	ownerUserID, tenantID string,
) ([]APIKey, error) {
	if _, err := s.ownedTenant(ctx, ownerUserID, tenantID); err != nil {
		return nil, err
	}
	keys, err := s.apiKeys.ListPlatformAPIKeys(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list platform api keys: %w", err)
	}
	return keys, nil
}

func (s *PlatformTenantService) RevokePlatformAPIKey(
	ctx context.Context,
	ownerUserID, tenantID, keyID string,
) (APIKey, error) {
	if _, err := s.ownedTenant(ctx, ownerUserID, tenantID); err != nil {
		return APIKey{}, err
	}
	key, err := s.apiKeys.RevokePlatformAPIKey(ctx, tenantID, keyID)
	if err != nil {
		return APIKey{}, fmt.Errorf("revoke platform api key: %w", err)
	}
	return key, nil
}

func (s *PlatformTenantService) ownedTenant(
	ctx context.Context,
	ownerUserID, tenantID string,
) (PlatformTenantRecord, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	tenantID = strings.TrimSpace(tenantID)
	if ownerUserID == "" || tenantID == "" {
		return PlatformTenantRecord{}, ErrInvalidRequest
	}

	record, found, err := s.repository.GetPlatformTenant(ctx, tenantID)
	if err != nil {
		return PlatformTenantRecord{}, fmt.Errorf("get platform tenant: %w", err)
	}
	if !found || strings.TrimSpace(record.OwnerUserID) != ownerUserID {
		return PlatformTenantRecord{}, ErrPlatformTenantNotFound
	}
	if strings.TrimSpace(record.Kind) == "" {
		record.Kind = APIKeyKindPlatform
	}
	return record, nil
}
