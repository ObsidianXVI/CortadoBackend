package filesync

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/cortado/daemon/internal/state"
)

func TestEngineApplyRemoteChangeWritesWhenNoConflict(t *testing.T) {
	store := mustOpenStore(t)
	defer store.Close()

	engine := mustNewEngine(t, store, nil)
	path := filepath.Join(t.TempDir(), "main.txt")

	result, err := engine.ApplyRemoteChange(context.Background(), RemoteFileChange{
		Path:        path,
		Content:     []byte("hello"),
		ModTimeUnix: 10,
		RemoteClock: 1,
	})
	if err != nil {
		t.Fatalf("apply remote change: %v", err)
	}
	if !result.Applied {
		t.Fatal("expected remote change to be applied")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read applied file: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected applied content: %q", content)
	}
	if result.State.RemoteClock != 1 || result.State.SyncedClock != 1 {
		t.Fatalf("unexpected clocks after apply: %#v", result.State)
	}
}

func TestEngineMergesTextConflictAndLogsResolution(t *testing.T) {
	store := mustOpenStore(t)
	defer store.Close()

	root := t.TempDir()
	path := filepath.Join(root, "main.txt")
	baseContent := []byte("hello\nworld\n")
	localContent := []byte("hello\nlocal\n")
	mergedContent := []byte("hello\nlocal\nremote\n")
	if err := os.WriteFile(path, localContent, 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	if err := store.UpsertFileState(state.FileState{
		Path:        path,
		Checksum:    checksumString(localContent),
		ModTimeUnix: 20,
		SyncedClock: 1,
		LocalClock:  2,
		RemoteClock: 1,
	}); err != nil {
		t.Fatalf("seed file state: %v", err)
	}

	engine := mustNewEngine(t, store, mergeRunnerStub{
		merged: mergedContent,
	})
	if _, err := engine.MarkSynced(path, baseContent); err != nil {
		t.Fatalf("mark synced: %v", err)
	}
	if err := store.UpsertFileState(state.FileState{
		Path:        path,
		Checksum:    checksumString(localContent),
		ModTimeUnix: 20,
		SyncedClock: 1,
		LocalClock:  2,
		RemoteClock: 1,
	}); err != nil {
		t.Fatalf("restore local clock state: %v", err)
	}

	result, err := engine.ApplyRemoteChange(context.Background(), RemoteFileChange{
		Path:        path,
		Content:     []byte("hello\nremote\n"),
		ModTimeUnix: 30,
		RemoteClock: 2,
	})
	if err != nil {
		t.Fatalf("apply remote conflict change: %v", err)
	}
	if !result.Applied || !result.Merged {
		t.Fatalf("expected merged apply result, got %#v", result)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read merged file: %v", err)
	}
	if string(content) != string(mergedContent) {
		t.Fatalf("unexpected merged content: %q", content)
	}
	if result.State.SyncedClock != result.State.LocalClock || result.State.SyncedClock != result.State.RemoteClock {
		t.Fatalf("expected merged clocks to converge, got %#v", result.State)
	}

	mergeLog, err := os.ReadFile(engine.mergeLogPath)
	if err != nil {
		t.Fatalf("read merge log: %v", err)
	}
	if !strings.Contains(string(mergeLog), "status=merged") {
		t.Fatalf("expected merged log entry, got %q", mergeLog)
	}
}

func TestEngineReturnsConflictNoticeWhenTextMergeFails(t *testing.T) {
	store := mustOpenStore(t)
	defer store.Close()

	root := t.TempDir()
	path := filepath.Join(root, "main.txt")
	baseContent := []byte("hello\nworld\n")
	localContent := []byte("hello\nlocal\n")
	if err := os.WriteFile(path, localContent, 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	if err := store.UpsertFileState(state.FileState{
		Path:        path,
		Checksum:    checksumString(localContent),
		ModTimeUnix: 20,
		SyncedClock: 1,
		LocalClock:  2,
		RemoteClock: 1,
	}); err != nil {
		t.Fatalf("seed file state: %v", err)
	}

	engine := mustNewEngine(t, store, mergeRunnerStub{
		err: ErrMergeConflict,
	})
	if _, err := engine.MarkSynced(path, baseContent); err != nil {
		t.Fatalf("mark synced: %v", err)
	}
	if err := store.UpsertFileState(state.FileState{
		Path:        path,
		Checksum:    checksumString(localContent),
		ModTimeUnix: 20,
		SyncedClock: 1,
		LocalClock:  2,
		RemoteClock: 1,
	}); err != nil {
		t.Fatalf("restore local clock state: %v", err)
	}

	result, err := engine.ApplyRemoteChange(context.Background(), RemoteFileChange{
		Path:        path,
		Content:     []byte("hello\nremote\n"),
		ModTimeUnix: 30,
		RemoteClock: 2,
	})
	if err != nil {
		t.Fatalf("apply remote conflict change: %v", err)
	}
	if result.Conflict == nil {
		t.Fatalf("expected conflict notice, got %#v", result)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read local file: %v", err)
	}
	if string(content) != string(localContent) {
		t.Fatalf("expected local file to remain unchanged, got %q", content)
	}

	mergeLog, err := os.ReadFile(engine.mergeLogPath)
	if err != nil {
		t.Fatalf("read merge log: %v", err)
	}
	if !strings.Contains(string(mergeLog), "status=conflict") {
		t.Fatalf("expected conflict log entry, got %q", mergeLog)
	}
}

func TestEngineUsesBinaryLastWriteWinsByModTime(t *testing.T) {
	store := mustOpenStore(t)
	defer store.Close()

	root := t.TempDir()
	path := filepath.Join(root, "asset.bin")
	localContent := []byte{0x00, 0x01, 0x02}
	remoteContent := []byte{0x03, 0x04, 0x05}
	if err := os.WriteFile(path, localContent, 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	if err := store.UpsertFileState(state.FileState{
		Path:        path,
		Checksum:    checksumString(localContent),
		ModTimeUnix: 20,
		SyncedClock: 1,
		LocalClock:  1,
		RemoteClock: 1,
	}); err != nil {
		t.Fatalf("seed file state: %v", err)
	}

	engine := mustNewEngine(t, store, nil)
	result, err := engine.ApplyRemoteChange(context.Background(), RemoteFileChange{
		Path:        path,
		Content:     remoteContent,
		ModTimeUnix: 30,
		RemoteClock: 2,
	})
	if err != nil {
		t.Fatalf("apply remote binary change: %v", err)
	}
	if !result.Applied {
		t.Fatalf("expected remote binary content to win, got %#v", result)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read binary file: %v", err)
	}
	if string(content) != string(remoteContent) {
		t.Fatalf("unexpected binary content: %v", content)
	}
}

type mergeRunnerStub struct {
	err    error
	merged []byte
}

func (m mergeRunnerStub) Merge(context.Context, []byte, []byte, []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return append([]byte(nil), m.merged...), nil
}

func mustNewEngine(t *testing.T, store *state.Store, mergeRunner MergeRunner) *Engine {
	t.Helper()

	engine, err := NewEngine(EngineConfig{
		Logger:      log.New(io.Discard, "", 0),
		MergeRunner: mergeRunner,
		StateStore:  store,
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return engine
}

func mustOpenStore(t *testing.T) *state.Store {
	t.Helper()

	store, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}
