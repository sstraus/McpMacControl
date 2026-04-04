package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sstraus/mcpmaccontrol/internal/input"
)

func TestValidateAction_MissingType(t *testing.T) {
	action := Action{}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for missing type")
	}
	if !strings.Contains(err.Error(), `Missing "type" field`) {
		t.Errorf("error should mention missing type field, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Valid types:") {
		t.Error("error should list valid types")
	}
}

func TestValidateAction_UnknownType(t *testing.T) {
	action := Action{Type: "unknown_action"}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), `Unknown action type: "unknown_action"`) {
		t.Errorf("error should mention unknown type, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Did you mean") {
		t.Error("error should suggest valid types")
	}
}

func TestValidateAction_ClickWithoutApp(t *testing.T) {
	// Click without app is valid — uses absolute screen coordinates.
	action := Action{Type: "click", X: 100, Y: 50}
	err := validateAction(0, action)
	assert.NoError(t, err, "click without app should be valid (absolute coords)")
}

func TestValidateAction_MoveWithoutApp(t *testing.T) {
	// Move without app is valid — uses absolute screen coordinates.
	action := Action{Type: "move", X: 100, Y: 50}
	err := validateAction(0, action)
	assert.NoError(t, err, "move without app should be valid (absolute coords)")
}

func TestValidateAction_ClickInvalidButton(t *testing.T) {
	action := Action{Type: "click", App: "Safari", X: 100, Y: 50, Button: "invalid"}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for invalid button")
	}
	if !strings.Contains(err.Error(), `Invalid button: "invalid"`) {
		t.Errorf("error should mention invalid button, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "left (default), right, middle") {
		t.Error("error should list valid buttons")
	}
}

