package auth

import (
	"context"
	"testing"
)

func TestDevFirebaseBootstrapServiceAssignsDefaultTenantClaim(t *testing.T) {
	t.Parallel()

	manager := &firebaseClaimsManagerStub{}
	service, err := NewDevFirebaseBootstrapService(DevFirebaseBootstrapConfig{
		DefaultTenantID: "demo-tenant",
		Enabled:         true,
		Manager:         manager,
		TenantClaim:     "tenant_id",
	})
	if err != nil {
		t.Fatalf("new dev firebase bootstrap service: %v", err)
	}

	assignment, err := service.AssignTenantClaim(
		context.Background(),
		&VerifiedFirebaseToken{
			UID:    "firebase-user-1",
			Claims: map[string]any{"role": "tester"},
		},
		"",
	)
	if err != nil {
		t.Fatalf("assign tenant claim: %v", err)
	}
	if assignment.TenantID != "demo-tenant" || assignment.UserID != "firebase-user-1" {
		t.Fatalf("unexpected assignment: %+v", assignment)
	}
	if manager.uid != "firebase-user-1" {
		t.Fatalf("unexpected manager uid: %q", manager.uid)
	}
	if got := manager.claims["tenant_id"]; got != "demo-tenant" {
		t.Fatalf("unexpected tenant claim: %v", got)
	}
	if got := manager.claims["role"]; got != "tester" {
		t.Fatalf("expected existing claims to be preserved, got %v", got)
	}
}

func TestDevFirebaseBootstrapServiceSkipsReservedTokenClaims(t *testing.T) {
	t.Parallel()

	manager := &firebaseClaimsManagerStub{}
	service, err := NewDevFirebaseBootstrapService(DevFirebaseBootstrapConfig{
		DefaultTenantID: "demo-tenant",
		Enabled:         true,
		Manager:         manager,
		TenantClaim:     "tenant_id",
	})
	if err != nil {
		t.Fatalf("new dev firebase bootstrap service: %v", err)
	}

	_, err = service.AssignTenantClaim(
		context.Background(),
		&VerifiedFirebaseToken{
			UID: "firebase-user-1",
			Claims: map[string]any{
				"auth_time": 1710000000,
				"firebase":  map[string]any{"sign_in_provider": "password"},
				"iss":       "https://securetoken.google.com/cortado-ide",
				"role":      "tester",
				"tenant_id": "old-tenant",
			},
		},
		"demo-tenant",
	)
	if err != nil {
		t.Fatalf("assign tenant claim: %v", err)
	}

	if _, found := manager.claims["auth_time"]; found {
		t.Fatalf("expected reserved auth_time claim to be removed: %#v", manager.claims)
	}
	if _, found := manager.claims["firebase"]; found {
		t.Fatalf("expected reserved firebase claim to be removed: %#v", manager.claims)
	}
	if _, found := manager.claims["iss"]; found {
		t.Fatalf("expected reserved iss claim to be removed: %#v", manager.claims)
	}
	if got := manager.claims["role"]; got != "tester" {
		t.Fatalf("expected custom role claim to be preserved, got %v", got)
	}
	if got := manager.claims["tenant_id"]; got != "demo-tenant" {
		t.Fatalf("expected tenant claim to be updated, got %v", got)
	}
}

func TestDevFirebaseBootstrapServiceRejectsWhenDisabled(t *testing.T) {
	t.Parallel()

	service, err := NewDevFirebaseBootstrapService(DevFirebaseBootstrapConfig{
		Enabled:     false,
		Manager:     &firebaseClaimsManagerStub{},
		TenantClaim: "tenant_id",
	})
	if err != nil {
		t.Fatalf("new dev firebase bootstrap service: %v", err)
	}

	if _, err := service.AssignTenantClaim(
		context.Background(),
		&VerifiedFirebaseToken{UID: "firebase-user-1"},
		"",
	); err != ErrDevBootstrapDisabled {
		t.Fatalf("expected disabled error, got %v", err)
	}
}

type firebaseClaimsManagerStub struct {
	claims map[string]any
	uid    string
}

func (s *firebaseClaimsManagerStub) SetCustomUserClaims(
	_ context.Context,
	uid string,
	claims map[string]any,
) error {
	s.uid = uid
	s.claims = claims
	return nil
}
