package store

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/your-org/cortado/control-plane/internal/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FirestoreAuthStoreConfig struct {
	APIKeysCollection         string
	FirstPartyUsersCollection string
	RefreshTokensCollection   string
	TenantsCollection         string
}

type FirestoreAuthStore struct {
	apiKeysCollection         string
	client                    *firestore.Client
	firstPartyUsersCollection string
	refreshTokensCollection   string
	tenantsCollection         string
}

func NewFirestoreAuthStore(client *firestore.Client, cfg FirestoreAuthStoreConfig) *FirestoreAuthStore {
	if cfg.APIKeysCollection == "" {
		cfg.APIKeysCollection = auth.DefaultAPIKeysCollection
	}
	if cfg.FirstPartyUsersCollection == "" {
		cfg.FirstPartyUsersCollection = auth.DefaultFirstPartyUsersCollection
	}
	if cfg.RefreshTokensCollection == "" {
		cfg.RefreshTokensCollection = auth.DefaultRefreshTokensCollection
	}
	if cfg.TenantsCollection == "" {
		cfg.TenantsCollection = "tenants"
	}

	return &FirestoreAuthStore{
		apiKeysCollection:         cfg.APIKeysCollection,
		client:                    client,
		firstPartyUsersCollection: cfg.FirstPartyUsersCollection,
		refreshTokensCollection:   cfg.RefreshTokensCollection,
		tenantsCollection:         cfg.TenantsCollection,
	}
}

func (s *FirestoreAuthStore) ListAPIKeys(ctx context.Context) ([]auth.APIKeyRecord, error) {
	iter := s.client.Collection(s.apiKeysCollection).Documents(ctx)
	defer iter.Stop()

	records := []auth.APIKeyRecord{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return records, nil
		}
		if err != nil {
			return nil, fmt.Errorf("iterate api key documents: %w", err)
		}

		var record auth.APIKeyRecord
		if err := doc.DataTo(&record); err != nil {
			return nil, fmt.Errorf("decode api key document %q: %w", doc.Ref.ID, err)
		}
		if record.ID == "" {
			record.ID = doc.Ref.ID
		}
		records = append(records, record)
	}
}

func (s *FirestoreAuthStore) SaveAPIKey(ctx context.Context, record auth.APIKeyRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("save api key document: id is required")
	}
	if _, err := s.client.Collection(s.apiKeysCollection).Doc(record.ID).Set(ctx, record); err != nil {
		return fmt.Errorf("save api key document: %w", err)
	}
	return nil
}

func (s *FirestoreAuthStore) GetFirstPartyAccount(ctx context.Context, firebaseUID string) (auth.FirstPartyAccount, bool, error) {
	doc, err := s.client.Collection(s.firstPartyUsersCollection).Doc(firebaseUID).Get(ctx)
	if isNotFound(err) {
		return auth.FirstPartyAccount{}, false, nil
	}
	if err != nil {
		return auth.FirstPartyAccount{}, false, fmt.Errorf("get first-party account document: %w", err)
	}

	var account auth.FirstPartyAccount
	if err := doc.DataTo(&account); err != nil {
		return auth.FirstPartyAccount{}, false, fmt.Errorf("decode first-party account document %q: %w", doc.Ref.ID, err)
	}
	if account.FirebaseUID == "" {
		account.FirebaseUID = doc.Ref.ID
	}

	return account, true, nil
}

func (s *FirestoreAuthStore) SaveFirstPartyAccount(ctx context.Context, account auth.FirstPartyAccount) error {
	if strings.TrimSpace(account.FirebaseUID) == "" {
		return fmt.Errorf("save first-party account document: firebase uid is required")
	}
	if _, err := s.client.Collection(s.firstPartyUsersCollection).Doc(account.FirebaseUID).Set(ctx, account); err != nil {
		return fmt.Errorf("save first-party account document: %w", err)
	}
	return nil
}

func (s *FirestoreAuthStore) EnsurePersonalTenant(ctx context.Context, tenant auth.PersonalTenantRecord) error {
	if strings.TrimSpace(tenant.TenantID) == "" {
		return fmt.Errorf("save personal tenant document: tenant id is required")
	}
	if _, err := s.client.Collection(s.tenantsCollection).Doc(tenant.TenantID).Set(ctx, tenant, firestore.MergeAll); err != nil {
		return fmt.Errorf("save personal tenant document: %w", err)
	}
	return nil
}

