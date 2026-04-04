package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
	"github.com/sstraus/mcpmaccontrol/internal/input"
	"github.com/sstraus/mcpmaccontrol/internal/notify"
	"github.com/sstraus/mcpmaccontrol/internal/permissions"
)

// StringOrArray accepts both "value" and ["value"] from JSON.
type StringOrArray []string

// UnmarshalJSON handles both string and array input.
func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	// Try array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}
	// Try single string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}
	return fmt.Errorf("modifiers must be a string or array of strings")
}

// Action represents a single action in a batch.
type Action struct {
	Type string `json:"type"` // click, move, type, key, wait, scroll, focus, minimize, close

	// Mouse actions
	App     string `json:"app,omitempty"`      // Target app name
	AppName string `json:"app_name,omitempty"` // Alias for app
	X       int    `json:"x,omitempty"`        // X coordinate
	Y       int    `json:"y,omitempty"`        // Y coordinate
	Button  string `json:"button,omitempty"`   // left, right, middle
	Double  bool   `json:"double,omitempty"`   // double-click

	// Keyboard actions
	Text      string        `json:"text,omitempty"`      // Text to type
	Key       string        `json:"key,omitempty"`       // Key to press
	Modifiers StringOrArray `json:"modifiers,omitempty"` // cmd, shift, alt, ctrl (accepts string or array)

	// Wait action
	Ms       int     `json:"ms,omitempty"`       // Milliseconds to wait
	Duration float64 `json:"duration,omitempty"` // Alias: seconds (converted to ms)

	// Scroll action
	DeltaY int `json:"delta_y,omitempty"` // Vertical scroll (negative = up)
	DeltaX int `json:"delta_x,omitempty"` // Horizontal scroll (negative = left)

	// Resize action
	Width  int `json:"width,omitempty"`  // New window width
	Height int `json:"height,omitempty"` // New window height

	// Drag action (destination coordinates, relative to window)
	ToX int `json:"to_x,omitempty"` // Drag destination X
	ToY int `json:"to_y,omitempty"` // Drag destination Y

	// Screenshot action
	Format  string `json:"format,omitempty"`  // Image format: "webp" (default) or "png"
	Quality int    `json:"quality,omitempty"` // WebP quality 1-100 (default 25; use 50+ for small glyphs/icons)
}

// normalizeAction converts aliases to canonical field names.
// Returns the original type before normalization (empty if unchanged).
func normalizeAction(action *Action) (originalType string) {
	rawType := action.Type

	// mouse_move → move, mouse_click → click, etc.
	if strings.HasPrefix(strings.ToLower(action.Type), "mouse_") {
		action.Type = strings.TrimPrefix(strings.ToLower(action.Type), "mouse_")
	}
	// hover → move
	if strings.EqualFold(action.Type, "hover") {
		action.Type = "move"
	}

	if !strings.EqualFold(rawType, action.Type) {
		originalType = rawType
	}

	// app_name → app
	if action.App == "" && action.AppName != "" {
		action.App = action.AppName
	}
	// duration (seconds) → ms (milliseconds)
	if action.Ms == 0 && action.Duration > 0 {
		action.Ms = int(action.Duration * 1000)
	}
	// Compound key shortcut: "cmd+shift+g" → key="g", modifiers=["cmd","shift"]
	if strings.Contains(action.Key, "+") {
		parts := strings.Split(action.Key, "+")
		action.Key = parts[len(parts)-1]
		action.Modifiers = append(parts[:len(parts)-1:len(parts)-1], action.Modifiers...)
	}
	return originalType
}

