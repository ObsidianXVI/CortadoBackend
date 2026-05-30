package store

import (
	"testing"
	"time"

	"github.com/your-org/cortado/control-plane/internal/auth"
	"github.com/your-org/cortado/control-plane/internal/tenant"
)

func TestPersonalTenantDocumentReturnsMapData(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 30, 6, 0, 0, 0, time.UTC)
	document := personalTenantDocument(auth.PersonalTenantRecord{
		CreatedAt:   now,
		DisplayName: "User One",
		Kind:        auth.APIKeyKindPersonal,
		OwnerUserID: "user-1",
		TenantID:    "tenant-1",
		UpdatedAt:   now,
	})

	if got := document["tenantId"]; got != "tenant-1" {
		t.Fatalf("unexpected tenantId: %#v", got)
	}
	if got := document["ownerUserId"]; got != "user-1" {
		t.Fatalf("unexpected ownerUserId: %#v", got)
	}
	if got := document["kind"]; got != auth.APIKeyKindPersonal {
		t.Fatalf("unexpected kind: %#v", got)
	}
}

func TestPlatformTenantDocumentReturnsMapData(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 30, 6, 0, 0, 0, time.UTC)
	document := platformTenantDocument(auth.PlatformTenantRecord{
		CreatedAt:   now,
		DisplayName: "Acme IDE",
		Kind:        auth.APIKeyKindPlatform,
		OwnerUserID: "user-1",
		TenantID:    "platform-1",
		UpdatedAt:   now,
	})

	if got := document["tenantId"]; got != "platform-1" {
		t.Fatalf("unexpected tenantId: %#v", got)
	}
	if got := document["kind"]; got != auth.APIKeyKindPlatform {
		t.Fatalf("unexpected kind: %#v", got)
	}
}

func TestTenantMetadataDocumentReturnsMapData(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 30, 6, 0, 0, 0, time.UTC)
	document := tenantMetadataDocument(tenant.Metadata{
		AuthProvider: &tenant.AuthProviderConfig{
			Type:        "oidc",
			Issuer:      "https://issuer.example.com",
			JWKSURI:     "https://issuer.example.com/jwks",
			UserIDClaim: "sub",
		},
		TenantID:  "tenant-1",
		UpdatedAt: now,
	})

	if got := document["tenantId"]; got != "tenant-1" {
		t.Fatalf("unexpected tenantId: %#v", got)
	}
	if got := document["authProvider"]; got == nil {
		t.Fatal("expected authProvider entry")
	}
}
