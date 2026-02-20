package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"

	"github.com/sstraus/mcpmaccontrol/internal/permissions"
	"github.com/sstraus/mcpmaccontrol/internal/shell"
)

// skipWithoutAccessibility skips the test if accessibility permission
// is not granted (e.g., CI or unprivileged test runner).
func skipWithoutAccessibility(t *testing.T) {
	t.Helper()
	if !permissions.HasAccessibilityPermission() {
		t.Skip("Accessibility permission not granted, skipping")
	}
}


// mockRequest creates a mock MCP call tool request
func mockRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "test",
			Arguments: args,
		},
	}
}

func TestHandleCaptureWindow_MissingAppName(t *testing.T) {
	req := mockRequest(map[string]any{})

	result, err := HandleCaptureWindow(context.Background(), req)

	assert.NoError(t, err) // Tool errors return as result, not error
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "app_name is required")
}

func TestHandleCaptureScreen_DefaultParams(t *testing.T) {
	// This test will fail to connect to agent, but tests parameter handling
	req := mockRequest(map[string]any{})

	result, err := HandleCaptureScreen(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Will error because no agent, but shouldn't panic
}

func TestHandleListWindows_EmptyFilter(t *testing.T) {
	// This test will fail to connect to agent, but tests parameter handling
	req := mockRequest(map[string]any{})

	result, err := HandleListWindows(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Will error because no agent, but shouldn't panic
}

func TestHandleHelp_NoTopic(t *testing.T) {
	req := mockRequest(map[string]any{})

	result, err := HandleHelp(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	// Help should return overview
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "macOS")
}

func TestHandleHelp_ActionsTopic(t *testing.T) {
	req := mockRequest(map[string]any{"topic": "actions"})

	result, err := HandleHelp(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "click")
	assert.Contains(t, text, "type")
}

func TestHandleHelp_ExamplesTopic(t *testing.T) {
	req := mockRequest(map[string]any{"topic": "examples"})

	result, err := HandleHelp(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
}

func TestHandleHelp_ShellTopic(t *testing.T) {
	req := mockRequest(map[string]any{"topic": "shell"})

	result, err := HandleHelp(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "shell")
}

func TestHandleHelp_ProcessesTopic(t *testing.T) {
	req := mockRequest(map[string]any{"topic": "processes"})

	result, err := HandleHelp(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "PROCESSES TOOL")
	assert.Contains(t, text, "name")
	assert.Contains(t, text, "pid")
}

func TestHandleHelp_UnknownTopic(t *testing.T) {
	req := mockRequest(map[string]any{"topic": "unknown_topic_xyz"})

	result, err := HandleHelp(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Unknown topic returns general help or error
}

func TestHandleDo_MissingActions(t *testing.T) {
	skipWithoutAccessibility(t)
	req := mockRequest(map[string]any{})

	result, err := HandleDo(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "actions")
}

func TestHandleDo_EmptyActions(t *testing.T) {
	skipWithoutAccessibility(t)
	req := mockRequest(map[string]any{"actions": []any{}})

	result, err := HandleDo(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "empty")
}

func TestHandleDo_InvalidActionFormat(t *testing.T) {
	skipWithoutAccessibility(t)
	req := mockRequest(map[string]any{"actions": "not an array"})

	result, err := HandleDo(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestHandleDo_ValidationError(t *testing.T) {
	skipWithoutAccessibility(t)
	// Action with missing required field
	req := mockRequest(map[string]any{
		"actions": []any{
			map[string]any{"type": "click", "x": 100, "y": 100}, // missing app
		},
	})

	result, err := HandleDo(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "app")
}

func TestHandleDo_WaitAction(t *testing.T) {
	skipWithoutAccessibility(t)
	// Wait action doesn't need agent connection
	req := mockRequest(map[string]any{
		"actions": []any{
			map[string]any{"type": "wait", "ms": 10},
		},
	})

	result, err := HandleDo(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "wait")
	assert.Contains(t, text, "10ms")
}

// --- Shell tool (consolidated) tests ---

// --- Alert tool tests ---

func TestHandleAlert_ActivateWarn(t *testing.T) {
	req := mockRequest(map[string]any{"active": true})

	result, err := HandleAlert(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "activated")
}

func TestHandleAlert_DeactivateWarn(t *testing.T) {
	req := mockRequest(map[string]any{"active": false})

	result, err := HandleAlert(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "deactivated")
}

func TestHandleAlert_MissingParam(t *testing.T) {
	req := mockRequest(map[string]any{})

	result, err := HandleAlert(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Default is false (deactivate) — should still succeed
	assert.False(t, result.IsError)
}

// --- Shell tool (consolidated) tests ---

func TestHandleShell_MissingAction(t *testing.T) {
	req := mockRequest(map[string]any{})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "action")
}

func TestHandleShell_InvalidAction(t *testing.T) {
	req := mockRequest(map[string]any{"action": "invalid"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "Unknown action")
	assert.Contains(t, text, "spawn")
}

func TestHandleShell_SpawnNoManager(t *testing.T) {
	// Ensure ShellManager is nil
	old := ShellManager
	ShellManager = nil
	defer func() { ShellManager = old }()

	req := mockRequest(map[string]any{"action": "spawn"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "shell manager not initialized")
}

func TestHandleShell_WriteNoManager(t *testing.T) {
	old := ShellManager
	ShellManager = nil
	defer func() { ShellManager = old }()

	req := mockRequest(map[string]any{"action": "write", "session_id": "abc", "input": "ls\n"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleShell_ReadNoManager(t *testing.T) {
	old := ShellManager
	ShellManager = nil
	defer func() { ShellManager = old }()

	req := mockRequest(map[string]any{"action": "read", "session_id": "abc"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleShell_ListNoManager(t *testing.T) {
	old := ShellManager
	ShellManager = nil
	defer func() { ShellManager = old }()

	req := mockRequest(map[string]any{"action": "list"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleShell_WriteMissingSessionID(t *testing.T) {
	ShellManager = shell.NewManager()
	defer func() { ShellManager.CloseAll(); ShellManager = nil }()

	req := mockRequest(map[string]any{"action": "write", "input": "ls\n"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "session_id")
}

func TestHandleShell_WriteMissingInput(t *testing.T) {
	ShellManager = shell.NewManager()
	defer func() { ShellManager.CloseAll(); ShellManager = nil }()

	req := mockRequest(map[string]any{"action": "write", "session_id": "abc"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "input")
}

func TestHandleShell_SpawnWithManager(t *testing.T) {
	ShellManager = shell.NewManager()
	defer func() {
		ShellManager.CloseAll()
		ShellManager = nil
	}()

	req := mockRequest(map[string]any{"action": "spawn"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "Session created")
	assert.Contains(t, text, "ID:")
}

func TestHandleShell_ListWithManager(t *testing.T) {
	ShellManager = shell.NewManager()
	defer func() {
		ShellManager.CloseAll()
		ShellManager = nil
	}()

	req := mockRequest(map[string]any{"action": "list"})

	result, err := HandleShell(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "No active sessions")
}
