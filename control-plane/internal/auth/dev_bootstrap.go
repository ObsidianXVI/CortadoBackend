package auth

import (
	"context"
	"errors"
	"strings"
)

var reservedFirebaseCustomClaimKeys = map[string]struct{}{
	"acr":       {},
	"amr":       {},
	"at_hash":   {},
	"aud":       {},
	"auth_time": {},
	"azp":       {},
	"cnf":       {},
	"c_hash":    {},
	"exp":       {},
	"firebase":  {},
	"iat":       {},
	"iss":       {},
	"jti":       {},
	"nbf":       {},
	"nonce":     {},
	"sub":       {},
	"user_id":   {},
}

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

	claims := MergeFirebaseClaims(sanitizeFirebaseCustomClaims(token.Claims), map[string]any{
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

func sanitizeFirebaseCustomClaims(claims map[string]any) map[string]any {
	sanitized := make(map[string]any, len(claims))
	for key, value := range claims {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			continue
		}
		if _, reserved := reservedFirebaseCustomClaimKeys[normalized]; reserved {
			continue
		}
		sanitized[normalized] = value
	}
	return sanitized
}
