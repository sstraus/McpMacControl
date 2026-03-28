package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const helpOverview = `macOS control via MCP — mouse, keyboard, screenshots, windows, shell, processes

NEVER use osascript or AppleScript. These tools replace it entirely.
Use list_windows() or processes() to discover correct app names first.

WORKFLOW:
1. list_windows("Safari") or capture_window("Safari") to find/see window
2. do([{type:"click",app:"Safari",x:100,y:50}]) — target element CENTER

BATCH actions in ONE do() call. Each call = 1 round-trip.
  BAD:  do([click]) → capture_window → do([click]) → capture_window  (4 calls)
  GOOD: do([click, wait, screenshot, click, wait, screenshot])       (1 call)

Use do(screenshot) for inline captures. Use capture_window only for region/title params.

help("actions") = action reference, help("shell") = PTY docs, help("examples") = examples`

const helpActions = `do({actions:[...]}) — execute actions in sequence

MOUSE: click, move, drag, scroll
  {type:"click", app:"App", x:N, y:N}  — window-relative coords
  {type:"click", x:N, y:N}             — absolute screen coords
  Optional: button:"right"|"middle", double:true
  {type:"move", app:"App", x:N, y:N}
  {type:"drag", app:"App", x:N, y:N, to_x:N, to_y:N}
  {type:"scroll", app:"App", x:N, y:N, delta_y:N}  — negative=up, positive=down

KEYBOARD: type, paste, clipboard, key  (app REQUIRED — prevents input to wrong window)
  {type:"type", app:"App", text:"hello"}
  {type:"paste", app:"App", text:"/path"}  — uses clipboard+⌘V, for autocomplete fields
  {type:"clipboard"}                       — read clipboard (no app needed)
  {type:"key", app:"App", key:"enter"}
  {type:"key", app:"App", key:"v", modifiers:["cmd"]}
  Keys: a-z, 0-9, tab, enter, escape, delete, space, backspace, arrows, f1-f12
  Modifiers: cmd, shift, alt, ctrl
  Shorthand: {type:"key", app:"App", key:"cmd+shift+g"}

UTILITY: wait, screenshot
  {type:"wait", ms:500}
  {type:"screenshot", app:"Safari"}     — inline window capture
  {type:"screenshot"}                   — inline full screen
  Optional: format:"webp"|"png", quality:1-100 (default 25; use 50+ for small glyphs/icons)

WINDOW: focus, minimize, restore, close, resize
  {type:"focus", app:"App"}
  {type:"minimize", app:"App"}
  {type:"restore", app:"App"}
  {type:"close", app:"App"}
  {type:"resize", app:"App", width:800, height:600}`

const helpExamples = `EXAMPLES — always batch into ONE do() call

Type + submit:
  do([{type:"click",app:"Safari",x:300,y:100},{type:"type",app:"Safari",text:"user@ex.com"},{type:"key",app:"Safari",key:"tab"},{type:"type",app:"Safari",text:"pass"},{type:"key",app:"Safari",key:"enter"}])

Copy-paste between apps:
  do([{type:"key",app:"Safari",key:"a",modifiers:["cmd"]},{type:"key",app:"Safari",key:"c",modifiers:["cmd"]},{type:"click",app:"Notes",x:100,y:100},{type:"key",app:"Notes",key:"v",modifiers:["cmd"]}])

Click + verify (visual checkpoint):
  do([{type:"click",app:"Safari",x:100,y:50},{type:"wait",ms:1000},{type:"screenshot",app:"Safari"},{type:"click",app:"Safari",x:200,y:100},{type:"wait",ms:1000},{type:"screenshot",app:"Safari"}])

Scroll:
  do([{type:"scroll",app:"Safari",x:400,y:300,delta_y:100}])

Right-click / double-click:
  do([{type:"click",app:"Finder",x:200,y:150,button:"right"}])
  do([{type:"click",app:"Finder",x:200,y:150,double:true}])`

