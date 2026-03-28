# Code Review: f32c83e — require app context for key/type/paste actions
**Date:** 2026-02-27
**Reviewers:** Multi-Agent (security, architecture, simplicity, silent-failure, test-quality, go)
**Target:** HEAD~1 (3 files: do.go, do_test.go, help.go)

## Summary
- **P1 Critical Issues:** 0
- **P2 Important Issues:** 5
- **P3 Nice-to-Have:** 3
- **Confidence Threshold:** 70
- **Filtered Out (below threshold):** 2

## P1 - Critical (Block Merge)

None.

## P2 - Important (Fix Before/After Merge)

- [x] **#1 [DRY]** Error message templates share identical boilerplate across two switch branches `do.go:623-657` (Confidence: 92)
  - Issue: The "Fix:" section with Option A / Option B / `list_windows()` footer is verbatim identical in both the coordinate-actions and keyboard-actions error branches. Only the 2-line explanation differs.
  - Fix: Extract a `noAppContextError(i int, actionType, reason, exampleSuffix string) error` helper
  - Agent: simplicity, go

- [x] **#2 [SILENT-FAILURE]** `validateAppContext` switch has no `default` — new action types silently bypass enforcement `do.go:621` (Confidence: 87)
  - Issue: Types not listed in the switch (wait, screenshot, clipboard) fall through silently. Currently correct, but a new action type added later would silently skip app-context enforcement with no signal.
  - Fix: Add explicit exempt cases + a `default` that fails closed
  - Agent: silent-failure-hunter

- [x] **#3 [TEST-GAP]** Error tests for key/type/paste don't verify fix-hint content `do_test.go:816-841` (Confidence: 82)
  - Issue: New tests assert only `"no app context"`. Existing click test asserts `"Option A"`, `"Option B"`, and `"absolute screen coordinates"`. Keyboard tests should also assert `"frontmost"` and the fix hints.
  - Fix: Add `assert.Contains(t, err.Error(), "frontmost")` and `"Option A"` / `"Option B"` to the three keyboard error tests
  - Agent: test-quality, go

- [x] **#4 [TEST-GAP]** Missing `type`-with-direct-App and `paste`-with-direct-App happy path tests `do_test.go:843` (Confidence: 92)
  - Issue: `key` has a direct-App test (`TestValidateAppContext_KeyWithApp`), but `type` and `paste` only test the focus-inheritance path. The direct-App path is untested for those two.
  - Fix: Add `TestValidateAppContext_TypeWithApp` and `TestValidateAppContext_PasteWithApp`
  - Agent: test-quality

- [x] **#5 [TEST-STALE]** `TestValidActionTypes` hardcoded list is stale — omits paste, clipboard, drag, screenshot `do_test.go:455` (Confidence: 95)
  - Issue: The expected slice only has 11 of 15 valid types. Separate tests partially compensate but no single test asserts the complete membership.
  - Fix: Replace with `require.ElementsMatch(t, expected, ValidActionTypes)` covering all 15 types
  - Agent: architecture, silent-failure-hunter

## P3 - Nice-to-Have

- [ ] **#6 [DOCS]** `help.go` says "app REQUIRED" for keyboard actions but doesn't mention focus-inheritance alternative `help.go:37` (Confidence: 72)
  - Issue: LLM consumers may always spell out `app` on every action instead of using the more concise focus-then-act pattern
  - Agent: architecture

- [ ] **#7 [REDUNDANCY]** `scroll` case in `validateAppContext` is unreachable — `validateScrollAction` rejects scroll-without-app first `do.go:621` (Confidence: 85)
  - Issue: `validateScrollAction` hard-requires `action.App != ""` before `validateAppContext` runs. Same applies to `drag`. Silent redundancy as defense-in-depth, but undocumented.
  - Agent: security

- [ ] **#8 [TEST-GAP]** No test for mid-batch re-focus with keyboard actions `do_test.go` (Confidence: 75)
  - Issue: `focus(A) -> type -> focus(B) -> key` is a valid real-world pattern but untested for the newly-restricted action types
  - Agent: test-quality

---

## Cross-Cutting Analysis

### Root Causes Identified

| Root Cause | Findings Affected | Suggested Fix |
|------------|-------------------|---------------|
| Duplicated error message template | #1, #3 | Extract `noAppContextError` helper — fixes duplication AND makes tests cleaner (one place to assert) |
| No exhaustive switch coverage | #2, #5 | Add `default` case + `ElementsMatch` test — both enforce completeness |
| Asymmetric test coverage for keyboard actions | #3, #4, #8 | Add missing happy/error path tests — small effort |

### Single-Fix Opportunities

1. **`noAppContextError` helper** — Fixes #1 and simplifies #3 (~20 lines net reduction)
2. **`default` case in switch + `ElementsMatch` test** — Fixes #2 and #5 (~10 lines added)
3. **4 small test functions** — Fixes #4 and #8 (~25 lines added)

### Context Files (Read Before Fixing)

| File | Reason | Referenced By |
|------|--------|---------------|
| `internal/tools/do.go:186-197` | Call site of validateAppContext — errors surfaced as tool-result errors | go, security |
| `internal/tools/do.go:220-248` | Execution loop focus propagation — must stay in sync with validation | architecture |
| `internal/tools/do.go:466` | validateScrollAction — pre-empts validateAppContext for scroll | security, architecture |

## Recommended Actions

1. **Immediate:** No P1 blockers — merge is safe
2. **This PR/follow-up:** Address #1 (DRY), #2 (default case), #3-#5 (test gaps)
3. **Later:** #6 (help docs), #7 (document redundancy), #8 (edge case test)
