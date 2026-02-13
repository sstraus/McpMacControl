package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
)

// HandleSaveScreen handles the save_screen tool call.
// Saves screenshot to a temp file and returns the path.
func HandleSaveScreen(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	displayIndex := request.GetInt("display", 0)

	// Capture and save to file
	path, err := capture.SaveScreenToFile(displayIndex)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("capture failed: %v", err)), nil
	}

	// Return the file path
	numDisplays := capture.NumDisplays()
	description := fmt.Sprintf("Screenshot saved to: %s", path)
	if numDisplays > 1 {
		description = fmt.Sprintf("Screenshot of display %d saved to: %s", displayIndex, path)
	}

	return mcp.NewToolResultText(description), nil
}
