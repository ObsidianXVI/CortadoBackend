package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/cortado/control-plane/internal/auth"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
	"github.com/your-org/cortado/control-plane/internal/tenant"
)

func TestTenantAuthProviderRoutesUseFirebaseSelfServiceSurface(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	service := &tenantAuthProviderServiceStub{
		config: tenant.AuthProviderConfig{
			AllowedAudiences:         []string{"client-1"},
			AllowedSigningAlgorithms: []string{"RS256"},
			DiscoveryURL:             "https://issuer.example.com/.well-known/openid-configuration",
			Issuer:                   "https://issuer.example.com",
			JWKSURI:                  "https://issuer.example.com/jwks",
			Type:                     "oidc",
			UserIDClaim:              "sub",
		},
	}
	router := NewRouter(RouterConfig{
		APIKeyAuth: cpmiddleware.NewFirebaseAuthMiddleware(cpmiddleware.FirebaseAuthConfig{
			TenantClaim: "tenant_id",
			Verifier: apiFirebaseVerifierStub{
				token: &auth.VerifiedFirebaseToken{
					UID:    "firebase-user-1",
					Claims: map[string]any{"tenant_id": "tenant-1"},
				},
			},
		}),
		TenantAuthSvc: service,
	})

	putReq := httptest.NewRequest(http.MethodPut, "/v1/tenant/auth-provider", bytes.NewBufferString(`{
		"discoveryUrl":"https://issuer.example.com/.well-known/openid-configuration",
		"allowedAudiences":["client-1"],
		"allowedSigningAlgorithms":["RS256"]
	}`))
	putReq.Header.Set("Authorization", "Bearer firebase-id-token")
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)

	if putRec.Code != http.StatusCreated {
		t.Fatalf("unexpected put status: got %d want %d", putRec.Code, http.StatusCreated)
	}
	if service.putTenantID != "tenant-1" {
		t.Fatalf("unexpected tenant for put: %q", service.putTenantID)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/tenant/auth-provider", nil)
	getReq.Header.Set("Authorization", "Bearer firebase-id-token")
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", getRec.Code, http.StatusOK)
	}
	if service.getTenantID != "tenant-1" {
		t.Fatalf("unexpected tenant for get: %q", service.getTenantID)
	}

	var response tenantAuthProviderEnvelope
	if err := json.Unmarshal(getRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if response.AuthProvider.Issuer != "https://issuer.example.com" {
		t.Fatalf("unexpected issuer: %q", response.AuthProvider.Issuer)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/tenant/auth-provider", nil)
	deleteReq.Header.Set("Authorization", "Bearer firebase-id-token")
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("unexpected delete status: got %d want %d", deleteRec.Code, http.StatusNoContent)
	}
	if service.deleteTenantID != "tenant-1" {
		t.Fatalf("unexpected tenant for delete: %q", service.deleteTenantID)
	}
}

func TestTenantAuthProviderRoutesRequireFirebaseAuth(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	router := NewRouter(RouterConfig{
		APIKeyAuth: cpmiddleware.NewFirebaseAuthMiddleware(cpmiddleware.FirebaseAuthConfig{
			TenantClaim: "tenant_id",
			Verifier: apiFirebaseVerifierStub{
				token: &auth.VerifiedFirebaseToken{
					UID:    "firebase-user-1",
					Claims: map[string]any{"tenant_id": "tenant-1"},
				},
			},
		}),
		TenantAuthSvc: &tenantAuthProviderServiceStub{},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenant/auth-provider", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusUnauthorized)
	}
}

type tenantAuthProviderServiceStub struct {
	config         tenant.AuthProviderConfig
	deleteTenantID string
	err            error
	getTenantID    string
	putCreated     bool
	putTenantID    string
}

func (s *tenantAuthProviderServiceStub) DeleteAuthProvider(_ context.Context, tenantID string) error {
	s.deleteTenantID = tenantID
	return s.err
}

func (s *tenantAuthProviderServiceStub) GetAuthProvider(_ context.Context, tenantID string) (tenant.AuthProviderConfig, error) {
	s.getTenantID = tenantID
	return s.config, s.err
}

func (s *tenantAuthProviderServiceStub) PutAuthProvider(_ context.Context, tenantID string, _ tenant.UpsertAuthProviderInput) (tenant.AuthProviderConfig, bool, error) {
	s.putTenantID = tenantID
	created := s.putCreated
	if !created {
		created = true
	}
	return s.config, created, s.err
}
