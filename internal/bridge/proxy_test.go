package bridge

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Suppress log output from mcp-go library during tests.
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// testBackend manages a real MCP server on a Unix socket for testing.
type testBackend struct {
	sockPath string
	listener net.Listener
	cancel   context.CancelFunc
}

// startTestBackend starts a real MCP server on a Unix socket with a simple
// echo tool. Returns the socket path. The backend accepts one connection.
func startTestBackend(t *testing.T) string {
	t.Helper()
	tb := startTestBackendMulti(t)
	return tb.sockPath
}

// newTestBackend creates a testBackend without starting it. Call tb.start(t)
// to begin accepting connections. Useful when the launcher callback should
// trigger the start.
func newTestBackend(t *testing.T) *testBackend {
	t.Helper()

	sockPath := filepath.Join("/tmp", "mcptest-"+t.Name()+".sock")
	os.Remove(sockPath)

	tb := &testBackend{sockPath: sockPath}
	t.Cleanup(func() {
		tb.stop()
		os.Remove(sockPath)
	})
	return tb
}

// startTestBackendMulti starts a real MCP server that accepts multiple
// sequential connections (one at a time). Returns a testBackend that can
// be restarted to simulate backend death and recovery.
func startTestBackendMulti(t *testing.T) *testBackend {
	t.Helper()

	tb := newTestBackend(t)
	tb.start(t)
	return tb
}

func (tb *testBackend) start(t *testing.T) {
	t.Helper()

	mcpServer := server.NewMCPServer("test-backend", "1.0.0",
		server.WithToolCapabilities(true),
	)
	mcpServer.AddTool(
		mcp.NewTool("echo", mcp.WithDescription("Echo test tool"),
			mcp.WithString("message", mcp.Required(), mcp.Description("Message to echo")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg := req.GetString("message", "")
			return mcp.NewToolResultText("echo: " + msg), nil
		},
	)

	listener, err := net.Listen("unix", tb.sockPath)
	require.NoError(t, err)
	tb.listener = listener

	stdioServer := server.NewStdioServer(mcpServer)
	ctx, cancel := context.WithCancel(context.Background())
	tb.cancel = cancel

	// Accept connections in a loop so the backend can serve multiple reconnects.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = stdioServer.Listen(ctx, conn, conn)
			conn.Close()
		}
	}()
}

func (tb *testBackend) stop() {
	if tb.cancel != nil {
		tb.cancel()
	}
	if tb.listener != nil {
		tb.listener.Close()
	}
}

// restart stops and restarts the backend, simulating a backend crash and recovery.
func (tb *testBackend) restart(t *testing.T) {
	t.Helper()
	tb.stop()
	os.Remove(tb.sockPath)
	tb.start(t)
}

func TestLazyProxy_NotLaunchedUntilToolCall(t *testing.T) {
	var launched atomic.Bool

	sockPath := startTestBackend(t)

	proxy := &lazyProxy{
		appPath:  "/fake/app.app",
		sockPath: sockPath,
		launcher: func(_, _ string) error {
			launched.Store(true)
			return nil
		},
	}
	defer proxy.close()

	// The backend should not be launched yet.
	assert.False(t, launched.Load(), "backend should not be launched before any tool call")
	assert.Nil(t, proxy.client, "client should be nil before any tool call")
}

func TestLazyProxy_LaunchOnFirstToolCall(t *testing.T) {
	var launchCount atomic.Int32

	// Prepare a backend but don't start it — the launcher will start it.
	tb := newTestBackend(t)

	proxy := &lazyProxy{
		appPath:  "/fake/app.app",
		sockPath: tb.sockPath,
		launcher: func(_, _ string) error {
			launchCount.Add(1)
			tb.start(t) // Start the backend when launcher is called
			return nil
		},
	}
	defer proxy.close()

	ctx := context.Background()

	// First tool call should trigger launch (no existing backend to reuse).
	result, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "hello"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int32(1), launchCount.Load(), "launcher should be called exactly once")

	// Verify the result from the backend.
	require.Len(t, result.Content, 1)
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected TextContent")
	assert.Equal(t, "echo: hello", textContent.Text)

	// Second tool call should reuse the existing connection, not re-launch.
	result2, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "world"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.Equal(t, int32(1), launchCount.Load(), "launcher should still be 1 after second call")

	textContent2, ok := result2.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "echo: world", textContent2.Text)
}

func TestLazyProxy_ConcurrentToolCalls(t *testing.T) {
	var launchCount atomic.Int32

	tb := newTestBackend(t)

	proxy := &lazyProxy{
		appPath:  "/fake/app.app",
		sockPath: tb.sockPath,
		launcher: func(_, _ string) error {
			launchCount.Add(1)
			tb.start(t)
			return nil
		},
	}
	defer proxy.close()

	ctx := context.Background()

	// Fire multiple tool calls concurrently.
	const numCalls = 10
	var wg sync.WaitGroup
	results := make([]*mcp.CallToolResult, numCalls)
	errs := make([]error, numCalls)

	for i := range numCalls {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = proxy.handleToolCall(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "echo",
					Arguments: map[string]any{"message": "concurrent"},
				},
			})
		}(i)
	}
	wg.Wait()

	// Launcher should only be called once despite concurrent calls.
	assert.Equal(t, int32(1), launchCount.Load(), "launcher should be called exactly once")

	// All calls should succeed.
	for i := range numCalls {
		assert.NoError(t, errs[i], "call %d should not error", i)
		require.NotNil(t, results[i], "call %d should have a result", i)
	}
}