func TestValidateAction_ClickValid(t *testing.T) {
	tests := []struct {
		name   string
		action Action
	}{
		{"basic click", Action{Type: "click", App: "Safari", X: 100, Y: 50}},
		{"right click", Action{Type: "click", App: "Safari", X: 100, Y: 50, Button: "right"}},
		{"middle click", Action{Type: "click", App: "Safari", X: 100, Y: 50, Button: "middle"}},
		{"double click", Action{Type: "click", App: "Safari", X: 100, Y: 50, Double: true}},
		{"right double click", Action{Type: "click", App: "Finder", X: 200, Y: 100, Button: "right", Double: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(0, tt.action)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}


func TestValidateAction_MoveValid(t *testing.T) {
	action := Action{Type: "move", App: "Safari", X: 300, Y: 200}
	err := validateAction(0, action)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateAction_TypeMissingText(t *testing.T) {
	action := Action{Type: "type"}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for type without text")
	}
	if !strings.Contains(err.Error(), `requires "text" field`) {
		t.Errorf("error should mention missing text, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Correct format:") {
		t.Error("error should show correct format")
	}
}

func TestValidateAction_TypeValid(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"simple text", "hello"},
		{"with spaces", "hello world"},
		{"with special chars", "user@example.com"},
		{"with punctuation", "Hello, World!"},
		{"with numbers", "test123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{Type: "type", Text: tt.text}
			err := validateAction(0, action)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateAction_KeyMissingKey(t *testing.T) {
	action := Action{Type: "key"}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for key without key field")
	}
	if !strings.Contains(err.Error(), `requires "key" field`) {
		t.Errorf("error should mention missing key, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Valid keys:") {
		t.Error("error should list valid keys")
	}
}

func TestValidateAction_KeyUnknownKey(t *testing.T) {
	action := Action{Type: "key", Key: "unknown_key"}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), `Unknown key: "unknown_key"`) {
		t.Errorf("error should mention unknown key, got: %s", err.Error())
	}
}

func TestValidateAction_KeyUnknownModifier(t *testing.T) {
	action := Action{Type: "key", Key: "a", Modifiers: []string{"invalid"}}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for unknown modifier")
	}
	if !strings.Contains(err.Error(), `Unknown modifier: "invalid"`) {
		t.Errorf("error should mention unknown modifier, got: %s", err.Error())
	}
}

func TestValidateAction_KeyValid(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		modifiers []string
	}{
		// Letters
		{"letter a", "a", nil},
		{"letter z", "z", nil},
		// Numbers
		{"number 0", "0", nil},
		{"number 9", "9", nil},
		// Special keys
		{"tab", "tab", nil},
		{"enter", "enter", nil},
		{"return", "return", nil},
		{"escape", "escape", nil},
		{"esc", "esc", nil},
		{"delete", "delete", nil},
		{"backspace", "backspace", nil},
		{"space", "space", nil},
		// Arrow keys
		{"left", "left", nil},
		{"right", "right", nil},
		{"up", "up", nil},
		{"down", "down", nil},
		// Function keys
		{"f1", "f1", nil},
		{"f12", "f12", nil},
		// With modifiers
		{"cmd+c", "c", []string{"cmd"}},
		{"cmd+v", "v", []string{"cmd"}},
		{"cmd+1", "1", []string{"cmd"}},
		{"shift+tab", "tab", []string{"shift"}},
		{"cmd+shift+z", "z", []string{"cmd", "shift"}},
		// Modifier aliases
		{"command+a", "a", []string{"command"}},
		{"option+a", "a", []string{"option"}},
		{"opt+a", "a", []string{"opt"}},
		{"control+a", "a", []string{"control"}},
		{"ctrl+a", "a", []string{"ctrl"}},
		{"alt+a", "a", []string{"alt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{Type: "key", Key: tt.key, Modifiers: tt.modifiers}
			err := validateAction(0, action)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateAction_WaitZeroMs(t *testing.T) {
	action := Action{Type: "wait", Ms: 0}
	err := validateAction(0, action)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and")
}

func TestValidateAction_WaitNegativeMs(t *testing.T) {
	action := Action{Type: "wait", Ms: -100}
	err := validateAction(0, action)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and")
}

func TestValidateAction_WaitExceedsMax(t *testing.T) {
	action := Action{Type: "wait", Ms: maxWaitMs + 1}
	err := validateAction(0, action)
	assert.Error(t, err, "wait exceeding maxWaitMs should be rejected")
	assert.Contains(t, err.Error(), "between 1 and")
}

func TestValidateAction_WaitValid(t *testing.T) {
	tests := []struct {
		name string
		ms   int
	}{
		{"100ms", 100},
		{"500ms", 500},
		{"1 second", 1000},
		{"5 seconds", 5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{Type: "wait", Ms: tt.ms}
			err := validateAction(0, action)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateAction_ScrollMissingApp(t *testing.T) {
	action := Action{Type: "scroll", X: 100, Y: 100, DeltaY: -50}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for scroll without app")
	}
	if !strings.Contains(err.Error(), `requires "app" field`) {
		t.Errorf("error should mention missing app, got: %s", err.Error())
	}
}

func TestValidateAction_ScrollMissingDelta(t *testing.T) {
	action := Action{Type: "scroll", App: "Safari", X: 100, Y: 100}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for scroll without delta")
	}
	if !strings.Contains(err.Error(), `requires "delta_y" and/or "delta_x"`) {
		t.Errorf("error should mention missing delta, got: %s", err.Error())
	}
}

func TestValidateAction_ScrollValid(t *testing.T) {
	tests := []struct {
		name   string
		deltaY int
		deltaX int
	}{
		{"scroll up", -100, 0},
		{"scroll down", 100, 0},
		{"scroll left", 0, -50},
		{"scroll right", 0, 50},
		{"diagonal scroll", -50, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{Type: "scroll", App: "Safari", X: 400, Y: 300, DeltaY: tt.deltaY, DeltaX: tt.deltaX}
			err := validateAction(0, action)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateAction_WindowActionsValid(t *testing.T) {
	// Window actions that only need app
	simpleWindowActions := []string{"focus", "minimize", "restore", "close"}

	for _, actionType := range simpleWindowActions {
		t.Run(actionType, func(t *testing.T) {
			action := Action{Type: actionType, App: "Safari"}
			err := validateAction(0, action)
			if err != nil {
				t.Errorf("validation should pass for %s, got: %v", actionType, err)
			}
		})
	}

	// Resize needs width and height
	t.Run("resize", func(t *testing.T) {
		action := Action{Type: "resize", App: "Safari", Width: 800, Height: 600}
		err := validateAction(0, action)
		if err != nil {
			t.Errorf("validation should pass for resize, got: %v", err)
		}
	})
}

func TestValidateAction_WindowActionsMissingApp(t *testing.T) {
	windowActions := []string{"focus", "minimize", "restore", "close"}

	for _, actionType := range windowActions {
		t.Run(actionType, func(t *testing.T) {
			action := Action{Type: actionType}
			err := validateAction(0, action)
			if err == nil {
				t.Errorf("expected error for %s without app", actionType)
			}
			if !strings.Contains(err.Error(), `requires "app" field`) {
				t.Errorf("error should mention missing app, got: %s", err.Error())
			}
		})
	}
}

func TestValidateAction_ResizeMissingDimensions(t *testing.T) {
	action := Action{Type: "resize", App: "Safari"}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for resize without dimensions")
	}
	if !strings.Contains(err.Error(), `requires positive "width" and "height"`) {
		t.Errorf("error should mention missing dimensions, got: %s", err.Error())
	}
}

func TestValidateAction_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name   string
		action Action
	}{
		{"uppercase type", Action{Type: "CLICK", App: "Safari", X: 100, Y: 50}},
		{"mixed case type", Action{Type: "Click", App: "Safari", X: 100, Y: 50}},
		{"uppercase key", Action{Type: "key", Key: "ENTER"}},
		{"mixed case button", Action{Type: "click", App: "Safari", X: 100, Y: 50, Button: "Right"}},
		{"uppercase modifier", Action{Type: "key", Key: "c", Modifiers: []string{"CMD"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(0, tt.action)
			if err != nil {
				t.Errorf("expected no error for case-insensitive input, got: %v", err)
			}
		})
	}
}

func TestValidateAction_ActionIndex(t *testing.T) {
	// Verify error messages include correct action index
	tests := []struct {
		index  int
		action Action
	}{
		{0, Action{Type: "click", Button: "invalid"}},  // invalid button
		{3, Action{Type: "click", Button: "invalid"}},
		{10, Action{Type: "click", Button: "invalid"}},
	}

	for _, tt := range tests {
		indexStr := fmt.Sprintf("%d", tt.index)
		t.Run("index "+indexStr, func(t *testing.T) {
			err := validateAction(tt.index, tt.action)
			if err == nil {
				t.Error("expected error")
			}
			expectedPrefix := "[Action " + indexStr + "]"
			if !strings.Contains(err.Error(), expectedPrefix) {
				t.Errorf("error should contain %s, got: %s", expectedPrefix, err.Error())
			}
		})
	}
}

func TestFormatMissingActionsError(t *testing.T) {
	err := formatMissingActionsError()
	if !strings.Contains(err, `Missing "actions" parameter`) {
		t.Error("should mention missing actions")
	}
	if !strings.Contains(err, "Correct format:") {
		t.Error("should show correct format")
	}
	if !strings.Contains(err, "help(") {
		t.Error("should reference help")
	}
}

func TestFormatEmptyActionsError(t *testing.T) {
	err := formatEmptyActionsError()
	if !strings.Contains(err, "empty") {
		t.Error("should mention empty")
	}
	if !strings.Contains(err, "Example:") {
		t.Error("should show example")
	}
}

func TestFormatParseError(t *testing.T) {
	err := formatParseError(nil)
	if !strings.Contains(err, "Common mistakes:") {
		t.Error("should show common mistakes")
	}
	if !strings.Contains(err, "Object instead of array") {
		t.Error("should mention object vs array mistake")
	}
}

// Test that ValidActionTypes contains exactly all supported action types
func TestValidActionTypes(t *testing.T) {
	expected := []string{
		"click", "move", "type", "key", "paste", "clipboard",
		"wait", "scroll", "drag", "focus", "minimize", "restore",
		"close", "resize", "screenshot",
	}
	require.ElementsMatch(t, expected, ValidActionTypes)
}

// Test that valid keys list includes common keys
func TestValidKeysComplete(t *testing.T) {
	required := []string{
		"a", "z", "0", "9",
		"tab", "enter", "escape", "delete", "space",
		"left", "right", "up", "down",
		"f1", "f12",
	}
	for _, k := range required {
		found := false
		for _, v := range ValidKeys {
			if v == k {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidKeys missing: %s", k)
		}
	}
}

// Test that valid modifiers list includes all aliases
func TestValidModifiersComplete(t *testing.T) {
	required := []string{"cmd", "command", "shift", "alt", "option", "opt", "ctrl", "control"}
	for _, m := range required {
		found := false
		for _, v := range ValidModifiers {
			if v == m {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidModifiers missing: %s", m)
		}
	}
}

func TestNormalizeAction_CompoundKey(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		modifiers     StringOrArray
		wantKey       string
		wantModifiers StringOrArray
	}{
		{
			name:          "cmd+shift+g",
			key:           "cmd+shift+g",
			wantKey:       "g",
			wantModifiers: StringOrArray{"cmd", "shift"},
		},
		{
			name:          "ctrl+c",
			key:           "ctrl+c",
			wantKey:       "c",
			wantModifiers: StringOrArray{"ctrl"},
		},
		{
			name:          "plain key unchanged",
			key:           "enter",
			wantKey:       "enter",
			wantModifiers: nil,
		},
		{
			name:          "compound with existing modifiers",
			key:           "shift+g",
			modifiers:     StringOrArray{"cmd"},
			wantKey:       "g",
			wantModifiers: StringOrArray{"shift", "cmd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{Type: "key", Key: tt.key, Modifiers: tt.modifiers}
			normalizeAction(&action)
			assert.Equal(t, tt.wantKey, action.Key)
			assert.Equal(t, tt.wantModifiers, action.Modifiers)
		})
	}
}

func TestNormalizeAction_AppNameToApp(t *testing.T) {
	action := Action{AppName: "Safari"}
	normalizeAction(&action)
	if action.App != "Safari" {
		t.Errorf("expected app to be 'Safari', got: %s", action.App)
	}
}

func TestNormalizeAction_AppNameIgnoredIfAppSet(t *testing.T) {
	action := Action{App: "Finder", AppName: "Safari"}
	normalizeAction(&action)
	if action.App != "Finder" {
		t.Errorf("expected app to remain 'Finder', got: %s", action.App)
	}
}

func TestNormalizeAction_DurationToMs(t *testing.T) {
	action := Action{Duration: 1.5} // 1.5 seconds
	normalizeAction(&action)
	if action.Ms != 1500 {
		t.Errorf("expected ms to be 1500, got: %d", action.Ms)
	}
}

func TestNormalizeAction_DurationIgnoredIfMsSet(t *testing.T) {
	action := Action{Ms: 500, Duration: 2.0}
	normalizeAction(&action)
	if action.Ms != 500 {
		t.Errorf("expected ms to remain 500, got: %d", action.Ms)
	}
}

func TestNormalizeAction_MousePrefixStripped(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"mouse_move", "move"},
		{"mouse_click", "click"},
		{"Mouse_Scroll", "scroll"},
		{"MOUSE_DRAG", "drag"},
		{"click", "click"}, // no prefix — unchanged
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			action := Action{Type: tt.input}
			normalizeAction(&action)
			assert.Equal(t, tt.want, action.Type)
		})
	}
}

func TestFormatActionsHelp(t *testing.T) {
	help := formatActionsHelp()
	if !strings.Contains(help, "Expected format:") {
		t.Error("should contain expected format header")
	}
	if !strings.Contains(help, "do({") {
		t.Error("should show do() call format")
	}
	if !strings.Contains(help, "help(\"actions\")") {
		t.Error("should reference help command")
	}
}

func TestValidateAction_ResizeWithOnlyWidth(t *testing.T) {
	action := Action{Type: "resize", App: "Safari", Width: 800}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for resize with only width")
	}
	if !strings.Contains(err.Error(), `requires positive "width" and "height"`) {
		t.Errorf("error should mention both dimensions required, got: %s", err.Error())
	}
}

func TestValidateAction_ResizeWithOnlyHeight(t *testing.T) {
	action := Action{Type: "resize", App: "Safari", Height: 600}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for resize with only height")
	}
}

func TestValidateAction_ResizeWithZeroWidth(t *testing.T) {
	action := Action{Type: "resize", App: "Safari", Width: 0, Height: 600}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for resize with zero width")
	}
}

func TestValidateAction_ResizeWithNegativeDimensions(t *testing.T) {
	action := Action{Type: "resize", App: "Safari", Width: -100, Height: 600}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for resize with negative width")
	}
}

