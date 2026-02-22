package tools

import (
	"context"
	"encoding/base64"
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

	// Optimize image
	result, err := capture.OptimizeImageWithFormat(img, format, quality)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("optimize failed: %v", err)), nil
	}

	// Encode to base64
	imageData := base64.StdEncoding.EncodeToString(result.Data)

	// Get dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Return image with description
	description := fmt.Sprintf("Screenshot of display %d (%dx%d)", displayIndex, width, height)

	return mcp.NewToolResultImage(description, imageData, result.MIMEType), nil
}
