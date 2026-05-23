package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAgentServerListDir(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "alpha.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(workspaceRoot, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.ListDir(ctx, &pb.ListDirRequest{Path: "."})
	if err != nil {
		t.Fatalf("list dir: %v", err)
	}
	if len(resp.GetEntries()) != 2 {
		t.Fatalf("unexpected entry count: got %d want 2", len(resp.GetEntries()))
	}
	if resp.GetEntries()[0].GetName() != "alpha.txt" || resp.GetEntries()[0].GetIsDir() {
		t.Fatalf("unexpected first entry: %#v", resp.GetEntries()[0])
	}
	if resp.GetEntries()[1].GetName() != "nested" || !resp.GetEntries()[1].GetIsDir() {
		t.Fatalf("unexpected second entry: %#v", resp.GetEntries()[1])
	}

	_, err = client.ListDir(ctx, &pb.ListDirRequest{Path: "../outside"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for traversal, got %v", err)
	}
}

func TestAgentServerReadFileStreamsChunkedContent(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	content := bytes.Repeat([]byte("cortado-"), (fileChunkSize/8)+12345)
	path := filepath.Join(workspaceRoot, "large.txt")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.ReadFile(ctx, &pb.ReadFileRequest{Path: "large.txt"})
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var (
		buf        bytes.Buffer
		lastChunk  *pb.ReadFileChunk
		chunkCount int
	)

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("recv read file chunk: %v", err)
		}
		chunk := resp.GetChunk()
		if chunk == nil {
			t.Fatalf("missing chunk payload")
		}
		if got, want := chunk.GetSeq(), int32(chunkCount); got != want {
			t.Fatalf("unexpected chunk seq: got %d want %d", got, want)
		}
		buf.Write(chunk.GetData())
		lastChunk = chunk
		chunkCount++
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Fatalf("unexpected read content length: got %d want %d", buf.Len(), len(content))
	}
	if chunkCount < 2 {
		t.Fatalf("expected multiple chunks, got %d", chunkCount)
	}
	if lastChunk == nil || !lastChunk.GetIsLast() {
		t.Fatalf("expected final chunk")
	}
	expectedChecksum := encodeXXHash64(xxhash.Sum64(content))
	if !bytes.Equal(lastChunk.GetChecksum(), expectedChecksum) {
		t.Fatalf("unexpected checksum: got %x want %x", lastChunk.GetChecksum(), expectedChecksum)
	}
}

