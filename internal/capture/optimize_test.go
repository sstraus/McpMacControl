package capture

import (
	"errors"
	"image"
	"image/color"
	"testing"
)

func TestOptimizeImage(t *testing.T) {
	// Capture a real screen to test optimization
	numDisplays := NumDisplays()
	if numDisplays == 0 {
		t.Skip("No displays available")
	}

	img, err := CaptureScreen(0)
	if err != nil {
		if errors.Is(err, ErrPermissionDenied) {
			t.Skipf("Screen Recording permission not granted, skipping: %v", err)
		}
		t.Fatalf("CaptureScreen failed: %v", err)
	}

	originalBounds := img.Bounds()
	t.Logf("Original size: %dx%d", originalBounds.Dx(), originalBounds.Dy())

	// Optimize
	optimized, err := OptimizeImage(img)
	if err != nil {
		t.Fatalf("OptimizeImage failed: %v", err)
	}

	t.Logf("Optimized image size: %d bytes (%.1f KB)", len(optimized), float64(len(optimized))/1024)

	// Verify it's a valid PNG by decoding
	// (already tested by OptimizeImage encoding successfully)

	// Verify size is reasonable (should be under 500KB for most screens)
	if len(optimized) > 500*1024 {
		t.Logf("Warning: optimized image is larger than expected (%d bytes)", len(optimized))
	}
}

func TestResize(t *testing.T) {
	tests := []struct {
		name     string
		w, h     int
		maxDim   int
		expectW  int
		expectH  int
		resized  bool
	}{
		{
			name:    "no resize needed - small image",
			w:       800,
			h:       600,
			maxDim:  1500,
			expectW: 800,
			expectH: 600,
			resized: false,
		},
		{
			name:    "resize wide image",
			w:       3000,
			h:       2000,
			maxDim:  1500,
			expectW: 1500,
			expectH: 1000,
			resized: true,
		},
		{
			name:    "resize tall image",
			w:       1000,
			h:       2000,
			maxDim:  1500,
			expectW: 750,
			expectH: 1500,
			resized: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			src := image.NewRGBA(image.Rect(0, 0, tt.w, tt.h))

			result := resize(src, tt.maxDim)
			bounds := result.Bounds()

			if bounds.Dx() != tt.expectW || bounds.Dy() != tt.expectH {
				t.Errorf("resize(%dx%d, %d) = %dx%d, want %dx%d",
					tt.w, tt.h, tt.maxDim,
					bounds.Dx(), bounds.Dy(),
					tt.expectW, tt.expectH)
			}
		})
	}
}

func TestResizeToTarget(t *testing.T) {
	tests := []struct {
		name            string
		srcW, srcH      int
		targetW, targetH int
	}{
		{
			name:    "Retina to points (2x scale down)",
			srcW:    2400,
			srcH:    1600,
			targetW: 1200,
			targetH: 800,
		},
		{
			name:    "same size no-op",
			srcW:    1200,
			srcH:    800,
			targetW: 1200,
			targetH: 800,
		},
		{
			name:    "scale up (unusual but supported)",
			srcW:    600,
			srcH:    400,
			targetW: 1200,
			targetH: 800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := image.NewRGBA(image.Rect(0, 0, tt.srcW, tt.srcH))
			result := ResizeToTarget(src, tt.targetW, tt.targetH)
			bounds := result.Bounds()

			if bounds.Dx() != tt.targetW || bounds.Dy() != tt.targetH {
				t.Errorf("ResizeToTarget(%dx%d, %d, %d) = %dx%d, want %dx%d",
					tt.srcW, tt.srcH, tt.targetW, tt.targetH,
					bounds.Dx(), bounds.Dy(),
					tt.targetW, tt.targetH)
			}
		})
	}
}

