package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultListenAddr = "127.0.0.1:9731"
	stateDirName      = ".cortado"
	stateFileName     = "state.db"
)

type Config struct {
	ListenAddr string
	StatePath  string
	WatchRoots []string
}

func FromEnv() (Config, error) {
	statePath, err := defaultStatePath()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ListenAddr: envOrDefault("CORTADO_DAEMON_LISTEN_ADDR", DefaultListenAddr),
		StatePath:  envOrDefault("CORTADO_DAEMON_STATE_PATH", statePath),
		WatchRoots: splitPaths(os.Getenv("CORTADO_DAEMON_WATCH_ROOTS")),
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ListenAddr) == "" {
		return fmt.Errorf("listen address must not be empty")
	}
	if strings.TrimSpace(c.StatePath) == "" {
		return fmt.Errorf("state path must not be empty")
	}

	host, port, err := net.SplitHostPort(c.ListenAddr)
	if err != nil {
		return fmt.Errorf("parse listen address %q: %w", c.ListenAddr, err)
	}
	if host != "127.0.0.1" {
		return fmt.Errorf("listen address host must be 127.0.0.1, got %q", host)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("parse listen port %q: %w", port, err)
	}
	if portNumber <= 0 || portNumber > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535, got %d", portNumber)
	}

	for _, root := range c.WatchRoots {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("watch root must be an absolute path: %q", root)
		}
	}

	return nil
}

func defaultStatePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, stateDirName, stateFileName), nil
}

func envOrDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func splitPaths(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, string(os.PathListSeparator))
	roots := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		roots = append(roots, filepath.Clean(part))
	}
	return roots
}
