package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	fileChunkSize   = 256 * 1024
	fileDebounce    = 50 * time.Millisecond
	fileHashDelay   = 10 * time.Millisecond
	defaultFileMode = 0o644
)

var excludedWatchDirs = map[string]struct{}{
	".git":         {},
	"build":        {},
	"node_modules": {},
}

func (s *AgentServer) ListDir(ctx context.Context, req *pb.ListDirRequest) (*pb.ListDirResponse, error) {
	path, err := s.resolveWorkspacePath(req.GetPath(), true)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, mapFilesystemError("list dir", err)
	}

	response := &pb.ListDirResponse{
		Entries: make([]*pb.DirectoryEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list dir entry info: %v", err)
		}
		response.Entries = append(response.Entries, &pb.DirectoryEntry{
			Name:        entry.Name(),
			Size:        info.Size(),
			IsDir:       entry.IsDir(),
			ModTime:     timestamppb.New(info.ModTime()),
			Permissions: uint32(info.Mode().Perm()),
		})
	}

	return response, nil
}

func (s *AgentServer) ReadFile(req *pb.ReadFileRequest, stream pb.WorkspaceAgentService_ReadFileServer) error {
	path, err := s.resolveWorkspacePath(req.GetPath(), false)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return mapFilesystemError("open file", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return status.Errorf(codes.Internal, "stat file: %v", err)
	}
	if info.IsDir() {
		return status.Error(codes.InvalidArgument, "path must reference a file")
	}

	hasher := xxhash.New()
	buf := make([]byte, fileChunkSize)
	var seq int32

	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			chunkData := append([]byte(nil), buf[:n]...)
			if _, err := hasher.Write(chunkData); err != nil {
				return status.Errorf(codes.Internal, "hash file chunk: %v", err)
			}

			chunk := &pb.ReadFileChunk{
				Data: chunkData,
				Seq:  seq,
			}
			seq++

			if errors.Is(readErr, io.EOF) {
				chunk.IsLast = true
				chunk.Checksum = encodeXXHash64(hasher.Sum64())
			}

			if err := stream.Send(&pb.ReadFileResponse{Chunk: chunk}); err != nil {
				return err
			}
		}

		if errors.Is(readErr, io.EOF) {
			if n == 0 {
				if err := stream.Send(&pb.ReadFileResponse{
					Chunk: &pb.ReadFileChunk{
						Seq:      seq,
						IsLast:   true,
						Checksum: encodeXXHash64(hasher.Sum64()),
					},
				}); err != nil {
					return err
				}
			}
			return nil
		}
		if readErr != nil {
			return status.Errorf(codes.Internal, "read file: %v", readErr)
		}
	}
}

func (s *AgentServer) WriteFile(stream pb.WorkspaceAgentService_WriteFileServer) (err error) {
	var (
		bytesWritten int64
		expectedSeq  int32
		sawLast      bool
		targetPath   string
		tempPath     string
		file         *os.File
	)

	hasher := xxhash.New()

	defer func() {
		if file != nil {
			_ = file.Close()
		}
		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
	}()

	for {
		req, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return recvErr
		}

		chunk := req.GetChunk()
		if chunk == nil {
			return status.Error(codes.InvalidArgument, "write file chunk is required")
		}
		if sawLast {
			return status.Error(codes.InvalidArgument, "received chunk after final chunk")
		}
		if chunk.GetSeq() != expectedSeq {
			return status.Errorf(codes.InvalidArgument, "unexpected chunk sequence: got %d want %d", chunk.GetSeq(), expectedSeq)
		}

		path, err := s.resolveWorkspacePath(chunk.GetPath(), false)
		if err != nil {
			return err
		}

		if targetPath == "" {
			targetPath = path
			tempPath, file, err = prepareWriteTarget(targetPath)
			if err != nil {
				return err
			}
		} else if path != targetPath {
			return status.Error(codes.InvalidArgument, "write file path cannot change within a stream")
		}

		if len(chunk.GetData()) > 0 {
			if _, err := file.Write(chunk.GetData()); err != nil {
				return status.Errorf(codes.Internal, "write temp file: %v", err)
			}
			if _, err := hasher.Write(chunk.GetData()); err != nil {
				return status.Errorf(codes.Internal, "hash written chunk: %v", err)
			}
			bytesWritten += int64(len(chunk.GetData()))
		}

		expectedSeq++

		if chunk.GetIsLast() {
			sawLast = true
			computedChecksum := encodeXXHash64(hasher.Sum64())
			if !bytes.Equal(chunk.GetChecksum(), computedChecksum) {
				return status.Error(codes.InvalidArgument, "write file checksum mismatch")
			}

			if err := file.Close(); err != nil {
				return status.Errorf(codes.Internal, "close temp file: %v", err)
			}
			file = nil

			if err := os.Rename(tempPath, targetPath); err != nil {
				return mapFilesystemError("rename temp file", err)
			}
			tempPath = ""

			return stream.SendAndClose(&pb.WriteFileResponse{
				BytesWritten: bytesWritten,
				Checksum:     computedChecksum,
			})
		}
	}

	if !sawLast {
		return status.Error(codes.InvalidArgument, "missing final chunk")
	}

	return nil
}

