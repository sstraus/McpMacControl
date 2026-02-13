// Package bridge implements the self-relaunch mechanism for macOS TCC permissions.
//
// When launched as a child process of a terminal (e.g., by Claude Code), macOS
// attributes accessibility/screen recording permissions to the parent terminal,
// not to MCPMacControl. Launching via `open -a` gives the .app its own TCC
// identity, but MCP needs stdio.
//
// The bridge acts as an MCP server on stdio, handling initialize and tools/list
// locally. On the first tools/call, it lazily launches the .app server and
// forwards all tool calls to it via an MCP client over a Unix socket.
package bridge

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/sstraus/mcpmaccontrol/internal/tools"
)

const (
	// SocketPath is the well-known Unix socket path shared by all bridges.
	// A single .app instance serves all bridges via this socket.
	SocketPath = "/tmp/mcpmaccontrol.sock"

	pollInterval = 100 * time.Millisecond
	pollTimeout  = 10 * time.Second
)

// Run is the bridge entry point. It creates an MCP server on stdio that
// handles initialize and tools/list locally. Tool calls are forwarded to
// the .app server, which is launched lazily on the first call.
//
// Returns an error if the .app bundle can't be found (indicating a bare
// binary with no bundle, e.g., `go run .`).
func Run() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	appPath, err := findAppBundle(exePath)
	if err != nil {
		return fmt.Errorf("find .app bundle: %w", err)
	}

	proxy := newLazyProxy(appPath, SocketPath)
	defer proxy.close()

	// Create bridge MCP server with proxy handlers for all tools.
	mcpServer := server.NewMCPServer(
		"mcpmaccontrol",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	for _, tool := range tools.Definitions() {
		mcpServer.AddTool(tool, proxy.handleToolCall)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Serve MCP on stdio. initialize and tools/list are handled by the
	// local MCP server; tool calls are forwarded to the .app via proxy.
	stdioServer := server.NewStdioServer(mcpServer)

	var stdout io.Writer = os.Stdout
	if os.Getenv("MCPMACCONTROL_DEBUG") == "1" {
		if f, err := os.OpenFile("/tmp/mcpmaccontrol-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			defer f.Close()
			log.SetOutput(f)
			log.Printf("=== bridge started (pid %d) ===", os.Getpid())
		}
		stdout = &debugWriter{w: os.Stdout}
	}

	if err := stdioServer.Listen(ctx, os.Stdin, stdout); err != nil {
		log.Printf("bridge session ended: %v", err)
	} else {
		log.Printf("bridge session ended normally (stdin closed)")
	}

	// Return nil: session end is not a startup failure.
	// main.go only falls back to direct stdio on non-nil errors
	// (indicating no .app bundle was found).
	return nil
}

// findAppBundle locates the .app bundle for the given binary. It first checks
// if the binary is inside a .app (e.g., Something.app/Contents/MacOS/binary),
// then falls back to looking for a sibling .app in the same directory (e.g.,
// build/mcpmaccontrol alongside build/MCPMacControl.app/).
func findAppBundle(binaryPath string) (string, error) {
	resolved, err := filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks: %w", err)
	}

	// Walk up looking for an enclosing .app directory.
	dir := filepath.Dir(resolved)
	for dir != "/" && dir != "." {
		if filepath.Ext(dir) == ".app" {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}

	// Not inside .app - look for a sibling .app in the same directory.
	entries, err := os.ReadDir(filepath.Dir(resolved))
	if err != nil {
		return "", fmt.Errorf("no .app bundle found near %q: %w", binaryPath, err)
	}
	for _, e := range entries {
		if e.IsDir() && filepath.Ext(e.Name()) == ".app" {
			return filepath.Join(filepath.Dir(resolved), e.Name()), nil
		}
	}

	return "", fmt.Errorf("no .app bundle found near %q", binaryPath)
}

// launchApp launches the .app server instance via `open -n -a`.
// The -n flag is needed because without it macOS won't launch a new instance.
// The server always listens on the well-known SocketPath.
func launchApp(appPath, _ string) error {
	cmd := exec.Command("open", "-n", "-a", appPath, "--args", "--server")
	return cmd.Start()
}

// debugWriter wraps an io.Writer and logs every write to stderr.
// Activate with MCPMACCONTROL_DEBUG=1 to capture the wire protocol.
type debugWriter struct {
	w io.Writer
}

func (d *debugWriter) Write(p []byte) (int, error) {
	log.Printf("[WIRE→] %s", p)
	return d.w.Write(p)
}

// waitForSocket polls until the Unix socket exists and is accepting connections.
func waitForSocket(path string) (net.Conn, error) {
	deadline := time.Now().Add(pollTimeout)

	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", path)
		if err == nil {
			return conn, nil
		}
		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("timeout waiting for socket %s after %v", path, pollTimeout)
}
