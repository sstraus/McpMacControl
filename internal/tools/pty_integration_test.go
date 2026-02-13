package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sstraus/mcpmaccontrol/internal/shell"
)

func callShell(t *testing.T, args map[string]any) string {
	t.Helper()
	result, err := HandleShell(context.Background(), mockRequest(args))
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(mcp.TextContent).Text
	if result.IsError {
		t.Fatalf("Tool error: %s", text)
	}
	return text
}

func TestPTYIntegration_SessionLifecycle(t *testing.T) {
	ShellManager = shell.NewManager()
	defer ShellManager.CloseAll()

	// STEP 1: Spawn
	t.Log("=== STEP 1: Spawn session ===")
	spawnResult := callShell(t, map[string]any{
		"action":  "spawn",
		"command": "/bin/bash",
		"cols":    float64(120),
		"rows":    float64(30),
	})
	t.Logf("Spawn result:\n%s", spawnResult)

	var sessionID string
	for _, line := range strings.Split(spawnResult, "\n") {
		if strings.HasPrefix(line, "ID: ") {
			sessionID = strings.TrimPrefix(line, "ID: ")
			break
		}
	}
	require.NotEmpty(t, sessionID, "Could not extract session ID")
	t.Logf("Session ID: %s", sessionID)

	// STEP 2: send_input with wait_ms
	t.Log("\n=== STEP 2: send_input with wait_ms ===")
	sendResult1 := callShell(t, map[string]any{
		"action":      "send_input",
		"session_id":  sessionID,
		"input":       "echo MARKER_AAA",
		"special_key": "enter",
		"wait_ms":     float64(500),
	})
	t.Logf("Send result 1:\n%s", sendResult1)
	assert.Contains(t, sendResult1, "MARKER_AAA", "snapshot after wait_ms should contain marker")

	// STEP 3: get_snapshot (session still alive)
	t.Log("\n=== STEP 3: get_snapshot (session still alive) ===")
	snapResult1 := callShell(t, map[string]any{
		"action":     "get_snapshot",
		"session_id": sessionID,
	})
	t.Logf("Snapshot 1:\n%s", snapResult1)
	assert.Contains(t, snapResult1, "MARKER_AAA", "previous marker should persist")

	// STEP 4: Session continuity (both markers visible)
	t.Log("\n=== STEP 4: Session continuity ===")
	sendResult2 := callShell(t, map[string]any{
		"action":      "send_input",
		"session_id":  sessionID,
		"input":       "echo MARKER_BBB",
		"special_key": "enter",
		"wait_ms":     float64(500),
	})
	t.Logf("Send result 2:\n%s", sendResult2)
	assert.Contains(t, sendResult2, "MARKER_BBB", "new marker should be present")
	assert.Contains(t, sendResult2, "MARKER_AAA", "old marker should still be present - same session")

	// STEP 5: Env variable persistence
	t.Log("\n=== STEP 5: Env variable persistence ===")
	callShell(t, map[string]any{
		"action":      "send_input",
		"session_id":  sessionID,
		"input":       "export MY_TEST_VAR=PERSISTENCE_CHECK",
		"special_key": "enter",
		"wait_ms":     float64(200),
	})
	sendResult3 := callShell(t, map[string]any{
		"action":      "send_input",
		"session_id":  sessionID,
		"input":       "echo $MY_TEST_VAR",
		"special_key": "enter",
		"wait_ms":     float64(500),
	})
	t.Logf("Env var result:\n%s", sendResult3)
	assert.Contains(t, sendResult3, "PERSISTENCE_CHECK", "env var should persist across commands")

	// STEP 6: Resize
	t.Log("\n=== STEP 6: Resize ===")
	resizeResult := callShell(t, map[string]any{
		"action":     "resize",
		"session_id": sessionID,
		"cols":       float64(80),
		"rows":       float64(20),
	})
	t.Logf("Resize result: %s", resizeResult)
	assert.Contains(t, resizeResult, "Resized")

	// STEP 7: get_snapshot with wait_ms
	t.Log("\n=== STEP 7: get_snapshot with wait_ms ===")
	callShell(t, map[string]any{
		"action":      "send_input",
		"session_id":  sessionID,
		"input":       "echo AFTER_RESIZE",
		"special_key": "enter",
	})
	snapResult2 := callShell(t, map[string]any{
		"action":     "get_snapshot",
		"session_id": sessionID,
		"wait_ms":    float64(500),
	})
	t.Logf("Snapshot with wait:\n%s", snapResult2)
	assert.Contains(t, snapResult2, "AFTER_RESIZE", "get_snapshot wait_ms should capture output")

	// STEP 8: List sessions (exactly 1)
	t.Log("\n=== STEP 8: List sessions ===")
	listResult := callShell(t, map[string]any{
		"action": "list",
	})
	t.Logf("List result:\n%s", listResult)
	assert.Contains(t, listResult, "1 active session")
	assert.Contains(t, listResult, sessionID)

	// STEP 9: Process still running
	t.Log("\n=== STEP 9: Process still running ===")
	session, err := ShellManager.Get(sessionID)
	require.NoError(t, err)
	assert.Greater(t, session.PID(), 0, "PID should be positive")
	assert.Nil(t, session.ExitCode(), "process should still be running")

	// STEP 10: Close
	t.Log("\n=== STEP 10: Close session ===")
	closeResult := callShell(t, map[string]any{
		"action":     "close",
		"session_id": sessionID,
	})
	t.Logf("Close result: %s", closeResult)
	assert.Contains(t, closeResult, "closed")

	// STEP 11: Verify gone
	t.Log("\n=== STEP 11: Session removed ===")
	listResult2 := callShell(t, map[string]any{
		"action": "list",
	})
	assert.NotContains(t, listResult2, sessionID, "session should be removed")

	result, err := HandleShell(context.Background(), mockRequest(map[string]any{
		"action":     "get_snapshot",
		"session_id": sessionID,
	}))
	require.NoError(t, err)
	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "not found", "closed session should return not found")

	t.Log("\n=== ALL 11 STEPS PASSED ===")
}
