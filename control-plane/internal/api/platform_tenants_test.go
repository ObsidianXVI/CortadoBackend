package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/your-org/cortado/control-plane/internal/auth"
)

func TestPlatformTenantRoutesCreateListAndManageKeys(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 26, 4, 30, 0, 0, time.UTC)
	service := &platformTenantServiceStub{
		createdTenant: auth.PlatformTenant{
			CreatedAt:   now,
			DisplayName: "Acme",
			Kind:        auth.APIKeyKindPlatform,
			TenantID:    "platform-tenant-1",
			UpdatedAt:   now,
		},
		listedTenants: []auth.PlatformTenant{
			{
				CreatedAt:   now,
				DisplayName: "Acme",
				Kind:        auth.APIKeyKindPlatform,
				TenantID:    "platform-tenant-1",
				UpdatedAt:   now,
			},
		},
		issuedKey: auth.IssuedAPIKey{
			APIKey: "cortado_platform",
			Record: auth.APIKey{
				CreatedAt: now,
				ID:        "key-1",
				Kind:      auth.APIKeyKindPlatform,
				TenantID:  "platform-tenant-1",
			},
		},
		listedKeys: []auth.APIKey{
			{
				CreatedAt: now,
				ID:        "key-1",
				Kind:      auth.APIKeyKindPlatform,
				TenantID:  "platform-tenant-1",
			},
		},
		revokedKey: auth.APIKey{
			CreatedAt: now,
			ID:        "key-1",
			Kind:      auth.APIKeyKindPlatform,
			Revoked:   true,
			TenantID:  "platform-tenant-1",
		},
	}
	authService := mustAuthService(t)
	sessionTokens, err := authService.CreateSession(context.Background(), "secret-api-key", "user-1")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	router := NewRouter(RouterConfig{
		JWKSProvider:      authService,
		PlatformTenantSvc: service,
	})

	createReq := httptest.NewRequest(http.MethodPost, "/v1/platform-tenants", jsonBody(t, map[string]any{
		"displayName": "Acme",
	}))
	createReq.Header.Set("Authorization", "Bearer "+sessionTokens.AccessToken)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d", createRec.Code, http.StatusCreated)
	}
	if service.createOwnerUserID != "user-1" || service.createDisplayName != "Acme" {
		t.Fatalf("unexpected create args: owner=%q display=%q", service.createOwnerUserID, service.createDisplayName)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/platform-tenants", nil)
	listReq.Header.Set("Authorization", "Bearer "+sessionTokens.AccessToken)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", listRec.Code, http.StatusOK)
	}
	if service.listOwnerUserID != "user-1" {
		t.Fatalf("unexpected list owner: %q", service.listOwnerUserID)
	}

	issueReq := httptest.NewRequest(http.MethodPost, "/v1/platform-tenants/platform-tenant-1/api-keys", nil)
	issueReq.Header.Set("Authorization", "Bearer "+sessionTokens.AccessToken)
	issueRec := httptest.NewRecorder()
	router.ServeHTTP(issueRec, issueReq)

	if issueRec.Code != http.StatusCreated {
		t.Fatalf("unexpected issue status: got %d want %d", issueRec.Code, http.StatusCreated)
	}
	if service.issueOwnerUserID != "user-1" || service.issueTenantID != "platform-tenant-1" {
		t.Fatalf("unexpected issue args: owner=%q tenant=%q", service.issueOwnerUserID, service.issueTenantID)
	}

	keyListReq := httptest.NewRequest(http.MethodGet, "/v1/platform-tenants/platform-tenant-1/api-keys", nil)
	keyListReq.Header.Set("Authorization", "Bearer "+sessionTokens.AccessToken)
	keyListRec := httptest.NewRecorder()
	router.ServeHTTP(keyListRec, keyListReq)

	if keyListRec.Code != http.StatusOK {
		t.Fatalf("unexpected key list status: got %d want %d", keyListRec.Code, http.StatusOK)
	}
	if service.listKeysOwnerUserID != "user-1" || service.listKeysTenantID != "platform-tenant-1" {
		t.Fatalf("unexpected key list args: owner=%q tenant=%q", service.listKeysOwnerUserID, service.listKeysTenantID)
	}

	revokeReq := httptest.NewRequest(http.MethodDelete, "/v1/platform-tenants/platform-tenant-1/api-keys/key-1", nil)
	revokeReq.Header.Set("Authorization", "Bearer "+sessionTokens.AccessToken)
	revokeRec := httptest.NewRecorder()
	router.ServeHTTP(revokeRec, revokeReq)

	if revokeRec.Code != http.StatusOK {
		t.Fatalf("unexpected revoke status: got %d want %d", revokeRec.Code, http.StatusOK)
	}
	if service.revokeOwnerUserID != "user-1" || service.revokeTenantID != "platform-tenant-1" || service.revokeKeyID != "key-1" {
		t.Fatalf("unexpected revoke args: owner=%q tenant=%q key=%q", service.revokeOwnerUserID, service.revokeTenantID, service.revokeKeyID)
	}
}

type platformTenantServiceStub struct {
	createDisplayName   string
	createOwnerUserID   string
	createdTenant       auth.PlatformTenant
	issueOwnerUserID    string
	issueTenantID       string
	issuedKey           auth.IssuedAPIKey
	listKeysOwnerUserID string
	listKeysTenantID    string
	listOwnerUserID     string
	listedKeys          []auth.APIKey
	listedTenants       []auth.PlatformTenant
	revokeKeyID         string
	revokeOwnerUserID   string
	revokeTenantID      string
	revokedKey          auth.APIKey
}

func (s *platformTenantServiceStub) CreatePlatformTenant(_ context.Context, ownerUserID, displayName string) (auth.PlatformTenant, error) {
	s.createOwnerUserID = ownerUserID
	s.createDisplayName = displayName
	return s.createdTenant, nil
}

func (s *platformTenantServiceStub) IssuePlatformAPIKey(_ context.Context, ownerUserID, tenantID string) (auth.IssuedAPIKey, error) {
	s.issueOwnerUserID = ownerUserID
	s.issueTenantID = tenantID
	return s.issuedKey, nil
}

func (s *platformTenantServiceStub) ListPlatformAPIKeys(_ context.Context, ownerUserID, tenantID string) ([]auth.APIKey, error) {
	s.listKeysOwnerUserID = ownerUserID
	s.listKeysTenantID = tenantID
	return s.listedKeys, nil
}

func (s *platformTenantServiceStub) ListPlatformTenants(_ context.Context, ownerUserID string) ([]auth.PlatformTenant, error) {
	s.listOwnerUserID = ownerUserID
	return s.listedTenants, nil
}

func (s *platformTenantServiceStub) RevokePlatformAPIKey(_ context.Context, ownerUserID, tenantID, keyID string) (auth.APIKey, error) {
	s.revokeOwnerUserID = ownerUserID
	s.revokeTenantID = tenantID
	s.revokeKeyID = keyID
	return s.revokedKey, nil
}

func jsonBody(t *testing.T, payload map[string]any) *bytes.Buffer {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json body: %v", err)
	}
	return bytes.NewBuffer(body)
}
