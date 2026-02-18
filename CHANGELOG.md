# Changelog

## Unreleased

### Features
- **Action notifications** — plays a "Tink" sound and flashes an orange border overlay on all displays before mouse, keyboard, scroll, and drag actions so the user knows automation is happening
- **Menu bar toggle** — "Notify on Actions" checkbox in the systray context menu to enable/disable notifications (enabled by default)

### Fixes
- **Stale socket recovery** — when the backend `.app` crashes and leaves a socket file behind, the bridge now detects the stale socket during launch (dial succeeds but MCP handshake fails), removes it, and starts a fresh backend instead of returning "broken pipe"

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
