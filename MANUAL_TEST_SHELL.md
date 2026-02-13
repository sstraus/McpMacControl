# Manual Testing Guide: Shell Tool with Terminal Emulation

## Prerequisites

1. Build complete: `make build`
2. MCP server configured in Claude Code settings
3. Claude Code session active with MCP connection

## Test Cases

### 1. Basic Shell Session

```javascript
// Spawn a shell
shell({"action": "spawn"})
// Note the session ID from response

// Send a command
shell({"action": "send_input", "session_id": "SESSION_ID", "input": "echo hello", "special_key": "enter", "wait_ms": 500})

// Get snapshot
shell({"action": "get_snapshot", "session_id": "SESSION_ID"})
// Should see "hello" in output

// Close session
shell({"action": "close", "session_id": "SESSION_ID"})
```

### 2. Working Directory and Args

```javascript
// Spawn with custom working directory and args
shell({"action": "spawn", "command": "/bin/bash", "args": ["-c", "pwd"], "cwd": "/tmp", "cols": 100, "rows": 30})
// Should see /tmp in initial snapshot

// Close session
shell({"action": "close", "session_id": "SESSION_ID"})
```

### 3. Special Keys

```javascript
// Test various special keys
shell({"action": "spawn"})

// Type without pressing enter
shell({"action": "send_input", "session_id": "SESSION_ID", "input": "ls -la"})

// Press enter key
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "enter", "wait_ms": 500})

// Get output
shell({"action": "get_snapshot", "session_id": "SESSION_ID"})

// Test ctrl+c
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "ctrl+c"})

// Close
shell({"action": "close", "session_id": "SESSION_ID"})
```

### 4. TUI Application - Vim (Critical Test)

```javascript
// Spawn vim editing a test file
shell({"action": "spawn", "command": "vim", "args": ["/tmp/test.txt"], "cols": 120, "rows": 40})
// Should see vim UI in initial snapshot

// Enter insert mode
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "i", "wait_ms": 200})

// Type some text
shell({"action": "send_input", "session_id": "SESSION_ID", "input": "Hello from vim!", "wait_ms": 200})

// Get snapshot with ANSI formatting
shell({"action": "get_snapshot", "session_id": "SESSION_ID", "format": "ansi"})
// Should see vim UI with typed text

// Exit insert mode
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "escape", "wait_ms": 200})

// Save and quit
shell({"action": "send_input", "session_id": "SESSION_ID", "input": ":wq", "wait_ms": 100})
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "enter", "wait_ms": 500})

// Close session
shell({"action": "close", "session_id": "SESSION_ID"})
// Should show exit code 0

// Verify file was created
shell({"action": "spawn", "command": "/bin/cat", "args": ["/tmp/test.txt"]})
// Should see "Hello from vim!" in initial snapshot
```

### 5. TUI Application - htop

```javascript
// Spawn htop
shell({"action": "spawn", "command": "htop", "cols": 140, "rows": 40})
// Should see htop UI with system info

// Get ANSI snapshot
shell({"action": "get_snapshot", "session_id": "SESSION_ID", "format": "ansi"})
// Should see colored htop interface

// Navigate down
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "down", "wait_ms": 100})
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "down", "wait_ms": 100})

// Get updated snapshot
shell({"action": "get_snapshot", "session_id": "SESSION_ID", "format": "ansi"})
// Cursor should have moved

// Quit htop
shell({"action": "send_input", "session_id": "SESSION_ID", "special_key": "q"})

// Close
shell({"action": "close", "session_id": "SESSION_ID"})
```

### 6. Terminal Resize

```javascript
// Spawn with default size
shell({"action": "spawn"})

// Resize terminal
shell({"action": "resize", "session_id": "SESSION_ID", "cols": 160, "rows": 50})

// Run command that shows terminal size
shell({"action": "send_input", "session_id": "SESSION_ID", "input": "echo $COLUMNS x $LINES", "special_key": "enter", "wait_ms": 300})

// Get snapshot
shell({"action": "get_snapshot", "session_id": "SESSION_ID"})
// Should show 160 x 50

// Close
shell({"action": "close", "session_id": "SESSION_ID"})
```

### 7. List Sessions

```javascript
// Spawn multiple sessions
shell({"action": "spawn"})  // Session 1
shell({"action": "spawn"})  // Session 2
shell({"action": "spawn"})  // Session 3

// List all active sessions
shell({"action": "list"})
// Should show 3 sessions with IDs, PIDs, commands, etc.

// Close all (one by one using their IDs)
shell({"action": "close", "session_id": "SESSION_ID_1"})
shell({"action": "close", "session_id": "SESSION_ID_2"})
shell({"action": "close", "session_id": "SESSION_ID_3"})

// List again
shell({"action": "list"})
// Should show "No active sessions"
```

### 8. Exit Code Tracking

```javascript
// Spawn a command that exits with success
shell({"action": "spawn", "command": "/bin/bash", "args": ["-c", "exit 0"]})
// Wait for it to complete
// (Wait 200ms)

// Get exit code
shell({"action": "close", "session_id": "SESSION_ID"})
// Should show "Exit code: 0"

// Spawn a command that exits with error
shell({"action": "spawn", "command": "/bin/bash", "args": ["-c", "exit 42"]})
// Wait for it to complete
// (Wait 200ms)

// Get exit code
shell({"action": "close", "session_id": "SESSION_ID"})
// Should show "Exit code: 42"
```

## Expected Outcomes

### Critical Success Criteria

1. ✅ Vim launches and displays its UI
2. ✅ Can type in vim and see changes in snapshots
3. ✅ Special keys (escape, arrows, etc.) work in TUI apps
4. ✅ ANSI format preserves colors and formatting
5. ✅ Non-destructive snapshots (can get_snapshot multiple times)
6. ✅ Exit codes are tracked correctly

### Known Limitations

- Wait times may need adjustment based on system performance
- Some ANSI sequences may not render perfectly in all terminals
- Very large screen buffers may be slow to serialize

## Troubleshooting

### Session Not Found

- Verify session ID is correct (copy from spawn response)
- Check session hasn't already been closed
- Use `list` action to see active sessions

### No Output in Snapshot

- Increase `wait_ms` parameter in send_input
- Some commands may take longer to produce output
- Check if command actually produces output (try locally first)

### TUI App Not Displaying Correctly

- Verify cols/rows are large enough for the app
- Try "ansi" format instead of "text"
- Some apps may need specific terminal settings

### Permission Issues

- Ensure MCP server has necessary permissions
- Check that the command exists and is executable
- Verify cwd path exists if specified
