package pty

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

const (
	defaultShell      = "/bin/bash"
	defaultTerm       = "TERM=xterm-256color"
	cpuSampleInterval = 5 * time.Second
	cpuSampleWindow   = 65 * time.Second
	maxCPUSampleCount = 16
)

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
	cpuSamples   []cpuSample
	lastActivity atomic.Int64
	sampleMu     sync.RWMutex
	samplerOnce  sync.Once
	sessions     sync.Map
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
	m.recordActivity(time.Now())

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
	if err == nil {
		m.recordActivity(time.Now())
	}
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

func (m *Manager) IdleStatus(now time.Time) (time.Time, float64, error) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()

	m.sampleMu.RLock()
	sampleCount := len(m.cpuSamples)
	m.sampleMu.RUnlock()
	if sampleCount == 0 {
		m.ensureCPUSampler()
	}

	var lastActivity time.Time
	if activityUnixNano := m.lastActivity.Load(); activityUnixNano > 0 {
		lastActivity = time.Unix(0, activityUnixNano).UTC()
	}

	cpuPercent, err := m.cpuPercent(now)
	if err != nil {
		return lastActivity, 0, err
	}

	return lastActivity, cpuPercent, nil
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

func (m *Manager) recordActivity(observedAt time.Time) {
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	m.lastActivity.Store(observedAt.UTC().UnixNano())
	m.ensureCPUSampler()
}

func (m *Manager) ensureCPUSampler() {
	m.samplerOnce.Do(func() {
		m.recordCPUSample()

		go func() {
			ticker := time.NewTicker(cpuSampleInterval)
			defer ticker.Stop()

			for range ticker.C {
				m.recordCPUSample()
			}
		}()
	})
}

func (m *Manager) recordCPUSample() {
	sample, err := readCPUSample(time.Now())
	if err != nil {
		return
	}

	m.sampleMu.Lock()
	defer m.sampleMu.Unlock()

	m.cpuSamples = append(m.cpuSamples, sample)
	cutoff := sample.at.Add(-cpuSampleWindow)
	firstValid := 0
	for firstValid < len(m.cpuSamples) && m.cpuSamples[firstValid].at.Before(cutoff) {
		firstValid++
	}
	if firstValid > 0 {
		m.cpuSamples = append([]cpuSample(nil), m.cpuSamples[firstValid:]...)
	}
	if len(m.cpuSamples) > maxCPUSampleCount {
		m.cpuSamples = append([]cpuSample(nil), m.cpuSamples[len(m.cpuSamples)-maxCPUSampleCount:]...)
	}
}

func (m *Manager) cpuPercent(now time.Time) (float64, error) {
	m.sampleMu.RLock()
	defer m.sampleMu.RUnlock()

	if len(m.cpuSamples) < 2 {
		return 0, nil
	}

	newest := m.cpuSamples[len(m.cpuSamples)-1]
	cutoff := now.Add(-60 * time.Second)
	oldest := m.cpuSamples[0]
	for _, sample := range m.cpuSamples {
		if sample.at.After(cutoff) {
			break
		}
		oldest = sample
	}

	if newest.total <= oldest.total || newest.idle < oldest.idle {
		return 0, nil
	}

	totalDelta := newest.total - oldest.total
	idleDelta := newest.idle - oldest.idle
	if totalDelta == 0 {
		return 0, nil
	}

	busyDelta := totalDelta - idleDelta
	return (float64(busyDelta) / float64(totalDelta)) * 100, nil
}

type cpuSample struct {
	at    time.Time
	idle  uint64
	total uint64
}

func readCPUSample(observedAt time.Time) (cpuSample, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuSample{}, fmt.Errorf("read /proc/stat: %w", err)
	}

	firstLine, _, _ := strings.Cut(string(data), "\n")
	return parseCPUSample(firstLine, observedAt)
}

func parseCPUSample(line string, observedAt time.Time) (cpuSample, error) {
	fields := strings.Fields(line)
	if len(fields) < 6 || fields[0] != "cpu" {
		return cpuSample{}, fmt.Errorf("unexpected cpu stat line: %q", line)
	}

	var (
		idleTicks  uint64
		totalTicks uint64
	)

	for index, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuSample{}, fmt.Errorf("parse cpu stat field %q: %w", field, err)
		}
		totalTicks += value
		if index == 3 || index == 4 {
			idleTicks += value
		}
	}

	return cpuSample{
		at:    observedAt.UTC(),
		idle:  idleTicks,
		total: totalTicks,
	}, nil
}
