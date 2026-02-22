# Code Review: wip/match-windows-by-title
**Date:** 2026-02-22
**Reviewers:** Multi-Agent (security, performance, architecture, simplicity, silent-failure, test-quality, go-idioms)
**Target:** Branch `wip/match-windows-by-title` vs `main` (16 commits, 28 files, +1316/-149 lines)
**Confidence Threshold:** 70

## Summary
- **P1 Critical Issues:** 2
- **P2 Important Issues:** 8
- **P3 Nice-to-Have:** 10
- **Filtered Out (below threshold):** 0

---

## P1 - Critical (Block Merge)

- [x] **#1 [GO]** Slice append mutates backing array in `normalizeAction` (Confidence: 95) — FIXED in 6d397fd
  - Location: `internal/tools/do.go:102`
  - Issue: `append(parts[:len(parts)-1], action.Modifiers...)` shares the backing array with `parts`. When `action.Modifiers` is non-empty, append overwrites `parts[len(parts)-1]` which was already assigned to `action.Key`.
  - Fix: Use full slice expression `parts[:len(parts)-1:len(parts)-1]` to cap the subslice
  - Agent: go-reviewer

- [x] **#2 [TEST]** No test for off-screen absolute coordinates rejection (Confidence: 95) — FIXED in 6d397fd
  - Location: `internal/tools/do_test.go` (missing)
  - Issue: The `IsOnDisplay` guard in `executeClick`/`executeMove` is the primary safety check for the new absolute-coords feature. Only the happy path is tested. A regression removing the guard would go undetected.
  - Fix: Add `TestExecuteClick_OffScreenCoords` and `TestExecuteMove_OffScreenCoords` with coords like `(-99999,-99999)`
  - Agent: test-quality-reviewer

---

## P2 - Important (Fix Before/After Merge)

- [x] **#3 [SECURITY]** Unbounded `wait` duration enables sleep-DoS (Confidence: 95) — FIXED in 6d397fd
  - Location: `internal/tools/do.go:755` and `internal/tools/shell.go:140,171`
  - Issue: `wait` action and shell `wait_ms` accept any positive integer. A call with `ms: 2147483647` blocks the goroutine for ~24 days.
  - Fix: Cap at 30,000ms (30 seconds) in validation
  - Agent: security-reviewer

- [ ] **#4 [ARCH]** Duplicated window-matching logic across 3 sites (Confidence: 95) — STORY 003-a1c3
  - Location: `internal/input/input.go:69-113`, `internal/capture/window.go:67-124`, `internal/tools/do.go:850-864`
  - Issue: The two-pass matching algorithm (owner name first, title second) is implemented twice. A third variant in `do.go:findWindowPID` uses case-sensitive comparison and has no title fallback, creating divergent behavior.
  - Fix: Extract `capture.FindWindow(appName string) (*WindowInfo, error)` as the canonical implementation
  - Agents: architecture, simplicity, performance (all flagged independently)

- [ ] **#5 [PERF]** Double CGWindowList fetch per app-targeted action (Confidence: 97) — STORY 003-a1c3
  - Location: `internal/tools/do.go:640,644`
  - Issue: Every click/move/scroll/drag calls `ListWindows` twice: once in `ensureWindowFocused` (via `findWindowPID`) and once in `ClickInWindow` (via `findWindow`). For a 10-action batch, that's 20 syscalls reducible to 1-2.
  - Fix: Unify `findWindowPID` and `findWindow` into a single lookup; pass resolved window info through the execution chain
  - Agent: performance-reviewer

- [x] **#6 [GO]** `log.Fatalf` inside goroutine bypasses deferred cleanup (Confidence: 88) — FIXED in 6d397fd
  - Location: `main.go:152,158`
  - Issue: `log.Fatalf` calls `os.Exit(1)` inside the server goroutine, skipping `defer listener.Close()` and `defer os.Remove(sockPath)`. Leaves stale socket on disk.
  - Fix: Replace with `log.Printf` + `close(quitCh)` + `systray.Quit()` + `return`
  - Agent: go-reviewer

- [x] **#7 [SILENT]** Handshake error discarded on reuse path (Confidence: 85) — FIXED in 6d397fd
  - Location: `internal/bridge/proxy.go:178`
  - Issue: When handshake fails on an existing socket, the error is silently dropped. Log says "Stale socket detected" but not why the handshake failed.
  - Fix: `log.Printf("Stale socket detected (%v), removing...", err, ...)`
  - Agent: silent-failure-hunter

