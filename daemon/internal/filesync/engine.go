package filesync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cespare/xxhash/v2"
	"github.com/your-org/cortado/daemon/internal/state"
)

var ErrMergeConflict = errors.New("text merge conflict")

type MergeRunner interface {
	Merge(ctx context.Context, base, local, remote []byte) ([]byte, error)
}

type EngineConfig struct {
	ConflictSink ConflictSink
	Logger       *log.Logger
	MergeLogPath string
	MergeRunner  MergeRunner
	SnapshotDir  string
	StateStore   *state.Store
}

type Engine struct {
	conflictSink ConflictSink
	logger       *log.Logger
	mergeLogPath string
	mergeRunner  MergeRunner
	snapshots    snapshotStore
	stateStore   *state.Store
}

type RemoteFileChange struct {
	Content     []byte
	ModTimeUnix int64
	Path        string
	RemoteClock int64
}

type ApplyResult struct {
	Applied  bool
	Conflict *ConflictNotice
	Merged   bool
	State    state.FileState
}

type ConflictNotice struct {
	LastSyncedClock int64  `json:"lastSyncedClock"`
	LocalClock      int64  `json:"localClock"`
	Path            string `json:"path"`
	Reason          string `json:"reason"`
	RemoteClock     int64  `json:"remoteClock"`
}

type ConflictSink interface {
	PublishConflict(ConflictNotice)
}

type diff3Runner struct{}

type snapshotStore struct {
	dir string
}

