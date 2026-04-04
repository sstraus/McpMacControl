package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
	"github.com/sstraus/mcpmaccontrol/internal/permissions"
)

// HandleCaptureScreen handles the capture_screen tool call.
func HandleCaptureScreen(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	displayIndex := request.GetInt("display", 0)
	formatStr := request.GetString("format", "webp")
	quality := request.GetInt("quality", 0)

	// Check Screen Recording permission before capturing.
	// Use HasScreenRecordingPermission() to avoid the 10-second blocking
	// RequestScreenRecordingPermission() call in EnsureAllPermissions().
	if !permissions.HasScreenRecordingPermission() {
		return mcp.NewToolResultError("Screen Recording permission required. Grant access in System Settings > Privacy & Security > Screen Recording"), nil
	}

	// Capture the screen
	img, err := capture.CaptureScreen(displayIndex)
	if err != nil {
		if errors.Is(err, capture.ErrPermissionDenied) {
			return mcp.NewToolResultError(permissionErrorMsg), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("capture failed: %v", err)), nil
	}

	// Determine format
	format := capture.FormatWebP
	if formatStr == "png" {
		format = capture.FormatPNG
	}

	// Encode image — inline if small, temp file if large
	prefix := fmt.Sprintf("screen%d", displayIndex)
	imgContent, err := imageResult(img, prefix, format, quality)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	description := fmt.Sprintf("Screenshot of display %d (%dx%d)", displayIndex, width, height)
	content := []mcp.Content{mcp.NewTextContent(description)}
	content = append(content, imgContent...)

	return &mcp.CallToolResult{Content: content}, nil
}
