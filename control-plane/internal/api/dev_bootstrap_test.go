package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/your-org/cortado/control-plane/internal/auth"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

func TestDevBootstrapRouteAssignsTenantClaim(t *testing.T) {
	t.Parallel()

	service := &devBootstrapServiceStub{
		assignment: auth.DevTenantClaimAssignment{
			TenantID: "demo-tenant",
			UserID:   "firebase-user-1",
		},
	}

	router := NewRouter(RouterConfig{
		DevBootstrapAuth: cpmiddleware.NewFirebaseAuthMiddleware(
			cpmiddleware.FirebaseAuthConfig{
				AllowMissingTenantClaim: true,
				TenantClaim:             "tenant_id",
				Verifier: apiFirebaseVerifierStub{
					token: &auth.VerifiedFirebaseToken{
						UID:    "firebase-user-1",
						Claims: map[string]any{"role": "tester"},
					},
				},
			},
		),
		DevBootstrapSvc: service,
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/dev/firebase/tenant-claim",
		strings.NewReader(`{"tenantId":"demo-tenant"}`),
	)
	req.Header.Set("Authorization", "Bearer firebase-id-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	if service.tenantID != "demo-tenant" {
		t.Fatalf("unexpected tenant assignment request: %q", service.tenantID)
	}
	if service.token == nil || service.token.UID != "firebase-user-1" {
		t.Fatalf("unexpected token forwarded to service: %+v", service.token)
	}

	var response assignTenantClaimResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Assignment.TenantID != "demo-tenant" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

type devBootstrapServiceStub struct {
	assignment auth.DevTenantClaimAssignment
	tenantID   string
	token      *auth.VerifiedFirebaseToken
}

func (s *devBootstrapServiceStub) AssignTenantClaim(
	_ context.Context,
	token *auth.VerifiedFirebaseToken,
	tenantID string,
) (auth.DevTenantClaimAssignment, error) {
	s.token = token
	s.tenantID = tenantID
	return s.assignment, nil
}
