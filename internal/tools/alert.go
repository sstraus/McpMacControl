package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/notify"
)

// HandleAlert activates or deactivates a visual alert overlay.
// When active=true, a flashing red border alerts the user.
// When active=false, the red flash stops and a brief green border confirms.
func HandleAlert(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	active := request.GetBool("active", false)

	if active {
		notify.StartWarn()
		return mcp.NewToolResultText("Alert activated — red flash on screen"), nil
	}

	notify.StopWarn()
	return mcp.NewToolResultText("Alert deactivated — green confirmation shown"), nil
}
