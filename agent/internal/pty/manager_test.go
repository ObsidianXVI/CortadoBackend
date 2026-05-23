package pty

import (
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestManagerCreateWriteReadAndKill(t *testing.T) {
	t.Parallel()

	manager := &Manager{}
	session, err := manager.Create("/bin/bash", 80, 24, nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() {
		manager.Kill(session.ID)
	})

	if err := manager.Resize(session.ID, 100, 40); err != nil {
		t.Fatalf("resize session: %v", err)
	}

	if err := manager.Write(session.ID, []byte("echo hello_cortado\nexit\n")); err != nil {
		t.Fatalf("write session: %v", err)
	}

	outputCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		buf := make([]byte, 4096)
		var output strings.Builder

		for {
			n, readErr := manager.Read(session.ID, buf)
			if n > 0 {
				output.Write(buf[:n])
				if strings.Contains(output.String(), "hello_cortado") {
					outputCh <- output.String()
					return
				}
			}
			if readErr != nil {
				if errors.Is(readErr, syscall.EIO) {
					errCh <- readErr
					return
				}
				errCh <- readErr
				return
			}
		}
	}()

	select {
	case output := <-outputCh:
		if !strings.Contains(output, "hello_cortado") {
			t.Fatalf("output missing expected marker: %q", output)
		}
	case readErr := <-errCh:
		t.Fatalf("read before expected output: %v", readErr)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
}

func TestManagerCreateMissingShell(t *testing.T) {
	t.Parallel()

	manager := &Manager{}
	_, err := manager.Create("/definitely/missing/bash", 80, 24, nil)
	if err == nil {
		t.Fatal("expected missing shell error")
	}

	if got, want := err.Error(), "shell /definitely/missing/bash not found in image"; got != want {
		t.Fatalf("unexpected error: got %q want %q", got, want)
	}
}

func TestPTYEnvAddsDefaultTERMWhenUnset(t *testing.T) {
	t.Setenv("TERM", "")
	env := ptyEnv(nil)

	if !hasEnvKey(env, "TERM") {
		t.Fatal("expected TERM to be present")
	}

	found := false
	for _, entry := range env {
		if entry == defaultTerm {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %q in env, got %v", defaultTerm, env)
	}
}

func TestPTYEnvPreservesExplicitTERM(t *testing.T) {
	t.Setenv("TERM", "")

	env := ptyEnv([]string{"TERM=screen-256color"})

	if !hasEnvKey(env, "TERM") {
		t.Fatal("expected TERM to be present")
	}

	if got := lastEnvValue(env, "TERM"); got != "screen-256color" {
		t.Fatalf("expected explicit TERM to win, got %q in %v", got, env)
	}

	for _, entry := range env {
		if entry == defaultTerm {
			t.Fatalf("expected default TERM to be omitted when explicit TERM is present: %v", env)
		}
	}
}

func lastEnvValue(env []string, key string) string {
	prefix := key + "="
	value := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			value = strings.TrimPrefix(entry, prefix)
		}
	}
	return value
}
