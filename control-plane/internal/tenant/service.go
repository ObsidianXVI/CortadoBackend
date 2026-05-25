package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
)

const authProviderTypeOIDC = "oidc"

var (
	ErrInvalidRequest = errors.New("invalid tenant auth provider configuration")
	ErrNotFound       = errors.New("tenant auth provider not found")
)

var supportedSigningAlgorithms = map[string]struct{}{
	jwt.SigningMethodRS256.Alg(): {},
	jwt.SigningMethodRS384.Alg(): {},
	jwt.SigningMethodRS512.Alg(): {},
	jwt.SigningMethodPS256.Alg(): {},
	jwt.SigningMethodPS384.Alg(): {},
	jwt.SigningMethodPS512.Alg(): {},
	jwt.SigningMethodES256.Alg(): {},
	jwt.SigningMethodES384.Alg(): {},
	jwt.SigningMethodES512.Alg(): {},
	jwt.SigningMethodEdDSA.Alg(): {},
}

type Repository interface {
	GetMetadata(ctx context.Context, tenantID string) (Metadata, bool, error)
	SaveMetadata(ctx context.Context, metadata Metadata) error
}

type ServiceConfig struct {
	HTTPClient *http.Client
	Now        func() time.Time
	Repository Repository
}

type Service struct {
	httpClient *http.Client
	now        func() time.Time
	repository Repository
}

type Metadata struct {
	AuthProvider *AuthProviderConfig `firestore:"authProvider,omitempty" json:"authProvider,omitempty"`
	TenantID     string              `firestore:"tenantId" json:"tenantId"`
	UpdatedAt    time.Time           `firestore:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type AuthProviderConfig struct {
	AllowedAudiences         []string           `firestore:"allowedAudiences" json:"allowedAudiences"`
	AllowedSigningAlgorithms []string           `firestore:"allowedSigningAlgorithms" json:"allowedSigningAlgorithms"`
	ClaimRequirements        []ClaimRequirement `firestore:"claimRequirements,omitempty" json:"claimRequirements,omitempty"`
	DiscoveryURL             string             `firestore:"discoveryUrl,omitempty" json:"discoveryUrl,omitempty"`
	Issuer                   string             `firestore:"issuer" json:"issuer"`
	JWKSURI                  string             `firestore:"jwksUri" json:"jwksUri"`
	Type                     string             `firestore:"type" json:"type"`
	UserIDClaim              string             `firestore:"userIdClaim" json:"userIdClaim"`
}

type ClaimRequirement struct {
	AnyOf []string `firestore:"anyOf" json:"anyOf"`
	Claim string   `firestore:"claim" json:"claim"`
}

type UpsertAuthProviderInput struct {
	AllowedAudiences         []string
	AllowedSigningAlgorithms []string
	ClaimRequirements        []ClaimRequirement
	DiscoveryURL             string
	Issuer                   string
	JWKSURI                  string
	UserIDClaim              string
}

type oidcDiscoveryDocument struct {
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	Issuer                           string   `json:"issuer"`
	JWKSURI                          string   `json:"jwks_uri"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Alg string `json:"alg"`
	Use string `json:"use"`
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Repository == nil {
		return nil, errors.New("repository is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	return &Service{
		httpClient: cfg.HTTPClient,
		now:        cfg.Now,
		repository: cfg.Repository,
	}, nil
}

func (s *Service) GetAuthProvider(ctx context.Context, tenantID string) (AuthProviderConfig, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return AuthProviderConfig{}, fmt.Errorf("%w: tenant id is required", ErrInvalidRequest)
	}

	metadata, found, err := s.repository.GetMetadata(ctx, tenantID)
	if err != nil {
		return AuthProviderConfig{}, fmt.Errorf("get tenant metadata: %w", err)
	}
	if !found || metadata.AuthProvider == nil {
		return AuthProviderConfig{}, ErrNotFound
	}

	return *metadata.AuthProvider, nil
}

func (s *Service) PutAuthProvider(ctx context.Context, tenantID string, input UpsertAuthProviderInput) (AuthProviderConfig, bool, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return AuthProviderConfig{}, false, fmt.Errorf("%w: tenant id is required", ErrInvalidRequest)
	}

	config, err := s.validateInput(ctx, input)
	if err != nil {
		return AuthProviderConfig{}, false, err
	}

	metadata, found, err := s.repository.GetMetadata(ctx, tenantID)
	if err != nil {
		return AuthProviderConfig{}, false, fmt.Errorf("get tenant metadata: %w", err)
	}
	created := !found || metadata.AuthProvider == nil
	metadata.TenantID = tenantID
	metadata.AuthProvider = &config
	metadata.UpdatedAt = s.now().UTC()

	if err := s.repository.SaveMetadata(ctx, metadata); err != nil {
		return AuthProviderConfig{}, false, fmt.Errorf("save tenant metadata: %w", err)
	}

	return config, created, nil
}

