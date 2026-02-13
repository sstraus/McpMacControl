package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mark3labs/mcp-go/mcp"
)

// fakeProcessLister returns a canned process list for testing.
func fakeProcessLister() ([]processInfo, error) {
	return []processInfo{
		{PID: 1, PPID: 0, Name: "launchd", User: "root", CPUPercent: 0.1, MemMB: 12.5, State: "Ss"},
		{PID: 100, PPID: 1, Name: "WindowServer", User: "root", CPUPercent: 5.2, MemMB: 210.3, State: "Ss"},
		{PID: 200, PPID: 1, Name: "Finder", User: "testuser", CPUPercent: 0.3, MemMB: 85.0, State: "S"},
		{PID: 201, PPID: 1, Name: "Safari", User: "testuser", CPUPercent: 12.1, MemMB: 450.7, State: "S"},
		{PID: 202, PPID: 201, Name: "Safari Web Content", User: "testuser", CPUPercent: 3.4, MemMB: 120.2, State: "S"},
		{PID: 300, PPID: 1, Name: "node", User: "testuser", CPUPercent: 0.0, MemMB: 60.0, State: "S"},
	}, nil
}

func setFakeProcesses(t *testing.T) {
	t.Helper()
	old := listProcessesFn
	listProcessesFn = fakeProcessLister
	t.Cleanup(func() { listProcessesFn = old })
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	return result.Content[0].(mcp.TextContent).Text
}

func TestHandleProcesses_DefaultListAll(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.IsError)
	text := resultText(t, result)
	// Should contain all processes
	assert.Contains(t, text, "launchd")
	assert.Contains(t, text, "WindowServer")
	assert.Contains(t, text, "Finder")
	assert.Contains(t, text, "Safari")
	assert.Contains(t, text, "node")
	// Should contain header
	assert.Contains(t, text, "PID")
}

func TestHandleProcesses_FilterByName(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{"name": "Safari"})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.IsError)
	text := resultText(t, result)
	// Should contain Safari and its child
	assert.Contains(t, text, "Safari")
	assert.Contains(t, text, "Safari Web Content")
	// Should NOT contain unrelated processes
	assert.NotContains(t, text, "Finder")
	assert.NotContains(t, text, "node")
}

func TestHandleProcesses_FilterByNameCaseInsensitive(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{"name": "safari"})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.IsError)
	text := resultText(t, result)
	assert.Contains(t, text, "Safari")
}

func TestHandleProcesses_FilterByPID(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{"pid": float64(201)})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.IsError)
	text := resultText(t, result)
	// Should contain the specific process
	assert.Contains(t, text, "Safari")
	assert.Contains(t, text, "201")
	// Should also include children
	assert.Contains(t, text, "Safari Web Content")
}

func TestHandleProcesses_FilterNoMatch(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{"name": "nonexistent_app"})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.IsError)
	text := resultText(t, result)
	assert.Contains(t, text, "No processes found")
	assert.Contains(t, text, "nonexistent_app")
}

func TestHandleProcesses_OutputFormat(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	text := resultText(t, result)
	// Should have column headers
	assert.Contains(t, text, "PID")
	assert.Contains(t, text, "PPID")
	assert.Contains(t, text, "CPU%")
	assert.Contains(t, text, "MEM(MB)")
	assert.Contains(t, text, "USER")
	assert.Contains(t, text, "STATE")
	assert.Contains(t, text, "NAME")
}

func TestHandleProcesses_ShowsParentChild(t *testing.T) {
	setFakeProcesses(t)
	// Filter by Safari PID, should include child "Safari Web Content"
	req := mockRequest(map[string]any{"pid": float64(201)})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	text := resultText(t, result)
	assert.Contains(t, text, "201")
	assert.Contains(t, text, "202") // child PID
	assert.Contains(t, text, "Safari Web Content")
}

func TestHandleProcesses_ProcessCount(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	text := resultText(t, result)
	// Should show total count
	assert.Contains(t, text, "6 processes")
}

func TestHandleProcesses_FilteredCount(t *testing.T) {
	setFakeProcesses(t)
	req := mockRequest(map[string]any{"name": "Safari"})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err)
	text := resultText(t, result)
	// Should show filtered count
	assert.Contains(t, text, "2 processes")
}

// --- parsePSLine unit tests ---

func TestParsePSLine_ValidLine(t *testing.T) {
	p, ok := parsePSLine("  123   1 testuser  S  2.5  51200 Google Chrome")
	require.True(t, ok)
	assert.Equal(t, 123, p.PID)
	assert.Equal(t, 1, p.PPID)
	assert.Equal(t, "testuser", p.User)
	assert.Equal(t, "S", p.State)
	assert.Equal(t, 2.5, p.CPUPercent)
	assert.InDelta(t, 50.0, p.MemMB, 0.1) // 51200 KB / 1024
	assert.Equal(t, "Google Chrome", p.Name)
}

