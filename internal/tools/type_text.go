package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/input"
)

// HandleTypeText handles the type_text tool call.
func HandleTypeText(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("text is required"), nil
	}

	if text == "" {
		return mcp.NewToolResultError("text cannot be empty"), nil
	}

	// Type the text
	input.TypeText(text)

	return mcp.NewToolResultText(fmt.Sprintf("Typed %d characters", len(text))), nil
}
