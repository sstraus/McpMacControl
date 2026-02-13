package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/input"
)

// HandleMouseMove handles the mouse_move tool call.
func HandleMouseMove(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := request.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("app_name is required"), nil
	}

	x, err := request.RequireInt("x")
	if err != nil {
		return mcp.NewToolResultError("x coordinate is required"), nil
	}

	y, err := request.RequireInt("y")
	if err != nil {
		return mcp.NewToolResultError("y coordinate is required"), nil
	}

	// Move mouse
	absX, absY, err := input.MoveMouseToWindow(appName, x, y)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("mouse move failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Mouse moved to (%d, %d) relative to %s (screen: %d, %d)", x, y, appName, absX, absY)), nil
}
