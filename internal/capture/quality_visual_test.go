package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestWebPVisualSamples writes sample images at different quality levels
// to /tmp/webp-quality/ for visual inspection.
// Run with: go test -run TestWebPVisualSamples -v ./internal/capture/
func TestWebPVisualSamples(t *testing.T) {
	dir := "/tmp/webp-quality"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	img := createDetailedUIImage(2560, 1600)

	// Save PNG baseline
	pngResult, err := OptimizeImageWithFormat(img, FormatPNG, 0)
	if err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(dir, "baseline.png")
	if err := os.WriteFile(pngPath, pngResult.Data, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("PNG: %s (%.1f KB)", pngPath, float64(len(pngResult.Data))/1024)

	// Save WebP at various quality levels
	qualities := []int{15, 25, 30, 35, 40, 50, 60, 75}
	for _, q := range qualities {
		result, err := OptimizeImageWithFormat(img, FormatWebP, q)
		if err != nil {
			t.Fatalf("q%d: %v", q, err)
		}
		path := filepath.Join(dir, fmt.Sprintf("q%02d.webp", q))
		if err := os.WriteFile(path, result.Data, 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("q%d: %s (%.1f KB)", q, path, float64(len(result.Data))/1024)
	}

	// Also try with a real screenshot if possible
	captured, err := CaptureScreen(0)
	if err != nil {
		t.Logf("No real screenshot available: %v", err)
		t.Logf("\nFiles saved to %s - open them to compare visually", dir)
		return
	}

	t.Logf("\n--- Real screenshot %dx%d ---", captured.Bounds().Dx(), captured.Bounds().Dy())
	realDir := filepath.Join(dir, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}

	pngResult, err = OptimizeImageWithFormat(captured, FormatPNG, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "baseline.png"), pngResult.Data, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("PNG: %.1f KB", float64(len(pngResult.Data))/1024)

	for _, q := range qualities {
		result, err := OptimizeImageWithFormat(captured, FormatWebP, q)
		if err != nil {
			t.Fatalf("q%d: %v", q, err)
		}
		path := filepath.Join(realDir, fmt.Sprintf("q%02d.webp", q))
		if err := os.WriteFile(path, result.Data, 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("q%d: %.1f KB", q, float64(len(result.Data))/1024)
	}

	t.Logf("\nFiles saved to %s - open them to compare visually", dir)
}
