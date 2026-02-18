package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const helpOverview = `MCPMacControl - Mouse, Keyboard, Screenshot, Window & Shell Control for macOS

TOOLS:
• help           → This documentation
• list_windows   → Find windows by app name
• capture_window → Screenshot + coordinates for clicking
• capture_screen → Full screen screenshot
• do             → Execute actions (click, type, paste, key, drag, scroll, wait, focus, etc.)
• shell          → PTY shell sessions with terminal emulation (spawn, send_input, get_snapshot, resize, close, list)
• processes       → List running processes with filtering for debugging

WORKFLOW:
1. list_windows("Safari")        → Find window
2. capture_window("Safari")      → Get screenshot + bounds
3. Analyze image, find the CENTER of the target element
4. do([{type:"click", app:"Safari", x:100, y:50}])

IMPORTANT: Always aim for the CENTER of buttons, links, and UI elements.
Clicking near edges often misses. Estimate the element's bounding box
and use its midpoint, not its border.

Call help("actions") for full action reference.
Call help("shell") for shell documentation.
Call help("processes") for process listing.
Call help("examples") for usage examples.`

const helpActions = `DO TOOL - ACTION REFERENCE

Execute one or more actions in sequence:
do({"actions": [...]})

MOUSE ACTIONS:
┌─────────────────────────────────────────────────────────────┐
│ click - Click at position in window                         │
│   {type:"click", app:"AppName", x:100, y:50}               │
│   Optional: button:"right"|"middle", double:true            │
├─────────────────────────────────────────────────────────────┤
│ move - Move mouse to position in window                     │
│   {type:"move", app:"AppName", x:100, y:50}                │
├─────────────────────────────────────────────────────────────┤
│ drag - Drag from one position to another in window          │
│   {type:"drag", app:"Finder", x:100, y:100,                │
│    to_x:300, to_y:300}                                      │
├─────────────────────────────────────────────────────────────┤
│ scroll - Scroll at position in window                       │
│   {type:"scroll", app:"AppName", x:100, y:50, delta_y:-50} │
│   delta_y: negative=up, positive=down                       │
│   delta_x: negative=left, positive=right                    │
└─────────────────────────────────────────────────────────────┘

KEYBOARD ACTIONS:
┌─────────────────────────────────────────────────────────────┐
│ type - Type text string (char by char)                      │
│   {type:"type", text:"Hello World"}                        │
├─────────────────────────────────────────────────────────────┤
│ paste - Paste text via clipboard (⌘V)                       │
│   {type:"paste", text:"/path/to/file"}                     │
│   Use instead of "type" when field has autocomplete         │
├─────────────────────────────────────────────────────────────┤
│ clipboard - Read current clipboard contents                 │
│   {type:"clipboard"}                                        │
├─────────────────────────────────────────────────────────────┤
│ key - Press key with optional modifiers                     │
│   {type:"key", key:"enter"}                                │
│   {type:"key", key:"cmd+v"}                    → ⌘V       │
│   {type:"key", key:"cmd+shift+g"}              → ⌘⇧G      │
│   {type:"key", key:"v", modifiers:["cmd"]}     → ⌘V       │
│                                                             │
│   Keys: a-z, 0-9, tab, enter, escape, delete, space,       │
│         left, right, up, down, f1-f12                       │
│   Modifiers: cmd, shift, alt, ctrl                          │
└─────────────────────────────────────────────────────────────┘

UTILITY ACTIONS:
┌─────────────────────────────────────────────────────────────┐
│ wait - Pause between actions                                │
│   {type:"wait", ms:500}  → Wait 500 milliseconds           │
└─────────────────────────────────────────────────────────────┘

WINDOW ACTIONS:
┌─────────────────────────────────────────────────────────────┐
│ focus - Bring window to front                               │
│   {type:"focus", app:"Safari"}                             │
├─────────────────────────────────────────────────────────────┤
│ minimize - Minimize window to dock                          │
│   {type:"minimize", app:"Safari"}                          │
├─────────────────────────────────────────────────────────────┤
│ restore - Restore minimized window                          │
│   {type:"restore", app:"Safari"}                           │
├─────────────────────────────────────────────────────────────┤
│ close - Close window                                        │
│   {type:"close", app:"Safari"}                             │
├─────────────────────────────────────────────────────────────┤
│ resize - Resize window                                      │
│   {type:"resize", app:"Safari", width:800, height:600}     │
└─────────────────────────────────────────────────────────────┘`

