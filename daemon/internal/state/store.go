package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const SchemaVersion = 1

type Store struct {
	db   *sql.DB
	path string
}

type FileState struct {
	Path        string
	Checksum    string
	ModTimeUnix int64
	SyncedClock int64
	LocalClock  int64
	RemoteClock int64
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{
		db:   db,
		path: path,
	}
	if err := store.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) SchemaVersion() int {
	return SchemaVersion
}

func (s *Store) UpsertFileState(fileState FileState) error {
	_, err := s.db.Exec(
		`INSERT INTO file_state (path, checksum, mod_time, synced_clock, local_clock, remote_clock)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET
		   checksum = excluded.checksum,
		   mod_time = excluded.mod_time,
		   synced_clock = excluded.synced_clock,
		   local_clock = excluded.local_clock,
		   remote_clock = excluded.remote_clock`,
		fileState.Path,
		fileState.Checksum,
		fileState.ModTimeUnix,
		fileState.SyncedClock,
		fileState.LocalClock,
		fileState.RemoteClock,
	)
	if err != nil {
		return fmt.Errorf("upsert file state for %s: %w", fileState.Path, err)
	}
	return nil
}

func (s *Store) DeleteFileState(path string) error {
	if _, err := s.db.Exec(`DELETE FROM file_state WHERE path = ?`, path); err != nil {
		return fmt.Errorf("delete file state for %s: %w", path, err)
	}
	return nil
}

func (s *Store) LookupFileState(path string) (FileState, bool, error) {
	var fileState FileState
	err := s.db.QueryRow(
		`SELECT path, checksum, mod_time, synced_clock, local_clock, remote_clock FROM file_state WHERE path = ?`,
		path,
	).Scan(
		&fileState.Path,
		&fileState.Checksum,
		&fileState.ModTimeUnix,
		&fileState.SyncedClock,
		&fileState.LocalClock,
		&fileState.RemoteClock,
	)
	if err == sql.ErrNoRows {
		return FileState{}, false, nil
	}
	if err != nil {
		return FileState{}, false, fmt.Errorf("lookup file state for %s: %w", path, err)
	}

	return fileState, true, nil
}

func NextLogicalClock(fileState FileState) int64 {
	next := fileState.SyncedClock
	if fileState.LocalClock > next {
		next = fileState.LocalClock
	}
	if fileState.RemoteClock > next {
		next = fileState.RemoteClock
	}
	return next + 1
}

func (s *Store) ensureSchema() error {
	statements := []string{
		`PRAGMA journal_mode = WAL;`,
		`CREATE TABLE IF NOT EXISTS daemon_metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		fmt.Sprintf(
			`INSERT INTO daemon_metadata (key, value)
			 VALUES ('schema_version', '%d')
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value;`,
			SchemaVersion,
		),
		`CREATE TABLE IF NOT EXISTS watched_roots (
			path TEXT PRIMARY KEY,
			display_name TEXT NOT NULL DEFAULT '',
			added_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS file_state (
			path TEXT PRIMARY KEY,
			checksum TEXT NOT NULL DEFAULT '',
			mod_time INTEGER NOT NULL DEFAULT 0,
			synced_clock INTEGER NOT NULL DEFAULT 0,
			local_clock INTEGER NOT NULL DEFAULT 0,
			remote_clock INTEGER NOT NULL DEFAULT 0
		);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("apply daemon state schema: %w", err)
		}
	}

	return nil
}
