package state_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/your-org/cortado/daemon/internal/state"
)

func TestOpenEnsuresSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")

	store, err := state.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite for verification: %v", err)
	}
	defer db.Close()

	for _, tableName := range []string{"daemon_metadata", "watched_roots", "file_state"} {
		var found string
		if err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
			tableName,
		).Scan(&found); err != nil {
			t.Fatalf("lookup table %s: %v", tableName, err)
		}
		if found != tableName {
			t.Fatalf("unexpected table lookup result: got %q want %q", found, tableName)
		}
	}

	if got := store.SchemaVersion(); got != state.SchemaVersion {
		t.Fatalf("unexpected schema version: got %d want %d", got, state.SchemaVersion)
	}
}

func TestUpsertAndLookupFileState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")

	store, err := state.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	input := state.FileState{
		Path:        "/workspace/lib/main.dart",
		Checksum:    "abc123",
		ModTimeUnix: 42,
		SyncedClock: 7,
	}
	if err := store.UpsertFileState(input); err != nil {
		t.Fatalf("upsert file state: %v", err)
	}

	got, found, err := store.LookupFileState(input.Path)
	if err != nil {
		t.Fatalf("lookup file state: %v", err)
	}
	if !found {
		t.Fatal("expected file state to be found")
	}
	if got != input {
		t.Fatalf("unexpected file state: got %#v want %#v", got, input)
	}

	if err := store.DeleteFileState(input.Path); err != nil {
		t.Fatalf("delete file state: %v", err)
	}
	if _, found, err := store.LookupFileState(input.Path); err != nil {
		t.Fatalf("lookup deleted file state: %v", err)
	} else if found {
		t.Fatal("expected deleted file state to be absent")
	}
}
