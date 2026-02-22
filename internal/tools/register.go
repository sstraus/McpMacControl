package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// toolDef pairs a tool definition with its handler.
type toolDef struct {
	tool    mcp.Tool
	handler server.ToolHandlerFunc
}

// allTools returns all active tool definitions paired with their handlers.
func allTools() []toolDef {
	return []toolDef{
		{
			mcp.NewTool("help",
				mcp.WithDescription("macOS control: mouse, keyboard, screenshots, windows. Call for usage."),
				mcp.WithString("topic",
					mcp.Description("actions, capture, examples, or action name"),
				),
			),
			HandleHelp,
		},
		{
			mcp.NewTool("list_windows",
				mcp.WithDescription("List visible windows"),
				mcp.WithString("app_filter",
					mcp.Description("Filter by app name"),
				),
			),
			HandleListWindows,
		},
		{
			mcp.NewTool("capture_window",
				mcp.WithDescription("Screenshot a window or a REGION within it. Use region_* params to capture only a portion (e.g., just a toolbar, sidebar, or dialog) - saves tokens. Returns image + click coordinates."),
				mcp.WithString("app_name",
					mcp.Required(),
					mcp.Description("App name (e.g., 'Safari', 'Finder')"),
				),
				mcp.WithString("window_title",
					mcp.Description("Window title substring to match specific window"),
				),
				mcp.WithBoolean("hide_shadow",
					mcp.Description("Hide window shadow (default: true)"),
				),
				mcp.WithString("format",
					mcp.Description("Image format: 'webp' (default, ~5x smaller) or 'png' (only if client cannot decode webp)"),
				),
				mcp.WithNumber("region_x",
					mcp.Description("Capture region: X offset from window top-left"),
				),
				mcp.WithNumber("region_y",
					mcp.Description("Capture region: Y offset from window top-left"),
				),
				mcp.WithNumber("region_width",
					mcp.Description("Capture region: width in pixels"),
				),
				mcp.WithNumber("region_height",
					mcp.Description("Capture region: height in pixels"),
				),
			),
			HandleCaptureWindow,
		},
		{
			mcp.NewTool("capture_screen",
				mcp.WithDescription("Screenshot entire screen"),
				mcp.WithNumber("display",
					mcp.Description("Display number (0=primary)"),
				),
				mcp.WithString("format",
					mcp.Description("Image format: 'webp' (default, ~5x smaller) or 'png' (only if client cannot decode webp)"),
				),
			),
			HandleCaptureScreen,
		},
		{
			mcp.NewTool("do",
				mcp.WithDescription("Execute actions: click, type, key, scroll, wait, screenshot. Call help() first."),
				mcp.WithArray("actions",
					mcp.Required(),
					mcp.Description("Array of actions to execute in sequence"),
				),
			),
			HandleDo,
		},
		{
			mcp.NewTool("alert",
				mcp.WithDescription("Visual alert overlay. Red flash when active=true (needs user attention), green flash when active=false (done)."),
				mcp.WithBoolean("active",
					mcp.Description("true = start red flashing border, false = stop and show green confirmation"),
				),
			),
			HandleAlert,
		},
		{
			mcp.NewTool("processes",
				mcp.WithDescription("List running processes with PID, CPU, memory, state. Filter by name or PID. Includes child processes."),
				mcp.WithString("name",
					mcp.Description("Filter by process name (case-insensitive substring match)"),
				),
				mcp.WithNumber("pid",
					mcp.Description("Filter by PID (includes child processes)"),
				),
			),
			HandleProcesses,
		},
		{
			mcp.NewTool("shell",
				mcp.WithDescription("PTY shell sessions with terminal emulation. Actions: spawn, send_input, get_snapshot, resize, close, list. Call help('shell') for usage."),
				mcp.WithString("action",
					mcp.Required(),
					mcp.Description("Action: spawn, send_input, get_snapshot, resize, close, list"),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID (required for send_input, get_snapshot, resize, close)"),
				),
				mcp.WithString("command",
					mcp.Description("Command to run (default: /bin/bash)"),
				),
				mcp.WithString("cwd",
					mcp.Description("Working directory"),
				),
				mcp.WithString("input",
					mcp.Description("Text to send (for send_input)"),
				),
				mcp.WithString("special_key",
					mcp.Description("Special key: enter, tab, ctrl+c, up, down, etc. (for send_input)"),
				),
				mcp.WithString("format",
					mcp.Description("Snapshot format: text (default), ansi (for get_snapshot)"),
				),
				mcp.WithNumber("wait_ms",
					mcp.Description("Wait ms after input before snapshot (default: 100)"),
				),
				mcp.WithNumber("cols",
					mcp.Description("Terminal columns (default: 80)"),
				),
				mcp.WithNumber("rows",
					mcp.Description("Terminal rows (default: 24)"),
				),
			),
			HandleShell,
		},
	}
}

// Definitions returns just the tool schemas without handlers.
// Used by the bridge to register tools with proxy handlers.
func Definitions() []mcp.Tool {
	defs := allTools()
	tools := make([]mcp.Tool, len(defs))
	for i, d := range defs {
		tools[i] = d.tool
	}
	return tools
}

// Register adds all Mac control tools with their real handlers to the MCP server.
func Register(s *server.MCPServer) {
	for _, td := range allTools() {
		s.AddTool(td.tool, td.handler)
	}
}
