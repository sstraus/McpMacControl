// Package bridge implements the self-relaunch mechanism for macOS TCC permissions.
//
// The bridge acts as an MCP server on stdio, handling initialize and tools/list
// locally. On the first tools/call, it lazily launches the .app server and
// forwards all tool calls to it via an MCP client over a Unix socket.
package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const defaultCallTimeout = 30 * time.Second

// lazyProxy manages the lifecycle of the .app backend and forwards tool calls.
// The backend is launched on the first tool call and reused for subsequent calls.
// If the backend dies, the proxy detects the transport error and reconnects.
type lazyProxy struct {
	appPath     string
	sockPath    string
	callTimeout time.Duration                       // timeout per tool call; 0 means defaultCallTimeout
	launcher    func(appPath, sockPath string) error // injected for testing

	mu        sync.Mutex
	connected bool
	client    *client.Client
	conn      net.Conn
}

// newLazyProxy creates a proxy that will launch the given .app on first use.
func newLazyProxy(appPath, sockPath string) *lazyProxy {
	return &lazyProxy{
		appPath:  appPath,
		sockPath: sockPath,
		launcher: launchApp,
	}
}

// getCallTimeout returns the per-call timeout, defaulting to 30s.
func (p *lazyProxy) getCallTimeout() time.Duration {
	if p.callTimeout > 0 {
		return p.callTimeout
	}
	return defaultCallTimeout
}

// callTool wraps client.CallTool with a timeout to prevent indefinite hangs.
func (p *lazyProxy) callTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, p.getCallTimeout())
	defer cancel()

	result, err := p.client.CallTool(callCtx, req)
	if err != nil && callCtx.Err() == context.DeadlineExceeded {
		return nil, context.DeadlineExceeded
	}
	return result, err
}

// handleToolCall forwards a tool call to the .app backend, launching it if needed.
// If the backend connection is dead (transport error), it resets and retries once.
// Each call has a timeout (default 30s) to prevent indefinite hangs.
func (p *lazyProxy) handleToolCall(ctx context.Context, req mcp.CallToolRequest) (result *mcp.CallToolResult, _ error) {
	if os.Getenv("MCPMACCONTROL_DEBUG") == "1" {
		defer func() {
			if result != nil && result.IsError {
				j, _ := json.Marshal(result)
				log.Printf("[DEBUG] tool=%s returning error result: %s", req.Params.Name, j)
			}
		}()
	}

	if err := p.ensureBackend(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Backend startup failed: %v", err)), nil
	}

	result, err := p.callTool(ctx, req)
	if err == nil {
		return result, nil
	}

	// Timeout: backend is unresponsive. Reset connection so the next call can retry fresh.
	if errors.Is(err, context.DeadlineExceeded) {
		p.resetConnection()
		return mcp.NewToolResultError(fmt.Sprintf("Backend did not respond within %s. The server may be unresponsive — try again.", p.getCallTimeout())), nil
	}

	// Check if the error is a transport error (e.g., broken pipe from dead backend).
	var transportErr *transport.Error
	if !errors.As(err, &transportErr) {
		return mcp.NewToolResultError(fmt.Sprintf("Backend call failed: %v", err)), nil
	}

	// Backend died - reset and retry once.
	log.Printf("Backend connection lost, reconnecting: %v", err)
	p.resetConnection()

	if err := p.ensureBackend(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Backend reconnection failed: %v", err)), nil
	}

	result, err = p.callTool(ctx, req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			p.resetConnection()
			return mcp.NewToolResultError(fmt.Sprintf("Backend did not respond within %s after reconnect.", p.getCallTimeout())), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Backend call failed after reconnect: %v", err)), nil
	}
	return result, nil
}

// ensureBackend launches the .app and connects the MCP client if not already connected.
func (p *lazyProxy) ensureBackend(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connected {
		return nil
	}

	if err := p.launchBackend(ctx); err != nil {
		return err
	}
	p.connected = true
	return nil
}

// resetConnection tears down the current backend connection so the next
// ensureBackend call will re-launch. Must not be called under p.mu.
func (p *lazyProxy) resetConnection() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		_ = p.client.Close()
		p.client = nil
	}
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
	p.connected = false
}

// launchBackend connects to an existing .app or launches a new one.
// It first tries to connect to the well-known socket. If that succeeds,
// the existing .app is reused. Otherwise, a new .app is launched.
func (p *lazyProxy) launchBackend(ctx context.Context) error {
	// Try connecting to an already-running .app instance.
	conn, err := net.DialTimeout("unix", p.sockPath, 500*time.Millisecond)
	if err != nil {
		// No existing instance — launch a new one.
		log.Printf("Launching backend: %s", p.appPath)
		if err := p.launcher(p.appPath, p.sockPath); err != nil {
			return fmt.Errorf("launch .app: %w", err)
		}

		conn, err = waitForSocket(p.sockPath)
		if err != nil {
			return fmt.Errorf("connect to server: %w", err)
		}
	} else {
		log.Printf("Reusing existing backend at %s", p.sockPath)
	}
	p.conn = conn

	// Create MCP client over the socket connection.
	t := transport.NewIO(conn, writeCloser{conn}, io.NopCloser(strings.NewReader("")))
	p.client = client.NewClient(t)

	if err := p.client.Start(ctx); err != nil {
		return fmt.Errorf("start MCP client: %w", err)
	}

	// Initialize the MCP session with the .app server.
	_, err = p.client.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcpmaccontrol-bridge",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize backend: %w", err)
	}

	log.Printf("Backend connected via %s", p.sockPath)
	return nil
}

// close shuts down the bridge's connection to the backend.
// The socket file is owned by the .app server, not the bridge.
func (p *lazyProxy) close() {
	p.resetConnection()
}

// writeCloser wraps a net.Conn to satisfy io.WriteCloser.
// We don't want Close() to close the underlying conn (the proxy manages that).
type writeCloser struct {
	w io.Writer
}

func (wc writeCloser) Write(b []byte) (int, error) { return wc.w.Write(b) }
func (wc writeCloser) Close() error                 { return nil }