const helpExamples = `EXAMPLES

1. SINGLE CLICK:
   do({"actions": [
     {"type": "click", "app": "Safari", "x": 100, "y": 50}
   ]})

2. TYPE AND SUBMIT:
   do({"actions": [
     {"type": "click", "app": "Safari", "x": 300, "y": 100},
     {"type": "type", "text": "hello@example.com"},
     {"type": "key", "key": "tab"},
     {"type": "type", "text": "mypassword"},
     {"type": "key", "key": "enter"}
   ]})

3. KEYBOARD SHORTCUTS:
   do({"actions": [
     {"type": "key", "key": "a", "modifiers": ["cmd"]},
     {"type": "key", "key": "c", "modifiers": ["cmd"]},
     {"type": "click", "app": "Notes", "x": 100, "y": 100},
     {"type": "key", "key": "v", "modifiers": ["cmd"]}
   ]})

4. SCROLL DOWN:
   do({"actions": [
     {"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": 100}
   ]})

5. RIGHT-CLICK (CONTEXT MENU):
   do({"actions": [
     {"type": "click", "app": "Finder", "x": 200, "y": 150, "button": "right"}
   ]})

6. DOUBLE-CLICK:
   do({"actions": [
     {"type": "click", "app": "Finder", "x": 200, "y": 150, "double": true}
   ]})

7. WAIT BETWEEN ACTIONS:
   do({"actions": [
     {"type": "click", "app": "Safari", "x": 100, "y": 50},
     {"type": "wait", "ms": 1000},
     {"type": "type", "text": "delayed typing"}
   ]})`

const helpCapture = `CAPTURE TOOLS

list_windows(app_filter?)
  List visible windows. Returns: [ID] AppName - Title
  Example: list_windows("Safari")
  Example: list_windows()  → all windows

capture_window(app_name, ...)
  Screenshot a window or specific region. Returns image + bounds.

  Parameters:
  • app_name: Required. App name (case-insensitive)
  • window_title: Optional. Title substring to match
  • format: "webp" (default, ~5x smaller) or "png" (only if client cannot decode webp)

  Region capture (optional - for smaller images):
  • region_x: X offset from window top-left
  • region_y: Y offset from window top-left
  • region_width: Width of region to capture
  • region_height: Height of region to capture

  Examples:
    capture_window("Safari")
      → Full window screenshot

    capture_window("Safari", region_x: 0, region_y: 0, region_width: 400, region_height: 300)
      → Only top-left 400x300 area
      → Smaller image, faster transfer
      → Click coords = region_x + x_in_image, region_y + y_in_image

capture_screen(display?, format?)
  Full screen capture.
  • display: 0=primary, 1=secondary, etc.`

