package tools

import (
	"encoding/base64"
	"fmt"
	"image"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
)

// maxInlineBase64 is the threshold in base64 characters below which images are
// returned inline. Above this, images are saved to a temp file and the path is
// returned instead. Claude Code truncates MCP results above ~80K characters;
// 50K gives comfortable headroom for surrounding text content.
const maxInlineBase64 = 50_000

// imageResult optimizes an image and returns MCP content. Small images are
// returned inline as base64; large ones are saved to a temp file.
func imageResult(img image.Image, prefix string, format capture.ImageFormat, quality int) ([]mcp.Content, error) {
	result, err := capture.OptimizeImageWithFormat(img, format, quality)
	if err != nil {
		return nil, fmt.Errorf("optimize failed: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(result.Data)

	if len(encoded) <= maxInlineBase64 {
		return []mcp.Content{mcp.NewImageContent(encoded, result.MIMEType)}, nil
	}

	// Too large for inline — save already-optimized bytes to temp file
	filePath, err := capture.SaveBytesToFile(result.Data, prefix, result.Format)
	if err != nil {
		return nil, fmt.Errorf("save failed: %w", err)
	}

	return []mcp.Content{
		mcp.NewTextContent(fmt.Sprintf("Screenshot saved to: %s\nUse the Read tool to view it.", filePath)),
	}, nil
}