func TestParsePSLine_EmptyLine(t *testing.T) {
	_, ok := parsePSLine("")
	assert.False(t, ok)
}

func TestParsePSLine_WhitespaceOnly(t *testing.T) {
	_, ok := parsePSLine("   \t  ")
	assert.False(t, ok)
}

func TestParsePSLine_TooFewFields(t *testing.T) {
	_, ok := parsePSLine("123 1 root S 0.0 1024")
	assert.False(t, ok)
}

func TestParsePSLine_InvalidPID(t *testing.T) {
	_, ok := parsePSLine("abc 1 root S 0.0 1024 bash")
	assert.False(t, ok)
}

func TestParsePSLine_InvalidPPID(t *testing.T) {
	_, ok := parsePSLine("123 abc root S 0.0 1024 bash")
	assert.False(t, ok)
}

func TestParsePSLine_InvalidCPUDefaultsToZero(t *testing.T) {
	p, ok := parsePSLine("123 1 root S bad 1024 bash")
	require.True(t, ok)
	assert.Equal(t, 0.0, p.CPUPercent)
}

func TestParsePSLine_InvalidMemDefaultsToZero(t *testing.T) {
	p, ok := parsePSLine("123 1 root S 0.5 bad bash")
	require.True(t, ok)
	assert.Equal(t, 0.0, p.MemMB)
}

func TestParsePSLine_NameWithSpaces(t *testing.T) {
	p, ok := parsePSLine("500 201 testuser S 3.4 122880 Safari Web Content")
	require.True(t, ok)
	assert.Equal(t, "Safari Web Content", p.Name)
}

// --- filterProcesses unit tests ---

func TestFilterProcesses_NoFilters(t *testing.T) {
	procs, _ := fakeProcessLister()
	result := filterProcesses(procs, "", 0)
	assert.Equal(t, procs, result)
}

func TestFilterProcesses_NameCaseInsensitive(t *testing.T) {
	procs, _ := fakeProcessLister()
	result := filterProcesses(procs, "FINDER", 0)
	require.Len(t, result, 1)
	assert.Equal(t, "Finder", result[0].Name)
}

func TestFilterProcesses_PIDNotFound(t *testing.T) {
	procs, _ := fakeProcessLister()
	result := filterProcesses(procs, "", 9999)
	assert.Empty(t, result)
}

func TestFilterProcesses_PIDWithChildren(t *testing.T) {
	procs, _ := fakeProcessLister()
	// Safari (201) has child Safari Web Content (202)
	result := filterProcesses(procs, "", 201)
	require.Len(t, result, 2)
	assert.Equal(t, 201, result[0].PID)
	assert.Equal(t, 202, result[1].PID)
}

func TestFilterProcesses_PIDTakesPrecedenceOverName(t *testing.T) {
	procs, _ := fakeProcessLister()
	// When both provided, PID filter is used (pidFilter > 0 branch)
	result := filterProcesses(procs, "Finder", 201)
	// Should match PID 201 (Safari), not name "Finder"
	require.Len(t, result, 2)
	assert.Equal(t, "Safari", result[0].Name)
}

func TestFilterProcesses_IncludesDescendants(t *testing.T) {
	// Single pass picks up children, grandchildren when ordered by PID
	procs := []processInfo{
		{PID: 1, PPID: 0, Name: "root"},
		{PID: 10, PPID: 1, Name: "parent"},
		{PID: 20, PPID: 10, Name: "child"},
		{PID: 30, PPID: 20, Name: "grandchild"},
	}
	result := filterProcesses(procs, "", 10)
	require.Len(t, result, 3) // parent + child + grandchild
	assert.Equal(t, 10, result[0].PID)
	assert.Equal(t, 20, result[1].PID)
	assert.Equal(t, 30, result[2].PID)
}

func TestFilterProcesses_NameDoesNotIncludeChildrenByRelationship(t *testing.T) {
	procs, _ := fakeProcessLister()
	// Name "node" matches only the node process, not its children by PPID
	result := filterProcesses(procs, "node", 0)
	require.Len(t, result, 1)
	assert.Equal(t, "node", result[0].Name)
}

// --- HandleProcesses error path ---

func TestHandleProcesses_ListError(t *testing.T) {
	old := listProcessesFn
	listProcessesFn = func() ([]processInfo, error) {
		return nil, fmt.Errorf("connection refused")
	}
	t.Cleanup(func() { listProcessesFn = old })

	req := mockRequest(map[string]any{})
	result, err := HandleProcesses(context.Background(), req)

	assert.NoError(t, err) // tool errors come as result, not Go error
	assert.True(t, result.IsError)
	text := resultText(t, result)
	assert.Contains(t, text, "Failed to list processes")
	assert.Contains(t, text, "connection refused")
}