const helpShell = `SHELL TOOL - PTY Session Management with Terminal Emulation

Start interactive shell sessions with full terminal emulation for TUI apps (vim, htop, etc.).
All operations use the single "shell" tool with an "action" parameter.

WORKFLOW:
1. shell({"action": "spawn", "command": "/bin/bash", "cols": 120, "rows": 30}) → Create session
2. shell({"action": "send_input", "session_id": "ID", "input": "ls"})          → Send text
3. shell({"action": "send_input", "session_id": "ID", "special_key": "enter"}) → Send special key
4. shell({"action": "get_snapshot", "session_id": "ID", "format": "ansi"})     → Get screen state
5. shell({"action": "close", "session_id": "ID"})                              → Close when done

ACTIONS:

spawn
  Start new PTY session with terminal emulation.
  • command: "/bin/bash" (default), "/bin/zsh", "vim", etc.
  • args: Command arguments (optional array)
  • cwd: Working directory (optional)
  • cols: Terminal columns (default: 80)
  • rows: Terminal rows (default: 24)
  • Returns: Session ID, PID, initial snapshot

send_input
  Send text input or special keys to session.
  • session_id: Required. ID from spawn.
  • input: Text to send (optional)
  • special_key: Special key name (optional)
  • wait_ms: Wait after input before snapshot (default: 100)

  Special keys: enter, tab, escape, backspace, delete, up, down, left, right,
                home, end, pageup, pagedown, f1-f12, ctrl+c, ctrl+d, ctrl+z,
                ctrl+l, shift+tab

get_snapshot
  Get current terminal screen state.
  • session_id: Required. ID from spawn.
  • format: "text" (plain, default) or "ansi" (with colors/formatting)
  • Returns: Current screen content (non-destructive)

resize
  Change terminal size.
  • session_id: Required. ID from spawn.
  • cols: Number of columns (default: 80)
  • rows: Number of rows (default: 24)

list
  List all active sessions with info. No parameters needed.

close
  Close session and cleanup.
  • session_id: Required. ID from spawn.
  • Returns: Exit code if available

EXAMPLES:

Basic shell:
  1. shell({"action": "spawn"})
  2. shell({"action": "send_input", "session_id": "abc", "input": "pwd", "special_key": "enter"})
  3. shell({"action": "get_snapshot", "session_id": "abc"})
  4. shell({"action": "close", "session_id": "abc"})

TUI app (vim):
  1. shell({"action": "spawn", "command": "vim", "args": ["file.txt"], "cols": 120, "rows": 40})
  2. shell({"action": "send_input", "session_id": "xyz", "special_key": "i"})  → Insert mode
  3. shell({"action": "send_input", "session_id": "xyz", "input": "Hello"})
  4. shell({"action": "send_input", "session_id": "xyz", "special_key": "escape"})
  5. shell({"action": "get_snapshot", "session_id": "xyz", "format": "ansi"})  → See vim UI
  6. shell({"action": "send_input", "session_id": "xyz", "input": ":wq", "special_key": "enter"})
  7. shell({"action": "close", "session_id": "xyz"})`

const helpProcesses = `PROCESSES TOOL - List and Filter Running Processes

List running processes with resource usage for debugging.
No permissions required.

PARAMETERS:
  • name: Filter by process name (case-insensitive substring match)
  • pid:  Filter by PID (includes child processes)

OUTPUT: Table with PID, PPID, CPU%, MEM(MB), USER, STATE, NAME

EXAMPLES:
  processes()                      → List all processes
  processes({"name": "Safari"})    → Find Safari and related processes
  processes({"name": "node"})      → Find all Node.js processes
  processes({"pid": 1234})         → Show process 1234 and its children

USE CASES:
  • Find hanging processes by name
  • Check resource usage (CPU/memory)
  • Identify parent-child process trees
  • Debug process state during failures`