func (s *AgentServer) MakeDir(ctx context.Context, req *pb.MakeDirRequest) (*pb.MakeDirResponse, error) {
	path, err := s.resolveWorkspacePath(req.GetPath(), false)
	if err != nil {
		return nil, err
	}

	if err := os.Mkdir(path, 0o755); err != nil {
		return nil, mapFilesystemError("make dir", err)
	}

	return &pb.MakeDirResponse{}, nil
}

func (s *AgentServer) RenamePath(ctx context.Context, req *pb.RenamePathRequest) (*pb.RenamePathResponse, error) {
	oldPath, err := s.resolveWorkspacePath(req.GetOldPath(), false)
	if err != nil {
		return nil, err
	}

	newPath, err := s.resolveWorkspacePath(req.GetNewPath(), false)
	if err != nil {
		return nil, err
	}
	if oldPath == newPath {
		return nil, status.Error(codes.InvalidArgument, "source and destination paths must differ")
	}

	if _, err := os.Stat(oldPath); err != nil {
		return nil, mapFilesystemError("stat source path", err)
	}

	parentInfo, err := os.Stat(filepath.Dir(newPath))
	if err != nil {
		return nil, mapFilesystemError("stat destination parent", err)
	}
	if !parentInfo.IsDir() {
		return nil, status.Error(codes.InvalidArgument, "destination parent must be a directory")
	}

	if _, err := os.Stat(newPath); err == nil {
		return nil, status.Error(codes.AlreadyExists, "destination path already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, mapFilesystemError("stat destination path", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, mapFilesystemError("rename path", err)
	}

	return &pb.RenamePathResponse{}, nil
}

func (s *AgentServer) DeletePath(ctx context.Context, req *pb.DeletePathRequest) (*pb.DeletePathResponse, error) {
	path, err := s.resolveWorkspacePath(req.GetPath(), false)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); err != nil {
		return nil, mapFilesystemError("stat path", err)
	}
	if err := os.RemoveAll(path); err != nil {
		return nil, mapFilesystemError("delete path", err)
	}

	return &pb.DeletePathResponse{}, nil
}

