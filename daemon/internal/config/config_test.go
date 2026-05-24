package config_test

import (
	"path/filepath"
	"testing"

	"github.com/your-org/cortado/daemon/internal/config"
)

func TestFromEnvUsesLoopbackDefaults(t *testing.T) {
	t.Setenv("HOME", "/tmp/cortado-home")
	t.Setenv("CORTADO_DAEMON_LISTEN_ADDR", "")
	t.Setenv("CORTADO_DAEMON_STATE_PATH", "")

	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatalf("from env: %v", err)
	}

	if cfg.ListenAddr != config.DefaultListenAddr {
		t.Fatalf("unexpected listen addr: got %q want %q", cfg.ListenAddr, config.DefaultListenAddr)
	}

	wantStatePath := filepath.Join("/tmp/cortado-home", ".cortado", "state.db")
	if cfg.StatePath != wantStatePath {
		t.Fatalf("unexpected state path: got %q want %q", cfg.StatePath, wantStatePath)
	}
}

func TestValidateRejectsNonLoopbackListener(t *testing.T) {
	cfg := config.Config{
		ListenAddr: "0.0.0.0:9731",
		StatePath:  "/tmp/state.db",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-loopback address to be rejected")
	}
}

func TestFromEnvParsesWatchRoots(t *testing.T) {
	t.Setenv("HOME", "/tmp/cortado-home")
	t.Setenv("CORTADO_DAEMON_WATCH_ROOTS", "/tmp/one:/tmp/two")

	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatalf("from env: %v", err)
	}

	if got, want := len(cfg.WatchRoots), 2; got != want {
		t.Fatalf("unexpected watch root count: got %d want %d", got, want)
	}
	if cfg.WatchRoots[0] != "/tmp/one" || cfg.WatchRoots[1] != "/tmp/two" {
		t.Fatalf("unexpected watch roots: %#v", cfg.WatchRoots)
	}
}