func NewEngine(cfg EngineConfig) (*Engine, error) {
	if cfg.StateStore == nil {
		return nil, fmt.Errorf("state store must not be nil")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	rootDir := filepath.Dir(cfg.StateStore.Path())
	snapshotDir := cfg.SnapshotDir
	if strings.TrimSpace(snapshotDir) == "" {
		snapshotDir = filepath.Join(rootDir, "snapshots")
	}
	mergeLogPath := cfg.MergeLogPath
	if strings.TrimSpace(mergeLogPath) == "" {
		mergeLogPath = filepath.Join(rootDir, "merge.log")
	}

	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return nil, fmt.Errorf("create snapshot dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(mergeLogPath), 0o755); err != nil {
		return nil, fmt.Errorf("create merge log dir: %w", err)
	}

	mergeRunner := cfg.MergeRunner
	if mergeRunner == nil {
		mergeRunner = diff3Runner{}
	}

	return &Engine{
		conflictSink: cfg.ConflictSink,
		logger:       logger,
		mergeLogPath: mergeLogPath,
		mergeRunner:  mergeRunner,
		snapshots: snapshotStore{
			dir: snapshotDir,
		},
		stateStore: cfg.StateStore,
	}, nil
}

func (e *Engine) MarkSynced(path string, content []byte) (state.FileState, error) {
	fileState, found, err := e.stateStore.LookupFileState(path)
	if err != nil {
		return state.FileState{}, err
	}
	if !found {
		fileState = state.FileState{Path: path}
	}

	if len(content) > 0 || fileState.Checksum == "" {
		fileState.Checksum = checksumString(content)
	}
	fileState.SyncedClock = maxClock(fileState.LocalClock, fileState.RemoteClock)
	if err := e.stateStore.UpsertFileState(fileState); err != nil {
		return state.FileState{}, err
	}
	if err := e.snapshots.Save(path, content); err != nil {
		return state.FileState{}, err
	}

	return fileState, nil
}

func (e *Engine) ApplyRemoteChange(ctx context.Context, change RemoteFileChange) (ApplyResult, error) {
	path := strings.TrimSpace(change.Path)
	if path == "" {
		return ApplyResult{}, fmt.Errorf("remote change path must not be empty")
	}
	if change.RemoteClock <= 0 {
		return ApplyResult{}, fmt.Errorf("remote clock must be positive")
	}

	fileState, found, err := e.stateStore.LookupFileState(path)
	if err != nil {
		return ApplyResult{}, err
	}
	if !found {
		fileState = state.FileState{Path: path}
	}

	if change.RemoteClock <= fileState.RemoteClock && change.RemoteClock <= fileState.SyncedClock {
		return ApplyResult{State: fileState}, nil
	}

	localContent, localInfo, _, err := readLocalFile(path)
	if err != nil {
		return ApplyResult{}, err
	}

	localChanged := fileState.LocalClock > fileState.SyncedClock
	remoteChanged := change.RemoteClock > fileState.SyncedClock
	if localChanged && remoteChanged {
		baseContent, ok, baseErr := e.snapshots.Load(path)
		if baseErr != nil {
			return ApplyResult{}, baseErr
		}
		if ok && isTextContent(baseContent) && isTextContent(localContent) && isTextContent(change.Content) {
			merged, mergeErr := e.mergeRunner.Merge(ctx, baseContent, localContent, change.Content)
			if mergeErr == nil {
				modTimeUnix, writeErr := writeLocalFile(path, merged, maxInt64(change.ModTimeUnix, currentModTimeUnix(localInfo)))
				if writeErr != nil {
					return ApplyResult{}, writeErr
				}

				resolvedClock := state.NextLogicalClock(fileState)
				fileState = state.FileState{
					Path:        path,
					Checksum:    checksumString(merged),
					ModTimeUnix: modTimeUnix,
					SyncedClock: resolvedClock,
					LocalClock:  resolvedClock,
					RemoteClock: resolvedClock,
				}
				if err := e.stateStore.UpsertFileState(fileState); err != nil {
					return ApplyResult{}, err
				}
				if err := e.snapshots.Save(path, merged); err != nil {
					return ApplyResult{}, err
				}
				_ = e.logResolution("merged", fileState, "")

				return ApplyResult{
					Applied: true,
					Merged:  true,
					State:   fileState,
				}, nil
			}
			if !errors.Is(mergeErr, ErrMergeConflict) {
				return ApplyResult{}, mergeErr
			}
		}

		if !isTextContent(localContent) || !isTextContent(change.Content) {
			if localInfo != nil && localInfo.ModTime().UnixMilli() > change.ModTimeUnix {
				fileState.RemoteClock = maxInt64(fileState.RemoteClock, change.RemoteClock)
				if err := e.stateStore.UpsertFileState(fileState); err != nil {
					return ApplyResult{}, err
				}
				_ = e.logResolution("binary_local_wins", fileState, "")
				return ApplyResult{State: fileState}, nil
			}

			modTimeUnix, writeErr := writeLocalFile(path, change.Content, change.ModTimeUnix)
			if writeErr != nil {
				return ApplyResult{}, writeErr
			}

			fileState.Path = path
			fileState.Checksum = checksumString(change.Content)
			fileState.ModTimeUnix = modTimeUnix
			fileState.RemoteClock = maxInt64(fileState.RemoteClock, change.RemoteClock)
			fileState.SyncedClock = fileState.RemoteClock
			if !localChanged {
				fileState.LocalClock = maxInt64(fileState.LocalClock, fileState.SyncedClock)
			}
			if err := e.stateStore.UpsertFileState(fileState); err != nil {
				return ApplyResult{}, err
			}
			if err := e.snapshots.Save(path, change.Content); err != nil {
				return ApplyResult{}, err
			}
			_ = e.logResolution("binary_remote_wins", fileState, "")
			return ApplyResult{
				Applied: true,
				State:   fileState,
			}, nil
		}

		fileState.RemoteClock = maxInt64(fileState.RemoteClock, change.RemoteClock)
		if err := e.stateStore.UpsertFileState(fileState); err != nil {
			return ApplyResult{}, err
		}

		conflict := &ConflictNotice{
			Path:            path,
			Reason:          "text conflict requires manual resolution",
			LocalClock:      fileState.LocalClock,
			RemoteClock:     change.RemoteClock,
			LastSyncedClock: fileState.SyncedClock,
		}
		_ = e.logResolution("conflict", fileState, conflict.Reason)
		if e.conflictSink != nil {
			e.conflictSink.PublishConflict(*conflict)
		}

		return ApplyResult{
			Conflict: conflict,
			State:    fileState,
		}, nil
	}

	if localChanged && !remoteChanged {
		return ApplyResult{State: fileState}, nil
	}

	modTimeUnix, err := writeLocalFile(path, change.Content, change.ModTimeUnix)
	if err != nil {
		return ApplyResult{}, err
	}

	fileState.Path = path
	fileState.Checksum = checksumString(change.Content)
	fileState.ModTimeUnix = modTimeUnix
	fileState.RemoteClock = maxInt64(fileState.RemoteClock, change.RemoteClock)
	fileState.SyncedClock = fileState.RemoteClock
	if !localChanged {
		fileState.LocalClock = maxInt64(fileState.LocalClock, fileState.SyncedClock)
	}

	if err := e.stateStore.UpsertFileState(fileState); err != nil {
		return ApplyResult{}, err
	}
	if err := e.snapshots.Save(path, change.Content); err != nil {
		return ApplyResult{}, err
	}
	_ = e.logResolution("applied_remote", fileState, "")

	return ApplyResult{
		Applied: true,
		State:   fileState,
	}, nil
}

func (e *Engine) logResolution(status string, fileState state.FileState, details string) error {
	entry := fmt.Sprintf(
		"%s path=%s status=%s local_clock=%d remote_clock=%d last_synced_clock=%d details=%q\n",
		time.Now().UTC().Format(time.RFC3339),
		fileState.Path,
		status,
		fileState.LocalClock,
		fileState.RemoteClock,
		fileState.SyncedClock,
		details,
	)
	if err := os.WriteFile(e.mergeLogPath, append(existingFile(e.mergeLogPath), []byte(entry)...), 0o644); err != nil {
		if e.logger != nil {
			e.logger.Printf("append merge log: %v", err)
		}
		return err
	}
	return nil
}

func readLocalFile(path string) ([]byte, fs.FileInfo, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("stat local file %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, info, false, fmt.Errorf("local path %s is a directory", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false, fmt.Errorf("read local file %s: %w", path, err)
	}
	return content, info, true, nil
}

func writeLocalFile(path string, content []byte, modTimeUnix int64) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, fmt.Errorf("create parent dir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return 0, fmt.Errorf("write local file %s: %w", path, err)
	}

	if modTimeUnix > 0 {
		modTime := time.UnixMilli(modTimeUnix)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			return 0, fmt.Errorf("set mod time for %s: %w", path, err)
		}
		return modTimeUnix, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat local file %s: %w", path, err)
	}
	return info.ModTime().UnixMilli(), nil
}

