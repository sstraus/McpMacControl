package protocol

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_StartStop(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	require.NotNil(t, srv)

	err := srv.Start()
	require.NoError(t, err)

	// Verify socket file created
	_, err = os.Stat(socketPath)
	assert.NoError(t, err)

	// Verify socket permissions (should be 0600)
	info, _ := os.Stat(socketPath)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm()&0777, "socket should have 0600 permissions")

	err = srv.Stop()
	assert.NoError(t, err)

	// Socket file should be removed
	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err))
}

func TestServer_ConnectionCount(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	srv.RegisterHandler(MethodPing, func(req *Request) *Response {
		return NewResponse(req, map[string]any{"pong": true})
	})

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	assert.Equal(t, int32(0), srv.ConnectionCount())

	// Connect client
	client := NewClient(socketPath)
	err = client.Connect()
	require.NoError(t, err)

	// Give server time to accept connection
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), srv.ConnectionCount())

	client.Close()

	// Give server time to detect disconnect
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), srv.ConnectionCount())
}

func TestClient_ConnectDisconnect(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := NewClient(socketPath)
	assert.False(t, client.IsConnected())

	err = client.Connect()
	require.NoError(t, err)
	assert.True(t, client.IsConnected())

	err = client.Close()
	assert.NoError(t, err)
	assert.False(t, client.IsConnected())
}

func TestClient_RequestResponse(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	srv.RegisterHandler(MethodPing, func(req *Request) *Response {
		return NewResponse(req, map[string]any{"pong": true})
	})
	srv.RegisterHandler(MethodHello, func(req *Request) *Response {
		version := "unknown"
		if req.Params != nil {
			if v, ok := req.Params["version"].(string); ok {
				version = v
			}
		}
		return NewResponse(req, map[string]any{
			"version": version,
			"permissions": map[string]any{
				"accessibility":    true,
				"screen_recording": true,
			},
		})
	})

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := NewClient(socketPath)
	err = client.Connect()
	require.NoError(t, err)
	defer client.Close()

	// Test ping
	resp, err := client.Call(MethodPing, nil)
	require.NoError(t, err)
	assert.Nil(t, resp.Error)
	result := resp.Result.(map[string]any)
	assert.Equal(t, true, result["pong"])

	// Test hello with params
	resp, err = client.Call(MethodHello, map[string]any{"version": "1.0.0"})
	require.NoError(t, err)
	assert.Nil(t, resp.Error)
	result = resp.Result.(map[string]any)
	assert.Equal(t, "1.0.0", result["version"])
}

func TestClient_ErrorResponse(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	srv.RegisterHandler("fail", func(req *Request) *Response {
		return NewErrorResponse(req, ErrWindowNotFound, "Window not found: Safari")
	})

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := NewClient(socketPath)
	err = client.Connect()
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Call("fail", nil)
	require.NoError(t, err)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrWindowNotFound, resp.Error.Code)
	assert.Equal(t, "Window not found: Safari", resp.Error.Message)
}

func TestServer_UnknownMethod(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := NewClient(socketPath)
	err = client.Connect()
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Call("unknown_method", nil)
	require.NoError(t, err)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrMethodNotFound, resp.Error.Code)
}

func TestServer_MultipleClients(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	var callCount atomic.Int32
	srv := NewServer(socketPath)
	srv.RegisterHandler(MethodPing, func(req *Request) *Response {
		callCount.Add(1)
		return NewResponse(req, map[string]any{"count": callCount.Load()})
	})

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	// Create two clients
	client1 := NewClient(socketPath)
	err = client1.Connect()
	require.NoError(t, err)
	defer client1.Close()

	client2 := NewClient(socketPath)
	err = client2.Connect()
	require.NoError(t, err)
	defer client2.Close()

	// Both should work independently
	resp1, err := client1.Call(MethodPing, nil)
	require.NoError(t, err)
	assert.Nil(t, resp1.Error)

	resp2, err := client2.Call(MethodPing, nil)
	require.NoError(t, err)
	assert.Nil(t, resp2.Error)

	assert.Equal(t, int32(2), callCount.Load())
}

func TestClient_Timeout(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	srv.RegisterHandler("slow", func(req *Request) *Response {
		time.Sleep(500 * time.Millisecond)
		return NewResponse(req, nil)
	})

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := NewClient(socketPath)
	client.SetTimeout(100 * time.Millisecond)
	err = client.Connect()
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Call("slow", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestServer_StaleSocketCleanup(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	// Create a stale socket file
	f, err := os.Create(socketPath)
	require.NoError(t, err)
	f.Close()

	// Server should clean it up and start successfully
	srv := NewServer(socketPath)
	err = srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	// Client should be able to connect
	client := NewClient(socketPath)
	err = client.Connect()
	require.NoError(t, err)
	client.Close()
}

func TestServer_SocketPath(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	assert.Equal(t, socketPath, srv.SocketPath())
}

func TestClient_Ping(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	srv := NewServer(socketPath)
	srv.RegisterHandler(MethodPing, func(req *Request) *Response {
		return NewResponse(req, map[string]any{"pong": true})
	})

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	client := NewClient(socketPath)
	err = client.Connect()
	require.NoError(t, err)
	defer client.Close()

	err = client.Ping()
	assert.NoError(t, err)
}

func TestClient_Ping_NotConnected(t *testing.T) {
	client := NewClient("/nonexistent/path.sock")
	err := client.Ping()
	assert.Error(t, err)
}

func TestServer_BroadcastNotification(t *testing.T) {
	// Use short path to avoid macOS socket path length limits
	socketPath := filepath.Join("/tmp", "test-broadcast.sock")
	defer os.Remove(socketPath)

	srv := NewServer(socketPath)
	srv.RegisterHandler(MethodPing, func(req *Request) *Response {
		return NewResponse(req, map[string]any{"pong": true})
	})

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	// Connect a client
	client := NewClient(socketPath)
	err = client.Connect()
	require.NoError(t, err)
	defer client.Close()

	// Give server time to accept connection
	time.Sleep(50 * time.Millisecond)

	// Broadcast should not error
	notif := NewNotification(MethodStopping, map[string]any{"reason": "test"})
	srv.BroadcastNotification(notif)

	// Server should still work after broadcast
	resp, err := client.Call(MethodPing, nil)
	require.NoError(t, err)
	assert.Nil(t, resp.Error)
}
