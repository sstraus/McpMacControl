# MCPMacControl

[![Release](https://img.shields.io/github/v/release/sstraus/mcpmaccontrol)](https://github.com/sstraus/mcpmaccontrol/releases/latest)
[![Build](https://img.shields.io/github/actions/workflow/status/sstraus/mcpmaccontrol/release.yml)](https://github.com/sstraus/mcpmaccontrol/actions)
[![Go](https://img.shields.io/github/go-mod/go-version/sstraus/mcpmaccontrol)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![macOS](https://img.shields.io/badge/macOS-12%2B-000000?logo=apple)](https://github.com/sstraus/mcpmaccontrol)

**Give your AI eyes and hands on your Mac.**

MCPMacControl is an [MCP server](https://modelcontextprotocol.io/) that turns any AI assistant into an **agentic computer user** — it can see your screen, move the mouse, type on the keyboard, and run shell commands across **any macOS application**, not just the browser.

Works with Claude Code, Cursor, Windsurf, and any MCP-compatible client.

No scripting. No AppleScript. Just tell the AI what you want in plain language and watch it drive your Mac.

### What can your AI do with it?

- **Test your app like a QA engineer** — click through flows, fill forms, verify visual state
- **Automate legacy software** that has no API — navigate menus, extract data from any window
- **Debug UI issues** by looking at screenshots and reproducing the problem step by step
- **Drive native apps** — Xcode, Figma, Finder, Mail, Excel — anything with a window
- **Run interactive terminal sessions** — full PTY with TUI support (vim, htop, docker logs)

> *"Take a screenshot of Safari, click the search bar, and type anthropic.com"*
>
> That's a real prompt. The AI does the rest.

## Install

### Download (recommended)

Download `MCPMacControl.app.zip` from the [latest release](https://github.com/sstraus/mcpmaccontrol/releases), unzip, and move to `/Applications/`.

Release builds are signed and notarized with an Apple Developer ID certificate, so macOS permissions persist across updates and no Gatekeeper warnings appear.

### Build from source

```bash
git clone https://github.com/sstraus/mcpmaccontrol
cd mcpmaccontrol
make build
make install
```

### Configure your MCP client

Add to your MCP client's config (e.g. `~/.claude.json` for Claude Code):

```json
{
  "mcpServers": {
    "mac-control": {
      "command": "/Applications/MCPMacControl.app/Contents/MacOS/mcpmaccontrol"
    }
  }
}
```

Then ask your AI: *"Take a screenshot of Safari and click the search box"*

## Why not Playwright / browser automation?

Browser automation tools only control browsers. MCPMacControl controls **everything on your Mac** — native apps, Electron apps, system dialogs, menu bars, the Dock. If it has pixels on screen, Claude can see it and interact with it.

It also includes full PTY shell sessions, so Claude can run `vim`, `docker`, `ssh`, or any interactive terminal tool — not just headless commands.

## Why an `.app` bundle?

Unlike most MCP servers, MCPMacControl needs macOS Screen Recording and Accessibility permissions. macOS grants these permissions per-application, not per-binary. When a plain binary runs from a terminal, macOS attributes the permission to the **terminal app** (iTerm, Terminal.app), not to the binary itself. This means:

- Permissions don't persist when the binary is rebuilt
- Every terminal app that launches the binary needs separate permission grants
- There's no way for users to manage the binary's permissions in System Settings

Wrapping the binary in a signed `.app` bundle gives it its own identity in macOS privacy settings. The binary still communicates via stdio like any MCP server - the `.app` bundle is just how macOS identifies it for permission management.

That's why the command path is `/Applications/MCPMacControl.app/Contents/MacOS/mcpmaccontrol` instead of `/usr/local/bin/mcpmaccontrol`.

## Permissions

On first launch, macOS will prompt for two permissions:

| Permission | Purpose |
|------------|---------|
| **Screen Recording** | Screenshot capture |
| **Accessibility** | Mouse/keyboard control |

Grant both in **System Settings > Privacy & Security**. The app registers itself automatically.

## Features

- **Visual feedback** — menu bar icon, native popover showing current operation, sound + orange border flash before automation
- **Input safety** — verifies the target app is focused before sending keystrokes, preventing input from reaching the wrong window
- **App context inheritance** — `focus` propagates to subsequent actions in a batch, so you write `app` once
- **Signed `.app` bundle** — macOS permissions persist across updates, no Gatekeeper warnings
- **Region capture** — screenshot only what you need to save tokens
- **Auto-exit** when the AI client disconnects

## Tools

| Tool | Description |
|------|-------------|
| `help` | Built-in documentation - call first! |
| `list_windows` | Find windows by app name |
| `capture_window` | Screenshot window or region with click coordinates |
| `capture_screen` | Screenshot entire screen |
| `do` | Execute actions: click, type, key, scroll, focus, minimize, etc. |
| `shell` | PTY shell sessions: spawn, send_input, get_snapshot, resize, close, list |
| `processes` | List running processes with filtering for debugging |

### Built-in Help

The AI calls `help()` to learn the API:
- `help()` → Overview and workflow
- `help("actions")` → All action types for `do()`
- `help("shell")` → Shell/PTY documentation
- `help("examples")` → Usage examples

## How it works

Claude operates in a see-think-act loop — the same way a human would use a computer:

```
1. capture_window("Safari")         → Screenshot saved to temp file
2. [Claude reads the image]         → "I see a search bar at (400, 50)"
3. do([                             → Click, type, press Enter
     {type:"click", app:"Safari", x:400, y:50},
     {type:"type", text:"anthropic.com"},
     {type:"key", key:"enter"}
   ])
4. capture_window("Safari")         → Verify the result
```

Pixel coordinates in the screenshot map directly to `do()` action coordinates — no conversion needed.

### Region capture

Capture a portion of a window to save tokens:

```
capture_window("Safari", region_x: 0, region_y: 0, region_width: 400, region_height: 300)
```

Click coordinates: `click_x = region_x + x_in_image`, `click_y = region_y + y_in_image`.

## The `do` Tool

Execute one or more actions in sequence:

```json
do({"actions": [
  {"type": "click", "app": "Safari", "x": 400, "y": 50},
  {"type": "type", "text": "anthropic.com"},
  {"type": "key", "key": "enter"}
]})
```

### Action Types

| Action | Required | Optional | Description |
|--------|----------|----------|-------------|
| `click` | app, x, y | button, double | Click at position |
| `move` | app, x, y | | Move mouse |
| `type` | text | | Type text string |
| `key` | key | modifiers | Press key (supports `"cmd+shift+g"` compound syntax) |
| `scroll` | app, x, y, delta_y/delta_x | | Scroll in window |
| `wait` | ms | | Pause (milliseconds) |
| `focus` | app | | Bring window to front |
| `minimize` | app | | Minimize to dock |
| `restore` | app | | Restore from dock |
| `close` | app | | Close window |
| `resize` | app, width, height | | Resize window |

### Examples

**Click:**
```json
{"type": "click", "app": "Safari", "x": 100, "y": 50}
{"type": "click", "app": "Finder", "x": 200, "y": 100, "button": "right"}
{"type": "click", "app": "Finder", "x": 200, "y": 100, "double": true}
```

**Keyboard:**
```json
{"type": "type", "text": "Hello World"}
{"type": "key", "key": "enter"}
{"type": "key", "key": "cmd+v"}
{"type": "key", "key": "cmd+shift+g"}
{"type": "key", "key": "v", "modifiers": ["cmd"]}
```

**Scroll:**
```json
{"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": -100}
```

**Window Control:**
```json
{"type": "focus", "app": "Safari"}
{"type": "minimize", "app": "Finder"}
{"type": "restore", "app": "Finder"}
{"type": "resize", "app": "Safari", "width": 1024, "height": 768}
{"type": "close", "app": "TextEdit"}
```

## Shell Sessions

Run interactive terminal sessions with full PTY and terminal emulation (supports TUI apps like vim, htop):

```
1. shell(action: "spawn")                                          → Get session ID
2. shell(action: "send_input", session_id: "ID", input: "ls")     → Send text
3. shell(action: "send_input", session_id: "ID", special_key: "enter")
4. shell(action: "get_snapshot", session_id: "ID", format: "ansi") → Get screen
5. shell(action: "close", session_id: "ID")                        → Cleanup
```

| Action | Parameters | Description |
|--------|------------|-------------|
| `spawn` | command?, cwd?, cols?, rows? | Start session (default: /bin/bash) |
| `send_input` | session_id, input?, special_key?, wait_ms? | Send text or keys |
| `get_snapshot` | session_id, format? | Get screen state (text or ansi) |
| `resize` | session_id, cols?, rows? | Resize terminal |
| `list` | | List active sessions |
| `close` | session_id | Close session |

## Headless Mode

Run without menu bar (for CI/SSH):

```json
{
  "mcpServers": {
    "mac-control": {
      "command": "/Applications/MCPMacControl.app/Contents/MacOS/mcpmaccontrol",
      "env": {
        "MCPMACCONTROL_HEADLESS": "1"
      }
    }
  }
}
```

## Image Optimization

Screenshots are optimized for AI vision:
- WebP lossy format (default quality 25, tunable 1-100 via `quality` parameter; use 50+ for small glyphs/icons)
- Scaled to window point dimensions (coordinates match clicks)
- Use **region capture** to further reduce size and tokens

## Development

```bash
make build          # Build signed .app bundle
make test           # Run tests
make install        # Install to /Applications
make clean          # Remove build artifacts
make verify-sign    # Check code signature
```

To sign with your own Apple Developer certificate:

```bash
SIGN_IDENTITY="Developer ID Application: YOUR NAME (TEAMID)" make build
```

## Requirements

- macOS 12+
- Go 1.24+ (for building)

## License

MIT
