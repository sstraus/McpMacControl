package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/shell"
)

// ShellManager is the global session manager, initialized by main.go
var ShellManager *shell.Manager

// ValidShellActions lists all supported shell action types.
var ValidShellActions = []string{"spawn", "send_input", "get_snapshot", "resize", "close", "list"}

// HandleShell dispatches shell actions: spawn, send_input, get_snapshot, resize, close, list.
func HandleShell(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := request.GetString("action", "")
	if action == "" {
		return mcp.NewToolResultError(fmt.Sprintf(`Missing "action" parameter.

The shell tool requires an "action" string specifying what to do.

Valid actions: %s

Examples:
  shell({"action": "spawn", "command": "/bin/bash", "cols": 120, "rows": 30})
  shell({"action": "send_input", "session_id": "abc", "input": "ls"})
  shell({"action": "send_input", "session_id": "abc", "special_key": "enter", "wait_ms": 500})
  shell({"action": "get_snapshot", "session_id": "abc", "format": "ansi"})
  shell({"action": "list"})`, fmt.Sprintf("%v", ValidShellActions))), nil
	}

	switch action {
	case "spawn":
		return handleShellSpawn(request)
	case "send_input":
		return handleShellSendInput(request)
	case "get_snapshot":
		return handleShellGetSnapshot(request)
	case "resize":
		return handleShellResize(request)
	case "close":
		return handleShellClose(request)
	case "list":
		return handleShellList()
	default:
		return mcp.NewToolResultError(fmt.Sprintf(`Unknown action: %q

Valid actions: %s

Examples:
  shell({"action": "spawn", "command": "/bin/bash"})
  shell({"action": "send_input", "session_id": "abc", "input": "ls\n"})
  shell({"action": "get_snapshot", "session_id": "abc"})
  shell({"action": "list"})`, action, fmt.Sprintf("%v", ValidShellActions))), nil
	}
}

func handleShellSpawn(request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if ShellManager == nil {
		return mcp.NewToolResultError("shell manager not initialized"), nil
	}

	// Extract parameters with defaults
	command := request.GetString("command", "/bin/bash")
	cwd := request.GetString("cwd", "")
	cols := request.GetInt("cols", 80)
	rows := request.GetInt("rows", 24)

	// Parse args array
	args := request.GetStringSlice("args", nil)

	session, err := ShellManager.Spawn(command, args, cwd, cols, rows)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to spawn session: %v", err)), nil
	}

	// Get initial snapshot
	snapshot := session.Snapshot("text")

	return mcp.NewToolResultText(fmt.Sprintf(`Session created.

ID: %s
PID: %d
Command: %s
Size: %dx%d

Initial snapshot:
%s

Use shell({"action": "send_input", "session_id": "%s", "input": "cmd\n"}) to send commands.
Use shell({"action": "get_snapshot", "session_id": "%s"}) to get output.
Use shell({"action": "close", "session_id": "%s"}) when done.`,
		session.ID, session.PID(), session.Command, cols, rows, snapshot, session.ID, session.ID, session.ID)), nil
}

func handleShellSendInput(request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if ShellManager == nil {
		return mcp.NewToolResultError("shell manager not initialized"), nil
	}

	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	session, err := ShellManager.Get(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s\n\nUse shell({\"action\": \"spawn\"}) to create a new session.", sessionID)), nil
	}

	// Either input text or special key (or both)
	input := request.GetString("input", "")
	specialKey := request.GetString("special_key", "")
	waitMs := request.GetInt("wait_ms", 100)

	if input != "" {
		if err := session.SendInput(input); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send input: %v", err)), nil
		}
	}

	if specialKey != "" {
		if err := session.SendSpecialKey(specialKey); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send special key: %v", err)), nil
		}
	}

	if input == "" && specialKey == "" {
		return mcp.NewToolResultError("either 'input' or 'special_key' is required"), nil
	}

	// Wait for output to appear in the terminal buffer before taking snapshot
	if waitMs > 0 {
		time.Sleep(time.Duration(waitMs) * time.Millisecond)
	}

	// Get snapshot after input
	snapshot := session.Snapshot("text")

	return mcp.NewToolResultText(fmt.Sprintf("Input sent to session %s.\n\nCurrent snapshot:\n%s", sessionID, snapshot)), nil
}

func handleShellGetSnapshot(request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if ShellManager == nil {
		return mcp.NewToolResultError("shell manager not initialized"), nil
	}

	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	session, err := ShellManager.Get(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s\n\nUse shell({\"action\": \"spawn\"}) to create a new session.", sessionID)), nil
	}

	format := request.GetString("format", "text")
	if format != "text" && format != "ansi" {
		return mcp.NewToolResultError("format must be 'text' or 'ansi'"), nil
	}

	// Wait before taking snapshot if requested
	waitMs := request.GetInt("wait_ms", 0)
	if waitMs > 0 {
		time.Sleep(time.Duration(waitMs) * time.Millisecond)
	}

	snapshot := session.Snapshot(format)

	if snapshot == "" {
		return mcp.NewToolResultText("[empty snapshot]"), nil
	}

	return mcp.NewToolResultText(snapshot), nil
}

func handleShellResize(request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if ShellManager == nil {
		return mcp.NewToolResultError("shell manager not initialized"), nil
	}

	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	cols := request.GetInt("cols", 80)
	rows := request.GetInt("rows", 24)

	if rows <= 0 || cols <= 0 {
		return mcp.NewToolResultError("rows and cols must be positive"), nil
	}

	session, err := ShellManager.Get(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s", sessionID)), nil
	}

	if err := session.Resize(cols, rows); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to resize: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Resized session %s to %dx%d", sessionID, cols, rows)), nil
}

func handleShellClose(request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if ShellManager == nil {
		return mcp.NewToolResultError("shell manager not initialized"), nil
	}

	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	session, err := ShellManager.Get(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s", sessionID)), nil
	}

	exitCode := session.ExitCode()
	exitCodeStr := "still running"
	if exitCode != nil {
		exitCodeStr = fmt.Sprintf("%d", *exitCode)
	}

	if err := ShellManager.Close(sessionID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to close session: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Session %s closed. Exit code: %s", sessionID, exitCodeStr)), nil
}

func handleShellList() (*mcp.CallToolResult, error) {
	if ShellManager == nil {
		return mcp.NewToolResultError("shell manager not initialized"), nil
	}

	sessions := ShellManager.List()
	if len(sessions) == 0 {
		return mcp.NewToolResultText("No active sessions.\n\nUse shell({\"action\": \"spawn\"}) to create a new session."), nil
	}

	output, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format sessions: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d active session(s):\n\n%s", len(sessions), string(output))), nil
}