func (s *AgentServer) WatchFiles(req *pb.WatchFilesRequest, stream pb.WorkspaceAgentService_WatchFilesServer) error {
	root, err := s.resolveWorkspacePath("", true)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return status.Errorf(codes.Internal, "create file watcher: %v", err)
	}
	defer watcher.Close()

	if err := addRecursiveWatch(watcher, root, root); err != nil {
		return mapFilesystemError("watch workspace", err)
	}

	ctx := stream.Context()
	eventsCh := make(chan *pb.FileEvent, 32)
	errCh := make(chan error, 1)

	type pendingEvent struct {
		eventType pb.FileEventType
		timer     *time.Timer
	}

	var (
		mu      sync.Mutex
		pending = make(map[string]*pendingEvent)
	)

	scheduleEvent := func(path string, eventType pb.FileEventType) {
		if shouldExcludePath(root, path) {
			return
		}
		if eventType == pb.FileEventType_FILE_EVENT_TYPE_UNSPECIFIED {
			return
		}

		if eventType == pb.FileEventType_FILE_EVENT_TYPE_CREATED {
			if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
				if err := addRecursiveWatch(watcher, root, path); err != nil {
					select {
					case errCh <- status.Errorf(codes.Internal, "watch new directory: %v", err):
					default:
					}
					return
				}
			}
		}

		mu.Lock()
		existing := pending[path]
		if existing != nil {
			existing.eventType = mergeEventType(existing.eventType, eventType)
			if existing.timer != nil {
				existing.timer.Stop()
			}
			eventType = existing.eventType
		}

		timer := time.AfterFunc(fileDebounce, func() {
			time.Sleep(fileHashDelay)

			event, buildErr := s.buildFileEvent(root, path, eventType)

			mu.Lock()
			delete(pending, path)
			mu.Unlock()

			if buildErr != nil {
				select {
				case errCh <- buildErr:
				default:
				}
				return
			}
			if event == nil {
				return
			}

			select {
			case eventsCh <- event:
			case <-ctx.Done():
			}
		})

		pending[path] = &pendingEvent{
			eventType: eventType,
			timer:     timer,
		}
		mu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, item := range pending {
				if item.timer != nil {
					item.timer.Stop()
				}
			}
			mu.Unlock()
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			scheduleEvent(filepath.Clean(event.Name), mapWatchEventType(event))
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return status.Errorf(codes.Internal, "watch files: %v", watchErr)
		case event := <-eventsCh:
			if event == nil {
				continue
			}
			if err := stream.Send(&pb.WatchFilesResponse{Event: event}); err != nil {
				return err
			}
		case streamErr := <-errCh:
			if streamErr != nil {
				return streamErr
			}
		}
	}
}

func (s *AgentServer) buildFileEvent(root, path string, eventType pb.FileEventType) (*pb.FileEvent, error) {
	relativePath, err := relativeWorkspacePath(root, path)
	if err != nil || relativePath == "." {
		return nil, err
	}

	event := &pb.FileEvent{
		Path: relativePath,
		Type: eventType,
	}

	if eventType == pb.FileEventType_FILE_EVENT_TYPE_DELETED || eventType == pb.FileEventType_FILE_EVENT_TYPE_RENAMED {
		return event, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			event.Type = pb.FileEventType_FILE_EVENT_TYPE_DELETED
			return event, nil
		}
		return nil, mapFilesystemError("stat watched path", err)
	}
	if info.IsDir() {
		return event, nil
	}

	checksum, err := checksumFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			event.Type = pb.FileEventType_FILE_EVENT_TYPE_DELETED
			return event, nil
		}
		return nil, mapFilesystemError("checksum watched path", err)
	}
	event.Checksum = checksum

	return event, nil
}

func (s *AgentServer) resolveWorkspacePath(requestPath string, allowRoot bool) (string, error) {
	root := filepath.Clean(s.workspaceRoot)
	if strings.TrimSpace(root) == "" {
		root = defaultWorkspaceRoot
	}

	requestPath = strings.TrimSpace(requestPath)

	var resolved string
	switch {
	case requestPath == "", requestPath == ".":
		resolved = root
	case filepath.IsAbs(requestPath):
		resolved = filepath.Clean(requestPath)
	default:
		resolved = filepath.Join(root, requestPath)
	}

	if resolved != root && !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
		return "", status.Error(codes.InvalidArgument, "path must stay within the workspace root")
	}
	if !allowRoot && resolved == root {
		return "", status.Error(codes.InvalidArgument, "path must not be the workspace root")
	}

	return resolved, nil
}

func relativeWorkspacePath(root, path string) (string, error) {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return "", status.Errorf(codes.Internal, "relative workspace path: %v", err)
	}
	relativePath = filepath.Clean(relativePath)
	if relativePath == "." {
		return ".", nil
	}
	if strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) || relativePath == ".." {
		return "", status.Error(codes.InvalidArgument, "path must stay within the workspace root")
	}
	return filepath.ToSlash(relativePath), nil
}

