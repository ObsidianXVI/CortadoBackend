package auth

import (
	"context"
	"errors"
	"strings"
)

var ErrDevBootstrapDisabled = errors.New("dev firebase bootstrap is disabled")

type FirebaseIdentityManager interface {
	FirebaseTokenVerifier
	FirebaseClaimsManager
}

type DevTenantClaimAssignment struct {
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId"`
}

type DevFirebaseBootstrapConfig struct {
	DefaultTenantID string
	Enabled         bool
	Manager         FirebaseClaimsManager
	TenantClaim     string
}

type DevFirebaseBootstrapService struct {
	defaultTenantID string
	enabled         bool
	manager         FirebaseClaimsManager
	tenantClaim     string
}

func NewDevFirebaseBootstrapService(
	cfg DevFirebaseBootstrapConfig,
) (*DevFirebaseBootstrapService, error) {
	if cfg.Manager == nil {
		return nil, errors.New("firebase manager is required")
	}
	if strings.TrimSpace(cfg.TenantClaim) == "" {
		cfg.TenantClaim = "tenant_id"
	}
	if strings.TrimSpace(cfg.DefaultTenantID) == "" {
		cfg.DefaultTenantID = "demo-tenant"
	}

	return &DevFirebaseBootstrapService{
		defaultTenantID: strings.TrimSpace(cfg.DefaultTenantID),
		enabled:         cfg.Enabled,
		manager:         cfg.Manager,
		tenantClaim:     strings.TrimSpace(cfg.TenantClaim),
	}, nil
}

func (s *DevFirebaseBootstrapService) AssignTenantClaim(
	ctx context.Context,
	token *VerifiedFirebaseToken,
	tenantID string,
) (DevTenantClaimAssignment, error) {
	if !s.enabled {
		return DevTenantClaimAssignment{}, ErrDevBootstrapDisabled
	}
	if token == nil || strings.TrimSpace(token.UID) == "" {
		return DevTenantClaimAssignment{}, ErrFirebaseTokenInvalid
	}

	resolvedTenantID := strings.TrimSpace(tenantID)
	if resolvedTenantID == "" {
		resolvedTenantID = s.defaultTenantID
	}

	claims := MergeFirebaseClaims(token.Claims, map[string]any{
		s.tenantClaim: resolvedTenantID,
	})
	if err := s.manager.SetCustomUserClaims(ctx, token.UID, claims); err != nil {
		return DevTenantClaimAssignment{}, err
	}

	return DevTenantClaimAssignment{
		TenantID: resolvedTenantID,
		UserID:   strings.TrimSpace(token.UID),
	}, nil
}
