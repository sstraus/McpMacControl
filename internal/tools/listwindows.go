package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
	"github.com/sstraus/mcpmaccontrol/internal/permissions"
)

// HandleListWindows handles the list_windows tool call.
func HandleListWindows(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appFilter := request.GetString("app_filter", "")

	// Check permission before listing windows
	perms := permissions.EnsureAllPermissions()
	if !perms.ScreenRecording {
		return mcp.NewToolResultError("Screen Recording permission required. Grant access in System Settings > Privacy & Security > Screen Recording"), nil
	}

	// Get window list
	windows, err := capture.ListWindows(appFilter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list windows failed: %v", err)), nil
	}

	if len(windows) == 0 {
		if appFilter != "" {
			return mcp.NewToolResultText(fmt.Sprintf("No windows found for app filter %q", appFilter)), nil
		}
		return mcp.NewToolResultText("No windows found"), nil
	}

	// Format output
	var sb strings.Builder
	for _, w := range windows {
		focused := ""
		if w.Focused {
			focused = " [focused]"
		}
		if w.Name != "" {
			fmt.Fprintf(&sb, "[%d] %s - %s%s\n", w.ID, w.OwnerName, w.Name, focused)
		} else {
			fmt.Fprintf(&sb, "[%d] %s%s\n", w.ID, w.OwnerName, focused)
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}
