package store

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/your-org/cortado/control-plane/internal/tenant"
)

const defaultTenantCollection = "tenants"

type FirestoreTenantStoreConfig struct {
	Collection string
}

type FirestoreTenantStore struct {
	client     *firestore.Client
	collection string
}

func NewFirestoreTenantStore(client *firestore.Client, cfg FirestoreTenantStoreConfig) *FirestoreTenantStore {
	if cfg.Collection == "" {
		cfg.Collection = defaultTenantCollection
	}

	return &FirestoreTenantStore{
		client:     client,
		collection: cfg.Collection,
	}
}

func (s *FirestoreTenantStore) GetMetadata(ctx context.Context, tenantID string) (tenant.Metadata, bool, error) {
	doc, err := s.collectionRef().Doc(tenantID).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			return tenant.Metadata{}, false, nil
		}
		return tenant.Metadata{}, false, fmt.Errorf("get tenant metadata document: %w", err)
	}

	var metadata tenant.Metadata
	if err := doc.DataTo(&metadata); err != nil {
		return tenant.Metadata{}, false, fmt.Errorf("decode tenant metadata document: %w", err)
	}
	if strings.TrimSpace(metadata.TenantID) == "" {
		metadata.TenantID = doc.Ref.ID
	}

	return metadata, true, nil
}

func (s *FirestoreTenantStore) SaveMetadata(ctx context.Context, metadata tenant.Metadata) error {
	if strings.TrimSpace(metadata.TenantID) == "" {
		return fmt.Errorf("save tenant metadata document: tenant id is required")
	}
	if _, err := s.collectionRef().Doc(metadata.TenantID).Set(
		ctx,
		tenantMetadataDocument(metadata),
		firestore.MergeAll,
	); err != nil {
		return fmt.Errorf("save tenant metadata document: %w", err)
	}
	return nil
}

func (s *FirestoreTenantStore) collectionRef() *firestore.CollectionRef {
	return s.client.Collection(s.collection)
}

func tenantMetadataDocument(metadata tenant.Metadata) map[string]any {
	return map[string]any{
		"authProvider": metadata.AuthProvider,
		"tenantId":     metadata.TenantID,
		"updatedAt":    metadata.UpdatedAt,
	}
}