// ActionResult represents the result of a single action.
type ActionResult struct {
	Index   int    `json:"index"`
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ValidActionTypes lists all supported action types
var ValidActionTypes = []string{"click", "move", "type", "key", "paste", "clipboard", "wait", "scroll", "drag", "focus", "minimize", "restore", "close", "resize", "screenshot"}

// ValidKeys lists all supported key names
var ValidKeys = []string{
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
	"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	"tab", "enter", "return", "escape", "esc", "delete", "backspace", "space",
	"left", "right", "up", "down",
	"home", "end", "pageup", "pagedown",
	"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
	"-", "=", "[", "]", "\\", ";", "'", ",", ".", "/", "`",
}

// ValidModifiers lists all supported modifier keys
var ValidModifiers = []string{"cmd", "command", "shift", "alt", "option", "opt", "ctrl", "control"}

// ValidButtons lists all supported mouse buttons
var ValidButtons = []string{"left", "right", "middle"}

// HandleDo handles the unified "do" tool call.
// Executes one or more actions in sequence.
func HandleDo(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	debug := os.Getenv("MCPMACCONTROL_DEBUG") == "1"
	if debug {
		start := time.Now()
		log.Printf("[DEBUG] HandleDo start")
		defer func() {
			log.Printf("[DEBUG] HandleDo finished in %s", time.Since(start))
		}()
	}

	// Validate parameters first so the user gets helpful errors
	args := request.GetArguments()
	actionsRaw := args["actions"]
	if actionsRaw == nil {
		return mcp.NewToolResultError(formatMissingActionsError()), nil
	}

	// Parse actions
	actionsJSON, err := json.Marshal(actionsRaw)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse actions: %v\n\n%s", err, formatActionsHelp())), nil
	}

	var actions []Action
	if err := json.Unmarshal(actionsJSON, &actions); err != nil {
		return mcp.NewToolResultError(formatParseError(err)), nil
	}

	if len(actions) == 0 {
		return mcp.NewToolResultError(formatEmptyActionsError()), nil
	}

	// Normalize aliases (app_name → app, duration → ms) and track originals
	originalTypes := make([]string, len(actions))
	for i := range actions {
		originalTypes[i] = normalizeAction(&actions[i])
	}

	// Validate all actions first (fail fast with helpful errors)
	for i, action := range actions {
		if err := validateAction(i, action); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	// Validate that coordinate-based actions have an app context.
	// Either via explicit "app" field or a preceding "focus" action.
	if err := validateAppContext(actions); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Check Accessibility permission only when the batch includes actions that need it.
	// Screenshot-only batches only need Screen Recording (checked lazily in executeScreenshot).
	// Use HasAccessibilityPermission() instead of EnsureAllPermissions() to avoid
	// triggering a 10-second blocking Screen Recording request via dispatch_semaphore_wait.
	needsAccessibility := false
	for _, action := range actions {
		if strings.ToLower(action.Type) != "screenshot" && strings.ToLower(action.Type) != "wait" {
			needsAccessibility = true
			break
		}
	}
	if needsAccessibility && !permissions.HasAccessibilityPermission() {
		return mcp.NewToolResultError("Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility"), nil
	}

	// Execute actions, propagating app context from focus actions.
	// When a focus action sets an app, subsequent coordinate-based actions
	// (click, move, scroll, drag, screenshot) inherit that app if they
	// don't specify one — so window-relative coordinates from screenshots
	// map correctly without requiring app on every action.
	results := make([]ActionResult, 0, len(actions))
	var focusedApp string
	for i, action := range actions {
		// Track focused app from focus actions
		if strings.ToLower(action.Type) == "focus" && action.App != "" {
			focusedApp = action.App
		}

		// Inherit focusedApp into actions missing app. This ensures
		// coordinate-based actions use window-relative coords and
		// keyboard actions target the correct app.
		if action.App == "" && focusedApp != "" {
			switch strings.ToLower(action.Type) {
			case "click", "move", "scroll", "drag", "screenshot",
				"key", "type", "paste":
				action.App = focusedApp
			}
		}

		result := executeAction(i, action)
		// Annotate result when an alias was used
		if originalTypes[i] != "" && result.Success {
			result.Message += fmt.Sprintf(" (note: use \"%s\" instead of \"%s\")", action.Type, originalTypes[i])
		}
		results = append(results, result)

		// Stop on first error
		if !result.Success {
			break
		}
	}

	// Format response
	return formatResults(results), nil
}

// validateAction performs semantic validation with educational error messages
func validateAction(index int, action Action) error {
	actionType := strings.ToLower(action.Type)

	// Check if type is provided
	if action.Type == "" {
		return fmt.Errorf(`[Action %d] Missing "type" field.

Each action must have a "type" field specifying what to do.

Valid types: %s

Example:
  {"type": "click", "app": "Safari", "x": 100, "y": 50}

Call help("actions") for full reference.`, index, strings.Join(ValidActionTypes, ", "))
	}

	// Check if type is valid
	validType := false
	for _, t := range ValidActionTypes {
		if actionType == t {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf(`[Action %d] Unknown action type: "%s"

Valid types: %s

Did you mean one of these?
  • "click" - Click at position in window
  • "type"  - Type text string
  • "key"   - Press keyboard key
  • "scroll" - Scroll in window
  • "wait"  - Pause execution

Example:
  {"type": "click", "app": "Safari", "x": 100, "y": 50}`, index, action.Type, strings.Join(ValidActionTypes, ", "))
	}

	// Type-specific validation
	switch actionType {
	case "click", "move":
		return validateMouseAction(index, action, actionType)
	case "type":
		return validateTypeAction(index, action)
	case "key":
		return validateKeyAction(index, action)
	case "paste":
		return validatePasteAction(index, action)
	case "clipboard":
		return nil // no parameters required
	case "wait":
		return validateWaitAction(index, action)
	case "scroll":
		return validateScrollAction(index, action)
	case "drag":
		return validateDragAction(index, action)
	case "focus", "minimize", "restore", "close":
		return validateWindowAction(index, action, actionType)
	case "resize":
		return validateResizeAction(index, action)
	case "screenshot":
		return validateScreenshotAction(index, action)
	}

	return nil
}

func validateMouseAction(index int, action Action, actionType string) error {
	// app is optional: when omitted, x/y are absolute screen coordinates.

	// Validate button if provided
	if action.Button != "" {
		validButton := false
		for _, b := range ValidButtons {
			if strings.ToLower(action.Button) == b {
				validButton = true
				break
			}
		}
		if !validButton {
			return fmt.Errorf(`[Action %d] Invalid button: "%s"

Valid buttons: left (default), right, middle

Examples:
  {"type": "click", "app": "Safari", "x": 100, "y": 50}                    ← left click (default)
  {"type": "click", "app": "Safari", "x": 100, "y": 50, "button": "right"} ← right click
  {"type": "click", "app": "Safari", "x": 100, "y": 50, "button": "middle"} ← middle click`, index, action.Button)
		}
	}

	return nil
}

func validateTypeAction(index int, action Action) error {
	if action.Text == "" {
		return fmt.Errorf(`[Action %d] type action requires "text" field.

The "text" field specifies what to type.

Correct format:
  {"type": "type", "text": "Hello World"}

Your action:
  {"type": "type"}  ← missing "text"

Example:
  {"type": "type", "text": "user@example.com"}`, index)
	}
	return nil
}

func validateKeyAction(index int, action Action) error {
	if action.Key == "" {
		return fmt.Errorf(`[Action %d] key action requires "key" field.

The "key" field specifies which key to press.

Correct format:
  {"type": "key", "key": "enter"}
  {"type": "key", "key": "v", "modifiers": ["cmd"]}

Your action:
  {"type": "key"}  ← missing "key"

Valid keys:
  Letters: a-z
  Numbers: 0-9
  Special: tab, enter, escape, delete, space
  Arrows: left, right, up, down
  Function: f1-f12

Modifiers: cmd, shift, alt, ctrl`, index)
	}

	// Validate key
	keyLower := strings.ToLower(action.Key)
	validKey := false
	for _, k := range ValidKeys {
		if keyLower == k {
			validKey = true
			break
		}
	}
	if !validKey {
		return fmt.Errorf(`[Action %d] Unknown key: "%s"

Valid keys:
  Letters: a, b, c, ... z
  Numbers: 0, 1, 2, ... 9
  Special: tab, enter, escape, delete, space, backspace
  Arrows: left, right, up, down
  Function: f1, f2, ... f12

Examples:
  {"type": "key", "key": "enter"}
  {"type": "key", "key": "tab"}
  {"type": "key", "key": "v", "modifiers": ["cmd"]}  ← ⌘V`, index, action.Key)
	}

	// Validate modifiers
	for _, mod := range action.Modifiers {
		modLower := strings.ToLower(mod)
		validMod := false
		for _, m := range ValidModifiers {
			if modLower == m {
				validMod = true
				break
			}
		}
		if !validMod {
			return fmt.Errorf(`[Action %d] Unknown modifier: "%s"

Valid modifiers: cmd, shift, alt, ctrl
  (aliases: command, option, opt, control)

Examples:
  {"type": "key", "key": "c", "modifiers": ["cmd"]}         ← ⌘C
  {"type": "key", "key": "z", "modifiers": ["cmd", "shift"]} ← ⌘⇧Z`, index, mod)
		}
	}

	return nil
}

const maxWaitMs = 30_000 // 30 seconds — prevents unbounded goroutine blocking

func validateWaitAction(index int, action Action) error {
	if action.Ms <= 0 || action.Ms > maxWaitMs {
		return fmt.Errorf(`[Action %d] wait action requires "ms" between 1 and %d.

The "ms" field specifies milliseconds to wait.

Correct format:
  {"type": "wait", "ms": 500}

Your action:
  {"type": "wait", "ms": %d}

Examples:
  {"type": "wait", "ms": 100}   ← 100ms
  {"type": "wait", "ms": 1000}  ← 1 second
  {"type": "wait", "ms": 5000}  ← 5 seconds`, index, maxWaitMs, action.Ms)
	}
	return nil
}

func validateScrollAction(index int, action Action) error {
	if action.App == "" {
		return fmt.Errorf(`[Action %d] scroll action requires "app" field.

Correct format:
  {"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": -100}

Your action:
  {"type": "scroll", ...}  ← missing "app"

Tip: Use list_windows() to find available app names.`, index)
	}

	if action.DeltaY == 0 && action.DeltaX == 0 {
		return fmt.Errorf(`[Action %d] scroll action requires "delta_y" and/or "delta_x".

The delta values specify scroll direction and amount:
  • delta_y: Vertical scroll (negative=up, positive=down)
  • delta_x: Horizontal scroll (negative=left, positive=right)

Correct format:
  {"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": -100}

Your action:
  {"type": "scroll", "app": "%s", "x": %d, "y": %d}  ← missing delta

Examples:
  {"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": -100}  ← scroll up
  {"type": "scroll", "app": "Safari", "x": 400, "y": 300, "delta_y": 100}   ← scroll down`, index, action.App, action.X, action.Y)
	}

	return nil
}

func validateWindowAction(index int, action Action, actionType string) error {
	if action.App == "" {
		return fmt.Errorf(`[Action %d] %s action requires "app" field.

Correct format:
  {"type": "%s", "app": "Safari"}

Your action:
  {"type": "%s"}  ← missing "app"

Tip: Use list_windows() to find available app names.`, index, actionType, actionType, actionType)
	}
	return nil
}

func validateResizeAction(index int, action Action) error {
	if action.App == "" {
		return fmt.Errorf(`[Action %d] resize action requires "app" field.

Correct format:
  {"type": "resize", "app": "Safari", "width": 800, "height": 600}

Your action:
  {"type": "resize", "width": %d, "height": %d}  ← missing "app"

Tip: Use list_windows() to find available app names.`, index, action.Width, action.Height)
	}

	if action.Width <= 0 || action.Height <= 0 {
		return fmt.Errorf(`[Action %d] resize action requires positive "width" and "height".

Correct format:
  {"type": "resize", "app": "Safari", "width": 800, "height": 600}

Your action:
  {"type": "resize", "app": "%s", "width": %d, "height": %d}  ← invalid dimensions

Width and height must be positive integers (pixels).`, index, action.App, action.Width, action.Height)
	}

	return nil
}

func validatePasteAction(index int, action Action) error {
	if action.Text == "" {
		return fmt.Errorf(`[Action %d] paste action requires "text" field.

The "text" field specifies what to paste via the clipboard.

Correct format:
  {"type": "paste", "text": "/path/to/file"}

Your action:
  {"type": "paste"}  ← missing "text"

Paste writes to the clipboard and simulates ⌘V. Use this instead of
"type" when the target field has autocomplete (e.g. Finder Go to Folder).`, index)
	}
	return nil
}

func validateDragAction(index int, action Action) error {
	if action.App == "" {
		return fmt.Errorf(`[Action %d] drag action requires "app" field.

Correct format:
  {"type": "drag", "app": "Finder", "x": 100, "y": 100, "to_x": 300, "to_y": 300}

Your action:
  {"type": "drag", ...}  ← missing "app"

Tip: Use list_windows() to find available app names.`, index)
	}
	return nil
}

var validScreenshotFormats = []string{"webp", "png"}

func validateScreenshotAction(index int, action Action) error {
	if action.Format != "" {
		formatLower := strings.ToLower(action.Format)
		valid := false
		for _, f := range validScreenshotFormats {
			if formatLower == f {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf(`[Action %d] Invalid screenshot format: "%s"

Valid formats: webp (default), png

Examples:
  {"type": "screenshot", "app": "Safari"}
  {"type": "screenshot", "app": "Safari", "format": "png"}
  {"type": "screenshot"}  ← full screen`, index, action.Format)
		}
	}
	return nil
}

// validateAppContext checks that actions have an app context, either via an
// explicit "app" field or a preceding "focus" action in the batch. This
// prevents coordinate-based actions from using window-relative coordinates as
// absolute screen coordinates, and prevents keyboard/paste actions from
// targeting the wrong application when focus shifts between do() calls.
func validateAppContext(actions []Action) error {
	var focusApp string
	for i, action := range actions {
		actionType := strings.ToLower(action.Type)

		if actionType == "focus" && action.App != "" {
			focusApp = action.App
			continue
		}

		if action.App != "" || focusApp != "" {
			continue // has app context
		}

		switch actionType {
		case "click", "move", "scroll", "drag":
			return noAppContextError(i, actionType,
				fmt.Sprintf("Coordinates (x=%d, y=%d) will be treated as absolute screen coordinates,\nwhich is almost never what you want. Screenshot coordinates are relative\nto the window — you must specify which window.", action.X, action.Y),
				fmt.Sprintf(`{"type": "%s", "app": "AppName", "x": %d, "y": %d}`, actionType, action.X, action.Y),
				fmt.Sprintf(`{"type": "%s", "x": %d, "y": %d}`, actionType, action.X, action.Y))

		case "key", "type", "paste":
			return noAppContextError(i, actionType,
				"Without an app target, input goes to whichever application happens to be\nfrontmost — which may have changed since the last do() call.",
				fmt.Sprintf(`{"type": "%s", "app": "AppName", ...}`, actionType),
				fmt.Sprintf(`{"type": "%s", ...}`, actionType))

		case "wait", "screenshot", "clipboard":
			// These actions don't target a specific window

		case "focus", "minimize", "restore", "close", "resize":
			// These validate app in their own handlers
		}
	}
	return nil
}

// noAppContextError builds a consistent error for actions missing app context.
// reason explains why app context matters for this action type.
// optionA/optionB show example JSON with/without explicit app field.
func noAppContextError(index int, actionType, reason, optionA, optionB string) error {
	return fmt.Errorf(`[Action %d] %s action has no app context.

%s

Fix: add "app" to the action, or add a focus action earlier in the batch:

  Option A — app on each action:
    %s

  Option B — focus once, then actions inherit it:
    {"type": "focus", "app": "AppName"},
    %s

The "app" field is matched against window owner names from list_windows().`,
		index, actionType, reason, optionA, optionB)
}

// Error message formatters
func formatMissingActionsError() string {
	return `Missing "actions" parameter.

The do() tool requires an "actions" array containing one or more actions.

Correct format:
  do({"actions": [
    {"type": "click", "app": "Safari", "x": 100, "y": 50}
  ]})

Call help("actions") for full reference.
Call help("examples") for more examples.`
}

func formatParseError(err error) string {
	return fmt.Sprintf(`Failed to parse actions array: %v

The "actions" parameter must be a JSON array of action objects.

Correct format:
  do({"actions": [
    {"type": "click", "app": "Safari", "x": 100, "y": 50},
    {"type": "type", "text": "hello"},
    {"type": "key", "key": "enter"}
  ]})

Common mistakes:
  ✗ do({"actions": {"type": "click", ...}})     ← Object instead of array
  ✗ do({"actions": "click"})                    ← String instead of array
  ✓ do({"actions": [{"type": "click", ...}]})  ← Correct: array of objects`, err)
}

func formatEmptyActionsError() string {
	return `The "actions" array is empty.

Provide at least one action to execute.

Example:
  do({"actions": [
    {"type": "click", "app": "Safari", "x": 100, "y": 50}
  ]})

Call help("actions") for action reference.`
}

func formatActionsHelp() string {
	return `Expected format:
  do({"actions": [
    {"type": "click", "app": "Safari", "x": 100, "y": 50}
  ]})

Call help("actions") for full reference.`
}

func executeAction(index int, action Action) ActionResult {
	result := ActionResult{
		Index: index,
		Type:  action.Type,
	}

	// Notify user before screen-affecting actions
	switch strings.ToLower(action.Type) {
	case "click", "move", "type", "key", "paste", "scroll", "drag":
		notify.BeforeAction()
	}

	switch strings.ToLower(action.Type) {
	case "click":
		result = executeClick(index, action)
	case "move":
		result = executeMove(index, action)
	case "type":
		result = executeType(index, action)
	case "key":
		result = executeKey(index, action)
	case "paste":
		result = executePaste(index, action)
	case "clipboard":
		result = executeClipboard(index, action)
	case "wait":
		result = executeWait(index, action)
	case "scroll":
		result = executeScroll(index, action)
	case "drag":
		result = executeDrag(index, action)
	// Window control actions
	case "focus":
		result = executeFocus(index, action)
	case "minimize":
		result = executeMinimize(index, action)
	case "restore":
		result = executeRestore(index, action)
	case "close":
		result = executeClose(index, action)
	case "resize":
		result = executeResize(index, action)
	case "screenshot":
		result = executeScreenshot(index, action)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unknown action type: %s", action.Type)
	}

	return result
}

func executeClick(index int, action Action) ActionResult {
	button := input.ParseMouseButton(action.Button)

	var absX, absY int
	if action.App == "" {
		// Absolute screen coordinates
		absX, absY = action.X, action.Y
		if !input.IsOnDisplay(absX, absY) {
			return ActionResult{
				Index:   index,
				Type:    "click",
				Success: false,
				Error:   fmt.Sprintf("coordinates (%d,%d) are outside all displays", absX, absY),
			}
		}
		input.ClickMouse(absX, absY, int(button), action.Double)
	} else {
		wi, errResult := ensureWindowFocused(index, "click", action.App)
		if errResult != nil {
			return *errResult
		}
		var err error
		absX, absY, err = input.ClickInWindowInfo(wi, action.X, action.Y, button, action.Double)
		if err != nil {
			return ActionResult{
				Index:   index,
				Type:    "click",
				Success: false,
				Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
			}
		}
	}

	clickType := "click"
	if action.Double {
		clickType = "double-click"
	}
	if action.Button == "right" {
		clickType = "right-" + clickType
	}

	target := action.App
	if target == "" {
		target = "screen"
	}

	return ActionResult{
		Index:   index,
		Type:    "click",
		Success: true,
		Message: fmt.Sprintf("%s at (%d,%d) in %s (screen: %d,%d)", clickType, action.X, action.Y, target, absX, absY),
	}
}

func executeMove(index int, action Action) ActionResult {
	var absX, absY int
	if action.App == "" {
		// Absolute screen coordinates
		absX, absY = action.X, action.Y
		if !input.IsOnDisplay(absX, absY) {
			return ActionResult{
				Index:   index,
				Type:    "move",
				Success: false,
				Error:   fmt.Sprintf("coordinates (%d,%d) are outside all displays", absX, absY),
			}
		}
		input.MoveMouse(absX, absY)
	} else {
		wi, errResult := ensureWindowFocused(index, "move", action.App)
		if errResult != nil {
			return *errResult
		}
		var err error
		absX, absY, err = input.MoveMouseToWindowInfo(wi, action.X, action.Y)
		if err != nil {
			return ActionResult{
				Index:   index,
				Type:    "move",
				Success: false,
				Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
			}
		}
	}

	target := action.App
	if target == "" {
		target = "screen"
	}

	return ActionResult{
		Index:   index,
		Type:    "move",
		Success: true,
		Message: fmt.Sprintf("moved to (%d,%d) in %s (screen: %d,%d)", action.X, action.Y, target, absX, absY),
	}
}

func executeType(index int, action Action) ActionResult {
	if action.App != "" {
		if _, errResult := ensureWindowFocused(index, "type", action.App); errResult != nil {
			return *errResult
		}
	}

	input.TypeText(action.Text)

	target := action.App
	if target == "" {
		target = "frontmost app"
	}

	return ActionResult{
		Index:   index,
		Type:    "type",
		Success: true,
		Message: fmt.Sprintf("typed %d chars in %s", len(action.Text), target),
	}
}

func executeKey(index int, action Action) ActionResult {
	if action.App != "" {
		if _, errResult := ensureWindowFocused(index, "key", action.App); errResult != nil {
			return *errResult
		}
	}

	success := input.KeyTap(action.Key, []string(action.Modifiers))
	if !success {
		return ActionResult{
			Index:   index,
			Type:    "key",
			Success: false,
			Error:   fmt.Sprintf("key tap failed for %s\n\nValid keys: a-z, 0-9, tab, enter, escape, delete, space, arrows, f1-f12", action.Key),
		}
	}

	msg := action.Key
	if len(action.Modifiers) > 0 {
		msg = strings.Join(action.Modifiers, "+") + "+" + action.Key
	}

	target := action.App
	if target == "" {
		target = "frontmost app"
	}

	return ActionResult{
		Index:   index,
		Type:    "key",
		Success: true,
		Message: fmt.Sprintf("pressed %s in %s", msg, target),
	}
}

func executeWait(index int, action Action) ActionResult {
	time.Sleep(time.Duration(action.Ms) * time.Millisecond)

	return ActionResult{
		Index:   index,
		Type:    "wait",
		Success: true,
		Message: fmt.Sprintf("waited %dms", action.Ms),
	}
}

func executeScroll(index int, action Action) ActionResult {
	wi, errResult := ensureWindowFocused(index, "scroll", action.App)
	if errResult != nil {
		return *errResult
	}
	// Move mouse to the scroll position in window, then scroll
	_, _, err := input.MoveMouseToWindowInfo(wi, action.X, action.Y)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "scroll",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	input.Scroll(action.DeltaY, action.DeltaX)

	return ActionResult{
		Index:   index,
		Type:    "scroll",
		Success: true,
		Message: fmt.Sprintf("scrolled y=%d x=%d at (%d,%d) in %s", action.DeltaY, action.DeltaX, action.X, action.Y, action.App),
	}
}

func executePaste(index int, action Action) ActionResult {
	if action.App != "" {
		if _, errResult := ensureWindowFocused(index, "paste", action.App); errResult != nil {
			return *errResult
		}
	}

	if err := input.PasteText(action.Text); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "paste",
			Success: false,
			Error:   fmt.Sprintf("paste failed: %v", err),
		}
	}

	target := action.App
	if target == "" {
		target = "frontmost app"
	}

	return ActionResult{
		Index:   index,
		Type:    "paste",
		Success: true,
		Message: fmt.Sprintf("pasted %d chars via clipboard in %s", len(action.Text), target),
	}
}

func executeClipboard(index int, _ Action) ActionResult {
	text, err := input.GetClipboard()
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "clipboard",
			Success: false,
			Error:   fmt.Sprintf("clipboard read failed: %v", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "clipboard",
		Success: true,
		Message: text,
	}
}

func executeDrag(index int, action Action) ActionResult {
	wi, errResult := ensureWindowFocused(index, "drag", action.App)
	if errResult != nil {
		return *errResult
	}
	fromAbsX, fromAbsY, toAbsX, toAbsY, err := input.DragInWindowInfo(wi, action.X, action.Y, action.ToX, action.ToY)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "drag",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "drag",
		Success: true,
		Message: fmt.Sprintf("dragged (%d,%d)→(%d,%d) in %s (screen: %d,%d→%d,%d)", action.X, action.Y, action.ToX, action.ToY, action.App, fromAbsX, fromAbsY, toAbsX, toAbsY),
	}
}

