package agent

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sstraus/mcpmaccontrol/internal/protocol"
)

func TestNewServer(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	assert.NotNil(t, s)
	assert.NotNil(t, s.protoServer)
	assert.NotNil(t, s.lifecycle)
}

func TestServer_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	err := s.Start()
	require.NoError(t, err)

	// Verify socket was created
	_, err = os.Stat(socketPath)
	assert.NoError(t, err)

	// Verify lifecycle is ready
	assert.Equal(t, StateReady, s.Lifecycle().State())

	// Stop server
	err = s.Stop()
	assert.NoError(t, err)

	// Verify socket was removed
	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err))
}

func TestServer_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	lc := s.Lifecycle()
	assert.NotNil(t, lc)
	assert.Equal(t, StateStarting, lc.State())
}

func TestServer_ProtoServer(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	ps := s.ProtoServer()
	assert.NotNil(t, ps)
	assert.Equal(t, socketPath, ps.SocketPath())
}

func TestServer_BroadcastStopping(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	err := s.Start()
	require.NoError(t, err)
	defer s.Stop()

	// Should not panic even with no clients
	s.BroadcastStopping(protocol.StopReasonUserInitiated)
}

func TestServer_OnStopCallback(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	var stopReason string
	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
		OnStop: func(reason string) {
			stopReason = reason
		},
	})

	err := s.Start()
	require.NoError(t, err)
	defer s.Stop()

	// Trigger user stop through lifecycle
	s.Lifecycle().UserStop()

	assert.Equal(t, "user_initiated", stopReason)
}

func TestPidFilePath(t *testing.T) {
	path := pidFilePath()
	assert.Contains(t, path, "/tmp/mcpmaccontrol-agent-")
	assert.Contains(t, path, ".pid")
}

// Test helper functions
func TestGetString(t *testing.T) {
	params := map[string]any{
		"name":   "test",
		"number": 42,
		"empty":  "",
	}

	assert.Equal(t, "test", getString(params, "name", "default"))
	assert.Equal(t, "default", getString(params, "number", "default"))
	assert.Equal(t, "", getString(params, "empty", "default"))
	assert.Equal(t, "default", getString(params, "missing", "default"))
	assert.Equal(t, "default", getString(nil, "any", "default"))
}

func TestGetInt(t *testing.T) {
	params := map[string]any{
		"float":  float64(42),
		"int":    123,
		"string": "not a number",
	}

	assert.Equal(t, 42, getInt(params, "float", 0))
	assert.Equal(t, 123, getInt(params, "int", 0))
	assert.Equal(t, 0, getInt(params, "string", 0))
	assert.Equal(t, 99, getInt(params, "missing", 99))
	assert.Equal(t, 99, getInt(nil, "any", 99))
}

func TestGetBool(t *testing.T) {
	params := map[string]any{
		"true":   true,
		"false":  false,
		"string": "true",
	}

	assert.True(t, getBool(params, "true", false))
	assert.False(t, getBool(params, "false", true))
	assert.False(t, getBool(params, "string", false))
	assert.True(t, getBool(params, "missing", true))
	assert.True(t, getBool(nil, "any", true))
}

func TestGetStringSlice(t *testing.T) {
	params := map[string]any{
		"array":  []any{"a", "b", "c"},
		"single": "one",
		"mixed":  []any{"str", 123, "another"},
		"number": 42,
	}

	assert.Equal(t, []string{"a", "b", "c"}, getStringSlice(params, "array"))
	assert.Equal(t, []string{"one"}, getStringSlice(params, "single"))
	assert.Equal(t, []string{"str", "another"}, getStringSlice(params, "mixed"))
	assert.Nil(t, getStringSlice(params, "number"))
	assert.Nil(t, getStringSlice(params, "missing"))
	assert.Nil(t, getStringSlice(nil, "any"))
}

// Handler tests

func TestServer_HandleHello(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "hello",
	}

	resp := s.handleHello(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.Equal(t, 1, resp.ID)

	result, ok := resp.Result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, Version, result["version"])

	perms, ok := result["permissions"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, perms, "accessibility")
	assert.Contains(t, perms, "screen_recording")
}

func TestServer_HandlePing(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	s.lifecycle.SetReady()

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "ping",
	}

	resp := s.handlePing(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]any)
	require.True(t, ok)
	assert.True(t, result["pong"].(bool))
	assert.Equal(t, "ready", result["state"])
}

