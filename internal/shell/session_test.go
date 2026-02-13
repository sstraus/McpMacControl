package shell

import (
	"strings"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	if session.Command != "/bin/bash" {
		t.Errorf("expected command '/bin/bash', got '%s'", session.Command)
	}
	if session.IsClosed() {
		t.Error("session should not be closed")
	}
}

func TestNewSessionWithArgs(t *testing.T) {
	session, err := NewSession("/bin/bash", []string{"-c", "echo test"}, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	if session.Command != "/bin/bash" {
		t.Errorf("expected command '/bin/bash', got '%s'", session.Command)
	}
	if len(session.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(session.Args))
	}
}

func TestNewSessionWithCwd(t *testing.T) {
	session, err := NewSession("/bin/bash", []string{"-c", "pwd"}, "/tmp", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	if session.Cwd != "/tmp" {
		t.Errorf("expected cwd '/tmp', got '%s'", session.Cwd)
	}

	// Wait a bit for command to execute
	time.Sleep(100 * time.Millisecond)

	// Get snapshot and verify pwd output contains /tmp
	snapshot := session.Snapshot("text")
	if !strings.Contains(snapshot, "/tmp") {
		t.Errorf("expected snapshot to contain '/tmp', got: %s", snapshot)
	}
}

func TestNewSessionWithCustomSize(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 120, 40)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Terminal size is set, but we can't easily verify it from outside
	// Just check that session was created successfully
	if session.IsClosed() {
		t.Error("session should not be closed")
	}
}

func TestSessionSendInput(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Send a simple command
	err = session.SendInput("echo hello\n")
	if err != nil {
		t.Fatalf("failed to send input: %v", err)
	}

	// Wait a bit for output
	time.Sleep(200 * time.Millisecond)

	// Get snapshot
	snapshot := session.Snapshot("text")
	if !strings.Contains(snapshot, "hello") {
		t.Errorf("expected snapshot to contain 'hello', got: %s", snapshot)
	}
}

func TestSessionSendSpecialKey(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Send text without newline
	err = session.SendInput("echo test")
	if err != nil {
		t.Fatalf("failed to send input: %v", err)
	}

	// Send enter key
	err = session.SendSpecialKey("enter")
	if err != nil {
		t.Fatalf("failed to send special key: %v", err)
	}

	// Wait for output
	time.Sleep(200 * time.Millisecond)

	// Get snapshot
	snapshot := session.Snapshot("text")
	if !strings.Contains(snapshot, "test") {
		t.Errorf("expected snapshot to contain 'test', got: %s", snapshot)
	}
}

func TestSessionSendSpecialKeyCtrlC(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Send ctrl+c - should not fail
	err = session.SendSpecialKey("ctrl+c")
	if err != nil {
		t.Fatalf("failed to send ctrl+c: %v", err)
	}
}

func TestSessionSnapshotText(t *testing.T) {
	session, err := NewSession("/bin/bash", []string{"-c", "echo 'line1'; echo 'line2'"}, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Wait for command to complete
	time.Sleep(200 * time.Millisecond)

	// Get text snapshot
	snapshot := session.Snapshot("text")
	if !strings.Contains(snapshot, "line1") || !strings.Contains(snapshot, "line2") {
		t.Errorf("expected snapshot to contain both lines, got: %s", snapshot)
	}
}

func TestSessionSnapshotAnsi(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Send a command with color output
	err = session.SendInput("echo 'test'\n")
	if err != nil {
		t.Fatalf("failed to send input: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Get ANSI snapshot
	snapshot := session.Snapshot("ansi")
	// ANSI snapshot should be non-empty
	if snapshot == "" {
		t.Error("expected non-empty ANSI snapshot")
	}
}

func TestSessionPID(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	pid := session.PID()
	if pid <= 0 {
		t.Errorf("expected positive PID, got %d", pid)
	}
}

func TestSessionExitCode(t *testing.T) {
	// Run a command that exits immediately with code 0
	session, err := NewSession("/bin/bash", []string{"-c", "exit 0"}, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Wait for process to exit
	time.Sleep(200 * time.Millisecond)

	exitCode := session.ExitCode()
	if exitCode == nil {
		t.Error("expected exit code to be set")
	} else if *exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", *exitCode)
	}
}

func TestSessionExitCodeNonZero(t *testing.T) {
	// Run a command that exits with non-zero code
	session, err := NewSession("/bin/bash", []string{"-c", "exit 42"}, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	// Wait for process to exit
	time.Sleep(200 * time.Millisecond)

	exitCode := session.ExitCode()
	if exitCode == nil {
		t.Error("expected exit code to be set")
	} else if *exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", *exitCode)
	}
}

func TestSessionResize(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	err = session.Resize(120, 40)
	if err != nil {
		t.Errorf("failed to resize: %v", err)
	}
}

func TestSessionClose(t *testing.T) {
	session, err := NewSession("/bin/bash", nil, "", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	err = session.Close()
	if err != nil {
		t.Errorf("failed to close: %v", err)
	}

	if !session.IsClosed() {
		t.Error("session should be closed")
	}

	// Sending input to closed session should fail
	err = session.SendInput("test\n")
	if err == nil {
		t.Error("expected error sending input to closed session")
	}
}

func TestSessionInfo(t *testing.T) {
	session, err := NewSession("/bin/bash", []string{"-l"}, "/tmp", 100, 30)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	info := session.Info()
	if info["command"] != "/bin/bash" {
		t.Errorf("expected command '/bin/bash', got %v", info["command"])
	}
	if info["closed"] != false {
		t.Error("expected closed to be false")
	}
}
