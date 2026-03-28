package capture

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

// TestWebPQualitySweep compares WebP output size at multiple quality levels
// to find the optimal tradeoff between readability and file size.
// Run with: go test -run TestWebPQualitySweep -v ./internal/capture/
func TestWebPQualitySweep(t *testing.T) {
	img := createDetailedUIImage(1500, 1000)

	// Also test with PNG as baseline
	pngResult, err := OptimizeImageWithFormat(img, FormatPNG, 0)
	if err != nil {
		t.Fatalf("PNG encoding failed: %v", err)
	}
	pngKB := float64(len(pngResult.Data)) / 1024

	qualities := []int{10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 75}
	t.Logf("PNG baseline: %.1f KB", pngKB)
	t.Logf("")
	t.Logf("%-8s %10s %10s", "Quality", "Size (KB)", "vs PNG")

	for _, q := range qualities {
		result, err := OptimizeImageWithFormat(img, FormatWebP, q)
		if err != nil {
			t.Fatalf("WebP q%d failed: %v", q, err)
		}
		kb := float64(len(result.Data)) / 1024
		ratio := kb / pngKB * 100
		t.Logf("q%-7d %9.1f %9.1f%%", q, kb, ratio)
	}
}

// createDetailedUIImage builds a synthetic image with fine details
// that mimic real UI elements: thin borders, small text-like patterns,
// icons, toolbar buttons, and high-frequency detail areas.
func createDetailedUIImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// Light gray background
	fill(img, 0, 0, w, h, color.RGBA{R: 242, G: 242, B: 245, A: 255})

	// Dark sidebar with thin divider
	fill(img, 0, 0, 250, h, color.RGBA{R: 45, G: 45, B: 55, A: 255})
	fill(img, 249, 0, 251, h, color.RGBA{R: 80, G: 80, B: 90, A: 255}) // 2px divider

	// Title bar with traffic lights
	fill(img, 0, 0, w, 28, color.RGBA{R: 55, G: 55, B: 65, A: 255})
	drawCircle(img, 20, 14, 6, color.RGBA{R: 255, G: 95, B: 86, A: 255})  // close
	drawCircle(img, 38, 14, 6, color.RGBA{R: 255, G: 189, B: 46, A: 255}) // minimize
	drawCircle(img, 56, 14, 6, color.RGBA{R: 39, G: 201, B: 63, A: 255})  // zoom

	// Tab bar with thin borders (simulates browser/editor tabs)
	fill(img, 250, 28, w, 58, color.RGBA{R: 235, G: 235, B: 238, A: 255})
	for i := 0; i < 5; i++ {
		x0 := 260 + i*150
		// Active tab is white, others are gray
		tabColor := color.RGBA{R: 210, G: 210, B: 215, A: 255}
		if i == 0 {
			tabColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		fill(img, x0, 32, x0+140, 57, tabColor)
		// 1px border on tabs
		fill(img, x0, 32, x0+140, 33, color.RGBA{R: 180, G: 180, B: 185, A: 255})
		fill(img, x0, 32, x0+1, 57, color.RGBA{R: 180, G: 180, B: 185, A: 255})
		fill(img, x0+139, 32, x0+140, 57, color.RGBA{R: 180, G: 180, B: 185, A: 255})
		// Tab text (small blocks simulating text)
		drawTextBlock(img, x0+8, 40, 80, 10, color.RGBA{R: 60, G: 60, B: 70, A: 255})
		// Close X on tab (tiny 6x6 pattern)
		drawTinyX(img, x0+125, 40)
	}

	// Toolbar with small icons
	fill(img, 250, 58, w, 88, color.RGBA{R: 248, G: 248, B: 250, A: 255})
	fill(img, 250, 87, w, 88, color.RGBA{R: 210, G: 210, B: 215, A: 255}) // 1px divider
	// Toolbar icons (small shapes)
	for i := 0; i < 8; i++ {
		x := 270 + i*36
		drawSmallIcon(img, x, 63, 20, color.RGBA{R: 100, G: 100, B: 110, A: 255})
	}

	// Sidebar items (small text with icons)
	for i := 0; i < 30; i++ {
		y0 := 40 + i*24
		// Folder/file icon (small triangle or square)
		if i%3 == 0 {
			drawSmallTriangle(img, 15, y0+4, 10, color.RGBA{R: 150, G: 150, B: 170, A: 255})
		} else {
			fill(img, 15, y0+4, 25, y0+14, color.RGBA{R: 120, G: 120, B: 140, A: 255})
		}
		// File name text
		nameLen := 80 + (i*13)%120
		drawTextBlock(img, 30, y0+4, nameLen, 10, color.RGBA{R: 200, G: 200, B: 210, A: 255})
	}

	// Main content: code-like lines with syntax coloring
	for line := 0; line < 35; line++ {
		y0 := 96 + line*24
		if y0+14 >= h {
			break
		}

		// Line number (small, low contrast)
		drawTextBlock(img, 260, y0+2, 30, 10, color.RGBA{R: 180, G: 180, B: 190, A: 255})

		// Indentation varies
		indent := (line % 5) * 20
		x := 300 + indent

		// Keyword in blue
		drawTextBlock(img, x, y0+2, 40, 10, color.RGBA{R: 50, G: 100, B: 200, A: 255})
		x += 48

		// Identifier in dark
		identLen := 60 + (line*11)%100
		drawTextBlock(img, x, y0+2, identLen, 10, color.RGBA{R: 40, G: 40, B: 50, A: 255})
		x += identLen + 8

		// String literal in green (every other line)
		if line%2 == 0 {
			strLen := 80 + (line*7)%120
			drawTextBlock(img, x, y0+2, strLen, 10, color.RGBA{R: 30, G: 140, B: 60, A: 255})
			x += strLen + 8
		}

		// Comment in gray (some lines)
		if line%4 == 0 {
			drawTextBlock(img, x, y0+2, 120, 10, color.RGBA{R: 140, G: 140, B: 150, A: 255})
		}
	}

	// Keyboard shortcut hints (simulates ⌘⌥⇧ style badges)
	shortcuts := []struct{ x, y int }{
		{1200, 100}, {1200, 130}, {1200, 160}, {1200, 190},
		{1200, 220}, {1200, 250}, {1200, 280},
	}
	for _, s := range shortcuts {
		drawShortcutBadge(img, s.x, s.y)
	}

	// Status bar at bottom with small text
	fill(img, 0, h-24, w, h, color.RGBA{R: 38, G: 38, B: 48, A: 255})
	for i := 0; i < 6; i++ {
		x := 10 + i*200
		drawTextBlock(img, x, h-18, 60+i*15, 10, color.RGBA{R: 180, G: 180, B: 200, A: 255})
	}

	// Minimap on right edge (tiny colored lines)
	for y := 96; y < h-24; y += 2 {
		lineLen := 20 + (y*3)%40
		c := color.RGBA{
			R: uint8(100 + (y*2)%80),
			G: uint8(100 + (y*3)%80),
			B: uint8(120 + (y*5)%80),
			A: 255,
		}
		for x := w - 60; x < w-60+lineLen && x < w; x++ {
			img.Set(x, y, c)
		}
	}

	return img
}