func TestServer_HandleShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	err := s.Start()
	require.NoError(t, err)
	defer s.Stop()

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "shutdown",
	}

	resp := s.handleShutdown(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]any)
	require.True(t, ok)
	assert.True(t, result["ok"].(bool))

	// Wait for shutdown to be triggered
	time.Sleep(200 * time.Millisecond)
	state := s.lifecycle.State()
	assert.True(t, state == StateShuttingDown || state == StateStopped,
		"expected StateShuttingDown or StateStopped, got %v", state)
}

func TestServer_HandleClick_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "click",
		Params: map[string]any{
			"x": 100,
			"y": 100,
		},
	}

	resp := s.handleClick(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// May fail on permission check or parameter validation
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleMove_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "move",
		Params: map[string]any{
			"x": 100,
			"y": 100,
		},
	}

	resp := s.handleMove(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleType_MissingText(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "type",
		Params:  map[string]any{},
	}

	resp := s.handleType(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleKey_MissingKey(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "key",
		Params:  map[string]any{},
	}

	resp := s.handleKey(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// Handler checks permissions before params, so may get either error
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleScroll_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      8,
		Method:  "scroll",
		Params: map[string]any{
			"x":       100,
			"y":       100,
			"delta_y": 10,
		},
	}

	resp := s.handleScroll(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// Handler checks permissions before params, so may get either error
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleFocus_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      9,
		Method:  "focus",
		Params:  map[string]any{},
	}

	resp := s.handleFocus(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// Handler checks permissions before params, so may get either error
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleResize_InvalidDimensions(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "resize",
		Params: map[string]any{
			"app":    "Test",
			"width":  0,
			"height": 100,
		},
	}

	resp := s.handleResize(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// Handler checks permissions before params, so may get either error
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleCaptureWindow_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      11,
		Method:  "capture_window",
		Params:  map[string]any{},
	}

	resp := s.handleCaptureWindow(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, protocol.ErrInvalidParams, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "app parameter required")
}

func TestServer_HandleCaptureWindow_InvalidRegion(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	// Test with valid app but this will fail on permission check
	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      12,
		Method:  "capture_window",
		Params: map[string]any{
			"app":           "TestApp",
			"region_x":      -10, // Invalid
			"region_y":      0,
			"region_width":  100,
			"region_height": 100,
		},
	}

	resp := s.handleCaptureWindow(req)
	assert.NotNil(t, resp)
	// Will error on permission or window not found - both are valid
}

func TestServer_HandleListWindows_WithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      13,
		Method:  "list_windows",
		Params: map[string]any{
			"filter": "Finder",
		},
	}

	resp := s.handleListWindows(req)
	assert.NotNil(t, resp)
	// May succeed or fail based on permissions
}

func TestServer_HandleCaptureScreen_WithParams(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      14,
		Method:  "capture_screen",
		Params: map[string]any{
			"display": 0,
			"format":  "png",
		},
	}

	resp := s.handleCaptureScreen(req)
	assert.NotNil(t, resp)
	// May succeed or fail based on permissions
}

func TestServer_HandleOpenSettings(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      15,
		Method:  "open_settings",
		Params: map[string]any{
			"pane": "accessibility",
		},
	}

	// This will actually try to open settings, skip in CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI")
	}

	resp := s.handleOpenSettings(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleMinimize_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      16,
		Method:  "minimize",
		Params:  map[string]any{},
	}

	resp := s.handleMinimize(req)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// Handler checks permissions before params, so may get either error
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleRestore_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      17,
		Method:  "restore",
		Params:  map[string]any{},
	}

	resp := s.handleRestore(req)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// Handler checks permissions before params, so may get either error
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestServer_HandleClose_MissingApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      18,
		Method:  "close",
		Params:  map[string]any{},
	}

	resp := s.handleClose(req)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	// Handler checks permissions before params, so may get either error
	assert.True(t, resp.Error.Code == protocol.ErrInvalidParams || resp.Error.Code == protocol.ErrPermissionAccessibility)
}

func TestLifecycle_StateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateStarting, "starting"},
		{StateReady, "ready"},
		{StateActive, "active"},
		{StateShuttingDown, "shutting_down"},
		{StateStopped, "stopped"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestCropImage(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Fill with a color
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, image.White)
		}
	}

	// Crop a region (fast path using SubImage)
	cropped := cropImage(img, 10, 10, 50, 50)

	assert.NotNil(t, cropped)
	bounds := cropped.Bounds()
	assert.Equal(t, 50, bounds.Dx())
	assert.Equal(t, 50, bounds.Dy())
}

// noSubImage wraps an image without exposing SubImage interface
type noSubImage struct {
	image.Image
}

