package capture

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/chai2010/webp"
	xdraw "golang.org/x/image/draw"
)

const (
	// MaxDimension is the maximum dimension for the longest edge.
	MaxDimension = 1500
	// MaxColors is the number of colors in the indexed palette.
	MaxColors = 256
)

// ImageFormat specifies the output format.
type ImageFormat string

const (
	FormatWebP ImageFormat = "webp"
	FormatPNG  ImageFormat = "png"
)

// OptimizeResult contains the optimized image data and metadata.
type OptimizeResult struct {
	Data     []byte
	MIMEType string
	Format   ImageFormat
}

// OptimizeImage resizes and encodes an image as WebP for AI consumption.
func OptimizeImage(img image.Image) ([]byte, error) {
	result, err := OptimizeImageWithFormat(img, FormatWebP, 0)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// OptimizeImageWithFormat resizes and optimizes an image in the specified format.
// quality controls WebP compression (1-100). When quality <= 0, uses DefaultWebPQuality.
// quality is ignored for PNG format.
func OptimizeImageWithFormat(img image.Image, format ImageFormat, quality int) (*OptimizeResult, error) {
	// Resize if needed
	resized := resize(img, MaxDimension)

	switch format {
	case FormatWebP:
		q := DefaultWebPQuality
		if quality > 0 {
			q = float32(quality)
		}
		data, err := encodeWebP(resized, q)
		if err != nil {
			return nil, fmt.Errorf("webp encoding failed: %w", err)
		}
		return &OptimizeResult{
			Data:     data,
			MIMEType: "image/webp",
			Format:   FormatWebP,
		}, nil

	default:
		// PNG encoding (with quantization for smaller size)
		quantized := quantize(resized, MaxColors)
		data, err := encodePNG(quantized)
		if err != nil {
			return nil, fmt.Errorf("png encoding failed: %w", err)
		}
		return &OptimizeResult{
			Data:     data,
			MIMEType: "image/png",
			Format:   FormatPNG,
		}, nil
	}
}

// DefaultWebPQuality is the default lossy encoding quality (0-100).
// 25 preserves fine UI details (glyphs, small icons, keyboard shortcuts)
// while maintaining ~3x size reduction vs PNG. Use 50+ for dense UIs.
const DefaultWebPQuality float32 = 25

// encodeWebP encodes image as lossy WebP at the given quality level.
func encodeWebP(img image.Image, quality float32) ([]byte, error) {
	var buf bytes.Buffer
	err := webp.Encode(&buf, img, &webp.Options{
		Lossless: false,
		Quality:  quality,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodePNG encodes image as PNG.
func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// resize scales the image so the longest edge is at most maxDim.
// Uses high-quality CatmullRom interpolation.
func resize(img image.Image, maxDim int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Check if resize is needed
	if w <= maxDim && h <= maxDim {
		return img
	}

	// Calculate new dimensions preserving aspect ratio
	var newW, newH int
	if w >= h {
		newW = maxDim
		newH = (h * maxDim) / w
	} else {
		newH = maxDim
		newW = (w * maxDim) / h
	}

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))

	// Scale with high-quality interpolation
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)

	return dst
}

// ResizeToTarget resizes the image to specific target dimensions.
// Used to match window point dimensions for accurate click coordinates.
func ResizeToTarget(img image.Image, targetW, targetH int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// No resize needed if already at target size
	if w == targetW && h == targetH {
		return img
	}

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))

	// Scale with high-quality interpolation
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)

	return dst
}

// quantize reduces the image to a limited color palette.
// Uses a simple median-cut-like algorithm optimized for UI screenshots.
func quantize(img image.Image, numColors int) *image.Paletted {
	bounds := img.Bounds()

	// Build color histogram
	colorCount := make(map[color.RGBA]int)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			c := color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			}
			colorCount[c]++
		}
	}

	// If already under limit, use exact colors
	if len(colorCount) <= numColors {
		palette := make(color.Palette, 0, len(colorCount))
		for c := range colorCount {
			palette = append(palette, c)
		}
		return createPalettedImage(img, palette)
	}

	// Simple quantization: reduce to 6-bit color (64 levels per channel)
	// This gives us 64*64*64 = 262144 possible colors, but in practice
	// UI screenshots use far fewer
	quantizedColors := make(map[color.RGBA]int)
	for c, count := range colorCount {
		qc := color.RGBA{
			R: (c.R >> 2) << 2,
			G: (c.G >> 2) << 2,
			B: (c.B >> 2) << 2,
			A: c.A,
		}
		quantizedColors[qc] += count
	}

	// Pick top colors by frequency
	palette := selectTopColors(quantizedColors, numColors)

	return createPalettedImage(img, palette)
}

// selectTopColors returns the most frequent colors.
func selectTopColors(colorCount map[color.RGBA]int, n int) color.Palette {
	// Simple approach: convert to slice and sort by frequency
	type colorFreq struct {
		c     color.RGBA
		count int
	}

	colors := make([]colorFreq, 0, len(colorCount))
	for c, count := range colorCount {
		colors = append(colors, colorFreq{c, count})
	}

	// Sort by frequency (descending) - simple bubble sort is fine for this
	for i := 0; i < len(colors)-1; i++ {
		for j := i + 1; j < len(colors); j++ {
			if colors[j].count > colors[i].count {
				colors[i], colors[j] = colors[j], colors[i]
			}
		}
	}

	// Take top n colors
	palette := make(color.Palette, 0, n)
	for i := 0; i < len(colors) && i < n; i++ {
		palette = append(palette, colors[i].c)
	}

	return palette
}

// createPalettedImage creates a paletted image from the source.
func createPalettedImage(src image.Image, palette color.Palette) *image.Paletted {
	bounds := src.Bounds()
	dst := image.NewPaletted(bounds, palette)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}