func TestAgentServerWriteFileAndDeletePath(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	content := []byte("hello filesystem api")
	expectedChecksum := encodeXXHash64(xxhash.Sum64(content))

	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writeStream, err := client.WriteFile(ctx)
	if err != nil {
		t.Fatalf("open write stream: %v", err)
	}

	if err := writeStream.Send(&pb.WriteFileRequest{
		Chunk: &pb.WriteFileChunk{
			Path:     "dir/file.txt",
			Seq:      0,
			Data:     content[:5],
			IsLast:   true,
			Checksum: expectedChecksum,
		},
	}); err != nil && status.Code(err) != codes.NotFound {
		t.Fatalf("send missing-parent chunk: %v", err)
	}

	if _, err := writeStream.CloseAndRecv(); status.Code(err) != codes.NotFound {
		t.Fatalf("expected missing parent directory failure, got %v", err)
	}

	if err := os.Mkdir(filepath.Join(workspaceRoot, "dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeStream, err = client.WriteFile(ctx)
	if err != nil {
		t.Fatalf("open write stream: %v", err)
	}

	if err := writeStream.Send(&pb.WriteFileRequest{
		Chunk: &pb.WriteFileChunk{
			Path: "dir/file.txt",
			Seq:  0,
			Data: content[:5],
		},
	}); err != nil {
		t.Fatalf("send first chunk: %v", err)
	}
	if err := writeStream.Send(&pb.WriteFileRequest{
		Chunk: &pb.WriteFileChunk{
			Path:     "dir/file.txt",
			Seq:      1,
			Data:     content[5:],
			IsLast:   true,
			Checksum: expectedChecksum,
		},
	}); err != nil {
		t.Fatalf("send final chunk: %v", err)
	}

	writeResp, err := writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("close write stream: %v", err)
	}
	if writeResp.GetBytesWritten() != int64(len(content)) {
		t.Fatalf("unexpected bytes written: got %d want %d", writeResp.GetBytesWritten(), len(content))
	}
	if !bytes.Equal(writeResp.GetChecksum(), expectedChecksum) {
		t.Fatalf("unexpected response checksum: got %x want %x", writeResp.GetChecksum(), expectedChecksum)
	}

	written, err := os.ReadFile(filepath.Join(workspaceRoot, "dir", "file.txt"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if !bytes.Equal(written, content) {
		t.Fatalf("unexpected file content: got %q want %q", written, content)
	}

	if _, err := client.DeletePath(ctx, &pb.DeletePathRequest{Path: "dir"}); err != nil {
		t.Fatalf("delete path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, "dir")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected path deletion, got stat err %v", err)
	}
}

func TestAgentServerWriteFileRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspaceRoot, "dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writeStream, err := client.WriteFile(ctx)
	if err != nil {
		t.Fatalf("open write stream: %v", err)
	}

	if err := writeStream.Send(&pb.WriteFileRequest{
		Chunk: &pb.WriteFileChunk{
			Path:     "dir/file.txt",
			Seq:      0,
			Data:     []byte("bad checksum"),
			IsLast:   true,
			Checksum: []byte("wrongsum"),
		},
	}); err != nil {
		t.Fatalf("send write chunk: %v", err)
	}

	_, err = writeStream.CloseAndRecv()
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(workspaceRoot, "dir", "file.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no target file, got stat err %v", statErr)
	}
}

func TestAgentServerWatchFilesCreatedEvent(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.WatchFiles(ctx, &pb.WatchFilesRequest{})
	if err != nil {
		t.Fatalf("watch files: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	content := []byte("created")
	if err := os.WriteFile(filepath.Join(workspaceRoot, "created.txt"), content, 0o644); err != nil {
		t.Fatalf("write created file: %v", err)
	}

	event := waitForWatchEvent(t, stream, func(event *pb.FileEvent) bool {
		return event.GetPath() == "created.txt" && event.GetType() == pb.FileEventType_FILE_EVENT_TYPE_CREATED
	})

	expectedChecksum := encodeXXHash64(xxhash.Sum64(content))
	if !bytes.Equal(event.GetChecksum(), expectedChecksum) {
		t.Fatalf("unexpected created checksum: got %x want %x", event.GetChecksum(), expectedChecksum)
	}
}

func TestAgentServerWatchFilesModifiedEvent(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "modified.txt"), []byte("before"), 0o644); err != nil {
		t.Fatalf("write modified seed: %v", err)
	}

	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.WatchFiles(ctx, &pb.WatchFilesRequest{})
	if err != nil {
		t.Fatalf("watch files: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	modifiedContent := []byte("after")
	if err := os.WriteFile(filepath.Join(workspaceRoot, "modified.txt"), modifiedContent, 0o644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

	modifiedEvent := waitForWatchEvent(t, stream, func(event *pb.FileEvent) bool {
		return event.GetPath() == "modified.txt" && event.GetType() == pb.FileEventType_FILE_EVENT_TYPE_MODIFIED
	})
	expectedModifiedChecksum := encodeXXHash64(xxhash.Sum64(modifiedContent))
	if !bytes.Equal(modifiedEvent.GetChecksum(), expectedModifiedChecksum) {
		t.Fatalf("unexpected modified checksum: got %x want %x", modifiedEvent.GetChecksum(), expectedModifiedChecksum)
	}

}

func TestAgentServerWatchFilesDeletedEvent(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "deleted.txt"), []byte("delete me"), 0o644); err != nil {
		t.Fatalf("write deleted seed: %v", err)
	}

	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.WatchFiles(ctx, &pb.WatchFilesRequest{})
	if err != nil {
		t.Fatalf("watch files: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := os.Remove(filepath.Join(workspaceRoot, "deleted.txt")); err != nil {
		t.Fatalf("delete file: %v", err)
	}

	deletedEvent := waitForWatchEvent(t, stream, func(event *pb.FileEvent) bool {
		return event.GetPath() == "deleted.txt" && event.GetType() == pb.FileEventType_FILE_EVENT_TYPE_DELETED
	})
	if len(deletedEvent.GetChecksum()) != 0 {
		t.Fatalf("expected deleted checksum to be empty, got %x", deletedEvent.GetChecksum())
	}
}

func TestAgentServerWatchFilesRenamedEvent(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "rename-old.txt"), []byte("rename me"), 0o644); err != nil {
		t.Fatalf("write rename seed: %v", err)
	}

	client, cleanup := newTestClientWithWorkspaceRoot(t, nil, workspaceRoot)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.WatchFiles(ctx, &pb.WatchFilesRequest{})
	if err != nil {
		t.Fatalf("watch files: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := os.Rename(
		filepath.Join(workspaceRoot, "rename-old.txt"),
		filepath.Join(workspaceRoot, "rename-new.txt"),
	); err != nil {
		t.Fatalf("rename file: %v", err)
	}

	renamedEvent := waitForWatchEvent(t, stream, func(event *pb.FileEvent) bool {
		return event.GetPath() == "rename-old.txt" && event.GetType() == pb.FileEventType_FILE_EVENT_TYPE_RENAMED
	})
	if len(renamedEvent.GetChecksum()) != 0 {
		t.Fatalf("expected renamed checksum to be empty, got %x", renamedEvent.GetChecksum())
	}
}

func waitForWatchEvent(t *testing.T, stream pb.WorkspaceAgentService_WatchFilesClient, match func(*pb.FileEvent) bool) *pb.FileEvent {
	t.Helper()

	seen := make([]string, 0, 8)

	for {
		resp, err := stream.Recv()
		if err != nil {
			t.Fatalf("recv watch event after seeing %v: %v", seen, err)
		}
		event := resp.GetEvent()
		if event == nil {
			continue
		}
		seen = append(seen, fmt.Sprintf("%s:%s:%x", event.GetType().String(), event.GetPath(), event.GetChecksum()))
		if match(event) {
			return event
		}
	}
}
