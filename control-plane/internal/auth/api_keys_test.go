package auth

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAPIKeyServiceIssuesListsAndRevokesBoundKeys(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 25, 3, 0, 0, 0, time.UTC)
	repository := &apiKeyRepositoryStub{}
	service, err := NewAPIKeyService(APIKeyServiceConfig{
		Now:        func() time.Time { return now },
		Repository: repository,
	})
	if err != nil {
		t.Fatalf("new api key service: %v", err)
	}

	issued, err := service.IssueAPIKey(context.Background(), "tenant-1", "firebase-user-1")
	if err != nil {
		t.Fatalf("issue api key: %v", err)
	}
	if !strings.HasPrefix(issued.APIKey, defaultAPIKeyPrefix) {
		t.Fatalf("unexpected api key prefix: %q", issued.APIKey)
	}
	if issued.Record.TenantID != "tenant-1" || issued.Record.UserID != "firebase-user-1" {
		t.Fatalf("unexpected issued record: %+v", issued.Record)
	}
	if len(repository.records) != 1 {
		t.Fatalf("unexpected repository record count: %d", len(repository.records))
	}
	if repository.records[0].Hash == "" || repository.records[0].Hash == issued.APIKey {
		t.Fatalf("expected stored hash to differ from raw key")
	}

	listed, err := service.ListAPIKeys(context.Background(), "tenant-1", "firebase-user-1")
	if err != nil {
		t.Fatalf("list api keys: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != issued.Record.ID {
		t.Fatalf("unexpected listed records: %+v", listed)
	}

	revoked, err := service.RevokeAPIKey(context.Background(), "tenant-1", "firebase-user-1", issued.Record.ID)
	if err != nil {
		t.Fatalf("revoke api key: %v", err)
	}
	if !revoked.Revoked || !repository.records[0].Revoked {
		t.Fatalf("expected revoked record, got %+v and %+v", revoked, repository.records[0])
	}
}

func TestAPIKeyServiceRevokeRejectsCrossUserAccess(t *testing.T) {
	t.Parallel()

	repository := &apiKeyRepositoryStub{
		records: []APIKeyRecord{
			{
				ID:        "key-1",
				Hash:      "hash",
				TenantID:  "tenant-1",
				UserID:    "firebase-user-1",
				CreatedAt: time.Date(2026, time.May, 25, 3, 0, 0, 0, time.UTC),
			},
		},
	}
	service, err := NewAPIKeyService(APIKeyServiceConfig{Repository: repository})
	if err != nil {
		t.Fatalf("new api key service: %v", err)
	}

	if _, err := service.RevokeAPIKey(context.Background(), "tenant-1", "firebase-user-2", "key-1"); err != ErrAPIKeyNotFound {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestAPIKeyServiceIssuesListsAndRevokesPlatformKeys(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 26, 3, 15, 0, 0, time.UTC)
	repository := &apiKeyRepositoryStub{}
	service, err := NewAPIKeyService(APIKeyServiceConfig{
		Now:        func() time.Time { return now },
		Repository: repository,
	})
	if err != nil {
		t.Fatalf("new api key service: %v", err)
	}

	issued, err := service.IssuePlatformAPIKey(context.Background(), "platform-tenant-1", "user-1")
	if err != nil {
		t.Fatalf("issue platform api key: %v", err)
	}
	if issued.Record.Kind != APIKeyKindPlatform || issued.Record.UserID != "" {
		t.Fatalf("unexpected issued record: %+v", issued.Record)
	}
	if repository.records[0].Kind != APIKeyKindPlatform || repository.records[0].CreatedByUserID != "user-1" {
		t.Fatalf("unexpected stored record: %+v", repository.records[0])
	}

	listed, err := service.ListPlatformAPIKeys(context.Background(), "platform-tenant-1")
	if err != nil {
		t.Fatalf("list platform api keys: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != issued.Record.ID || listed[0].Kind != APIKeyKindPlatform {
		t.Fatalf("unexpected listed records: %+v", listed)
	}

	revoked, err := service.RevokePlatformAPIKey(context.Background(), "platform-tenant-1", issued.Record.ID)
	if err != nil {
		t.Fatalf("revoke platform api key: %v", err)
	}
	if !revoked.Revoked || !repository.records[0].Revoked {
		t.Fatalf("expected revoked platform key, got %+v and %+v", revoked, repository.records[0])
	}
}

func TestTenantIDFromFirebaseClaims(t *testing.T) {
	t.Parallel()

	tenantID, err := TenantIDFromFirebaseClaims(map[string]any{"tenant_id": "tenant-1"}, "tenant_id")
	if err != nil {
		t.Fatalf("tenant id from claims: %v", err)
	}
	if tenantID != "tenant-1" {
		t.Fatalf("unexpected tenant id: %q", tenantID)
	}

	if _, err := TenantIDFromFirebaseClaims(map[string]any{"tenant_id": true}, "tenant_id"); err != ErrTenantClaimMissing {
		t.Fatalf("expected missing tenant claim error, got %v", err)
	}
}

type apiKeyRepositoryStub struct {
	records []APIKeyRecord
}

func (r *apiKeyRepositoryStub) ListAPIKeys(_ context.Context) ([]APIKeyRecord, error) {
	return append([]APIKeyRecord(nil), r.records...), nil
}

func (r *apiKeyRepositoryStub) SaveAPIKey(_ context.Context, record APIKeyRecord) error {
	for index, existing := range r.records {
		if existing.ID == record.ID {
			r.records[index] = record
			return nil
		}
	}
	r.records = append(r.records, record)
	return nil
}