// --- paste action tests ---

func TestValidateAction_PasteMissingText(t *testing.T) {
	action := Action{Type: "paste"}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for paste without text")
	}
	if !strings.Contains(err.Error(), `requires "text" field`) {
		t.Errorf("error should mention missing text, got: %s", err.Error())
	}
}

func TestValidateAction_PasteValid(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"simple path", "/tmp/file.txt"},
		{"url", "https://example.com"},
		{"unicode", "Hello 世界"},
		{"multiline", "line1\nline2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{Type: "paste", Text: tt.text}
			err := validateAction(0, action)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// --- clipboard action tests ---

func TestValidateAction_ClipboardValid(t *testing.T) {
	action := Action{Type: "clipboard"}
	err := validateAction(0, action)
	if err != nil {
		t.Errorf("expected no error for clipboard read, got: %v", err)
	}
}

// --- drag action tests ---

func TestValidateAction_DragMissingApp(t *testing.T) {
	action := Action{Type: "drag", X: 100, Y: 100, Width: 200, Height: 200}
	err := validateAction(0, action)
	if err == nil {
		t.Error("expected error for drag without app")
	}
	if !strings.Contains(err.Error(), `requires "app" field`) {
		t.Errorf("error should mention missing app, got: %s", err.Error())
	}
}

func TestValidateAction_DragValid(t *testing.T) {
	action := Action{Type: "drag", App: "Finder", X: 100, Y: 100, ToX: 300, ToY: 300}
	err := validateAction(0, action)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// --- ValidActionTypes includes new actions (covered by TestValidActionTypes) ---

// --- screenshot validation tests ---

func TestValidateAction_ScreenshotValid(t *testing.T) {
	tests := []struct {
		name   string
		action Action
	}{
		{"with app", Action{Type: "screenshot", App: "Safari"}},
		{"without app (full screen)", Action{Type: "screenshot"}},
		{"with format webp", Action{Type: "screenshot", App: "Safari", Format: "webp"}},
		{"with format png", Action{Type: "screenshot", App: "Safari", Format: "png"}},
		{"format case insensitive", Action{Type: "screenshot", Format: "PNG"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAction(0, tt.action)
			assert.NoError(t, err)
		})
	}
}

func TestValidateAction_ScreenshotInvalidFormat(t *testing.T) {
	action := Action{Type: "screenshot", App: "Safari", Format: "jpeg"}
	err := validateAction(0, action)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `Invalid screenshot format: "jpeg"`)
	assert.Contains(t, err.Error(), "Valid formats:")
}

// TestValidActionTypes_IncludesScreenshot covered by TestValidActionTypes

// --- validateAppContext tests ---

func TestValidateAppContext_ClickWithApp(t *testing.T) {
	actions := []Action{
		{Type: "click", App: "Safari", X: 100, Y: 50},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_ClickWithPrecedingFocus(t *testing.T) {
	actions := []Action{
		{Type: "focus", App: "Safari"},
		{Type: "click", X: 100, Y: 50},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_ClickWithoutAppOrFocus(t *testing.T) {
	actions := []Action{
		{Type: "click", X: 100, Y: 50},
	}
	err := validateAppContext(actions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no app context")
	assert.Contains(t, err.Error(), "absolute screen coordinates")
	assert.Contains(t, err.Error(), "Option A")
	assert.Contains(t, err.Error(), "Option B")
}

func TestValidateAppContext_MoveWithoutAppOrFocus(t *testing.T) {
	actions := []Action{
		{Type: "move", X: 200, Y: 300},
	}
	err := validateAppContext(actions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no app context")
}

func TestValidateAppContext_TypeWithoutApp(t *testing.T) {
	actions := []Action{
		{Type: "type", Text: "hello"},
	}
	err := validateAppContext(actions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no app context")
	assert.Contains(t, err.Error(), "frontmost")
	assert.Contains(t, err.Error(), "Option A")
	assert.Contains(t, err.Error(), "Option B")
}

func TestValidateAppContext_KeyWithoutApp(t *testing.T) {
	actions := []Action{
		{Type: "key", Key: "enter"},
	}
	err := validateAppContext(actions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no app context")
	assert.Contains(t, err.Error(), "frontmost")
	assert.Contains(t, err.Error(), "Option A")
	assert.Contains(t, err.Error(), "Option B")
}

func TestValidateAppContext_PasteWithoutApp(t *testing.T) {
	actions := []Action{
		{Type: "paste", Text: "hello"},
	}
	err := validateAppContext(actions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no app context")
	assert.Contains(t, err.Error(), "frontmost")
	assert.Contains(t, err.Error(), "Option A")
	assert.Contains(t, err.Error(), "Option B")
}

func TestValidateAppContext_KeyWithApp(t *testing.T) {
	actions := []Action{
		{Type: "key", Key: "enter", App: "Terminal"},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_TypeWithApp(t *testing.T) {
	actions := []Action{
		{Type: "type", App: "Terminal", Text: "ls"},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_PasteWithApp(t *testing.T) {
	actions := []Action{
		{Type: "paste", App: "Finder", Text: "/tmp/file"},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_TypeWithPrecedingFocus(t *testing.T) {
	actions := []Action{
		{Type: "focus", App: "Safari"},
		{Type: "type", Text: "hello"},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_PasteWithPrecedingFocus(t *testing.T) {
	actions := []Action{
		{Type: "focus", App: "Terminal"},
		{Type: "paste", Text: "/tmp/file"},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_FocusThenMultipleClicks(t *testing.T) {
	actions := []Action{
		{Type: "focus", App: "Safari"},
		{Type: "click", X: 100, Y: 50},
		{Type: "click", X: 200, Y: 100},
		{Type: "click", X: 300, Y: 150},
	}
	assert.NoError(t, validateAppContext(actions))
}

func TestValidateAppContext_DragWithoutApp(t *testing.T) {
	actions := []Action{
		{Type: "drag", X: 10, Y: 10, ToX: 50, ToY: 50},
	}
	err := validateAppContext(actions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no app context")
}

func TestValidateAppContext_ScrollWithoutApp(t *testing.T) {
	actions := []Action{
		{Type: "scroll", X: 100, Y: 100, DeltaY: -3},
	}
	err := validateAppContext(actions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no app context")
}

func TestValidateAppContext_MixedBatch(t *testing.T) {
	// focus provides app context for all subsequent actions in the batch
	actions := []Action{
		{Type: "focus", App: "Terminal"},
		{Type: "click", X: 100, Y: 200},
		{Type: "type", Text: "ls"},
		{Type: "key", Key: "enter"},
		{Type: "wait", Ms: 500},
		{Type: "screenshot"},
	}
	assert.NoError(t, validateAppContext(actions))
}

// --- StringOrArray UnmarshalJSON tests ---

func TestStringOrArray_UnmarshalString(t *testing.T) {
	var s StringOrArray
	err := json.Unmarshal([]byte(`"cmd"`), &s)
	require.NoError(t, err)
	assert.Equal(t, StringOrArray{"cmd"}, s)
}

func TestStringOrArray_UnmarshalArray(t *testing.T) {
	var s StringOrArray
	err := json.Unmarshal([]byte(`["cmd", "shift"]`), &s)
	require.NoError(t, err)
	assert.Equal(t, StringOrArray{"cmd", "shift"}, s)
}

func TestStringOrArray_UnmarshalInvalid(t *testing.T) {
	var s StringOrArray
	err := json.Unmarshal([]byte(`123`), &s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "modifiers must be a string or array")
}

func TestStringOrArray_UnmarshalEmptyArray(t *testing.T) {
	var s StringOrArray
	err := json.Unmarshal([]byte(`[]`), &s)
	require.NoError(t, err)
	assert.Equal(t, StringOrArray{}, s)
}

// --- formatResults tests ---

func TestFormatResults_AllSuccess(t *testing.T) {
	results := []ActionResult{
		{Index: 0, Type: "click", Success: true, Message: "click at (100,50) in Safari"},
		{Index: 1, Type: "type", Success: true, Message: "typed 5 chars"},
	}
	out := formatResults(results)
	text := out.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "[0] click: click at (100,50)")
	assert.Contains(t, text, "[1] type: typed 5 chars")
	assert.NotContains(t, text, "stopped on first error")
}

func TestFormatResults_WithError(t *testing.T) {
	results := []ActionResult{
		{Index: 0, Type: "click", Success: true, Message: "click ok"},
		{Index: 1, Type: "type", Success: false, Error: "something failed"},
	}
	out := formatResults(results)
	text := out.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "[0] click: click ok")
	assert.Contains(t, text, "[1] type: ERROR")
	assert.Contains(t, text, "something failed")
	assert.Contains(t, text, "stopped on first error")
}

func TestFormatResults_SingleError(t *testing.T) {
	results := []ActionResult{
		{Index: 0, Type: "key", Success: false, Error: "key tap failed"},
	}
	out := formatResults(results)
	text := out.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "key tap failed")
	assert.Contains(t, text, "stopped on first error")
}

func TestFormatResults_WithScreenshot(t *testing.T) {
	results := []ActionResult{
		{Index: 0, Type: "click", Success: true, Message: "click at (100,50) in Safari"},
		{Index: 1, Type: "screenshot", Success: true, Message: "Safari (800x600)\nScreenshot saved to: /tmp/mcpmaccontrol-Safari-20260404-120000.webp\nUse the Read tool to view it."},
		{Index: 2, Type: "click", Success: true, Message: "click at (200,100) in Safari"},
	}
	out := formatResults(results)

	// All text now — single text content block
	require.Len(t, out.Content, 1)

	text := out.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "[0] click:")
	assert.Contains(t, text, "[1] screenshot:")
	assert.Contains(t, text, "Screenshot saved to:")
	assert.Contains(t, text, "[2] click:")
}

func TestFormatResults_WithScreenshot_Error(t *testing.T) {
	results := []ActionResult{
		{Index: 0, Type: "screenshot", Success: true, Message: "Safari (800x600)\nScreenshot saved to: /tmp/mcpmaccontrol-Safari-20260404-120000.webp"},
		{Index: 1, Type: "click", Success: false, Error: "window not found"},
	}
	out := formatResults(results)

	require.Len(t, out.Content, 1)

	text := out.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "Screenshot saved to:")
	assert.Contains(t, text, "stopped on first error")
}

// --- executeWait test ---

func TestExecuteWait_ReturnsCorrectMessage(t *testing.T) {
	result := executeWait(0, Action{Type: "wait", Ms: 50})
	assert.True(t, result.Success)
	assert.Equal(t, "wait", result.Type)
	assert.Equal(t, "waited 50ms", result.Message)
	assert.Equal(t, 0, result.Index)
}

// --- executeAction routing tests ---

func TestExecuteAction_UnknownType(t *testing.T) {
	result := executeAction(0, Action{Type: "nonexistent"})
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unknown action type")
}

func TestExecuteAction_WaitRoute(t *testing.T) {
	result := executeAction(0, Action{Type: "wait", Ms: 10})
	assert.True(t, result.Success)
	assert.Equal(t, "wait", result.Type)
}

func TestExecuteClick_OffScreenCoords(t *testing.T) {
	// Off-screen coordinates should be rejected without needing accessibility.
	result := executeClick(0, Action{Type: "click", X: -99999, Y: -99999})
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "outside all displays")
}

func TestExecuteMove_OffScreenCoords(t *testing.T) {
	result := executeMove(0, Action{Type: "move", X: -99999, Y: -99999})
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "outside all displays")
}

// --- executeClipboard test (no permissions needed) ---

func TestExecuteClipboard_ReadsClipboard(t *testing.T) {
	// Set clipboard to known value, then read via executeClipboard
	err := input.SetClipboard("test-clipboard-content")
	require.NoError(t, err)

	result := executeClipboard(0, Action{Type: "clipboard"})
	assert.True(t, result.Success)
	assert.Equal(t, "clipboard", result.Type)
	assert.Equal(t, "test-clipboard-content", result.Message)
}

func TestExecuteClipboard_EmptyClipboard(t *testing.T) {
	err := input.SetClipboard("")
	require.NoError(t, err)

	result := executeClipboard(0, Action{Type: "clipboard"})
	assert.True(t, result.Success)
	assert.Equal(t, "", result.Message)
}

// --- executeDrag test (requires window; test error path) ---

func TestExecuteDrag_NoSuchApp(t *testing.T) {
	result := executeDrag(0, Action{Type: "drag", App: "NonExistentApp12345", X: 10, Y: 10, ToX: 50, ToY: 50})
	assert.False(t, result.Success)
	assert.Equal(t, "drag", result.Type)
	assert.Contains(t, result.Error, "list_windows()")
}

// --- executeScreenshot tests ---

func TestExecuteScreenshot_NoSuchApp(t *testing.T) {
	skipWithoutScreenRecording(t)
	result := executeScreenshot(0, Action{Type: "screenshot", App: "NonExistentApp12345"})
	assert.False(t, result.Success)
	assert.Equal(t, "screenshot", result.Type)
	assert.Contains(t, result.Error, "capture failed")
}