func fill(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.Set(x, y, c)
		}
	}
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, c)
			}
		}
	}
}

// drawTextBlock draws a block of alternating pixels to simulate text rendering.
func drawTextBlock(img *image.RGBA, x0, y0, w, h int, c color.RGBA) {
	bg := color.RGBA{A: 0} // transparent (won't be drawn)
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			// Simulate character shapes with varying density
			charPos := (x - x0) % 7
			linePos := y - y0
			// Create gaps between "characters" and within "glyphs"
			if charPos == 6 { // gap between chars
				continue
			}
			// Simulate ascenders/descenders and varying glyph shapes
			draw := false
			switch {
			case linePos >= 2 && linePos <= 8: // main body
				draw = (charPos+linePos)%3 != 0
			case linePos <= 1: // ascender
				draw = charPos >= 1 && charPos <= 3 && (x-x0)/7%3 == 0
			case linePos >= 9: // descender
				draw = charPos >= 2 && charPos <= 4 && (x-x0)/7%4 == 0
			}
			if draw {
				_ = bg
				img.Set(x, y, c)
			}
		}
	}
}

func drawTinyX(img *image.RGBA, cx, cy int) {
	c := color.RGBA{R: 140, G: 140, B: 150, A: 255}
	for i := 0; i < 6; i++ {
		img.Set(cx+i, cy+i, c)
		img.Set(cx+5-i, cy+i, c)
	}
}

