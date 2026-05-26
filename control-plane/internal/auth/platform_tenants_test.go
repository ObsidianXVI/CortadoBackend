package auth

import (
	"context"
	"testing"
	"time"
)

func TestPlatformTenantServiceCreatesListsAndManagesKeys(t *testing.T) {
	t.Parallel()

	repository := &platformTenantRepositoryStub{}
	apiKeys, err := NewAPIKeyService(APIKeyServiceConfig{
		Now:        func() time.Time { return time.Date(2026, time.May, 26, 4, 0, 0, 0, time.UTC) },
		Repository: repository,
	})
	if err != nil {
		t.Fatalf("new api key service: %v", err)
	}

	service, err := NewPlatformTenantService(PlatformTenantServiceConfig{
		APIKeys:     apiKeys,
		IDGenerator: func() string { return "platform-tenant-1" },
		Now:         func() time.Time { return time.Date(2026, time.May, 26, 4, 0, 0, 0, time.UTC) },
		Repository:  repository,
	})
	if err != nil {
		t.Fatalf("new platform tenant service: %v", err)
	}

	tenant, err := service.CreatePlatformTenant(context.Background(), "user-1", "Acme")
	if err != nil {
		t.Fatalf("create platform tenant: %v", err)
	}
	if tenant.TenantID != "platform-tenant-1" || tenant.Kind != APIKeyKindPlatform {
		t.Fatalf("unexpected tenant: %+v", tenant)
	}

	listedTenants, err := service.ListPlatformTenants(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list platform tenants: %v", err)
	}
	if len(listedTenants) != 1 || listedTenants[0].TenantID != "platform-tenant-1" {
		t.Fatalf("unexpected listed tenants: %+v", listedTenants)
	}

	issued, err := service.IssuePlatformAPIKey(context.Background(), "user-1", "platform-tenant-1")
	if err != nil {
		t.Fatalf("issue platform api key: %v", err)
	}
	if issued.Record.Kind != APIKeyKindPlatform {
		t.Fatalf("unexpected issued key: %+v", issued.Record)
	}

	listedKeys, err := service.ListPlatformAPIKeys(context.Background(), "user-1", "platform-tenant-1")
	if err != nil {
		t.Fatalf("list platform api keys: %v", err)
	}
	if len(listedKeys) != 1 || listedKeys[0].ID != issued.Record.ID {
		t.Fatalf("unexpected listed keys: %+v", listedKeys)
	}

	revoked, err := service.RevokePlatformAPIKey(context.Background(), "user-1", "platform-tenant-1", issued.Record.ID)
	if err != nil {
		t.Fatalf("revoke platform api key: %v", err)
	}
	if !revoked.Revoked {
		t.Fatalf("expected revoked platform key: %+v", revoked)
	}
}

func TestPlatformTenantServiceRejectsCrossOwnerAccess(t *testing.T) {
	t.Parallel()

	repository := &platformTenantRepositoryStub{
		tenants: map[string]PlatformTenantRecord{
			"platform-tenant-1": {
				Kind:        APIKeyKindPlatform,
				OwnerUserID: "user-1",
				TenantID:    "platform-tenant-1",
			},
		},
	}
	apiKeys, err := NewAPIKeyService(APIKeyServiceConfig{Repository: repository})
	if err != nil {
		t.Fatalf("new api key service: %v", err)
	}
	service, err := NewPlatformTenantService(PlatformTenantServiceConfig{
		APIKeys:    apiKeys,
		Repository: repository,
	})
	if err != nil {
		t.Fatalf("new platform tenant service: %v", err)
	}

	if _, err := service.IssuePlatformAPIKey(context.Background(), "user-2", "platform-tenant-1"); err != ErrPlatformTenantNotFound {
		t.Fatalf("expected platform tenant not found, got %v", err)
	}
}

type platformTenantRepositoryStub struct {
	records []APIKeyRecord
	tenants map[string]PlatformTenantRecord
}

func (r *platformTenantRepositoryStub) GetPlatformTenant(_ context.Context, tenantID string) (PlatformTenantRecord, bool, error) {
	record, ok := r.tenants[tenantID]
	return record, ok, nil
}

func (r *platformTenantRepositoryStub) ListPlatformTenants(_ context.Context, ownerUserID string) ([]PlatformTenantRecord, error) {
	records := []PlatformTenantRecord{}
	for _, record := range r.tenants {
		if record.OwnerUserID == ownerUserID {
			records = append(records, record)
		}
	}
	return records, nil
}

func (r *platformTenantRepositoryStub) SavePlatformTenant(_ context.Context, tenant PlatformTenantRecord) error {
	if r.tenants == nil {
		r.tenants = map[string]PlatformTenantRecord{}
	}
	r.tenants[tenant.TenantID] = tenant
	return nil
}

func (r *platformTenantRepositoryStub) ListAPIKeys(_ context.Context) ([]APIKeyRecord, error) {
	return append([]APIKeyRecord(nil), r.records...), nil
}

func (r *platformTenantRepositoryStub) SaveAPIKey(_ context.Context, record APIKeyRecord) error {
	for index, existing := range r.records {
		if existing.ID == record.ID {
			r.records[index] = record
			return nil
		}
	}
	r.records = append(r.records, record)
	return nil
}
