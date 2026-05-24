package filesync

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	filesyncpb "github.com/your-org/cortado/agent/gen/filesync/v1"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	suppressWindow = 5 * time.Second
)

type WorkspaceFileService interface {
	DeletePath(ctx context.Context, workspaceID, path string) error
	ListDir(ctx context.Context, workspaceID, path string) ([]*agentpb.DirectoryEntry, error)
	ReadFile(ctx context.Context, workspaceID, path string, writer io.Writer) error
	RenamePath(ctx context.Context, workspaceID, oldPath, newPath string) error
	WatchFiles(ctx context.Context, workspaceID string, send func(*agentpb.FileEvent) error) error
	WriteFile(ctx context.Context, workspaceID, path string, createMissingDirs bool, reader io.Reader) (*agentpb.WriteFileResponse, error)
}

type ServiceConfig struct {
	Logger         *log.Logger
	WorkspaceFiles WorkspaceFileService
}

type Service struct {
	filesyncpb.UnimplementedFileSyncServiceServer

	logger         *log.Logger
	workspaceFiles WorkspaceFileService
}

type recvResult struct {
	err error
	msg *filesyncpb.SyncMessage
}

type suppressedEvent struct {
	checksum  string
	eventType filesyncpb.OpType
	expiresAt time.Time
}

type opSuppressor struct {
	entries map[string]suppressedEvent
	mu      sync.Mutex
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.WorkspaceFiles == nil {
		return nil, fmt.Errorf("workspace files service must not be nil")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	return &Service{
		logger:         logger,
		workspaceFiles: cfg.WorkspaceFiles,
	}, nil
}

func (s *Service) Sync(stream filesyncpb.FileSyncService_SyncServer) error {
	firstMessage, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}

	stateVector := firstMessage.GetStateVector()
	if stateVector == nil {
		return status.Error(codes.InvalidArgument, "first sync message must be a state_vector payload")
	}

	workspaceID := strings.TrimSpace(stateVector.GetWorkspaceId())
	if workspaceID == "" {
		return status.Error(codes.InvalidArgument, "state_vector.workspace_id is required")
	}

	remoteChecksums, err := s.collectWorkspaceStateVector(stream.Context(), workspaceID)
	if err != nil {
		return status.Errorf(codes.Internal, "collect workspace state vector: %v", err)
	}

	sendMu := &sync.Mutex{}
	send := func(message *filesyncpb.SyncMessage) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return stream.Send(message)
	}

	if err := send(&filesyncpb.SyncMessage{
		Payload: &filesyncpb.SyncMessage_SyncPlan{
			SyncPlan: buildSyncPlan(stateVector.GetChecksums(), remoteChecksums),
		},
	}); err != nil {
		return err
	}

	relayCtx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	recvCh := make(chan recvResult, 1)
	go func() {
		defer close(recvCh)
		for {
			message, recvErr := stream.Recv()
			recvCh <- recvResult{msg: message, err: recvErr}
			if recvErr != nil {
				return
			}
		}
	}()

	relayErrCh := make(chan error, 1)
	suppressor := newOpSuppressor()
	go func() {
		err := s.workspaceFiles.WatchFiles(relayCtx, workspaceID, func(event *agentpb.FileEvent) error {
			if suppressor.ShouldSuppress(event) {
				return nil
			}

			message, buildErr := s.syncMessageFromEvent(relayCtx, workspaceID, event)
			if buildErr != nil {
				return buildErr
			}
			if message == nil {
				return nil
			}
			return send(message)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			select {
			case relayErrCh <- err:
			default:
			}
		}
		close(relayErrCh)
	}()

	for {
		select {
		case relayErr, ok := <-relayErrCh:
			if ok && relayErr != nil {
				return status.Errorf(codes.Internal, "relay workspace file sync: %v", relayErr)
			}
			relayErrCh = nil
		case result, ok := <-recvCh:
			if !ok {
				return nil
			}
			if result.err == io.EOF {
				return nil
			}
			if result.err != nil {
				return result.err
			}

			fileOp := result.msg.GetFileOp()
			if fileOp == nil {
				return status.Error(codes.InvalidArgument, "only file_op payloads are accepted after the initial state vector")
			}

			if err := s.applyFileOp(stream.Context(), workspaceID, fileOp); err != nil {
				return withStatusPrefix(fmt.Sprintf("apply file op for %s", fileOp.GetPath()), err)
			}
			suppressor.Remember(fileOp)

			if err := send(&filesyncpb.SyncMessage{
				Payload: &filesyncpb.SyncMessage_Ack{
					Ack: &filesyncpb.Ack{OpId: fileOp.GetOpId()},
				},
			}); err != nil {
				return err
			}
		}
	}
}

