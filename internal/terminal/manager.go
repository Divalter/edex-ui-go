package terminal

import (
	"errors"
	"os"
	"sync"
	"edex-ui-go/internal/config"
)

type Manager struct {
	cfg      *config.Config
	sessions map[string]*Session
	mu       sync.Mutex
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:      cfg,
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) CreateSession(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= 5 {
		return nil, errors.New("maximum number of sessions reached")
	}

	if _, exists := m.sessions[id]; exists {
		return nil, errors.New("session already exists")
	}

	env := os.Environ()
	for k, v := range m.cfg.Env {
		env = append(env, k+"="+v)
	}

	f, cmd, err := startPTY(m.cfg.Shell, m.cfg.ShellArgs, env, m.cfg.Cwd, 80, 24)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:  id,
		pty: f,
		cmd: cmd,
	}

	m.sessions[id] = session
	return session, nil
}

func (m *Manager) GetSession(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return nil
	}

	err := session.Close()
	delete(m.sessions, id)
	return err
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range m.sessions {
		session.Close()
		delete(m.sessions, id)
	}
}
