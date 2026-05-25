package tenant

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServicePutAuthProviderUsesDiscoveryMetadata(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"issuer":"` + server.URL + `",
				"jwks_uri":"` + server.URL + `/jwks",
				"id_token_signing_alg_values_supported":["RS256","ES256"]
			}`))
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[{"kty":"RSA","use":"sig","kid":"key-1","alg":"RS256"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repository := &memoryRepository{}
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	service, err := NewService(ServiceConfig{
		HTTPClient: server.Client(),
		Now: func() time.Time {
			return now
		},
		Repository: repository,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	config, created, err := service.PutAuthProvider(context.Background(), "tenant-1", UpsertAuthProviderInput{
		AllowedAudiences:         []string{"client-1", "client-2", "client-1"},
		AllowedSigningAlgorithms: []string{"RS256", "RS256"},
		ClaimRequirements: []ClaimRequirement{
			{Claim: "https://example.com/claims/group", AnyOf: []string{"admin", "editor", "admin"}},
		},
		DiscoveryURL: server.URL + "/.well-known/openid-configuration",
	})
	if err != nil {
		t.Fatalf("put auth provider: %v", err)
	}
	if !created {
		t.Fatal("expected create result")
	}
	if config.Type != authProviderTypeOIDC {
		t.Fatalf("unexpected provider type: %q", config.Type)
	}
	if config.DiscoveryURL != server.URL+"/.well-known/openid-configuration" {
		t.Fatalf("unexpected discovery url: %q", config.DiscoveryURL)
	}
	if config.Issuer != server.URL {
		t.Fatalf("unexpected issuer: %q", config.Issuer)
	}
	if config.JWKSURI != server.URL+"/jwks" {
		t.Fatalf("unexpected jwks uri: %q", config.JWKSURI)
	}
	if len(config.AllowedAudiences) != 2 || config.AllowedAudiences[0] != "client-1" || config.AllowedAudiences[1] != "client-2" {
		t.Fatalf("unexpected audiences: %#v", config.AllowedAudiences)
	}
	if len(config.AllowedSigningAlgorithms) != 1 || config.AllowedSigningAlgorithms[0] != "RS256" {
		t.Fatalf("unexpected algorithms: %#v", config.AllowedSigningAlgorithms)
	}
	if config.UserIDClaim != "sub" {
		t.Fatalf("unexpected user id claim: %q", config.UserIDClaim)
	}
	if len(config.ClaimRequirements) != 1 || len(config.ClaimRequirements[0].AnyOf) != 2 {
		t.Fatalf("unexpected claim requirements: %#v", config.ClaimRequirements)
	}

	metadata, found, err := repository.GetMetadata(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if !found || metadata.AuthProvider == nil {
		t.Fatal("expected stored auth provider")
	}
	if metadata.UpdatedAt != now {
		t.Fatalf("unexpected updatedAt: got %v want %v", metadata.UpdatedAt, now)
	}
}

func TestServicePutAuthProviderRejectsMixedDiscoveryAndExplicitMetadata(t *testing.T) {
	t.Parallel()

	service := mustTenantService(t)

	_, _, err := service.PutAuthProvider(context.Background(), "tenant-1", UpsertAuthProviderInput{
		AllowedAudiences:         []string{"client-1"},
		AllowedSigningAlgorithms: []string{"RS256"},
		DiscoveryURL:             "https://issuer.example.com/.well-known/openid-configuration",
		Issuer:                   "https://issuer.example.com",
		JWKSURI:                  "https://issuer.example.com/jwks",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestServicePutAuthProviderRejectsAlgorithmsMissingFromDiscovery(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"issuer":"` + server.URL + `",
				"jwks_uri":"` + server.URL + `/jwks",
				"id_token_signing_alg_values_supported":["ES256"]
			}`))
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[{"kty":"EC","use":"sig","kid":"key-1","alg":"ES256"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service, err := NewService(ServiceConfig{
		HTTPClient: server.Client(),
		Repository: &memoryRepository{},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = service.PutAuthProvider(context.Background(), "tenant-1", UpsertAuthProviderInput{
		AllowedAudiences:         []string{"client-1"},
		AllowedSigningAlgorithms: []string{"RS256"},
		DiscoveryURL:             server.URL + "/.well-known/openid-configuration",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestServiceGetAndDeleteAuthProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[{"kty":"RSA","use":"sig","kid":"key-1"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service, err := NewService(ServiceConfig{
		HTTPClient: server.Client(),
		Repository: &memoryRepository{},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = service.PutAuthProvider(context.Background(), "tenant-1", UpsertAuthProviderInput{
		AllowedAudiences:         []string{"client-1"},
		AllowedSigningAlgorithms: []string{"RS256"},
		Issuer:                   server.URL,
		JWKSURI:                  server.URL + "/jwks",
		UserIDClaim:              "custom.sub",
	})
	if err != nil {
		t.Fatalf("put auth provider: %v", err)
	}

	config, err := service.GetAuthProvider(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("get auth provider: %v", err)
	}
	if config.UserIDClaim != "custom.sub" {
		t.Fatalf("unexpected user id claim: %q", config.UserIDClaim)
	}

	if err := service.DeleteAuthProvider(context.Background(), "tenant-1"); err != nil {
		t.Fatalf("delete auth provider: %v", err)
	}
	if _, err := service.GetAuthProvider(context.Background(), "tenant-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func mustTenantService(t *testing.T) *Service {
	t.Helper()

	service, err := NewService(ServiceConfig{Repository: &memoryRepository{}})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return service
}

type memoryRepository struct {
	metadata map[string]Metadata
}

func (r *memoryRepository) GetMetadata(_ context.Context, tenantID string) (Metadata, bool, error) {
	if r.metadata == nil {
		return Metadata{}, false, nil
	}
	metadata, ok := r.metadata[tenantID]
	return metadata, ok, nil
}

func (r *memoryRepository) SaveMetadata(_ context.Context, metadata Metadata) error {
	if r.metadata == nil {
		r.metadata = make(map[string]Metadata)
	}
	r.metadata[metadata.TenantID] = metadata
	return nil
}
