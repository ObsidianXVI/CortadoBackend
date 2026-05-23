package store

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/your-org/cortado/control-plane/internal/auth"
	"google.golang.org/api/iterator"
)

type FirestoreAuthStoreConfig struct {
	APIKeysCollection       string
	RefreshTokensCollection string
}

type FirestoreAuthStore struct {
	apiKeysCollection       string
	client                  *firestore.Client
	refreshTokensCollection string
}

func NewFirestoreAuthStore(client *firestore.Client, cfg FirestoreAuthStoreConfig) *FirestoreAuthStore {
	if cfg.APIKeysCollection == "" {
		cfg.APIKeysCollection = auth.DefaultAPIKeysCollection
	}
	if cfg.RefreshTokensCollection == "" {
		cfg.RefreshTokensCollection = auth.DefaultRefreshTokensCollection
	}

	return &FirestoreAuthStore{
		apiKeysCollection:       cfg.APIKeysCollection,
		client:                  client,
		refreshTokensCollection: cfg.RefreshTokensCollection,
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

func (s *FirestoreAuthStore) SaveRefreshToken(ctx context.Context, token auth.RefreshTokenRecord) error {
	if _, err := s.client.Collection(s.refreshTokensCollection).Doc(token.RefreshToken).Create(ctx, token); err != nil {
		return fmt.Errorf("create refresh token document: %w", err)
	}
	return nil
}