func (s *Service) DeleteAuthProvider(ctx context.Context, tenantID string) error {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return fmt.Errorf("%w: tenant id is required", ErrInvalidRequest)
	}

	metadata, found, err := s.repository.GetMetadata(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("get tenant metadata: %w", err)
	}
	if !found || metadata.AuthProvider == nil {
		return ErrNotFound
	}

	metadata.AuthProvider = nil
	metadata.UpdatedAt = s.now().UTC()
	if err := s.repository.SaveMetadata(ctx, metadata); err != nil {
		return fmt.Errorf("save tenant metadata: %w", err)
	}

	return nil
}

func (s *Service) validateInput(ctx context.Context, input UpsertAuthProviderInput) (AuthProviderConfig, error) {
	discoveryURL := strings.TrimSpace(input.DiscoveryURL)
	issuer := strings.TrimSpace(input.Issuer)
	jwksURI := strings.TrimSpace(input.JWKSURI)

	if discoveryURL != "" && (issuer != "" || jwksURI != "") {
		return AuthProviderConfig{}, fmt.Errorf("%w: configure either discoveryUrl or issuer+jwksUri", ErrInvalidRequest)
	}
	if discoveryURL == "" && (issuer == "" || jwksURI == "") {
		return AuthProviderConfig{}, fmt.Errorf("%w: issuer and jwksUri are required when discoveryUrl is not provided", ErrInvalidRequest)
	}

	allowedAudiences, err := normalizeNonEmptyList(input.AllowedAudiences, "allowedAudiences")
	if err != nil {
		return AuthProviderConfig{}, err
	}
	allowedAlgs, err := normalizeSigningAlgorithms(input.AllowedSigningAlgorithms)
	if err != nil {
		return AuthProviderConfig{}, err
	}
	userIDClaim, err := normalizeClaimName(input.UserIDClaim, "userIdClaim", true)
	if err != nil {
		return AuthProviderConfig{}, err
	}
	if userIDClaim == "" {
		userIDClaim = "sub"
	}
	claimRequirements, err := normalizeClaimRequirements(input.ClaimRequirements)
	if err != nil {
		return AuthProviderConfig{}, err
	}

	var discovery oidcDiscoveryDocument
	if discoveryURL != "" {
		if _, err := parseHTTPSURL(discoveryURL, "discoveryUrl"); err != nil {
			return AuthProviderConfig{}, err
		}

		discovery, err = s.fetchDiscovery(ctx, discoveryURL)
		if err != nil {
			return AuthProviderConfig{}, err
		}
		if len(discovery.IDTokenSigningAlgValuesSupported) > 0 {
			discoveryAlgs := make(map[string]struct{}, len(discovery.IDTokenSigningAlgValuesSupported))
			for _, alg := range discovery.IDTokenSigningAlgValuesSupported {
				discoveryAlgs[strings.TrimSpace(alg)] = struct{}{}
			}
			for _, alg := range allowedAlgs {
				if _, ok := discoveryAlgs[alg]; !ok {
					return AuthProviderConfig{}, fmt.Errorf("%w: allowed signing algorithm %q is not advertised by discovery metadata", ErrInvalidRequest, alg)
				}
			}
		}
		issuer = strings.TrimSpace(discovery.Issuer)
		jwksURI = strings.TrimSpace(discovery.JWKSURI)
	}

	if _, err := parseHTTPSURL(issuer, "issuer"); err != nil {
		return AuthProviderConfig{}, err
	}
	if _, err := parseHTTPSURL(jwksURI, "jwksUri"); err != nil {
		return AuthProviderConfig{}, err
	}
	if err := s.validateJWKS(ctx, jwksURI, allowedAlgs); err != nil {
		return AuthProviderConfig{}, err
	}

	return AuthProviderConfig{
		AllowedAudiences:         allowedAudiences,
		AllowedSigningAlgorithms: allowedAlgs,
		ClaimRequirements:        claimRequirements,
		DiscoveryURL:             discoveryURL,
		Issuer:                   issuer,
		JWKSURI:                  jwksURI,
		Type:                     authProviderTypeOIDC,
		UserIDClaim:              userIDClaim,
	}, nil
}