var actionHelp = map[string]string{
	"click": `CLICK ACTION

Click at position relative to window top-left.
IMPORTANT: Always target the CENTER of the element, not its edge.
Estimate the element's bounding box and use its midpoint.

Format:
  {"type": "click", "app": "AppName", "x": 100, "y": 50}

Required:
  • type: "click"
  • app: Application name (case-insensitive)
  • x: X coordinate from window top-left (center of target element)
  • y: Y coordinate from window top-left (center of target element)

Optional:
  • button: "left" (default), "right", "middle"
  • double: true for double-click

Examples:
  {"type": "click", "app": "Safari", "x": 100, "y": 50}
  {"type": "click", "app": "Finder", "x": 200, "y": 100, "button": "right"}
  {"type": "click", "app": "Finder", "x": 200, "y": 100, "double": true}`,

	"move": `MOVE ACTION

Move mouse cursor to position relative to window.

Format:
  {"type": "move", "app": "AppName", "x": 100, "y": 50}

Required:
  • type: "move"
  • app: Application name
  • x: X coordinate from window top-left
  • y: Y coordinate from window top-left

Example:
  {"type": "move", "app": "Safari", "x": 300, "y": 200}`,

	"type": `TYPE ACTION

Type a text string. Works in the currently focused input.

Format:
  {"type": "type", "text": "Hello World"}

Required:
  • type: "type"
  • text: String to type (can include spaces, punctuation)

Example:
  {"type": "type", "text": "user@example.com"}
  {"type": "type", "text": "Hello, World!"}`,

	"key": `KEY ACTION

Press a key with optional modifiers.

Format:
  {"type": "key", "key": "enter"}
  {"type": "key", "key": "v", "modifiers": ["cmd"]}

Required:
  • type: "key"
  • key: Key name (see list below)

Optional:
  • modifiers: Array of modifier names

Keys:
  Letters: a, b, c, ... z
  Numbers: 0, 1, 2, ... 9
  Special: tab, enter, escape, delete, space, backspace
  Arrows: left, right, up, down
  Function: f1, f2, ... f12

Modifiers:
  cmd (⌘), shift (⇧), alt/option (⌥), ctrl (⌃)

Examples:
  {"type": "key", "key": "enter"}           → Enter
  {"type": "key", "key": "tab"}             → Tab
  {"type": "key", "key": "tab", "modifiers": ["shift"]}  → Shift+Tab
  {"type": "key", "key": "c", "modifiers": ["cmd"]}      → ⌘C (copy)
  {"type": "key", "key": "v", "modifiers": ["cmd"]}      → ⌘V (paste)
  {"type": "key", "key": "1", "modifiers": ["cmd"]}      → ⌘1
  {"type": "key", "key": "a", "modifiers": ["cmd"]}      → ⌘A (select all)
  {"type": "key", "key": "z", "modifiers": ["cmd", "shift"]}  → ⌘⇧Z (redo)`,

	"paste": `PASTE ACTION

Paste text via the clipboard and ⌘V. Use instead of "type" when the target
field has autocomplete or interferes with char-by-char typing (e.g. Finder
Go to Folder dialog).

Format:
  {"type": "paste", "text": "/path/to/file"}

Required:
  • type: "paste"
  • text: String to paste

Examples:
  {"type": "paste", "text": "/Users/me/Documents"}
  {"type": "paste", "text": "https://long-url.example.com/path"}

Note: Overwrites the current clipboard contents.`,

	"clipboard": `CLIPBOARD ACTION

Read the current clipboard (pasteboard) contents.

Format:
  {"type": "clipboard"}

Required:
  • type: "clipboard"

Returns the text currently on the macOS general pasteboard.
Useful for reading text that was copied via ⌘C.`,

	"drag": `DRAG ACTION

Drag from one position to another within a window.

Format:
  {"type": "drag", "app": "Finder", "x": 100, "y": 100, "to_x": 300, "to_y": 300}

Required:
  • type: "drag"
  • app: Application name (case-insensitive)
  • x, y: Start position (relative to window top-left)
  • to_x, to_y: End position (relative to window top-left)

Example:
  {"type": "drag", "app": "Finder", "x": 50, "y": 100, "to_x": 50, "to_y": 300}

Performs: mouseDown at (x,y) → mouseDrag to (to_x,to_y) → mouseUp.`,

	"scroll": `SCROLL ACTION

Scroll at a position in window.

Format:
  {"type": "scroll", "app": "AppName", "x": 100, "y": 100, "delta_y": -50}

Required:
  • type: "scroll"
  • app: Application name
  • x, y: Position to scroll at (mouse moves here first)

Optional (at least one):
  • delta_y: Vertical scroll amount
    - Negative = scroll UP (content moves down)
    - Positive = scroll DOWN (content moves up)
  • delta_x: Horizontal scroll amount
    - Negative = scroll LEFT
    - Positive = scroll RIGHT

Examples:
  {"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": -100}  → Scroll up
  {"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": 100}   → Scroll down
  {"type": "scroll", "app": "Finder", "x": 200, "y": 200, "delta_x": 50}    → Scroll right`,

	"wait": `WAIT ACTION

Pause execution between actions.

Format:
  {"type": "wait", "ms": 500}

Required:
  • type: "wait"
  • ms: Milliseconds to wait (positive integer)

Example:
  {"type": "wait", "ms": 1000}  → Wait 1 second

Use for:
  • Waiting for UI to update after click
  • Waiting for page load
  • Timing-sensitive operations`,

	"focus": `FOCUS ACTION

Bring a window to the front and activate it.

Format:
  {"type": "focus", "app": "Safari"}

Required:
  • type: "focus"
  • app: Application name (case-insensitive)

Example:
  {"type": "focus", "app": "Finder"}

Note: Also activates the application (makes it frontmost).`,

	"minimize": `MINIMIZE ACTION

Minimize a window to the dock.

Format:
  {"type": "minimize", "app": "Safari"}

Required:
  • type: "minimize"
  • app: Application name (case-insensitive)

Example:
  {"type": "minimize", "app": "Safari"}

Use restore to bring the window back.`,

	"restore": `RESTORE ACTION

Restore a minimized window from the dock.

Format:
  {"type": "restore", "app": "Safari"}

Required:
  • type: "restore"
  • app: Application name (case-insensitive)

Example:
  {"type": "restore", "app": "Safari"}`,

	"close": `CLOSE ACTION

Close a window.

Format:
  {"type": "close", "app": "Safari"}

Required:
  • type: "close"
  • app: Application name (case-insensitive)

Example:
  {"type": "close", "app": "TextEdit"}

Warning: This will close the window. Unsaved work may be lost.`,

	"resize": `RESIZE ACTION

Change the size of a window.

Format:
  {"type": "resize", "app": "Safari", "width": 800, "height": 600}

Required:
  • type: "resize"
  • app: Application name (case-insensitive)
  • width: New window width in pixels
  • height: New window height in pixels

Example:
  {"type": "resize", "app": "Finder", "width": 1024, "height": 768}`,
}