func existingFile(path string) []byte {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return content
}

func currentModTimeUnix(info fs.FileInfo) int64 {
	if info == nil {
		return 0
	}
	return info.ModTime().UnixMilli()
}

func checksumString(content []byte) string {
	return fmt.Sprintf("%x", xxhash.Sum64(content))
}

func isTextContent(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	if !utf8.Valid(content) {
		return false
	}
	for _, b := range content {
		if b == 0 {
			return false
		}
	}
	return true
}

func maxClock(localClock, remoteClock int64) int64 {
	if localClock > remoteClock {
		return localClock
	}
	return remoteClock
}

func maxInt64(values ...int64) int64 {
	var max int64
	for i, value := range values {
		if i == 0 || value > max {
			max = value
		}
	}
	return max
}

func (s snapshotStore) Save(path string, content []byte) error {
	return os.WriteFile(s.filePath(path), content, 0o644)
}

func (s snapshotStore) Load(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(s.filePath(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read snapshot for %s: %w", path, err)
	}
	return content, true, nil
}

func (s snapshotStore) filePath(path string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(path)))
	return filepath.Join(s.dir, hex.EncodeToString(sum[:])+".base")
}

func (diff3Runner) Merge(ctx context.Context, base, local, remote []byte) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "cortado-diff3-*")
	if err != nil {
		return nil, fmt.Errorf("create diff3 temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	basePath := filepath.Join(tempDir, "base.txt")
	localPath := filepath.Join(tempDir, "local.txt")
	remotePath := filepath.Join(tempDir, "remote.txt")
	for path, content := range map[string][]byte{
		basePath:   base,
		localPath:  local,
		remotePath: remote,
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return nil, fmt.Errorf("write diff3 input %s: %w", path, err)
		}
	}

	cmd := exec.CommandContext(ctx, "diff3", "-m", localPath, basePath, remotePath)
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return nil, ErrMergeConflict
	}
	return nil, fmt.Errorf("run diff3: %w", err)
}
