package pty

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

const defaultShell = "/bin/bash"
const defaultTerm = "TERM=xterm-256color"

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	ID string

	ptm       *os.File
	cmd       *exec.Cmd
	exitCh    chan int32
	mu        sync.Mutex
	closeOnce sync.Once
}

func (s *Session) closePTY() {
	s.closeOnce.Do(func() {
		if s.ptm != nil {
			_ = s.ptm.Close()
		}
	})
}

type Manager struct {
	sessions sync.Map
}

func (m *Manager) Create(shell string, cols, rows uint16, env []string) (*Session, error) {
	if shell == "" {
		shell = defaultShell
	}

	resolvedShell, err := exec.LookPath(shell)
	if err != nil {
		return nil, fmt.Errorf("shell %s not found in image", shell)
	}

	cmd := exec.Command(resolvedShell)
	cmd.Env = ptyEnv(env)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	ptm, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}

	session := &Session{
		ID:     uuid.NewString(),
		ptm:    ptm,
		cmd:    cmd,
		exitCh: make(chan int32, 1),
	}
	m.sessions.Store(session.ID, session)

	go func(id string, s *Session) {
		exitCode := int32(0)
		if waitErr := s.cmd.Wait(); waitErr != nil {
			var exitErr *exec.ExitError
			if errors.As(waitErr, &exitErr) {
				exitCode = int32(exitErr.ExitCode())
			} else {
				exitCode = -1
			}
		} else if s.cmd.ProcessState != nil {
			exitCode = int32(s.cmd.ProcessState.ExitCode())
		}

		s.exitCh <- exitCode
		close(s.exitCh)
		s.closePTY()
		m.sessions.Delete(id)
	}(session.ID, session)

	return session, nil
}

func (m *Manager) Write(id string, data []byte) error {
	session, err := m.session(id)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	_, err = session.ptm.Write(data)
	return err
}

func (m *Manager) Read(id string, buf []byte) (int, error) {
	session, err := m.session(id)
	if err != nil {
		return 0, err
	}

	return session.ptm.Read(buf)
}

func (m *Manager) Resize(id string, cols, rows uint16) error {
	session, err := m.session(id)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	return pty.Setsize(session.ptm, &pty.Winsize{Cols: cols, Rows: rows})
}

func (m *Manager) Signal(id string, signal syscall.Signal) error {
	session, err := m.session(id)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.cmd.Process == nil {
		return errors.New("session process not started")
	}

	return syscall.Kill(-session.cmd.Process.Pid, signal)
}

func (m *Manager) OnExit(id string) (<-chan int32, error) {
	session, err := m.session(id)
	if err != nil {
		return nil, err
	}

	return session.exitCh, nil
}

func (m *Manager) Kill(id string) {
	value, ok := m.sessions.LoadAndDelete(id)
	if !ok {
		return
	}

	session := value.(*Session)

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.cmd.Process != nil {
		_ = session.cmd.Process.Kill()
	}
	session.closePTY()
}

func (m *Manager) session(id string) (*Session, error) {
	value, ok := m.sessions.Load(id)
	if !ok {
		return nil, ErrSessionNotFound
	}

	session, ok := value.(*Session)
	if !ok {
		return nil, errors.New("invalid session type")
	}

	return session, nil
}

func ptyEnv(env []string) []string {
	base := append([]string(nil), os.Environ()...)
	if !hasEnvKey(base, "TERM") && !hasEnvKey(env, "TERM") {
		base = append(base, defaultTerm)
	}
	return append(base, env...)
}

func hasEnvKey(env []string, key string) bool {
	prefix := key + "="
	for _, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix && entry != prefix {
			return true
		}
	}
	return false
}