func (s *Service) fetchDiscovery(ctx context.Context, discoveryURL string) (oidcDiscoveryDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("build discovery request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("%w: fetch discovery document: %v", ErrInvalidRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oidcDiscoveryDocument{}, fmt.Errorf("%w: discovery document returned status %d", ErrInvalidRequest, resp.StatusCode)
	}

	var document oidcDiscoveryDocument
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&document); err != nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("%w: decode discovery document: %v", ErrInvalidRequest, err)
	}
	if strings.TrimSpace(document.Issuer) == "" || strings.TrimSpace(document.JWKSURI) == "" {
		return oidcDiscoveryDocument{}, fmt.Errorf("%w: discovery document must include issuer and jwks_uri", ErrInvalidRequest)
	}

	return document, nil
}

func (s *Service) validateJWKS(ctx context.Context, jwksURI string, allowedAlgorithms []string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return fmt.Errorf("build jwks request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: fetch jwks document: %v", ErrInvalidRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: jwks document returned status %d", ErrInvalidRequest, resp.StatusCode)
	}

	var document jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return fmt.Errorf("%w: decode jwks document: %v", ErrInvalidRequest, err)
	}
	if len(document.Keys) == 0 {
		return fmt.Errorf("%w: jwks document must include at least one key", ErrInvalidRequest)
	}

	allowed := make(map[string]struct{}, len(allowedAlgorithms))
	for _, alg := range allowedAlgorithms {
		allowed[alg] = struct{}{}
	}

	usableKeyFound := false
	algHintSeen := false
	algMatched := false
	for _, key := range document.Keys {
		if key.Use != "" && key.Use != "sig" {
			continue
		}
		usableKeyFound = true
		if strings.TrimSpace(key.Alg) == "" {
			continue
		}
		algHintSeen = true
		if _, ok := allowed[strings.TrimSpace(key.Alg)]; ok {
			algMatched = true
		}
	}
	if !usableKeyFound {
		return fmt.Errorf("%w: jwks document does not expose any signature keys", ErrInvalidRequest)
	}
	if algHintSeen && !algMatched {
		return fmt.Errorf("%w: jwks document does not advertise any allowed signing algorithm", ErrInvalidRequest)
	}

	return nil
}

func normalizeNonEmptyList(values []string, field string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if containsWhitespace(trimmed) {
			return nil, fmt.Errorf("%w: %s entries must not contain whitespace", ErrInvalidRequest, field)
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("%w: %s must include at least one value", ErrInvalidRequest, field)
	}
	return normalized, nil
}

func normalizeSigningAlgorithms(values []string) ([]string, error) {
	normalized, err := normalizeNonEmptyList(values, "allowedSigningAlgorithms")
	if err != nil {
		return nil, err
	}
	for _, alg := range normalized {
		if _, ok := supportedSigningAlgorithms[alg]; !ok {
			return nil, fmt.Errorf("%w: unsupported signing algorithm %q", ErrInvalidRequest, alg)
		}
	}
	return normalized, nil
}

func normalizeClaimRequirements(values []ClaimRequirement) ([]ClaimRequirement, error) {
	normalized := make([]ClaimRequirement, 0, len(values))
	for _, requirement := range values {
		claim, err := normalizeClaimName(requirement.Claim, "claimRequirements.claim", false)
		if err != nil {
			return nil, err
		}
		anyOf, err := normalizeNonEmptyList(requirement.AnyOf, "claimRequirements.anyOf")
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, ClaimRequirement{
			AnyOf: anyOf,
			Claim: claim,
		})
	}
	return normalized, nil
}

func normalizeClaimName(value, field string, allowEmpty bool) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("%w: %s is required", ErrInvalidRequest, field)
	}
	if containsWhitespace(trimmed) {
		return "", fmt.Errorf("%w: %s must not contain whitespace", ErrInvalidRequest, field)
	}
	return trimmed, nil
}

func parseHTTPSURL(rawValue, field string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawValue))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid %s: %v", ErrInvalidRequest, field, err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: %s must be an https URL", ErrInvalidRequest, field)
	}
	return parsed, nil
}

func containsWhitespace(value string) bool {
	return strings.ContainsFunc(value, unicode.IsSpace)
}