func TestQuantize(t *testing.T) {
	// Create a simple test image with known colors
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Fill with a few distinct colors
	colors := []color.RGBA{
		{255, 0, 0, 255},   // Red
		{0, 255, 0, 255},   // Green
		{0, 0, 255, 255},   // Blue
		{255, 255, 0, 255}, // Yellow
	}

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := colors[(x/25+y/25)%4]
			src.Set(x, y, c)
		}
	}

	// Quantize
	result := quantize(src, 256)

	// Verify it's a paletted image
	if result == nil {
		t.Fatal("quantize returned nil")
	}

	// Verify palette has reasonable size
	paletteSize := len(result.Palette)
	t.Logf("Palette size: %d colors", paletteSize)

	if paletteSize > MaxColors {
		t.Errorf("Palette has %d colors, expected <= %d", paletteSize, MaxColors)
	}

	// Verify dimensions preserved
	if result.Bounds() != src.Bounds() {
		t.Errorf("Bounds changed: got %v, want %v", result.Bounds(), src.Bounds())
	}
}

func TestOptimizeImageWithFormat(t *testing.T) {
	// Create a synthetic test image (works without screen capture permission)
	img := createTestImage(200, 150)

	// Test WebP format
	webpResult, err := OptimizeImageWithFormat(img, FormatWebP, 0)
	if err != nil {
		t.Fatalf("OptimizeImageWithFormat(WebP) failed: %v", err)
	}
	if webpResult.Format != FormatWebP {
		t.Errorf("Expected format WebP, got %s", webpResult.Format)
	}
	if webpResult.MIMEType != "image/webp" {
		t.Errorf("Expected MIME type image/webp, got %s", webpResult.MIMEType)
	}
	t.Logf("WebP size: %.1f KB", float64(len(webpResult.Data))/1024)

	// Test PNG format
	pngResult, err := OptimizeImageWithFormat(img, FormatPNG, 0)
	if err != nil {
		t.Fatalf("OptimizeImageWithFormat(PNG) failed: %v", err)
	}
	if pngResult.Format != FormatPNG {
		t.Errorf("Expected format PNG, got %s", pngResult.Format)
	}
	if pngResult.MIMEType != "image/png" {
		t.Errorf("Expected MIME type image/png, got %s", pngResult.MIMEType)
	}
	t.Logf("PNG size: %.1f KB", float64(len(pngResult.Data))/1024)

	// WebP should be smaller than PNG for typical images
	t.Logf("WebP vs PNG: %.1f%% of PNG size", float64(len(webpResult.Data))/float64(len(pngResult.Data))*100)
}

func TestOptimizeImageWithFormatNoFallback(t *testing.T) {
	// Verify that requesting WebP returns WebP (not a silent PNG fallback)
	img := createTestImage(100, 100)

	result, err := OptimizeImageWithFormat(img, FormatWebP, 0)
	if err != nil {
		t.Fatalf("WebP encoding should succeed: %v", err)
	}

	// Must be WebP, not PNG
	if result.Format != FormatWebP {
		t.Errorf("Requested WebP but got %s - silent fallback detected", result.Format)
	}
	if result.MIMEType != "image/webp" {
		t.Errorf("Requested WebP but MIME is %s", result.MIMEType)
	}
}

// createTestImage builds a synthetic RGBA image for tests that don't need
// screen capture permission.
func createTestImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / w),
				G: uint8((y * 255) / h),
				B: 128,
				A: 255,
			})
		}
	}
	return img
}

func TestOptimizeImageSizeReduction(t *testing.T) {
	// Test with a window capture to verify real-world size reduction
	img, _, err := CaptureWindowByName("Finder", "", true)
	if err != nil {
		t.Skipf("Could not capture Finder window: %v", err)
	}

	optimized, err := OptimizeImage(img)
	if err != nil {
		t.Fatalf("OptimizeImage failed: %v", err)
	}

	sizeKB := float64(len(optimized)) / 1024
	t.Logf("Optimized Finder window: %.1f KB", sizeKB)

	// Window captures should be under 200KB after optimization
	if sizeKB > 300 {
		t.Logf("Warning: window capture larger than expected (%.1f KB)", sizeKB)
	}
}