func drawSmallIcon(img *image.RGBA, x, y, size int, c color.RGBA) {
	// Draw a small shape (varies by position for variety)
	variant := (x / 36) % 4
	switch variant {
	case 0: // filled square
		fill(img, x, y, x+size, y+size, c)
	case 1: // hollow square
		fill(img, x, y, x+size, y+1, c)
		fill(img, x, y+size-1, x+size, y+size, c)
		fill(img, x, y, x+1, y+size, c)
		fill(img, x+size-1, y, x+size, y+size, c)
	case 2: // circle
		drawCircle(img, x+size/2, y+size/2, size/2, c)
	case 3: // triangle
		drawSmallTriangle(img, x, y, size, c)
	}
}

func drawSmallTriangle(img *image.RGBA, x, y, size int, c color.RGBA) {
	for row := 0; row < size; row++ {
		halfWidth := row * size / (2 * size)
		cx := x + size/2
		for dx := -halfWidth; dx <= halfWidth; dx++ {
			img.Set(cx+dx, y+row, c)
		}
	}
}

// drawShortcutBadge simulates a keyboard shortcut badge like [⌘ ⇧ P]
func drawShortcutBadge(img *image.RGBA, x, y int) {
	badgeW := 80
	badgeH := 18

	// Badge background (rounded rect approximation)
	fill(img, x, y, x+badgeW, y+badgeH, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	// 1px border
	fill(img, x, y, x+badgeW, y+1, color.RGBA{R: 190, G: 190, B: 195, A: 255})
	fill(img, x, y+badgeH-1, x+badgeW, y+badgeH, color.RGBA{R: 190, G: 190, B: 195, A: 255})
	fill(img, x, y, x+1, y+badgeH, color.RGBA{R: 190, G: 190, B: 195, A: 255})
	fill(img, x+badgeW-1, y, x+badgeW, y+badgeH, color.RGBA{R: 190, G: 190, B: 195, A: 255})

	// Glyph-like patterns inside (simulating ⌘, ⇧, etc.)
	drawGlyphPattern(img, x+6, y+3, color.RGBA{R: 60, G: 60, B: 70, A: 255})
	drawGlyphPattern(img, x+26, y+3, color.RGBA{R: 60, G: 60, B: 70, A: 255})
	// Letter
	drawTextBlock(img, x+46, y+4, 12, 10, color.RGBA{R: 60, G: 60, B: 70, A: 255})
}

// drawGlyphPattern draws a small intricate pattern simulating a modifier key glyph.
func drawGlyphPattern(img *image.RGBA, x, y int, c color.RGBA) {
	// 12x12 pattern with fine detail (like ⌘ symbol)
	pattern := []string{
		"..####..",
		".#....#.",
		"##.##.##",
		"#..##..#",
		"#..##..#",
		"##.##.##",
		".#....#.",
		"..####..",
	}
	for dy, row := range pattern {
		for dx, ch := range row {
			if ch == '#' {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}

func TestWebPQualitySweepWithRealScreenshot(t *testing.T) {
	// This test captures an actual screenshot if permissions allow,
	// otherwise falls back to the synthetic image.
	var img image.Image

	captured, err := CaptureScreen(0)
	if err != nil {
		t.Logf("Cannot capture real screenshot (%v), using synthetic image", err)
		img = createDetailedUIImage(2560, 1600)
	} else {
		img = captured
		t.Logf("Using real screenshot: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	pngResult, err := OptimizeImageWithFormat(img, FormatPNG, 0)
	if err != nil {
		t.Fatalf("PNG encoding failed: %v", err)
	}
	pngKB := float64(len(pngResult.Data)) / 1024

	qualities := []int{10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 75, 85}
	fmt.Printf("\nPNG baseline: %.1f KB\n", pngKB)
	fmt.Printf("%-8s %10s %10s %10s\n", "Quality", "Size (KB)", "vs PNG", "vs q15")

	var q15Size float64
	for _, q := range qualities {
		result, err := OptimizeImageWithFormat(img, FormatWebP, q)
		if err != nil {
			t.Fatalf("WebP q%d failed: %v", q, err)
		}
		kb := float64(len(result.Data)) / 1024
		if q == 15 {
			q15Size = kb
		}
		vsQ15 := ""
		if q15Size > 0 {
			vsQ15 = fmt.Sprintf("%.1fx", kb/q15Size)
		}
		fmt.Printf("q%-7d %9.1f %9.1f%% %10s\n", q, kb, kb/pngKB*100, vsQ15)
	}
}