func (s *FirestoreAuthStore) GetPlatformTenant(ctx context.Context, tenantID string) (auth.PlatformTenantRecord, bool, error) {
	doc, err := s.client.Collection(s.tenantsCollection).Doc(tenantID).Get(ctx)
	if isNotFound(err) {
		return auth.PlatformTenantRecord{}, false, nil
	}
	if err != nil {
		return auth.PlatformTenantRecord{}, false, fmt.Errorf("get platform tenant document: %w", err)
	}

	var record auth.PlatformTenantRecord
	if err := doc.DataTo(&record); err != nil {
		return auth.PlatformTenantRecord{}, false, fmt.Errorf("decode platform tenant document %q: %w", doc.Ref.ID, err)
	}
	if strings.TrimSpace(record.TenantID) == "" {
		record.TenantID = doc.Ref.ID
	}
	if record.Kind == "" {
		record.Kind = auth.APIKeyKindPlatform
	}
	if record.Kind != auth.APIKeyKindPlatform {
		return auth.PlatformTenantRecord{}, false, nil
	}

	return record, true, nil
}

func (s *FirestoreAuthStore) ListPlatformTenants(ctx context.Context, ownerUserID string) ([]auth.PlatformTenantRecord, error) {
	iter := s.client.Collection(s.tenantsCollection).
		Where("kind", "==", auth.APIKeyKindPlatform).
		Where("ownerUserId", "==", strings.TrimSpace(ownerUserID)).
		Documents(ctx)
	defer iter.Stop()

	records := []auth.PlatformTenantRecord{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate platform tenant documents: %w", err)
		}

		var record auth.PlatformTenantRecord
		if err := doc.DataTo(&record); err != nil {
			return nil, fmt.Errorf("decode platform tenant document %q: %w", doc.Ref.ID, err)
		}
		if strings.TrimSpace(record.TenantID) == "" {
			record.TenantID = doc.Ref.ID
		}
		if record.Kind == "" {
			record.Kind = auth.APIKeyKindPlatform
		}
		records = append(records, record)
	}

	slices.SortFunc(records, func(left, right auth.PlatformTenantRecord) int {
		return right.CreatedAt.Compare(left.CreatedAt)
	})
	return records, nil
}

func (s *FirestoreAuthStore) SavePlatformTenant(ctx context.Context, tenant auth.PlatformTenantRecord) error {
	if strings.TrimSpace(tenant.TenantID) == "" {
		return fmt.Errorf("save platform tenant document: tenant id is required")
	}
	if _, err := s.client.Collection(s.tenantsCollection).Doc(tenant.TenantID).Set(ctx, tenant, firestore.MergeAll); err != nil {
		return fmt.Errorf("save platform tenant document: %w", err)
	}
	return nil
}

func (s *FirestoreAuthStore) SaveRefreshToken(ctx context.Context, token auth.RefreshTokenRecord) error {
	if _, err := s.client.Collection(s.refreshTokensCollection).Doc(token.RefreshToken).Create(ctx, token); err != nil {
		return fmt.Errorf("create refresh token document: %w", err)
	}
	return nil
}

func (s *FirestoreAuthStore) GetRefreshToken(ctx context.Context, refreshToken string) (auth.RefreshTokenRecord, bool, error) {
	doc, err := s.client.Collection(s.refreshTokensCollection).Doc(refreshToken).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return auth.RefreshTokenRecord{}, false, nil
	}
	if err != nil {
		return auth.RefreshTokenRecord{}, false, fmt.Errorf("get refresh token document: %w", err)
	}

	var record auth.RefreshTokenRecord
	if err := doc.DataTo(&record); err != nil {
		return auth.RefreshTokenRecord{}, false, fmt.Errorf("decode refresh token document %q: %w", doc.Ref.ID, err)
	}
	if record.RefreshToken == "" {
		record.RefreshToken = doc.Ref.ID
	}

	return record, true, nil
}