func prepareWriteTarget(targetPath string) (string, *os.File, error) {
	parentDir := filepath.Dir(targetPath)
	parentInfo, err := os.Stat(parentDir)
	if err != nil {
		return "", nil, mapFilesystemError("stat parent directory", err)
	}
	if !parentInfo.IsDir() {
		return "", nil, status.Error(codes.InvalidArgument, "parent path must be a directory")
	}

	mode := fs.FileMode(defaultFileMode)
	if targetInfo, err := os.Stat(targetPath); err == nil {
		if targetInfo.IsDir() {
			return "", nil, status.Error(codes.InvalidArgument, "path must reference a file")
		}
		mode = targetInfo.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", nil, mapFilesystemError("stat target file", err)
	}

	tempPath := filepath.Join(parentDir, fmt.Sprintf(".cortado-tmp-%s", uuid.NewString()))
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return "", nil, mapFilesystemError("create temp file", err)
	}

	return tempPath, file, nil
}

func addRecursiveWatch(watcher *fsnotify.Watcher, root, start string) error {
	return filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && shouldExcludePath(root, path) {
			return filepath.SkipDir
		}
		if err := watcher.Add(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		return nil
	})
}

func shouldExcludePath(root, path string) bool {
	relativePath, err := filepath.Rel(root, path)
	if err != nil || relativePath == "." {
		return false
	}

	for _, part := range strings.Split(filepath.Clean(relativePath), string(os.PathSeparator)) {
		if _, excluded := excludedWatchDirs[part]; excluded {
			return true
		}
	}
	return false
}

func mapWatchEventType(event fsnotify.Event) pb.FileEventType {
	switch {
	case event.Has(fsnotify.Rename):
		return pb.FileEventType_FILE_EVENT_TYPE_RENAMED
	case event.Has(fsnotify.Remove):
		return pb.FileEventType_FILE_EVENT_TYPE_DELETED
	case event.Has(fsnotify.Create):
		return pb.FileEventType_FILE_EVENT_TYPE_CREATED
	case event.Has(fsnotify.Write), event.Has(fsnotify.Chmod):
		return pb.FileEventType_FILE_EVENT_TYPE_MODIFIED
	default:
		return pb.FileEventType_FILE_EVENT_TYPE_UNSPECIFIED
	}
}

func mergeEventType(current, next pb.FileEventType) pb.FileEventType {
	switch {
	case next == pb.FileEventType_FILE_EVENT_TYPE_RENAMED || current == pb.FileEventType_FILE_EVENT_TYPE_RENAMED:
		return pb.FileEventType_FILE_EVENT_TYPE_RENAMED
	case next == pb.FileEventType_FILE_EVENT_TYPE_DELETED:
		return pb.FileEventType_FILE_EVENT_TYPE_DELETED
	case current == pb.FileEventType_FILE_EVENT_TYPE_CREATED:
		return pb.FileEventType_FILE_EVENT_TYPE_CREATED
	case next != pb.FileEventType_FILE_EVENT_TYPE_UNSPECIFIED:
		return next
	default:
		return current
	}
}

func checksumFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := xxhash.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}

	return encodeXXHash64(hasher.Sum64()), nil
}

func encodeXXHash64(value uint64) []byte {
	checksum := make([]byte, 8)
	binary.BigEndian.PutUint64(checksum, value)
	return checksum
}

func mapFilesystemError(action string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrExist):
		return status.Errorf(codes.AlreadyExists, "%s: %v", action, err)
	case errors.Is(err, os.ErrNotExist):
		return status.Errorf(codes.NotFound, "%s: %v", action, err)
	case errors.Is(err, os.ErrPermission):
		return status.Errorf(codes.PermissionDenied, "%s: %v", action, err)
	default:
		return status.Errorf(codes.Internal, "%s: %v", action, err)
	}
}
