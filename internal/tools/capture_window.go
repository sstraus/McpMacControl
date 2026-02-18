package tools

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
	"github.com/sstraus/mcpmaccontrol/internal/permissions"
)

const permissionErrorMsg = "Screen Recording permission is not granted for MCPMacControl.app. " +
	"The user must enable it in System Settings > Privacy & Security > Screen Recording, " +
	"then restart the MCP server."

// HandleCaptureWindow handles the capture_window tool call.
func HandleCaptureWindow(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Validate parameters first so the user gets helpful errors
	appName, err := request.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("app_name is required"), nil
	}

	windowTitle := request.GetString("window_title", "")
	hideShadow := request.GetBool("hide_shadow", true)
	formatStr := request.GetString("format", "webp")
	quality := request.GetInt("quality", 0)

	// Region parameters (optional - capture specific area of window)
	regionX := request.GetInt("region_x", 0)
	regionY := request.GetInt("region_y", 0)
	regionW := request.GetInt("region_width", 0)
	regionH := request.GetInt("region_height", 0)
	hasRegion := regionW > 0 && regionH > 0

	// Check permission before capturing
	perms := permissions.EnsureAllPermissions()
	if !perms.ScreenRecording {
		return mcp.NewToolResultError("Screen Recording permission required. Grant access in System Settings > Privacy & Security > Screen Recording"), nil
	}

	// Capture the window
	img, winInfo, err := capture.CaptureWindowByName(appName, windowTitle, hideShadow)
	if err != nil {
		if errors.Is(err, capture.ErrPermissionDenied) {
			return mcp.NewToolResultError(permissionErrorMsg), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("capture failed: %v", err)), nil
	}

	// Resize to window point dimensions so image pixels match click coordinates.
	// screencapture produces Retina-resolution images (2x pixels), but window
	// bounds and mouse events use points.
	img = capture.ResizeToTarget(img, winInfo.Width, winInfo.Height)

	// Crop to region if specified (coordinates are in point space)
	if hasRegion {
		img = cropImage(img, regionX, regionY, regionW, regionH)
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

	// Build bounds info for click targeting
	boundsInfo := fmt.Sprintf("Window: %s", winInfo.OwnerName)
	if winInfo.Name != "" {
		boundsInfo = fmt.Sprintf("Window: %s - %s", winInfo.OwnerName, winInfo.Name)
	}

	boundsInfo += fmt.Sprintf(" (%dx%d)", winInfo.Width, winInfo.Height)

	if winInfo.Focused {
		boundsInfo += " [focused]"
	} else {
		boundsInfo += " [not focused — do() will auto-focus before mouse actions]"
	}

	if hasRegion {
		// Region capture - explain coordinate mapping
		boundsInfo += fmt.Sprintf("\nRegion: x=%d, y=%d, %dx%d", regionX, regionY, regionW, regionH)
		boundsInfo += fmt.Sprintf("\nTo click: use x=%d+px, y=%d+py (where px,py are pixel coords in this image)", regionX, regionY)
	} else {
		boundsInfo += "\nPixel coordinates in this image map directly to do() x,y values."
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(boundsInfo),
			mcp.NewImageContent(imageData, result.MIMEType),
		},
	}, nil
}

// cropImage extracts a rectangular region from an image.
func cropImage(img image.Image, x, y, w, h int) image.Image {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	if si, ok := img.(subImager); ok {
		return si.SubImage(image.Rect(x, y, x+w, y+h))
	}

	// Fallback: copy pixels manually
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			dst.Set(dx, dy, img.At(x+dx, y+dy))
		}
	}
	return dst
}
