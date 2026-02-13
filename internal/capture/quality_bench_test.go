package capture

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebPQualityParameter(t *testing.T) {
	img := createUITestImage(1500, 1000)

	// Quality 0 should use the default (same as DefaultWebPQuality)
	defaultResult, err := OptimizeImageWithFormat(img, FormatWebP, 0)
	require.NoError(t, err)

	explicitDefault, err := OptimizeImageWithFormat(img, FormatWebP, int(DefaultWebPQuality))
	require.NoError(t, err)

	assert.Equal(t, len(defaultResult.Data), len(explicitDefault.Data),
		"quality=0 should produce same size as quality=%d", int(DefaultWebPQuality))

	// Lower quality should produce smaller files
	lowResult, err := OptimizeImageWithFormat(img, FormatWebP, 5)
	require.NoError(t, err)

	highResult, err := OptimizeImageWithFormat(img, FormatWebP, 85)
	require.NoError(t, err)

	assert.Less(t, len(lowResult.Data), len(highResult.Data),
		"quality=5 should be smaller than quality=85")

	t.Logf("q5=%.1fKB  default(q%d)=%.1fKB  q85=%.1fKB",
		float64(len(lowResult.Data))/1024,
		int(DefaultWebPQuality),
		float64(len(defaultResult.Data))/1024,
		float64(len(highResult.Data))/1024)
}

// createUITestImage builds a synthetic image resembling a code editor UI.
func createUITestImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// Light background
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 242, G: 242, B: 245, A: 255})
		}
	}

	// Dark sidebar
	for y := 0; y < h; y++ {
		for x := 0; x < 250; x++ {
			img.Set(x, y, color.RGBA{R: 45, G: 45, B: 55, A: 255})
		}
	}

	// Title bar
	for y := 0; y < 30; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 60, G: 60, B: 70, A: 255})
		}
	}

	// Simulated text lines
	for line := 0; line < 40; line++ {
		y0 := 50 + line*22
		lineLen := 300 + (line*7)%500
		for y := y0; y < y0+12 && y < h; y++ {
			for x := 280; x < 280+lineLen && x < w; x++ {
				r := uint8(30 + (x*3)%60)
				g := uint8(30 + (line*5)%80)
				b := uint8(50 + (x*7)%100)
				img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	return img
}
