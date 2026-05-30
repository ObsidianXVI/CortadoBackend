package main

import (
	"fmt"
	"os"
	"strings"
)

var runtimeDirEnvVars = []string{
	"HOME",
	"TMPDIR",
	"XDG_CACHE_HOME",
	"PUB_CACHE",
	"GRADLE_USER_HOME",
	"npm_config_cache",
}

func ensureRuntimeDirsFromEnv() error {
	created := map[string]struct{}{}
	for _, key := range runtimeDirEnvVars {
		dir := strings.TrimSpace(os.Getenv(key))
		if dir == "" {
			continue
		}
		if _, seen := created[dir]; seen {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create runtime directory for %s (%s): %w", key, dir, err)
		}
		created[dir] = struct{}{}
	}
	return nil
}
