package lsp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const defaultMaxRestarts = 3

var (
	ErrMissingContentLength = errors.New("missing Content-Length header")
	ErrServerClosed         = errors.New("lsp server closed")
	ErrStreamAttached       = errors.New("lsp stream already attached")
)

type Event struct {
	Data []byte
	Err  error
}

type CommandConfig struct {
	Args []string
	Dir  string
	Env  []string
	Path string
}

type CommandFactory func(language, workspaceRoot string) (CommandConfig, error)

type ManagerConfig struct {
	CommandFactory CommandFactory
	MaxRestarts    int
	WorkspaceRoot  string
}

type Manager struct {
	commandFactory CommandFactory
	maxRestarts    int
	workspaceRoot  string

	mu      sync.Mutex
	servers map[string]*Server
}

type Server struct {
	commandFactory CommandFactory
	language       string
	maxRestarts    int
	onClosed       func(*Server)
	workspaceRoot  string

	mu       sync.Mutex
	attached bool
	closeErr error
	closed   bool
	cmd      *exec.Cmd
	events   chan Event
	restarts int
	stdin    io.WriteCloser
	stderr   *bytes.Buffer
}

func NewManager() *Manager {
	return NewManagerWithConfig(ManagerConfig{})
}

func NewManagerWithConfig(cfg ManagerConfig) *Manager {
	if strings.TrimSpace(cfg.WorkspaceRoot) == "" {
		cfg.WorkspaceRoot = "/workspace"
	}
	if cfg.CommandFactory == nil {
		cfg.CommandFactory = defaultCommandFactory
	}
	if cfg.MaxRestarts <= 0 {
		cfg.MaxRestarts = defaultMaxRestarts
	}

	return &Manager{
		commandFactory: cfg.CommandFactory,
		maxRestarts:    cfg.MaxRestarts,
		workspaceRoot:  cfg.WorkspaceRoot,
		servers:        make(map[string]*Server),
	}
}

func (m *Manager) GetOrStart(language string) (*Server, error) {
	language = normalizeLanguage(language)
	if language == "" {
		return nil, fmt.Errorf("language is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if server, ok := m.servers[language]; ok {
		server.mu.Lock()
		closed := server.closed
		server.mu.Unlock()
		if !closed {
			return server, nil
		}
		delete(m.servers, language)
	}

	server := &Server{
		commandFactory: m.commandFactory,
		events:         make(chan Event, 128),
		language:       language,
		maxRestarts:    m.maxRestarts,
		onClosed: func(closed *Server) {
			m.mu.Lock()
			defer m.mu.Unlock()
			if current, ok := m.servers[language]; ok && current == closed {
				delete(m.servers, language)
			}
		},
		workspaceRoot: m.workspaceRoot,
	}
	if err := server.startLocked(); err != nil {
		return nil, err
	}

	m.servers[language] = server
	return server, nil
}

func (m *Manager) Languages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	languages := make([]string, 0, len(m.servers))
	for language := range m.servers {
		languages = append(languages, language)
	}
	return languages
}

func (s *Server) Attach() (<-chan Event, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.attached {
		return nil, nil, ErrStreamAttached
	}
	if s.closed {
		return nil, nil, s.terminalErrorLocked()
	}

	s.attached = true
	return s.events, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.attached = false
	}, nil
}

func (s *Server) Write(data []byte) error {
	s.mu.Lock()
	stdin := s.stdin
	closed := s.closed
	err := s.terminalErrorLocked()
	s.mu.Unlock()

	if closed || stdin == nil {
		return err
	}
	return writeFrame(stdin, data)
}

func (s *Server) startLocked() error {
	command, err := s.commandFactory(s.language, s.workspaceRoot)
	if err != nil {
		return err
	}

	cmd := exec.Command(command.Path, command.Args...)
	if len(command.Env) > 0 {
		cmd.Env = append([]string(nil), command.Env...)
	}
	if strings.TrimSpace(command.Dir) != "" {
		cmd.Dir = command.Dir
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open lsp stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open lsp stdout: %w", err)
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s lsp server: %w", s.language, err)
	}

	s.cmd = cmd
	s.stdin = stdin
	s.stderr = stderr

	go s.readLoop(cmd, stdout)
	go s.waitLoop(cmd)
	return nil
}

func (s *Server) readLoop(cmd *exec.Cmd, stdout io.Reader) {
	reader := bufio.NewReader(stdout)

	for {
		body, err := readFrame(reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			return
		}

		select {
		case s.events <- Event{Data: body}:
		default:
			s.events <- Event{Data: body}
		}
	}
}

func (s *Server) waitLoop(cmd *exec.Cmd) {
	waitErr := cmd.Wait()

	s.mu.Lock()
	if s.cmd != cmd || s.closed {
		s.mu.Unlock()
		return
	}

	s.cmd = nil
	s.stdin = nil

	if s.restarts < s.maxRestarts {
		s.restarts++
		if err := s.startLocked(); err == nil {
			s.mu.Unlock()
			return
		} else {
			waitErr = fmt.Errorf("restart %s lsp server: %w", s.language, err)
		}
	}

	closeErr := formatExitError(s.language, waitErr, s.stderr)
	s.closed = true
	s.closeErr = closeErr
	close(s.events)
	onClosed := s.onClosed
	s.mu.Unlock()

	if onClosed != nil {
		onClosed(s)
	}
}

func (s *Server) terminalErrorLocked() error {
	if s.closeErr != nil {
		return s.closeErr
	}
	return ErrServerClosed
}

func defaultCommandFactory(language, workspaceRoot string) (CommandConfig, error) {
	language = normalizeLanguage(language)
	if language != "dart" {
		return CommandConfig{}, fmt.Errorf("unsupported language %q", language)
	}

	dartPath, err := resolveDartPath()
	if err != nil {
		return CommandConfig{}, err
	}

	return CommandConfig{
		Args: []string{"language-server", "--protocol=lsp"},
		Dir:  workspaceRoot,
		Env:  os.Environ(),
		Path: dartPath,
	}, nil
}

func resolveDartPath() (string, error) {
	if sdkPath := strings.TrimSpace(os.Getenv("DART_SDK_PATH")); sdkPath != "" {
		dartPath := filepath.Join(sdkPath, "bin", "dart")
		if _, err := os.Stat(dartPath); err != nil {
			return "", fmt.Errorf("stat dart binary: %w", err)
		}
		return dartPath, nil
	}

	dartPath, err := exec.LookPath("dart")
	if err != nil {
		return "", fmt.Errorf("find dart executable: %w", err)
	}
	return dartPath, nil
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			continue
		}

		contentLength, err = strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("parse content length: %w", err)
		}
	}

	if contentLength < 0 {
		return nil, ErrMissingContentLength
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeFrame(writer io.Writer, data []byte) error {
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err := writer.Write(data)
	return err
}

func normalizeLanguage(language string) string {
	return strings.ToLower(strings.TrimSpace(language))
}

func formatExitError(language string, err error, stderr *bytes.Buffer) error {
	if err == nil && stderr != nil && strings.TrimSpace(stderr.String()) == "" {
		return fmt.Errorf("%s lsp server exited", language)
	}
	if stderr != nil && strings.TrimSpace(stderr.String()) != "" {
		if err != nil {
			return fmt.Errorf("%s lsp server exited: %v: %s", language, err, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("%s lsp server exited: %s", language, strings.TrimSpace(stderr.String()))
	}
	if err != nil {
		return fmt.Errorf("%s lsp server exited: %w", language, err)
	}
	return fmt.Errorf("%s lsp server exited", language)
}
