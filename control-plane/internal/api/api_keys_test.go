package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/your-org/cortado/control-plane/internal/auth"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

func TestAPIKeyRoutesIssueListAndRevokeWithFirebaseAuth(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 25, 3, 45, 0, 0, time.UTC)
	service := &apiKeyServiceStub{
		issued: auth.IssuedAPIKey{
			APIKey: "cortado_issued",
			Record: auth.APIKey{
				ID:        "key-1",
				TenantID:  "tenant-1",
				UserID:    "firebase-user-1",
				CreatedAt: now,
			},
		},
		listed: []auth.APIKey{
			{
				ID:        "key-1",
				TenantID:  "tenant-1",
				UserID:    "firebase-user-1",
				CreatedAt: now,
			},
		},
		revoked: auth.APIKey{
			ID:        "key-1",
			TenantID:  "tenant-1",
			UserID:    "firebase-user-1",
			Revoked:   true,
			CreatedAt: now,
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
		APIKeySvc: service,
	})

	issueReq := httptest.NewRequest(http.MethodPost, "/v1/api-keys", nil)
	issueReq.Header.Set("Authorization", "Bearer firebase-id-token")
	issueRec := httptest.NewRecorder()
	router.ServeHTTP(issueRec, issueReq)

	if issueRec.Code != http.StatusCreated {
		t.Fatalf("unexpected issue status: got %d want %d", issueRec.Code, http.StatusCreated)
	}
	if service.issueTenantID != "tenant-1" || service.issueUserID != "firebase-user-1" {
		t.Fatalf("unexpected issue actor: tenant=%q user=%q", service.issueTenantID, service.issueUserID)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	listReq.Header.Set("Authorization", "Bearer firebase-id-token")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", listRec.Code, http.StatusOK)
	}

	revokeReq := httptest.NewRequest(http.MethodDelete, "/v1/api-keys/key-1", nil)
	revokeReq.Header.Set("Authorization", "Bearer firebase-id-token")
	revokeRec := httptest.NewRecorder()
	router.ServeHTTP(revokeRec, revokeReq)

	if revokeRec.Code != http.StatusOK {
		t.Fatalf("unexpected revoke status: got %d want %d", revokeRec.Code, http.StatusOK)
	}
	if service.revokeTenantID != "tenant-1" || service.revokeUserID != "firebase-user-1" || service.revokedID != "key-1" {
		t.Fatalf("unexpected revoke actor: tenant=%q user=%q id=%q", service.revokeTenantID, service.revokeUserID, service.revokedID)
	}

	var issued issueAPIKeyResponse
	if err := json.Unmarshal(issueRec.Body.Bytes(), &issued); err != nil {
		t.Fatalf("decode issue response: %v", err)
	}
	if issued.APIKey != "cortado_issued" || issued.Record.ID != "key-1" {
		t.Fatalf("unexpected issue response: %+v", issued)
	}
}

func TestAPIKeyRoutesRejectMissingFirebaseAuth(t *testing.T) {
	t.Parallel()

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
		APIKeySvc: &apiKeyServiceStub{},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusUnauthorized)
	}
}

type apiKeyServiceStub struct {
	issueTenantID  string
	issueUserID    string
	issued         auth.IssuedAPIKey
	listTenantID   string
	listUserID     string
	listed         []auth.APIKey
	revokeTenantID string
	revokeUserID   string
	revokedID      string
	revoked        auth.APIKey
}

func (s *apiKeyServiceStub) IssueAPIKey(_ context.Context, tenantID, userID string) (auth.IssuedAPIKey, error) {
	s.issueTenantID = tenantID
	s.issueUserID = userID
	return s.issued, nil
}

func (s *apiKeyServiceStub) ListAPIKeys(_ context.Context, tenantID, userID string) ([]auth.APIKey, error) {
	s.listTenantID = tenantID
	s.listUserID = userID
	return s.listed, nil
}

func (s *apiKeyServiceStub) RevokeAPIKey(_ context.Context, tenantID, userID, keyID string) (auth.APIKey, error) {
	s.revokeTenantID = tenantID
	s.revokeUserID = userID
	s.revokedID = keyID
	return s.revoked, nil
}

type apiFirebaseVerifierStub struct {
	token *auth.VerifiedFirebaseToken
	err   error
}

func (s apiFirebaseVerifierStub) VerifyIDToken(_ context.Context, _ string) (*auth.VerifiedFirebaseToken, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.token, nil
}
