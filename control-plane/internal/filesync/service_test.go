package filesync_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	filesyncpb "github.com/your-org/cortado/agent/gen/filesync/v1"
	"github.com/your-org/cortado/control-plane/internal/filesync"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestServiceBuildsInitialSyncPlanFromStateVector(t *testing.T) {
	t.Parallel()

	stub := newWorkspaceFilesStub(map[string][]byte{
		"same.txt":        []byte("same"),
		"remote-only.txt": []byte("remote"),
		"remote-diff.txt": []byte("cloud"),
	})
	client, cleanup := newSyncClient(t, stub)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Sync(ctx)
	if err != nil {
		t.Fatalf("open sync stream: %v", err)
	}

	if err := stream.Send(&filesyncpb.SyncMessage{
		Payload: &filesyncpb.SyncMessage_StateVector{
			StateVector: &filesyncpb.StateVector{
				WorkspaceId: "ws-123",
				Checksums: map[string]string{
					"local-only.txt":  checksumString([]byte("local")),
					"remote-diff.txt": checksumString([]byte("local-wins")),
					"same.txt":        checksumString([]byte("same")),
				},
			},
		},
	}); err != nil {
		t.Fatalf("send state vector: %v", err)
	}

	response, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv sync plan: %v", err)
	}

	plan := response.GetSyncPlan()
	if plan == nil {
		t.Fatalf("expected sync plan, got %#v", response)
	}

	got := make([]string, 0, len(plan.GetEntries()))
	for _, entry := range plan.GetEntries() {
		got = append(got, entry.GetPath()+":"+entry.GetDirection().String())
	}

	want := []string{
		"local-only.txt:SYNC_DIRECTION_LOCAL_TO_CLOUD",
		"remote-diff.txt:SYNC_DIRECTION_LOCAL_TO_CLOUD",
		"remote-only.txt:SYNC_DIRECTION_CLOUD_TO_LOCAL",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected sync plan: got %v want %v", got, want)
	}
}

func TestServiceAppliesInboundFileOpsAndAcks(t *testing.T) {
	t.Parallel()

	stub := newWorkspaceFilesStub(nil)
	client, cleanup := newSyncClient(t, stub)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Sync(ctx)
	if err != nil {
		t.Fatalf("open sync stream: %v", err)
	}

	if err := stream.Send(&filesyncpb.SyncMessage{
		Payload: &filesyncpb.SyncMessage_StateVector{
			StateVector: &filesyncpb.StateVector{WorkspaceId: "ws-456"},
		},
	}); err != nil {
		t.Fatalf("send state vector: %v", err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("recv initial sync plan: %v", err)
	}

	ops := []*filesyncpb.FileOp{
		{
			OpId:    "op-create",
			Path:    "docs/note.txt",
			OpType:  filesyncpb.OpType_OP_TYPE_CREATE,
			Content: []byte("hello"),
		},
		{
			OpId:     "op-rename",
			OldPath:  "docs/note.txt",
			Path:     "docs/todo.txt",
			OpType:   filesyncpb.OpType_OP_TYPE_RENAME,
			Checksum: checksumBytes([]byte("hello")),
		},
		{
			OpId:   "op-delete",
			Path:   "docs/todo.txt",
			OpType: filesyncpb.OpType_OP_TYPE_DELETE,
		},
	}

	for _, op := range ops {
		if err := stream.Send(&filesyncpb.SyncMessage{
			Payload: &filesyncpb.SyncMessage_FileOp{FileOp: op},
		}); err != nil {
			t.Fatalf("send file op %s: %v", op.GetOpId(), err)
		}

		response, err := stream.Recv()
		if err != nil {
			t.Fatalf("recv ack for %s: %v", op.GetOpId(), err)
		}
		if response.GetAck() == nil || response.GetAck().GetOpId() != op.GetOpId() {
			t.Fatalf("unexpected ack for %s: %#v", op.GetOpId(), response)
		}
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()

	if got := string(stub.files["docs/note.txt"]); got != "" {
		t.Fatalf("expected note.txt to be renamed away, got %q", got)
	}
	if _, ok := stub.files["docs/todo.txt"]; ok {
		t.Fatalf("expected todo.txt to be deleted")
	}
	if got := strings.Join(stub.writes, ","); got != "docs/note.txt" {
		t.Fatalf("unexpected writes: %s", got)
	}
	if got := strings.Join(stub.renames, ","); got != "docs/note.txt->docs/todo.txt" {
		t.Fatalf("unexpected renames: %s", got)
	}
	if got := strings.Join(stub.deletes, ","); got != "docs/todo.txt" {
		t.Fatalf("unexpected deletes: %s", got)
	}
}

func TestServiceForwardsWorkspaceWatchEvents(t *testing.T) {
	t.Parallel()

	stub := newWorkspaceFilesStub(map[string][]byte{
		"src/main.go": []byte("package main\n"),
	})
	client, cleanup := newSyncClient(t, stub)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Sync(ctx)
	if err != nil {
		t.Fatalf("open sync stream: %v", err)
	}

	if err := stream.Send(&filesyncpb.SyncMessage{
		Payload: &filesyncpb.SyncMessage_StateVector{
			StateVector: &filesyncpb.StateVector{WorkspaceId: "ws-789"},
		},
	}); err != nil {
		t.Fatalf("send state vector: %v", err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("recv initial sync plan: %v", err)
	}

	stub.EmitWatchEvent(&agentpb.FileEvent{
		Path:     "src/main.go",
		Type:     agentpb.FileEventType_FILE_EVENT_TYPE_MODIFIED,
		Checksum: checksumBytes([]byte("package main\n")),
	})

	response, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv forwarded file op: %v", err)
	}

	fileOp := response.GetFileOp()
	if fileOp == nil {
		t.Fatalf("expected file op, got %#v", response)
	}
	if fileOp.GetPath() != "src/main.go" {
		t.Fatalf("unexpected file path: %q", fileOp.GetPath())
	}
	if fileOp.GetOpType() != filesyncpb.OpType_OP_TYPE_MODIFY {
		t.Fatalf("unexpected op type: %v", fileOp.GetOpType())
	}
	if got := string(fileOp.GetContent()); got != "package main\n" {
		t.Fatalf("unexpected file content: %q", got)
	}
}

type workspaceFilesStub struct {
	deletes []string
	files   map[string][]byte
	renames []string
	watchCh chan *agentpb.FileEvent
	writes  []string
	mu      sync.Mutex
}

func newWorkspaceFilesStub(files map[string][]byte) *workspaceFilesStub {
	cloned := make(map[string][]byte, len(files))
	for path, content := range files {
		cloned[path] = append([]byte(nil), content...)
	}
	return &workspaceFilesStub{
		files:   cloned,
		watchCh: make(chan *agentpb.FileEvent, 8),
	}
}

func (s *workspaceFilesStub) DeletePath(_ context.Context, _ string, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.files, path)
	s.deletes = append(s.deletes, path)
	return nil
}

func (s *workspaceFilesStub) ListDir(_ context.Context, _ string, path string) ([]*agentpb.DirectoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := ""
	if path != "" {
		prefix = path + "/"
	}

	entriesByName := make(map[string]*agentpb.DirectoryEntry)
	for filePath, content := range s.files {
		if prefix != "" && !strings.HasPrefix(filePath, prefix) {
			continue
		}
		rest := strings.TrimPrefix(filePath, prefix)
		if prefix == "" && strings.HasPrefix(rest, filePath+"/") {
			continue
		}
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		name := parts[0]
		if len(parts) == 1 {
			entriesByName[name] = &agentpb.DirectoryEntry{
				Name:  name,
				Size:  int64(len(content)),
				IsDir: false,
			}
			continue
		}
		if _, ok := entriesByName[name]; !ok {
			entriesByName[name] = &agentpb.DirectoryEntry{
				Name:  name,
				IsDir: true,
			}
		}
	}

	names := make([]string, 0, len(entriesByName))
	for name := range entriesByName {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]*agentpb.DirectoryEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, entriesByName[name])
	}
	return entries, nil
}

