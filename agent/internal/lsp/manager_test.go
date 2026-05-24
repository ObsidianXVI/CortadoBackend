package lsp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadFrameTrimsCRLF(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader("Content-Length: 17\r\n\r\n{\"jsonrpc\":\"2.0\"}"))
	body, err := readFrame(reader)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if got, want := string(body), "{\"jsonrpc\":\"2.0\"}"; got != want {
		t.Fatalf("unexpected frame body: got %q want %q", got, want)
	}
}

func TestManagerRestartsServerAfterCrash(t *testing.T) {
	t.Parallel()

	stateFile := filepathJoin(t.TempDir(), "lsp-helper-state")
	manager := NewManagerWithConfig(ManagerConfig{
		CommandFactory: helperCommandFactory(t, "crash-once", stateFile),
		MaxRestarts:    3,
		WorkspaceRoot:  t.TempDir(),
	})

	server, err := manager.GetOrStart("dart")
	if err != nil {
		t.Fatalf("get or start lsp server: %v", err)
	}

	events, release, err := server.Attach()
	if err != nil {
		t.Fatalf("attach stream: %v", err)
	}
	defer release()

	if err := server.Write([]byte("first")); err != nil {
		t.Fatalf("write first message: %v", err)
	}
	if event := waitForEvent(t, events); string(event.Data) != "run-1:first" {
		t.Fatalf("unexpected first event: %q", event.Data)
	}

	waitForHelperRun(t, stateFile, 2)

	if err := server.Write([]byte("second")); err != nil {
		t.Fatalf("write second message after restart: %v", err)
	}
	if event := waitForEvent(t, events); string(event.Data) != "run-2:second" {
		t.Fatalf("unexpected second event: %q", event.Data)
	}
}

func TestManagerClosesAfterMaxRestarts(t *testing.T) {
	t.Parallel()

	manager := NewManagerWithConfig(ManagerConfig{
		CommandFactory: helperCommandFactory(t, "crash-always", ""),
		MaxRestarts:    1,
		WorkspaceRoot:  t.TempDir(),
	})

	server, err := manager.GetOrStart("dart")
	if err != nil {
		t.Fatalf("get or start lsp server: %v", err)
	}

	events, release, err := server.Attach()
	if err != nil {
		t.Fatalf("attach stream: %v", err)
	}
	defer release()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatalf("expected events channel to close after restart exhaustion")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for lsp server to close")
	}
}

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mode := os.Getenv("LSP_HELPER_MODE")
	stateFile := os.Getenv("LSP_HELPER_STATE_FILE")

	switch mode {
	case "crash-always":
		os.Exit(1)
	case "crash-once":
		runCrashOnceHelper(stateFile)
	case "echo":
		runEchoHelper()
	default:
		os.Exit(2)
	}
}

func helperCommandFactory(t *testing.T, mode, stateFile string) CommandFactory {
	t.Helper()

	return func(language, workspaceRoot string) (CommandConfig, error) {
		return CommandConfig{
			Args: []string{"-test.run=TestHelperProcess"},
			Dir:  workspaceRoot,
			Env: append(os.Environ(),
				"GO_WANT_HELPER_PROCESS=1",
				"LSP_HELPER_MODE="+mode,
				"LSP_HELPER_STATE_FILE="+stateFile,
			),
			Path: os.Args[0],
		}, nil
	}
}

func runCrashOnceHelper(stateFile string) {
	run := incrementHelperRun(stateFile)
	reader := bufio.NewReader(os.Stdin)

	body, err := readFrame(reader)
	if err != nil {
		os.Exit(3)
	}
	if err := writeFrame(os.Stdout, []byte(fmt.Sprintf("run-%d:%s", run, body))); err != nil {
		os.Exit(4)
	}
	if run == 1 {
		os.Exit(1)
	}

	for {
		body, err := readFrame(reader)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				os.Exit(0)
			}
			os.Exit(5)
		}
		if err := writeFrame(os.Stdout, []byte(fmt.Sprintf("run-%d:%s", run, body))); err != nil {
			os.Exit(6)
		}
	}
}

func runEchoHelper() {
	reader := bufio.NewReader(os.Stdin)
	for {
		body, err := readFrame(reader)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				os.Exit(0)
			}
			os.Exit(7)
		}
		if err := writeFrame(os.Stdout, bytes.TrimSpace(append([]byte("echo:"), body...))); err != nil {
			os.Exit(8)
		}
	}
}

func incrementHelperRun(path string) int {
	value := 0
	if data, err := os.ReadFile(path); err == nil {
		value, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}
	value++
	_ = os.WriteFile(path, []byte(strconv.Itoa(value)), 0o644)
	return value
}

func waitForEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("lsp events channel closed unexpectedly")
		}
		if event.Err != nil {
			t.Fatalf("unexpected lsp event error: %v", event.Err)
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for lsp event")
	}
	return Event{}
}

func waitForHelperRun(t *testing.T, path string, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			got, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr == nil && got >= want {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for helper run %d", want)
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}
