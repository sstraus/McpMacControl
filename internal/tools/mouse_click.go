package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/input"
)

// HandleMouseClick handles the mouse_click tool call.
func HandleMouseClick(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	buttonStr := request.GetString("button", "left")
	button := input.ParseMouseButton(buttonStr)

	doubleClick := request.GetBool("double_click", false)

	// Click
	absX, absY, err := input.ClickInWindow(appName, x, y, button, doubleClick)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("click failed: %v", err)), nil
	}

	clickType := "Click"
	if doubleClick {
		clickType = "Double-click"
	}

	return mcp.NewToolResultText(fmt.Sprintf("%s at (%d, %d) relative to %s (screen: %d, %d)", clickType, x, y, appName, absX, absY)), nil
}