func (s *workspaceFilesStub) ReadFile(_ context.Context, _ string, path string, writer io.Writer) error {
	s.mu.Lock()
	content, ok := s.files[path]
	s.mu.Unlock()
	if !ok {
		return workspace.ErrNotFound
	}

	_, err := io.Copy(writer, bytes.NewReader(content))
	return err
}

func (s *workspaceFilesStub) RenamePath(_ context.Context, _ string, oldPath, newPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, ok := s.files[oldPath]
	if !ok {
		return workspace.ErrNotFound
	}
	delete(s.files, oldPath)
	s.files[newPath] = content
	s.renames = append(s.renames, oldPath+"->"+newPath)
	return nil
}

func (s *workspaceFilesStub) WatchFiles(ctx context.Context, _ string, send func(*agentpb.FileEvent) error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-s.watchCh:
			if err := send(event); err != nil {
				return err
			}
		}
	}
}

func (s *workspaceFilesStub) WriteFile(_ context.Context, _ string, path string, _ bool, reader io.Reader) (*agentpb.WriteFileResponse, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.files[path] = content
	s.writes = append(s.writes, path)
	return &agentpb.WriteFileResponse{
		BytesWritten: int64(len(content)),
		Checksum:     checksumBytes(content),
	}, nil
}

func (s *workspaceFilesStub) EmitWatchEvent(event *agentpb.FileEvent) {
	s.watchCh <- event
}

func newSyncClient(t *testing.T, workspaceFiles *workspaceFilesStub) (filesyncpb.FileSyncServiceClient, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()

	service, err := filesync.NewService(filesync.ServiceConfig{WorkspaceFiles: workspaceFiles})
	if err != nil {
		t.Fatalf("new file sync service: %v", err)
	}
	filesyncpb.RegisterFileSyncServiceServer(grpcServer, service)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Errorf("serve grpc sync server: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return filesyncpb.NewFileSyncServiceClient(conn), cleanup
}

func checksumString(content []byte) string {
	return strconv.FormatUint(xxhash.Sum64(content), 16)
}

func checksumBytes(content []byte) []byte {
	return binary.BigEndian.AppendUint64(nil, xxhash.Sum64(content))
}
