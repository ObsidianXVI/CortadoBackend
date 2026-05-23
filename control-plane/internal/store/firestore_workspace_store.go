package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

const defaultWorkspaceCollection = "workspaces"

type FirestoreWorkspaceStoreConfig struct {
	Collection string
}

type FirestoreWorkspaceStore struct {
	client     *firestore.Client
	collection string
}

func NewFirestoreWorkspaceStore(client *firestore.Client, cfg FirestoreWorkspaceStoreConfig) *FirestoreWorkspaceStore {
	if cfg.Collection == "" {
		cfg.Collection = defaultWorkspaceCollection
	}

	return &FirestoreWorkspaceStore{
		client:     client,
		collection: cfg.Collection,
	}
}

func (s *FirestoreWorkspaceStore) Create(ctx context.Context, ws workspace.Workspace) error {
	_, err := s.collectionRef().Doc(ws.ID).Create(ctx, ws)
	if err != nil {
		return fmt.Errorf("create workspace document: %w", err)
	}
	return nil
}

func (s *FirestoreWorkspaceStore) Delete(ctx context.Context, workspaceID string) error {
	_, err := s.collectionRef().Doc(workspaceID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("delete workspace document: %w", err)
	}
	return nil
}

func (s *FirestoreWorkspaceStore) Get(ctx context.Context, workspaceID string) (workspace.Workspace, error) {
	doc, err := s.collectionRef().Doc(workspaceID).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			return workspace.Workspace{}, workspace.ErrNotFound
		}
		return workspace.Workspace{}, fmt.Errorf("get workspace document: %w", err)
	}

	var ws workspace.Workspace
	if err := doc.DataTo(&ws); err != nil {
		return workspace.Workspace{}, fmt.Errorf("decode workspace document: %w", err)
	}
	return ws, nil
}

func (s *FirestoreWorkspaceStore) ListByTenant(ctx context.Context, tenantID string) ([]workspace.Workspace, error) {
	iter := s.collectionRef().Where("tenantId", "==", tenantID).Documents(ctx)
	defer iter.Stop()

	workspaces, err := collectWorkspaceDocuments(iter)
	if err != nil {
		return nil, err
	}
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].CreatedAt.After(workspaces[j].CreatedAt)
	})

	return workspaces, nil
}

func (s *FirestoreWorkspaceStore) ListByStatus(ctx context.Context, status workspace.Status) ([]workspace.Workspace, error) {
	iter := s.collectionRef().Where("status", "==", status).Documents(ctx)
	defer iter.Stop()

	return collectWorkspaceDocuments(iter)
}

func (s *FirestoreWorkspaceStore) ListInactiveSince(ctx context.Context, threshold time.Time) ([]workspace.Workspace, error) {
	iter := s.collectionRef().Where("lastActiveAt", "<=", threshold).Documents(ctx)
	defer iter.Stop()

	workspaces, err := collectWorkspaceDocuments(iter)
	if err != nil {
		return nil, err
	}

	filtered := workspaces[:0]
	for _, ws := range workspaces {
		switch ws.Status {
		case workspace.StatusDeleted, workspace.StatusStopped:
			continue
		default:
			filtered = append(filtered, ws)
		}
	}

	return filtered, nil
}

func (s *FirestoreWorkspaceStore) UpdateLastActive(ctx context.Context, workspaceID string, observedAt time.Time) (workspace.Workspace, error) {
	_, err := s.collectionRef().Doc(workspaceID).Update(ctx, []firestore.Update{
		{Path: "lastActiveAt", Value: observedAt},
		{Path: "updatedAt", Value: observedAt},
	})
	if err != nil {
		if isNotFound(err) {
			return workspace.Workspace{}, workspace.ErrNotFound
		}
		return workspace.Workspace{}, fmt.Errorf("update workspace last activity: %w", err)
	}

	return s.Get(ctx, workspaceID)
}

func (s *FirestoreWorkspaceStore) UpdateStatus(ctx context.Context, workspaceID string, status workspace.Status, updatedAt time.Time) (workspace.Workspace, error) {
	_, err := s.collectionRef().Doc(workspaceID).Update(ctx, []firestore.Update{
		{Path: "status", Value: status},
		{Path: "updatedAt", Value: updatedAt},
	})
	if err != nil {
		if isNotFound(err) {
			return workspace.Workspace{}, workspace.ErrNotFound
		}
		return workspace.Workspace{}, fmt.Errorf("update workspace status: %w", err)
	}

	return s.Get(ctx, workspaceID)
}

func (s *FirestoreWorkspaceStore) collectionRef() *firestore.CollectionRef {
	return s.client.Collection(s.collection)
}

func isNotFound(err error) bool {
	return grpcstatus.Code(err) == codes.NotFound
}

func collectWorkspaceDocuments(iter *firestore.DocumentIterator) ([]workspace.Workspace, error) {
	workspaces := make([]workspace.Workspace, 0)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return workspaces, nil
		}
		if err != nil {
			return nil, fmt.Errorf("iterate workspace documents: %w", err)
		}

		var ws workspace.Workspace
		if err := doc.DataTo(&ws); err != nil {
			return nil, fmt.Errorf("decode workspace document: %w", err)
		}
		workspaces = append(workspaces, ws)
	}
}
