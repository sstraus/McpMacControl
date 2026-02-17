package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
	"github.com/sstraus/mcpmaccontrol/internal/input"
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
}

// normalizeAction converts aliases to canonical field names.
func normalizeAction(action *Action) {
	// mouse_move → move, mouse_click → click, etc.
	if strings.HasPrefix(strings.ToLower(action.Type), "mouse_") {
		action.Type = strings.TrimPrefix(strings.ToLower(action.Type), "mouse_")
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
		action.Modifiers = append(parts[:len(parts)-1], action.Modifiers...)
	}
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
var ValidActionTypes = []string{"click", "move", "type", "key", "paste", "clipboard", "wait", "scroll", "drag", "focus", "minimize", "restore", "close", "resize"}

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

	// Normalize aliases (app_name → app, duration → ms)
	for i := range actions {
		normalizeAction(&actions[i])
	}

	// Validate all actions first (fail fast with helpful errors)
	for i, action := range actions {
		if err := validateAction(i, action); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	// Check permission before executing
	perms := permissions.EnsureAllPermissions()
	if !perms.Accessibility {
		return mcp.NewToolResultError("Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility"), nil
	}

	// Execute actions
	results := make([]ActionResult, 0, len(actions))
	for i, action := range actions {
		result := executeAction(i, action)
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
	}

	return nil
}

func validateMouseAction(index int, action Action, actionType string) error {
	if action.App == "" {
		return fmt.Errorf(`[Action %d] %s requires "app" field.

The "app" field specifies which window to target.

Correct format:
  {"type": "%s", "app": "Safari", "x": 100, "y": 50}

Your action:
  {"type": "%s", "x": %d, "y": %d}  ← missing "app"

Tip: Use list_windows() to find available app names.`, index, actionType, actionType, actionType, action.X, action.Y)
	}

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

func validateWaitAction(index int, action Action) error {
	if action.Ms <= 0 {
		return fmt.Errorf(`[Action %d] wait action requires positive "ms" value.

The "ms" field specifies milliseconds to wait.

Correct format:
  {"type": "wait", "ms": 500}

Your action:
  {"type": "wait", "ms": %d}  ← must be positive

Examples:
  {"type": "wait", "ms": 100}   ← 100ms
  {"type": "wait", "ms": 1000}  ← 1 second
  {"type": "wait", "ms": 5000}  ← 5 seconds`, index, action.Ms)
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
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unknown action type: %s", action.Type)
	}

	return result
}

func executeClick(index int, action Action) ActionResult {
	button := input.ParseMouseButton(action.Button)
	absX, absY, err := input.ClickInWindow(action.App, action.X, action.Y, button, action.Double)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "click",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	clickType := "click"
	if action.Double {
		clickType = "double-click"
	}
	if action.Button == "right" {
		clickType = "right-" + clickType
	}

	return ActionResult{
		Index:   index,
		Type:    "click",
		Success: true,
		Message: fmt.Sprintf("%s at (%d,%d) in %s (screen: %d,%d)", clickType, action.X, action.Y, action.App, absX, absY),
	}
}

func executeMove(index int, action Action) ActionResult {
	absX, absY, err := input.MoveMouseToWindow(action.App, action.X, action.Y)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "move",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "move",
		Success: true,
		Message: fmt.Sprintf("moved to (%d,%d) in %s (screen: %d,%d)", action.X, action.Y, action.App, absX, absY),
	}
}

func executeType(index int, action Action) ActionResult {
	input.TypeText(action.Text)

	return ActionResult{
		Index:   index,
		Type:    "type",
		Success: true,
		Message: fmt.Sprintf("typed %d chars", len(action.Text)),
	}
}

func executeKey(index int, action Action) ActionResult {
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

	return ActionResult{
		Index:   index,
		Type:    "key",
		Success: true,
		Message: fmt.Sprintf("pressed %s", msg),
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
	// Move mouse to the scroll position in window, then scroll
	_, _, err := input.MoveMouseToWindow(action.App, action.X, action.Y)
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
	if err := input.PasteText(action.Text); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "paste",
			Success: false,
			Error:   fmt.Sprintf("paste failed: %v", err),
		}
	}

	return ActionResult{
		Index:   index,
		Type:    "paste",
		Success: true,
		Message: fmt.Sprintf("pasted %d chars via clipboard", len(action.Text)),
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
	// Focus the target window first. On macOS, dragging a non-frontmost
	// window's title bar activates it (consuming the mouseDown) instead
	// of starting a drag gesture.
	pid, windowIndex, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "drag",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}
	if err := input.FocusWindow(pid, windowIndex); err != nil {
		return ActionResult{
			Index:   index,
			Type:    "drag",
			Success: false,
			Error:   fmt.Sprintf("failed to focus window: %v", err),
		}
	}
	time.Sleep(100 * time.Millisecond) // let focus settle

	fromAbsX, fromAbsY, toAbsX, toAbsY, err := input.DragInWindow(action.App, action.X, action.Y, action.ToX, action.ToY)
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

// findWindowPID finds the PID for an app by name, returning the PID and window index (always 0 for first window).
func findWindowPID(appName string) (pid int, windowIndex int, err error) {
	windows, err := capture.ListWindows(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list windows: %w", err)
	}

	// Find exact match
	for _, w := range windows {
		if w.OwnerName == appName {
			return w.OwnerPID, 0, nil
		}
	}

	return 0, 0, fmt.Errorf("window not found for app: %s", appName)
}

func executeFocus(index int, action Action) ActionResult {
	pid, windowIndex, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "focus",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.FocusWindow(pid, windowIndex); err != nil {
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
	pid, windowIndex, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "minimize",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.MinimizeWindow(pid, windowIndex); err != nil {
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
	pid, windowIndex, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "restore",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.RestoreWindow(pid, windowIndex); err != nil {
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
	pid, windowIndex, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "close",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.CloseWindow(pid, windowIndex); err != nil {
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
	pid, windowIndex, err := findWindowPID(action.App)
	if err != nil {
		return ActionResult{
			Index:   index,
			Type:    "resize",
			Success: false,
			Error:   fmt.Sprintf("%v\n\nTip: Use list_windows() to verify the app name exists.", err),
		}
	}

	if err := input.ResizeWindow(pid, windowIndex, action.Width, action.Height); err != nil {
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
