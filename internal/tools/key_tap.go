package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/input"
)

// HandleKeyTap handles the key_tap tool call.
func HandleKeyTap(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError("key is required"), nil
	}

	// Get modifiers (optional array of strings)
	modifiers := request.GetStringSlice("modifiers", nil)

	// Tap the key
	if !input.KeyTap(key, modifiers) {
		return mcp.NewToolResultError(fmt.Sprintf("Unknown key: %s. Valid keys: a-z, 0-9, tab, enter, escape, delete, space, arrows, f1-f12", key)), nil
	}

	// Build response
	if len(modifiers) > 0 {
		return mcp.NewToolResultText(fmt.Sprintf("Pressed %s+%s", strings.Join(modifiers, "+"), key)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Pressed %s", key)), nil
}