func (s *Service) collectWorkspaceStateVector(ctx context.Context, workspaceID string) (map[string]string, error) {
	checksums := make(map[string]string)

	var walk func(path string) error
	walk = func(path string) error {
		entries, err := s.workspaceFiles.ListDir(ctx, workspaceID, path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry == nil {
				continue
			}

			childPath := entry.GetName()
			if path != "" {
				childPath = path + "/" + entry.GetName()
			}

			if entry.GetIsDir() {
				if err := walk(childPath); err != nil {
					return err
				}
				continue
			}

			hasher := newXXHashWriter()
			if err := s.workspaceFiles.ReadFile(ctx, workspaceID, childPath, hasher); err != nil {
				return err
			}
			checksums[childPath] = hasher.HexChecksum()
		}

		return nil
	}

	if err := walk(""); err != nil {
		return nil, err
	}
	return checksums, nil
}

func buildSyncPlan(localChecksums, remoteChecksums map[string]string) *filesyncpb.SyncPlan {
	paths := make([]string, 0, len(localChecksums)+len(remoteChecksums))
	seen := make(map[string]struct{}, len(localChecksums)+len(remoteChecksums))
	for path := range localChecksums {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for path := range remoteChecksums {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	entries := make([]*filesyncpb.SyncPlanEntry, 0, len(paths))
	for _, path := range paths {
		localChecksum, hasLocal := localChecksums[path]
		remoteChecksum, hasRemote := remoteChecksums[path]

		var direction filesyncpb.SyncDirection
		switch {
		case hasLocal && !hasRemote:
			direction = filesyncpb.SyncDirection_SYNC_DIRECTION_LOCAL_TO_CLOUD
		case !hasLocal && hasRemote:
			direction = filesyncpb.SyncDirection_SYNC_DIRECTION_CLOUD_TO_LOCAL
		case hasLocal && hasRemote && localChecksum != remoteChecksum:
			// Feature 6.1 starts in local-mirror mode, so the local daemon wins
			// the first flat-map diff when both sides already have content.
			direction = filesyncpb.SyncDirection_SYNC_DIRECTION_LOCAL_TO_CLOUD
		default:
			continue
		}

		entries = append(entries, &filesyncpb.SyncPlanEntry{
			Path:      path,
			Direction: direction,
		})
	}

	return &filesyncpb.SyncPlan{Entries: entries}
}

func (s *Service) applyFileOp(ctx context.Context, workspaceID string, fileOp *filesyncpb.FileOp) error {
	if fileOp == nil {
		return status.Error(codes.InvalidArgument, "file_op payload is required")
	}

	path := strings.TrimSpace(fileOp.GetPath())
	if path == "" && fileOp.GetOpType() != filesyncpb.OpType_OP_TYPE_DELETE {
		return status.Error(codes.InvalidArgument, "file_op.path is required")
	}

	switch fileOp.GetOpType() {
	case filesyncpb.OpType_OP_TYPE_CREATE, filesyncpb.OpType_OP_TYPE_MODIFY:
		if len(fileOp.GetPatch()) > 0 && len(fileOp.GetContent()) == 0 {
			return status.Error(codes.Unimplemented, "patch-only file ops are not implemented yet")
		}
		if len(fileOp.GetContent()) == 0 {
			return status.Error(codes.InvalidArgument, "file_op.content is required for create/modify operations")
		}
		_, err := s.workspaceFiles.WriteFile(ctx, workspaceID, path, true, bytes.NewReader(fileOp.GetContent()))
		return err
	case filesyncpb.OpType_OP_TYPE_DELETE:
		if path == "" {
			return status.Error(codes.InvalidArgument, "file_op.path is required for delete operations")
		}
		return s.workspaceFiles.DeletePath(ctx, workspaceID, path)
	case filesyncpb.OpType_OP_TYPE_RENAME:
		oldPath := strings.TrimSpace(fileOp.GetOldPath())
		if oldPath == "" || path == "" {
			return status.Error(codes.InvalidArgument, "file_op.old_path and file_op.path are required for rename operations")
		}
		return s.workspaceFiles.RenamePath(ctx, workspaceID, oldPath, path)
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported op type %v", fileOp.GetOpType())
	}
}

func (s *Service) syncMessageFromEvent(ctx context.Context, workspaceID string, event *agentpb.FileEvent) (*filesyncpb.SyncMessage, error) {
	if event == nil {
		return nil, nil
	}

	fileOp := &filesyncpb.FileOp{
		OpId:     uuid.NewString(),
		Path:     event.GetPath(),
		OpType:   mapEventType(event.GetType()),
		Checksum: append([]byte(nil), event.GetChecksum()...),
	}
	if fileOp.GetOpType() == filesyncpb.OpType_OP_TYPE_UNSPECIFIED {
		return nil, nil
	}

	switch fileOp.GetOpType() {
	case filesyncpb.OpType_OP_TYPE_CREATE, filesyncpb.OpType_OP_TYPE_MODIFY:
		content, checksum, err := s.readWorkspaceFile(ctx, workspaceID, event.GetPath())
		if err != nil {
			if errors.Is(err, workspace.ErrNotFound) {
				fileOp.OpType = filesyncpb.OpType_OP_TYPE_DELETE
				fileOp.Content = nil
				fileOp.Checksum = nil
			} else {
				return nil, err
			}
		} else {
			fileOp.Content = content
			if len(fileOp.GetChecksum()) == 0 {
				fileOp.Checksum = checksum
			}
		}
	case filesyncpb.OpType_OP_TYPE_DELETE:
		fileOp.Content = nil
	case filesyncpb.OpType_OP_TYPE_RENAME:
		// The agent watcher currently reports the old path only for rename events.
		// Emit a delete so the daemon can converge without inventing a second path.
		fileOp.OpType = filesyncpb.OpType_OP_TYPE_DELETE
		fileOp.Content = nil
		fileOp.Checksum = nil
	}

	return &filesyncpb.SyncMessage{
		Payload: &filesyncpb.SyncMessage_FileOp{FileOp: fileOp},
	}, nil
}

func (s *Service) readWorkspaceFile(ctx context.Context, workspaceID, path string) ([]byte, []byte, error) {
	var buffer bytes.Buffer
	if err := s.workspaceFiles.ReadFile(ctx, workspaceID, path, &buffer); err != nil {
		return nil, nil, err
	}

	checksum := binary.BigEndian.AppendUint64(nil, xxhash64(buffer.Bytes()))
	return buffer.Bytes(), checksum, nil
}

func mapEventType(eventType agentpb.FileEventType) filesyncpb.OpType {
	switch eventType {
	case agentpb.FileEventType_FILE_EVENT_TYPE_CREATED:
		return filesyncpb.OpType_OP_TYPE_CREATE
	case agentpb.FileEventType_FILE_EVENT_TYPE_MODIFIED:
		return filesyncpb.OpType_OP_TYPE_MODIFY
	case agentpb.FileEventType_FILE_EVENT_TYPE_DELETED:
		return filesyncpb.OpType_OP_TYPE_DELETE
	case agentpb.FileEventType_FILE_EVENT_TYPE_RENAMED:
		return filesyncpb.OpType_OP_TYPE_RENAME
	default:
		return filesyncpb.OpType_OP_TYPE_UNSPECIFIED
	}
}

func newOpSuppressor() *opSuppressor {
	return &opSuppressor{
		entries: make(map[string]suppressedEvent),
	}
}

func (s *opSuppressor) Remember(fileOp *filesyncpb.FileOp) {
	if fileOp == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.rememberLocked(fileOp.GetPath(), suppressedEvent{
		checksum:  checksumKey(fileOp.GetChecksum()),
		eventType: fileOp.GetOpType(),
		expiresAt: now.Add(suppressWindow),
	})

	if fileOp.GetOpType() == filesyncpb.OpType_OP_TYPE_RENAME {
		s.rememberLocked(fileOp.GetOldPath(), suppressedEvent{
			eventType: filesyncpb.OpType_OP_TYPE_DELETE,
			expiresAt: now.Add(suppressWindow),
		})
		s.rememberLocked(fileOp.GetPath(), suppressedEvent{
			checksum:  checksumKey(fileOp.GetChecksum()),
			eventType: filesyncpb.OpType_OP_TYPE_CREATE,
			expiresAt: now.Add(suppressWindow),
		})
	}
}

func (s *opSuppressor) rememberLocked(path string, event suppressedEvent) {
	if strings.TrimSpace(path) == "" {
		return
	}
	s.entries[path] = event
}

func (s *opSuppressor) ShouldSuppress(event *agentpb.FileEvent) bool {
	if event == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := event.GetPath()
	entry, ok := s.entries[path]
	if !ok {
		return false
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.entries, path)
		return false
	}

	if entry.eventType != mapEventType(event.GetType()) && !(entry.eventType == filesyncpb.OpType_OP_TYPE_DELETE && event.GetType() == agentpb.FileEventType_FILE_EVENT_TYPE_RENAMED) {
		return false
	}
	if entry.checksum != "" && checksumKey(event.GetChecksum()) != entry.checksum {
		return false
	}

	delete(s.entries, path)
	return true
}

func checksumKey(checksum []byte) string {
	if len(checksum) == 0 {
		return ""
	}
	return hex.EncodeToString(checksum)
}

type xxhashWriter struct {
	hasher hash.Hash64
}

func newXXHashWriter() *xxhashWriter {
	return &xxhashWriter{hasher: xxhash.New()}
}

func (w *xxhashWriter) Write(p []byte) (int, error) {
	return w.hasher.Write(p)
}

func (w *xxhashWriter) HexChecksum() string {
	return strconv.FormatUint(w.hasher.Sum64(), 16)
}

func xxhash64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

func withStatusPrefix(prefix string, err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return status.Errorf(st.Code(), "%s: %s", prefix, st.Message())
	}
	return status.Errorf(codes.Internal, "%s: %v", prefix, err)
}