// HandleHelp handles the help tool call.
func HandleHelp(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	topic := strings.ToLower(request.GetString("topic", ""))

	switch topic {
	case "":
		return mcp.NewToolResultText(helpOverview), nil
	case "actions", "action", "do":
		return mcp.NewToolResultText(helpActions), nil
	case "examples", "example":
		return mcp.NewToolResultText(helpExamples), nil
	case "capture", "screenshot":
		return mcp.NewToolResultText(helpCapture), nil
	case "shell", "pty", "terminal":
		return mcp.NewToolResultText(helpShell), nil
	case "processes", "process", "ps":
		return mcp.NewToolResultText(helpProcesses), nil
	default:
		// Check for specific action help
		if help, ok := actionHelp[topic]; ok {
			return mcp.NewToolResultText(help), nil
		}
		// Suggest valid topics
		return mcp.NewToolResultText(`Unknown topic: "` + topic + `"

Valid topics:
  help()              → Overview
  help("actions")     → All action types for do()
  help("examples")    → Usage examples
  help("capture")     → Screenshot tools
  help("shell")       → PTY shell tools
  help("processes")   → Process listing

Action-specific help:
  help("click")       → Click action
  help("move")        → Move action
  help("drag")        → Drag action
  help("type")        → Type action
  help("paste")       → Paste via clipboard
  help("clipboard")   → Read clipboard
  help("key")         → Key action
  help("scroll")      → Scroll action
  help("wait")        → Wait action
  help("focus")       → Focus window
  help("minimize")    → Minimize window
  help("restore")     → Restore window
  help("close")       → Close window
  help("resize")      → Resize window`), nil
	}
}
