package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

// Session represents a PTY shell session with terminal emulation.
type Session struct {
	ID      string
	Command string
	Args    []string
	Cwd     string
	PTY     *os.File
	Cmd     *exec.Cmd
	cancel  context.CancelFunc
	term    *vt.SafeEmulator
	mu      sync.Mutex
	closed  bool
	created time.Time
}

// specialKeyMap maps pty-mcp key names to terminal escape sequences
var specialKeyMap = map[string]string{
	"enter":     "\r",
	"tab":       "\t",
	"escape":    "\x1b",
	"backspace": "\x7f",
	"delete":    "\x1b[3~",
	"up":        "\x1b[A",
	"down":      "\x1b[B",
	"left":      "\x1b[D",
	"right":     "\x1b[C",
	"home":      "\x1b[H",
	"end":       "\x1b[F",
	"pageup":    "\x1b[5~",
	"pagedown":  "\x1b[6~",
	"f1":        "\x1bOP",
	"f2":        "\x1bOQ",
	"f3":        "\x1bOR",
	"f4":        "\x1bOS",
	"f5":        "\x1b[15~",
	"f6":        "\x1b[17~",
	"f7":        "\x1b[18~",
	"f8":        "\x1b[19~",
	"f9":        "\x1b[20~",
	"f10":       "\x1b[21~",
	"f11":       "\x1b[23~",
	"f12":       "\x1b[24~",
	"ctrl+c":    "\x03",
	"ctrl+d":    "\x04",
	"ctrl+z":    "\x1a",
	"ctrl+l":    "\x0c",
	"shift+tab": "\x1b[Z",
}

// NewSession creates a new PTY session with terminal emulation.
// command: the command to execute (e.g., "/bin/bash")
// args: command arguments (can be nil)
// cwd: working directory (empty string for current dir)
// cols, rows: terminal dimensions
func NewSession(command string, args []string, cwd string, cols, rows int) (*Session, error) {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()
	if cwd != "" {
		cmd.Dir = cwd
	}

	// Create terminal emulator
	term := vt.NewSafeEmulator(cols, rows)

	// Start the command with a PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Set PTY window size
	err = pty.Setsize(ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		cancel()
		ptmx.Close()
		return nil, fmt.Errorf("failed to set PTY size: %w", err)
	}

	s := &Session{
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Command: command,
		Args:    args,
		Cwd:     cwd,
		PTY:     ptmx,
		Cmd:     cmd,
		cancel:  cancel,
		term:    term,
		created: time.Now(),
	}

	// Start reading output in background
	go s.readLoop()

	return s, nil
}

// readLoop continuously reads from the PTY and feeds it to the terminal emulator
func (s *Session) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.PTY.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.term.Write(buf[:n])
			s.mu.Unlock()
		}
		if err != nil {
			if err != io.EOF {
				// Log error but don't crash - session may be closing
			}
			// Wait for process to exit and populate ProcessState
			s.Cmd.Wait()
			return
		}
	}
}

// SendInput sends text input to the PTY.
func (s *Session) SendInput(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	_, err := s.PTY.WriteString(text)
	return err
}

// SendSpecialKey sends a special key to the PTY.
// Supported keys: enter, tab, escape, backspace, delete, up, down, left, right,
// home, end, pageup, pagedown, f1-f12, ctrl+c, ctrl+d, ctrl+z, ctrl+l, shift+tab
func (s *Session) SendSpecialKey(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	seq, ok := specialKeyMap[key]
	if !ok {
		return fmt.Errorf("unknown special key: %s", key)
	}

	_, err := s.PTY.WriteString(seq)
	return err
}

// Snapshot returns the current terminal screen state.
// format can be "text" (plain text) or "ansi" (with ANSI escape codes)
func (s *Session) Snapshot(format string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch format {
	case "ansi":
		return s.term.Render()
	case "text", "":
		return s.term.String()
	default:
		return s.term.String()
	}
}

// PID returns the process ID of the running command.
func (s *Session) PID() int {
	if s.Cmd == nil || s.Cmd.Process == nil {
		return 0
	}
	return s.Cmd.Process.Pid
}

// ExitCode returns the exit code if the process has exited, nil otherwise.
// This is a non-blocking check - it won't wait for the process to exit.
func (s *Session) ExitCode() *int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Cmd == nil || s.Cmd.Process == nil {
		return nil
	}

	// Check if process has exited without blocking
	// ProcessState is only set after Wait() returns
	if s.Cmd.ProcessState != nil {
		code := s.Cmd.ProcessState.ExitCode()
		return &code
	}

	return nil
}

// Resize changes the PTY window size.
func (s *Session) Resize(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	// Resize terminal emulator
	s.term.Resize(cols, rows)

	// Resize PTY
	return pty.Setsize(s.PTY, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// Close terminates the session.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.cancel()
	s.PTY.Close()

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- s.Cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		// Force kill if it doesn't exit
		if s.Cmd.Process != nil {
			s.Cmd.Process.Kill()
		}
	}

	return nil
}

// IsClosed returns whether the session is closed.
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Info returns session information.
func (s *Session) Info() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"id":      s.ID,
		"command": s.Command,
		"args":    s.Args,
		"cwd":     s.Cwd,
		"created": s.created.Format(time.RFC3339),
		"closed":  s.closed,
		"pid":     s.PID(),
	}
}
