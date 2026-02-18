package input

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
)

func TestParseMouseButton(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected MouseButton
	}{
		{"left explicit", "left", ButtonLeft},
		{"right", "right", ButtonRight},
		{"middle", "middle", ButtonMiddle},
		{"empty defaults to left", "", ButtonLeft},
		{"unknown defaults to left", "invalid", ButtonLeft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseMouseButton(tt.input))
		})
	}
}

func TestIsOnDisplay(t *testing.T) {
	// The machine running tests must have at least one display.
	// Primary display starts at (0,0), so (10,10) should always be on-screen.
	assert.True(t, isOnDisplay(10, 10), "point near origin should be on-screen")

	// A point far off-screen should not be on any display.
	assert.False(t, isOnDisplay(-99999, -99999), "point far off-screen should not be on any display")
}

func TestWindowOverlapsDisplay(t *testing.T) {
	displays := []capture.DisplayBounds{
		{X: 0, Y: 0, Width: 1920, Height: 1080},
	}

	tests := []struct {
		name    string
		window  capture.WindowInfo
		overlap bool
	}{
		{"fully on-screen", capture.WindowInfo{X: 100, Y: 100, Width: 800, Height: 600}, true},
		{"partially on-screen", capture.WindowInfo{X: -400, Y: 100, Width: 800, Height: 600}, true},
		{"fully off-screen left", capture.WindowInfo{X: -900, Y: 100, Width: 800, Height: 600}, false},
		{"fully off-screen right", capture.WindowInfo{X: 2000, Y: 100, Width: 800, Height: 600}, false},
		{"fully off-screen above", capture.WindowInfo{X: 100, Y: -700, Width: 800, Height: 600}, false},
		{"fully off-screen below", capture.WindowInfo{X: 100, Y: 1200, Width: 800, Height: 600}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.overlap, windowOverlapsDisplay(&tt.window, displays))
		})
	}
}