const helpCapture = `list_windows(app_filter?) — returns [ID] AppName - Title
capture_window(app_name, window_title?, format?, region_x/y/width/height?) — screenshot + coords
capture_screen(display?, format?) — full screen

Region capture: set region_x/y/width/height to capture a portion.
Click coords for region = region_x + pixel_x, region_y + pixel_y.

Prefer do([{type:"screenshot",app:"App"}]) when no region/title matching needed.`

const helpShell = `shell({action:"spawn|send_input|get_snapshot|resize|close|list", ...})

spawn: command?, args?, cwd?, cols?, rows? → session_id + initial snapshot
send_input: session_id, input?, special_key?, wait_ms? → snapshot after input
get_snapshot: session_id, format? (text|ansi) → current screen
resize: session_id, cols?, rows?
close: session_id → exit code
list: → all sessions

Special keys: enter, tab, escape, backspace, delete, up, down, left, right, home, end, pageup, pagedown, f1-f12, ctrl+c, ctrl+d, ctrl+z, ctrl+l, shift+tab

Example:
  shell({action:"spawn"}) → ID
  shell({action:"send_input",session_id:"ID",input:"ls",special_key:"enter"})
  shell({action:"get_snapshot",session_id:"ID",format:"ansi"})
  shell({action:"close",session_id:"ID"})`

const helpProcesses = `processes(name?, pid?) — PID, PPID, CPU%, MEM(MB), USER, STATE, NAME
  name: case-insensitive substring filter
  pid: filter by PID (includes children)`

var actionHelp = map[string]string{
	"click": `{type:"click", app:"App", x:N, y:N}
{type:"click", x:N, y:N}  — absolute screen coords (no app)
Optional: button:"right"|"middle", double:true
Always target element CENTER, not edge.`,

	"move": `{type:"move", app:"App", x:N, y:N}
{type:"move", x:N, y:N}  — absolute screen coords`,

	"type": `{type:"type", app:"App", text:"Hello"}
Types char by char. App required — ensures target window has focus.`,

	"key": `{type:"key", app:"App", key:"enter"}
{type:"key", app:"App", key:"v", modifiers:["cmd"]}
{type:"key", app:"App", key:"cmd+shift+g"}  — shorthand
Keys: a-z, 0-9, tab, enter, escape, delete, space, backspace, arrows, f1-f12
Modifiers: cmd, shift, alt, ctrl
App required — ensures target window has focus.`,

	"paste": `{type:"paste", app:"App", text:"/path/to/file"}
Pastes via clipboard+⌘V. Use instead of type when field has autocomplete.
App required — ensures target window has focus. Overwrites clipboard.`,

	"clipboard": `{type:"clipboard"}
Returns current clipboard text.`,

	"drag": `{type:"drag", app:"App", x:N, y:N, to_x:N, to_y:N}
mouseDown→drag→mouseUp. Coords relative to window.`,

	"scroll": `{type:"scroll", app:"App", x:N, y:N, delta_y:N}
delta_y: negative=up, positive=down. delta_x: negative=left, positive=right.`,

	"wait": `{type:"wait", ms:500}
Pause between actions. Use for UI updates, page loads.`,

	"screenshot": `{type:"screenshot", app:"Safari"} — window capture
{type:"screenshot"} — full screen
Optional: format:"webp"|"png", quality:1-100 (default 25; use 50+ for small glyphs/icons)
Use for visual checkpoints within do() batches.`,

	"focus":    `{type:"focus", app:"App"} — activate app and bring window to front (replaces osascript activate)`,
	"minimize": `{type:"minimize", app:"App"} — minimize to dock`,
	"restore":  `{type:"restore", app:"App"} — restore from dock`,
	"close":    `{type:"close", app:"App"} — close window`,
	"resize":   `{type:"resize", app:"App", width:N, height:N}`,
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
		if help, ok := actionHelp[topic]; ok {
			return mcp.NewToolResultText(help), nil
		}
		return mcp.NewToolResultText(`Unknown topic. Valid: actions, examples, capture, shell, processes, click, move, type, key, paste, clipboard, drag, scroll, wait, screenshot, focus, minimize, restore, close, resize`), nil
	}
}