// findWindowPID finds the PID for an app by name using the canonical
// two-pass matching algorithm.
func findWindowPID(appName string) (int, error) {
	w, err := capture.FindWindow(appName)
	if err != nil {
		return 0, err
	}
	return w.OwnerPID, nil
}

// ensureWindowFocused brings the target app's window to front if it is not
// already the frontmost application. Returns the resolved WindowInfo on
// success (eliminating the need for a second window lookup). Returns a
// non-nil ActionResult only on error (caller should return it).
func ensureWindowFocused(index int, actionType string, appName string) (*capture.WindowInfo, *ActionResult) {
	wi, err := capture.FindWindow(appName)
	if err != nil {
		return nil, &ActionResult{
			Index:   index,
			Type:    actionType,
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}
	if input.IsFrontmostApp(wi.OwnerPID) {
		return wi, nil
	}
	if err := input.FocusWindow(wi.OwnerPID, 0); err != nil {
		return nil, &ActionResult{
			Index:   index,
			Type:    actionType,
			Success: false,
			Error:   fmt.Sprintf("failed to focus window: %v", err),
		}
	}
	time.Sleep(100 * time.Millisecond) // let focus settle
	return wi, nil
}

func executeFocus(index int, action Action) ActionResult {
	pid, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "focus",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.FocusWindow(pid, 0); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "focus",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "focus",
		Success: true,
		Message: fmt.Sprintf("focused %s", action.App),
	}
}

