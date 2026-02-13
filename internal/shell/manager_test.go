package shell

import (
	"testing"
)

func TestManagerSpawn(t *testing.T) {
	m := NewManager()
	defer m.CloseAll()

	session, err := m.Spawn("", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	if session.Command != "/bin/bash" {
		t.Errorf("expected default command /bin/bash, got %s", session.Command)
	}

	if m.Count() != 1 {
		t.Errorf("expected 1 session, got %d", m.Count())
	}
}

func TestManagerSpawnWithArgs(t *testing.T) {
	m := NewManager()
	defer m.CloseAll()

	session, err := m.Spawn("/bin/bash", []string{"-l"}, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	if session.Command != "/bin/bash" {
		t.Errorf("expected command /bin/bash, got %s", session.Command)
	}

	if len(session.Args) != 1 || session.Args[0] != "-l" {
		t.Errorf("expected args [-l], got %v", session.Args)
	}

	if m.Count() != 1 {
		t.Errorf("expected 1 session, got %d", m.Count())
	}
}

func TestManagerSpawnWithCwd(t *testing.T) {
	m := NewManager()
	defer m.CloseAll()

	session, err := m.Spawn("/bin/bash", nil, "/tmp", 80, 24)
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	if session.Cwd != "/tmp" {
		t.Errorf("expected cwd /tmp, got %s", session.Cwd)
	}
}

func TestManagerSpawnWithCustomSize(t *testing.T) {
	m := NewManager()
	defer m.CloseAll()

	session, err := m.Spawn("", nil, "", 120, 40)
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	// Can't directly verify terminal size, just check session was created
	if session == nil {
		t.Error("expected non-nil session")
	}
}

func TestManagerGet(t *testing.T) {
	m := NewManager()
	defer m.CloseAll()

	session, err := m.Spawn("", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	got, err := m.Get(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if got.ID != session.ID {
		t.Errorf("expected session %s, got %s", session.ID, got.ID)
	}
}

func TestManagerGetNotFound(t *testing.T) {
	m := NewManager()

	_, err := m.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestManagerClose(t *testing.T) {
	m := NewManager()

	session, err := m.Spawn("", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	err = m.Close(session.ID)
	if err != nil {
		t.Errorf("failed to close: %v", err)
	}

	if m.Count() != 0 {
		t.Errorf("expected 0 sessions after close, got %d", m.Count())
	}

	// Get should fail after close
	_, err = m.Get(session.ID)
	if err == nil {
		t.Error("expected error getting closed session")
	}
}

func TestManagerCloseAll(t *testing.T) {
	m := NewManager()

	// Spawn multiple sessions
	for i := 0; i < 3; i++ {
		_, err := m.Spawn("", nil, "", 80, 24)
		if err != nil {
			t.Fatalf("failed to spawn session %d: %v", i, err)
		}
	}

	if m.Count() != 3 {
		t.Errorf("expected 3 sessions, got %d", m.Count())
	}

	m.CloseAll()

	if m.Count() != 0 {
		t.Errorf("expected 0 sessions after CloseAll, got %d", m.Count())
	}
}

func TestManagerList(t *testing.T) {
	m := NewManager()
	defer m.CloseAll()

	// Empty list
	list := m.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	// Spawn and list
	_, err := m.Spawn("", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to spawn: %v", err)
	}

	list = m.List()
	if len(list) != 1 {
		t.Errorf("expected 1 item, got %d", len(list))
	}
}
