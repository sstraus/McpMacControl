package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestWebPQualityTUICommander captures TUICommander window and saves
// samples at different quality levels for visual comparison.
// Run with: go test -run TestWebPQualityTUICommander -v ./internal/capture/
func TestWebPQualityTUICommander(t *testing.T) {
	dir := "/tmp/webp-quality/tuicommander"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	img, info, err := CaptureWindowByName("TUICommander", "", true)
	if err != nil {
		t.Skipf("Cannot capture TUICommander: %v", err)
	}
	t.Logf("Captured: %s (window %d) %dx%d", info.OwnerName, info.ID, img.Bounds().Dx(), img.Bounds().Dy())

	// Save PNG baseline
	pngResult, err := OptimizeImageWithFormat(img, FormatPNG, 0)
	if err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(dir, "baseline.png")
	if err := os.WriteFile(pngPath, pngResult.Data, 0644); err != nil {
		t.Fatal(err)
	}
	pngKB := float64(len(pngResult.Data)) / 1024

	// Fine-grained sweep focusing on the low range
	qualities := []int{10, 12, 15, 18, 20, 22, 25, 28, 30, 35, 40, 50}
	fmt.Printf("\nTUICommander window: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())
	fmt.Printf("PNG baseline: %.1f KB\n\n", pngKB)
	fmt.Printf("%-8s %10s %10s\n", "Quality", "Size (KB)", "vs PNG")

	for _, q := range qualities {
		result, err := OptimizeImageWithFormat(img, FormatWebP, q)
		if err != nil {
			t.Fatalf("q%d: %v", q, err)
		}
		kb := float64(len(result.Data)) / 1024
		path := filepath.Join(dir, fmt.Sprintf("q%02d.webp", q))
		if err := os.WriteFile(path, result.Data, 0644); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("q%-7d %9.1f %9.1f%%\n", q, kb, kb/pngKB*100)
	}

	fmt.Printf("\nFiles saved to %s\n", dir)
}
