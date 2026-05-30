package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRuntimeDirsFromEnvCreatesConfiguredDirectories(t *testing.T) {
	root := t.TempDir()
	homeDir := filepath.Join(root, "home")
	cacheDir := filepath.Join(root, "cache")
	tmpDir := filepath.Join(root, "tmp")

	t.Setenv("HOME", homeDir)
	t.Setenv("TMPDIR", tmpDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("PUB_CACHE", cacheDir)
	t.Setenv("GRADLE_USER_HOME", filepath.Join(root, "gradle"))
	t.Setenv("npm_config_cache", filepath.Join(root, "npm"))

	if err := ensureRuntimeDirsFromEnv(); err != nil {
		t.Fatalf("ensure runtime dirs: %v", err)
	}

	for _, dir := range []string{
		homeDir,
		tmpDir,
		cacheDir,
		filepath.Join(root, "gradle"),
		filepath.Join(root, "npm"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}
}
