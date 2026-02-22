# Changelog

## 1.2.0

### Features
- **Native popover** — replaced floating HUD with an NSPopover anchored to the systray icon. Two-line layout: project path (bold) and structured tool description. Fades out after 5 seconds
- **Project identification** — popover shows the calling project's path (via MCP ListRoots), so you can tell which Claude Code session is triggering actions
- **Structured tool descriptions** — popover shows target app, session IDs, and action chains (e.g. `do [Safari]: focus → click → type`)
- **App context inheritance** — `focus` action propagates app to subsequent `click`, `move`, `key`, `type`, `paste`, `scroll`, `drag`, and `screenshot` actions in the same batch, so coordinates from screenshots map correctly without repeating `app` on every action
- **Input target verification** — `key`, `type`, and `paste` actions now verify the target app is focused before sending input, preventing keystrokes from reaching the wrong app
- **App context validation** — coordinate-based actions without `app` or a preceding `focus` are rejected with a rich error explaining both fix options

### Fixes
- **Screenshot popover timing** — popover appears after captures instead of before, so it doesn't pollute screenshots

### Removed
- System-interacting tests that sent real clicks and keystrokes during `go test`

## 1.1.0

### Features
- **Action notifications** — plays a "Tink" sound and flashes an orange border overlay on all displays before mouse, keyboard, scroll, and drag actions so the user knows automation is happening
- **Menu bar toggle** — "Notify on Actions" checkbox in the systray context menu to enable/disable notifications (enabled by default)
- **Window matching by title** — `list_windows`, `capture_window`, and `do` actions now match windows by title substring in addition to app/owner name
- **Auto-focus before interaction** — mouse actions automatically focus the target window and report whether it was already focused
- **Display bounds API** — off-screen coordinates are detected and reported before actions execute
- **Action type aliases** — `hover` accepted as alias for `move`; `mouse_click`, `mouse_move`, etc. accepted as aliases for `click`, `move`, etc.

### Fixes
- **Stale socket recovery** — when the backend `.app` crashes and leaves a socket file behind, the bridge now detects the stale socket during launch (dial succeeds but MCP handshake fails), removes it, and starts a fresh backend instead of returning "broken pipe"
- **Help tool accuracy** — help output now educates on canonical action types instead of documenting aliases

## 1.0.0

Initial release.

### Tools
- **`do`** — batched mouse, keyboard, scroll, and window actions in a single call
- **`capture_window`** — screenshot a window or region, with coordinates mapped for clicking
- **`capture_screen`** — screenshot entire display
- **`list_windows`** — find windows by app name
- **`shell`** — interactive PTY sessions with terminal emulation (vim, htop, etc.)
- **`processes`** — list running processes with filtering by name or PID
- **`help`** — built-in API documentation

### Screenshots
- WebP lossy at quality 15 — fully readable UI text at ~3x smaller than q75
- Region capture to reduce token usage
- Coordinates in screenshots map directly to `do()` click positions

### Distribution
- Signed `.app` bundle for stable macOS permission identity
- `make release` — build, sign with Developer ID, notarize, staple, zip
- Auto-detects best signing cert: Developer ID > dev cert > ad-hoc

### Permissions
- Native wizard window showing live Accessibility/Screen Recording status
- Lazy `.app` startup — only spawns when a tool actually needs OS access