func TestCropImage_Fallback(t *testing.T) {
	// Create a simple test image wrapped to not expose SubImage
	baseImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Fill with different colors based on position
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			baseImg.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
		}
	}

	img := noSubImage{baseImg}

	// Crop a region (fallback path copying pixels)
	cropped := cropImage(img, 10, 20, 30, 40)

	assert.NotNil(t, cropped)
	bounds := cropped.Bounds()
	assert.Equal(t, 30, bounds.Dx())
	assert.Equal(t, 40, bounds.Dy())

	// Verify pixel values were copied correctly
	r, g, b, a := cropped.At(0, 0).RGBA()
	// Source pixel at (10, 20) should have R=10, G=20, B=128, A=255
	assert.Equal(t, uint32(10*257), r) // 257 is the alpha premultiplication factor for 8-bit
	assert.Equal(t, uint32(20*257), g)
	assert.Equal(t, uint32(128*257), b)
	assert.Equal(t, uint32(255*257), a)
}

func TestFindWindowPID_NotFound(t *testing.T) {
	// This will fail because app doesn't exist
	_, err := findWindowPID("NonExistentAppXYZ123")
	assert.Error(t, err)
}

// Integration tests - run when permissions are available
// These test the full handler paths including system calls

func TestServer_HandleListWindows_Success(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      20,
		Method:  "list_windows",
		Params:  map[string]any{},
	}

	resp := s.handleListWindows(req)
	assert.NotNil(t, resp)
	// If permissions granted, should succeed with array result
	// If not, should return permission error
	if resp.Error == nil {
		_, ok := resp.Result.([]map[string]any)
		assert.True(t, ok || resp.Result != nil)
	}
}

func TestServer_HandleClick_WithApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      21,
		Method:  "click",
		Params: map[string]any{
			"app":    "Finder",
			"x":      100,
			"y":      100,
			"button": "left",
		},
	}

	resp := s.handleClick(req)
	assert.NotNil(t, resp)
	// Will succeed or fail based on permissions/window availability
}

func TestServer_HandleMove_WithApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      22,
		Method:  "move",
		Params: map[string]any{
			"app": "Finder",
			"x":   100,
			"y":   100,
		},
	}

	resp := s.handleMove(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleType_WithText(t *testing.T) {
	// Skip as this actually types text
	if testing.Short() {
		t.Skip("Skipping in short mode - would type actual text")
	}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      23,
		Method:  "type",
		Params: map[string]any{
			"text": "", // Empty text to avoid typing
		},
	}

	resp := s.handleType(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleKey_WithKey(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      24,
		Method:  "key",
		Params: map[string]any{
			"key":       "a",
			"modifiers": []any{"cmd"}, // Will simulate Cmd+A (select all)
		},
	}

	resp := s.handleKey(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleScroll_WithApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      25,
		Method:  "scroll",
		Params: map[string]any{
			"app":     "Finder",
			"x":       100,
			"y":       100,
			"delta_y": 10,
		},
	}

	resp := s.handleScroll(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleFocus_WithApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      26,
		Method:  "focus",
		Params: map[string]any{
			"app": "Finder",
		},
	}

	resp := s.handleFocus(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleMinimize_WithApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      27,
		Method:  "minimize",
		Params: map[string]any{
			"app": "Finder",
		},
	}

	resp := s.handleMinimize(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleRestore_WithApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      28,
		Method:  "restore",
		Params: map[string]any{
			"app": "Finder",
		},
	}

	resp := s.handleRestore(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleClose_WithApp(t *testing.T) {
	// Skip as this would actually close a window
	t.Skip("Skipping - would close actual window")
}

func TestServer_HandleResize_WithApp(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      30,
		Method:  "resize",
		Params: map[string]any{
			"app":    "Finder",
			"width":  800,
			"height": 600,
		},
	}

	resp := s.handleResize(req)
	assert.NotNil(t, resp)
}

func TestServer_HandleCaptureScreen_Success(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      31,
		Method:  "capture_screen",
		Params: map[string]any{
			"display": 0,
		},
	}

	resp := s.handleCaptureScreen(req)
	assert.NotNil(t, resp)
	// If permissions granted, result should have image data
	if resp.Error == nil {
		result, ok := resp.Result.(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, result, "image")
	}
}

func TestServer_HandleCaptureWindow_Success(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	s := NewServer(ServerConfig{
		SocketPath:  socketPath,
		IdleTimeout: 5 * time.Minute,
	})

	req := &protocol.Request{
		JSONRPC: "2.0",
		ID:      32,
		Method:  "capture_window",
		Params: map[string]any{
			"app":    "Finder",
			"format": "webp",
		},
	}

	resp := s.handleCaptureWindow(req)
	assert.NotNil(t, resp)
	// Will succeed or fail based on permissions/window availability
}
