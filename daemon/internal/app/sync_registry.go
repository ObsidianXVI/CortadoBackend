package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/your-org/cortado/daemon/internal/filesync"
)

type SyncState string

const (
	SyncStateIdle       SyncState = "IDLE"
	SyncStateSyncing    SyncState = "SYNCING"
	SyncStateConflicted SyncState = "CONFLICTED"
)

type SyncStatus struct {
	LocalPath     string    `json:"localPath"`
	Message       string    `json:"message,omitempty"`
	State         SyncState `json:"state"`
	WorkspaceID   string    `json:"workspaceId"`
	WorkspacePath string    `json:"workspacePath,omitempty"`
}

type syncKey struct {
	localPath   string
	workspaceID string
}

type SyncRegistry struct {
	mu       sync.RWMutex
	statuses map[syncKey]SyncStatus
}

func NewSyncRegistry() *SyncRegistry {
	return &SyncRegistry{
		statuses: make(map[syncKey]SyncStatus),
	}
}

func (r *SyncRegistry) StartSync(localPath, workspaceID string) (SyncStatus, error) {
	key, err := newSyncKey(localPath, workspaceID)
	if err != nil {
		return SyncStatus{}, err
	}

	status := SyncStatus{
		LocalPath:     key.localPath,
		State:         SyncStateSyncing,
		WorkspaceID:   key.workspaceID,
		WorkspacePath: "/",
	}

	r.mu.Lock()
	r.statuses[key] = status
	r.mu.Unlock()

	return status, nil
}

func (r *SyncRegistry) StopSync(localPath, workspaceID string) (SyncStatus, error) {
	key, err := newSyncKey(localPath, workspaceID)
	if err != nil {
		return SyncStatus{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	status, ok := r.statuses[key]
	if !ok {
		status = SyncStatus{
			LocalPath:     key.localPath,
			State:         SyncStateIdle,
			WorkspaceID:   key.workspaceID,
			WorkspacePath: "/",
		}
	} else {
		status.State = SyncStateIdle
		status.Message = ""
	}
	r.statuses[key] = status
	return status, nil
}

func (r *SyncRegistry) GetSyncStatus(localPath, workspaceID string) (SyncStatus, error) {
	key, err := newSyncKey(localPath, workspaceID)
	if err != nil {
		return SyncStatus{}, err
	}

	r.mu.RLock()
	status, ok := r.statuses[key]
	r.mu.RUnlock()
	if ok {
		return status, nil
	}

	return SyncStatus{
		LocalPath:     key.localPath,
		State:         SyncStateIdle,
		WorkspaceID:   key.workspaceID,
		WorkspacePath: "/",
	}, nil
}

func (r *SyncRegistry) MarkConflict(notice filesync.ConflictNotice) {
	conflictPath := filepath.Clean(notice.Path)

	r.mu.Lock()
	defer r.mu.Unlock()

	var (
		bestKey   syncKey
		bestMatch SyncStatus
		found     bool
	)
	for key, status := range r.statuses {
		relativePath, ok := relativeWorkspacePath(key.localPath, conflictPath)
		if !ok {
			continue
		}
		if !found || len(key.localPath) > len(bestKey.localPath) {
			found = true
			bestKey = key
			bestMatch = status
			bestMatch.WorkspacePath = relativePath
		}
	}

	if !found {
		return
	}

	bestMatch.State = SyncStateConflicted
	bestMatch.Message = notice.Reason
	r.statuses[bestKey] = bestMatch
}

func newSyncKey(localPath, workspaceID string) (syncKey, error) {
	trimmedPath := strings.TrimSpace(localPath)
	if trimmedPath == "" {
		return syncKey{}, fmt.Errorf("localPath must not be empty")
	}

	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if trimmedWorkspaceID == "" {
		return syncKey{}, fmt.Errorf("workspaceId must not be empty")
	}

	cleanLocalPath := filepath.Clean(trimmedPath)
	if !filepath.IsAbs(cleanLocalPath) {
		return syncKey{}, fmt.Errorf("localPath must be absolute")
	}

	return syncKey{
		localPath:   cleanLocalPath,
		workspaceID: trimmedWorkspaceID,
	}, nil
}

func relativeWorkspacePath(rootPath, targetPath string) (string, bool) {
	relativePath, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return "", false
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", false
	}
	if relativePath == "." {
		return "/", true
	}
	return "/" + filepath.ToSlash(relativePath), true
}
