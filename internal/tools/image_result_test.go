package tools

import (
	"image"
	"image/color"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
)

func TestImageResult_SmallImage_ReturnsInline(t *testing.T) {
	// 10x10 image → tiny base64, well under threshold
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	content, err := imageResult(img, "test", capture.FormatWebP, 0)
	require.NoError(t, err)
	require.Len(t, content, 1)

	_, ok := content[0].(mcp.ImageContent)
	assert.True(t, ok, "small image should return inline ImageContent")
}
