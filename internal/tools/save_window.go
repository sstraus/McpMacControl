package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
)

// HandleSaveWindow handles the save_window tool call.
// Saves screenshot to a temp file and returns the path.
func HandleSaveWindow(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := request.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("app_name is required"), nil
	}

	windowTitle := request.GetString("window_title", "")
	hideShadow := request.GetBool("hide_shadow", true)

	// Capture and save to file
	path, info, err := capture.SaveWindowToFile(appName, windowTitle, hideShadow)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("capture failed: %v", err)), nil
	}

	// Return the file path with window info
	description := fmt.Sprintf("Screenshot of %s saved to: %s", info.OwnerName, path)
	if info.Name != "" {
		description = fmt.Sprintf("Screenshot of %s - %s saved to: %s", info.OwnerName, info.Name, path)
	}

	return mcp.NewToolResultText(description), nil
}