func executeMinimize(index int, action Action) ActionResult {
	pid, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "minimize",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.MinimizeWindow(pid, 0); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "minimize",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "minimize",
		Success: true,
		Message: fmt.Sprintf("minimized %s", action.App),
	}
}

func executeRestore(index int, action Action) ActionResult {
	pid, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "restore",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.RestoreWindow(pid, 0); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "restore",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "restore",
		Success: true,
		Message: fmt.Sprintf("restored %s", action.App),
	}
}

func executeClose(index int, action Action) ActionResult {
	pid, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "close",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.CloseWindow(pid, 0); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "close",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "close",
		Success: true,
		Message: fmt.Sprintf("closed %s", action.App),
	}
}

func executeResize(index int, action Action) ActionResult {
	pid, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "resize",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.ResizeWindow(pid, 0, action.Width, action.Height); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "resize",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "resize",
		Success: true,
		Message: fmt.Sprintf("resized %s to %dx%d", action.App, action.Width, action.Height),
	}
}

func executeScreenshot(index int, action Action) ActionResult {
	// Check Screen Recording permission lazily — only screenshot needs it.
	// Use HasScreenRecordingPermission() to avoid the 10-second blocking
	// RequestScreenRecordingPermission() call in EnsureAllPermissions().
	if !permissions.HasScreenRecordingPermission() {
		return ActionResult{
			Index:   index,
			Type:    "screenshot",
			Success: false,
			Error:   "Screen Recording permission required. Grant access in System Settings > Privacy & Security > Screen Recording",
		}
	}

	formatStr := strings.ToLower(action.Format)
	if formatStr == "" {
		formatStr = "webp"
	}
	format := capture.FormatWebP
	if formatStr == "png" {
		format = capture.FormatPNG
	}

	var img image.Image
	var description string

	if action.App != "" {
		// Capture specific window
		capturedImg, winInfo, err := capture.CaptureWindowByName(action.App, "", true)
		if err != nil {
			return ActionResult{
				Index:   index,
				Type:    "screenshot",
				Success: false,
				Error:   fmt.Sprintf("capture failed: %v", err),
			}
		}
		img = capture.ResizeToTarget(capturedImg, winInfo.Width, winInfo.Height)
		description = fmt.Sprintf("%s (%dx%d) — pixel coords in image = x,y for do() actions with app:\"%s\"",
			winInfo.OwnerName, winInfo.Width, winInfo.Height, action.App)
	} else {
		// Capture full screen
		capturedImg, err := capture.CaptureScreen(0)
		if err != nil {
			return ActionResult{
				Index:   index,
				Type:    "screenshot",
				Success: false,
				Error:   fmt.Sprintf("capture failed: %v", err),
			}
		}
		bounds := capturedImg.Bounds()
		img = capturedImg
		description = fmt.Sprintf("screen (%dx%d) — pixel coords in image = absolute x,y for do() actions",
			bounds.Dx(), bounds.Dy())
	}

	// Save to temp file instead of returning base64 (avoids token limit issues)
	prefix := "screen0"
	if action.App != "" {
		prefix = capture.SanitizeFilename(action.App)
	}
	filePath, err := capture.SaveToFileWithFormat(img, prefix, format, action.Quality)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "screenshot",
			Success: false,
			Error:   fmt.Sprintf("save failed: %v", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "screenshot",
		Success: true,
		Message: fmt.Sprintf("%s\nScreenshot saved to: %s\nUse the Read tool to view it.", description, filePath),
	}
}

func formatResults(results []ActionResult) *mcp.CallToolResult {
	var sb strings.Builder
	allSuccess := true
	for _, r := range results {
		if !r.Success {
			allSuccess = false
		}
		if r.Success {
			sb.WriteString(fmt.Sprintf("[%d] %s: %s\n", r.Index, r.Type, r.Message))
		} else {
			sb.WriteString(fmt.Sprintf("[%d] %s: ERROR\n%s\n", r.Index, r.Type, r.Error))
		}
	}
	if !allSuccess && len(results) > 0 {
		sb.WriteString("\nExecution stopped on first error.")
	}
	return mcp.NewToolResultText(sb.String())
}