- [x] **#8 [TEST]** Confirmed duplicate test (Confidence: 100) — FIXED in 6d397fd
  - Location: `internal/tools/do_test.go:94`
  - Issue: `TestValidateAction_MoveWithoutApp_Duplicate` is byte-for-byte identical to `TestValidateAction_MoveWithoutApp` at line 51.
  - Fix: Delete the duplicate
  - Agents: test-quality, simplicity (both flagged)

- [x] **#9 [PERF]** `transparentIcon()` re-encodes PNG on every call (Confidence: 95) — FIXED in 6d397fd
  - Location: `main.go:375-380`, called at `main.go:227`
  - Issue: Allocates image + PNG-encodes on every capture call (called twice per screenshot). Output is always identical.
  - Fix: Pre-compute as `var transparentIconData = func() []byte { ... }()`
  - Agents: performance, simplicity, silent-failure (all flagged)

- [x] **#10 [SILENT]** `findWindow()` silently falls back to off-screen window (Confidence: 80) — FIXED in 6d397fd
  - Location: `internal/input/input.go:111`
  - Issue: When all matched windows are off-screen, returns first match without logging. Action appears to succeed but has no visible effect.
  - Fix: Add `log.Printf` before the fallback return
  - Agent: silent-failure-hunter

---

## P3 - Nice-to-Have

- [ ] **#11 [SECURITY]** Session ID is predictable (timestamp-based) — `internal/shell/session.go:108` (Confidence: 85)
- [ ] **#12 [SECURITY]** Window title substring match enables cross-app confusion — `internal/input/input.go:87` (Confidence: 88)
- [ ] **#13 [GO]** `do.go` is 1057 lines, exceeds 300-line split threshold (Confidence: 92)
- [ ] **#14 [TEST]** `describeTool()` in main.go has zero test coverage (Confidence: 90)
- [ ] **#15 [TEST]** `hover` alias normalization untested in `normalizeAction` tests (Confidence: 92)
- [ ] **#16 [SILENT]** CGGetActiveDisplayList return value not checked in C code (Confidence: 82)
- [ ] **#17 [GO]** `init()` used to set `enabled` default; prefer explicit initialization (Confidence: 72)
- [ ] **#18 [ARCH]** `describeTool` in main.go has feature envy on do tool's action format (Confidence: 82)
- [ ] **#19 [SIMPLICITY]** `findWindowPID` returns constant `windowIndex=0` (Confidence: 90)
- [ ] **#20 [PERF]** Two-loop CGWindow list could be merged into single pass (Confidence: 85)

---

## Cross-Cutting Analysis

### Root Causes Identified

| Root Cause | Findings Affected | Suggested Fix |
|------------|-------------------|---------------|
| Window matching not centralized | #4, #5, #10, #12, #19, #20 | Extract `capture.FindWindow()` as canonical implementation |
| No upper bound on sleep params | #3 | Add `maxWaitMs = 30000` validation in both `do` and `shell` |
| `transparentIcon` not cached | #9 | One-line change to package-level `var` |
| Slice mutation safety | #1 | One-line fix with full slice expression |

### Single-Fix Opportunities

1. **`capture.FindWindow()`** — Fixes #4, #5, #10, #12, #19 (~80 lines net reduction)
2. **`maxWaitMs` validation** — Fixes #3 (~5 lines)
3. **`transparentIconData` var** — Fixes #9 (~3 lines)
4. **Full slice expression** — Fixes #1 (~1 line)

### Context Files (Read Before Fixing)

| File | Reason | Referenced By |
|------|--------|---------------|
| `internal/agent/server.go` | Third `findWindowPID` implementation (possibly dead code) | architecture, go-reviewer |
| `internal/capture/windows.go` | `ListWindows` contract — shared primitive | architecture, performance |
| `internal/tools/shell.go` | `wait_ms` unbounded sleep | security |

---

## Agent Highlights

- **Security:** Unbounded wait/sleep durations; window title confusion attack surface; no critical vulns
- **Performance:** Double CGWindowList syscall per mouse action; transparentIcon re-encoding
- **Architecture:** Three divergent window-matching implementations; feature envy in describeTool
- **Simplicity:** Duplicate test; transparentIcon not cached; two-loop CGWindow merge opportunity
- **Silent Failures:** Handshake error discarded; off-screen window fallback unlogged; CGError unchecked
- **Test Quality:** Missing off-screen rejection test; duplicate test; hover alias untested; describeTool untestable in main
- **Go Idioms:** Slice append mutation (P1); log.Fatal in goroutine; init() for defaults
- **TypeScript/Rust/C#/Helm/Data Safety:** N/A — no matching files