func TestLazyProxy_LaunchFailure(t *testing.T) {
	proxy := &lazyProxy{
		appPath:  "/fake/app.app",
		sockPath: "/nonexistent/socket.sock",
		launcher: func(_, _ string) error {
			return assert.AnError
		},
	}
	defer proxy.close()

	ctx := context.Background()

	result, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "echo",
		},
	})

	// handleToolCall returns tool error, not Go error.
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "result should be an error")
	require.Len(t, result.Content, 1)
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Backend startup failed")
}

func TestLazyProxy_ReconnectAfterBackendDeath(t *testing.T) {
	var launchCount atomic.Int32

	tb := newTestBackend(t)

	proxy := &lazyProxy{
		appPath:  "/fake/app.app",
		sockPath: tb.sockPath,
		launcher: func(_, _ string) error {
			launchCount.Add(1)
			tb.start(t)
			return nil
		},
	}
	defer proxy.close()

	ctx := context.Background()

	// First call should launch and succeed.
	result, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "before-crash"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Equal(t, int32(1), launchCount.Load())

	// Kill the backend to simulate a crash.
	tb.restart(t)

	// Next call should detect the broken pipe, reconnect, and succeed.
	// The reconnect path tries the socket first (which now has a new backend
	// from restart), so the launcher may or may not be called depending on
	// whether the restarted backend is already accepting.
	result2, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "after-crash"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.False(t, result2.IsError, "retry after reconnect should succeed")

	// Verify the result is from the new backend.
	textContent, ok := result2.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "echo: after-crash", textContent.Text)
}

func TestLazyProxy_ReconnectFailureReturnsCleanError(t *testing.T) {
	tb := startTestBackendMulti(t)

	proxy := &lazyProxy{
		appPath:  "/fake/app.app",
		sockPath: tb.sockPath,
		launcher: func(_, _ string) error {
			return nil
		},
	}
	defer proxy.close()

	ctx := context.Background()

	// First call should succeed.
	result, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "hello"},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	// Kill the backend and DON'T restart - simulate permanent failure.
	tb.stop()
	os.Remove(tb.sockPath)

	// Make the launcher fail too so reconnect can't succeed.
	proxy.launcher = func(_, _ string) error {
		return fmt.Errorf("app bundle not found")
	}

	// Next call should return a clean MCP tool error, not a raw Go error.
	result2, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "will-fail"},
		},
	})
	require.NoError(t, err, "handleToolCall should not return Go errors")
	require.NotNil(t, result2)
	assert.True(t, result2.IsError, "result should be an error")

	textContent, ok := result2.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Backend")
	// Should NOT contain raw Go error internals like "broken pipe"
	assert.NotContains(t, textContent.Text, "transport error")
}

func TestLazyProxy_ReusesExistingBackend(t *testing.T) {
	var launchCount atomic.Int32

	sockPath := startTestBackend(t)

	proxy := &lazyProxy{
		appPath:  "/fake/app.app",
		sockPath: sockPath,
		launcher: func(_, _ string) error {
			launchCount.Add(1)
			return nil
		},
	}
	defer proxy.close()

	ctx := context.Background()

	// First tool call: backend already listening, so launcher should NOT be called.
	result, err := proxy.handleToolCall(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "reuse"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Equal(t, int32(0), launchCount.Load(), "launcher should NOT be called when backend is already running")

	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "echo: reuse", textContent.Text)
}

func TestLazyProxy_ToolCallTimeout(t *testing.T) {
	// Create a backend with a tool handler that never responds (simulates hung backend).
	sockPath := filepath.Join("/tmp", "mcptest-"+t.Name()+".sock")
	os.Remove(sockPath)
	t.Cleanup(func() { os.Remove(sockPath) })

	mcpServer := server.NewMCPServer("test-backend", "1.0.0",
		server.WithToolCapabilities(true),
	)
	mcpServer.AddTool(
		mcp.NewTool("echo", mcp.WithDescription("Echo test tool"),
			mcp.WithString("message", mcp.Required(), mcp.Description("Message to echo")),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			<-ctx.Done() // Block until context cancelled
			return nil, ctx.Err()
		},
	)

	listener, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	stdioServer := server.NewStdioServer(mcpServer)
	bgCtx, bgCancel := context.WithCancel(context.Background())
	t.Cleanup(bgCancel)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = stdioServer.Listen(bgCtx, conn, conn)
	}()

	proxy := &lazyProxy{
		appPath:     "/fake/app.app",
		sockPath:    sockPath,
		callTimeout: 1 * time.Second,
		launcher:    func(_, _ string) error { return nil },
	}
	defer proxy.close()

	start := time.Now()
	result, err := proxy.handleToolCall(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"message": "timeout-test"},
		},
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "timed-out call should return error result")
	assert.Less(t, elapsed, 5*time.Second, "should complete near the 1s timeout, not hang")

	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "not respond")
}

func TestLazyProxy_Close_DoesNotRemoveSocket(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "cleanup-test.sock")

	// Create a dummy file to represent the socket.
	require.NoError(t, os.WriteFile(sockPath, []byte("placeholder"), 0o600))

	proxy := &lazyProxy{
		sockPath: sockPath,
	}
	proxy.close()

	// Socket file should still exist — it's owned by the .app server, not the bridge.
	_, err := os.Stat(sockPath)
	assert.NoError(t, err, "socket file should NOT be removed by bridge close")
}
