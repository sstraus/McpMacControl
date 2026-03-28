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
					mcp.Description("actions, capture, shell, processes, examples, or action name"),
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
				mcp.WithDescription("Screenshot a window or region. For simple captures, prefer do([{type:screenshot, app:Name}]) to batch with actions."),
				mcp.WithString("app_name",
					mcp.Required(),
					mcp.Description("App name"),
				),
				mcp.WithString("window_title",
					mcp.Description("Title substring match"),
				),
				mcp.WithBoolean("hide_shadow",
					mcp.Description("Hide shadow (default: true)"),
				),
				mcp.WithString("format",
					mcp.Description("webp (default) or png"),
				),
				mcp.WithNumber("quality",
					mcp.Description("WebP quality 1-100 (default 25; use 50+ for small glyphs/icons)"),
				),
				mcp.WithNumber("region_x",
					mcp.Description("Region X offset"),
				),
				mcp.WithNumber("region_y",
					mcp.Description("Region Y offset"),
				),
				mcp.WithNumber("region_width",
					mcp.Description("Region width"),
				),
				mcp.WithNumber("region_height",
					mcp.Description("Region height"),
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
					mcp.Description("webp (default) or png"),
				),
				mcp.WithNumber("quality",
					mcp.Description("WebP quality 1-100 (default 25; use 50+ for small glyphs/icons)"),
				),
			),
			HandleCaptureScreen,
		},
		{
			mcp.NewTool("do",
				mcp.WithDescription("Execute actions: click, type, key, scroll, wait, screenshot, focus, minimize, restore, close, resize, drag, paste, clipboard, move. ALWAYS batch into one call. Use screenshot action instead of capture_window. NEVER use osascript — use do(focus) to activate apps, list_windows/processes to find names. Call help() first."),
				mcp.WithArray("actions",
					mcp.Required(),
					mcp.Description("Array of action objects"),
				),
			),
			HandleDo,
		},
		{
			mcp.NewTool("alert",
				mcp.WithDescription("Visual alert: active=true red flash, active=false green confirmation"),
				mcp.WithBoolean("active",
					mcp.Description("true=start red flash, false=stop with green"),
				),
			),
			HandleAlert,
		},
		{
			mcp.NewTool("processes",
				mcp.WithDescription("List running processes with PID, CPU, memory. Filter by name or PID."),
				mcp.WithString("name",
					mcp.Description("Filter by name (case-insensitive)"),
				),
				mcp.WithNumber("pid",
					mcp.Description("Filter by PID (includes children)"),
				),
			),
			HandleProcesses,
		},
		{
			mcp.NewTool("shell",
				mcp.WithDescription("PTY shell sessions. Actions: spawn, send_input, get_snapshot, resize, close, list. Call help('shell') first."),
				mcp.WithString("action",
					mcp.Required(),
					mcp.Description("spawn|send_input|get_snapshot|resize|close|list"),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID"),
				),
				mcp.WithString("command",
					mcp.Description("Command (default: /bin/bash)"),
				),
				mcp.WithString("cwd",
					mcp.Description("Working directory"),
				),
				mcp.WithString("input",
					mcp.Description("Text to send"),
				),
				mcp.WithString("special_key",
					mcp.Description("enter, tab, ctrl+c, up, down, etc."),
				),
				mcp.WithString("format",
					mcp.Description("text (default) or ansi"),
				),
				mcp.WithNumber("wait_ms",
					mcp.Description("Wait ms after input (default: 100)"),
				),
				mcp.WithNumber("cols",
					mcp.Description("Columns (default: 80)"),
				),
				mcp.WithNumber("rows",
					mcp.Description("Rows (default: 24)"),
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
