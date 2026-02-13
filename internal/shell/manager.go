package shell

import (
	"fmt"
	"sync"
)

// Manager manages multiple PTY sessions.
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a new session manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// Spawn creates a new shell session.
// If command is empty, defaults to /bin/bash.
// cols and rows default to 80x24 if not specified or invalid.
func (m *Manager) Spawn(command string, args []string, cwd string, cols, rows int) (*Session, error) {
	if command == "" {
		command = "/bin/bash"
	}
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	session, err := NewSession(command, args, cwd, cols, rows)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	return session, nil
}

// Get retrieves a session by ID.
func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return session, nil
}

// Close closes a session and removes it from the manager.
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", id)
	}
	delete(m.sessions, id)
	m.mu.Unlock()

	return session.Close()
}

// CloseAll closes all sessions.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, s := range sessions {
		s.Close()
	}
}

// List returns information about all sessions.
func (m *Manager) List() []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]map[string]any, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s.Info())
	}
	return result
}

// Count returns the number of active sessions.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
