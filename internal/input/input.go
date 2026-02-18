package input

import (
	"fmt"
	"strings"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
)

// MouseButton represents a mouse button.
type MouseButton int

const (
	ButtonLeft   MouseButton = 0
	ButtonRight  MouseButton = 1
	ButtonMiddle MouseButton = 2
)

// ParseMouseButton converts a string to MouseButton.
func ParseMouseButton(s string) MouseButton {
	switch s {
	case "right":
		return ButtonRight
	case "middle":
		return ButtonMiddle
	default:
		return ButtonLeft
	}
}

// isOnDisplay returns true if the point (x, y) falls within any active display.
func isOnDisplay(x, y int) bool {
	for _, d := range capture.ActiveDisplayBounds() {
		if x >= d.X && x < d.X+d.Width && y >= d.Y && y < d.Y+d.Height {
			return true
		}
	}
	return false
}

// windowOverlapsDisplay returns true if any part of the window overlaps any
// active display.
func windowOverlapsDisplay(w *capture.WindowInfo, displays []capture.DisplayBounds) bool {
	for _, d := range displays {
		if w.X < d.X+d.Width && w.X+w.Width > d.X &&
			w.Y < d.Y+d.Height && w.Y+w.Height > d.Y {
			return true
		}
	}
	return false
}

// windowOrigin returns the top-left screen coordinates of the best matching
// window for appName. It uses a two-pass match (owner name first, then window
// title) and prefers windows whose bounds overlap an active display.
func windowOrigin(appName string) (x, y int, err error) {
	windows, err := capture.ListWindows(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list windows: %w", err)
	}

	appNameLower := strings.ToLower(appName)
	displays := capture.ActiveDisplayBounds()

	// Collect matches in two passes: owner name first, title second.
	var matches []capture.WindowInfo

	// Pass 1: exact owner name match (case-insensitive)
	for i := range windows {
		if strings.EqualFold(windows[i].OwnerName, appName) {
			matches = append(matches, windows[i])
		}
	}
	// Pass 2: title substring match (only if no owner matches found)
	if len(matches) == 0 {
		for i := range windows {
			if strings.Contains(strings.ToLower(windows[i].Name), appNameLower) {
				matches = append(matches, windows[i])
			}
		}
	}

	if len(matches) == 0 {
		return 0, 0, fmt.Errorf("window not found for app: %s", appName)
	}

	// Prefer a window that overlaps an active display.
	for i := range matches {
		if windowOverlapsDisplay(&matches[i], displays) {
			return matches[i].X, matches[i].Y, nil
		}
	}

	// Fall back to first match if none overlap a display.
	return matches[0].X, matches[0].Y, nil
}

// MoveMouseToWindow moves the mouse to a position relative to a window.
// Returns the absolute screen coordinates used.
func MoveMouseToWindow(appName string, relX, relY int) (absX, absY int, err error) {
	ox, oy, err := windowOrigin(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("move mouse: %w", err)
	}

	absX, absY = ox+relX, oy+relY
	if !isOnDisplay(absX, absY) {
		return 0, 0, fmt.Errorf("target coordinates (%d,%d) are outside all displays — window may be off-screen", absX, absY)
	}
	MoveMouse(absX, absY)
	return absX, absY, nil
}

// ClickInWindow clicks at a position relative to a window.
// Returns the absolute screen coordinates used.
func ClickInWindow(appName string, relX, relY int, button MouseButton, doubleClick bool) (absX, absY int, err error) {
	ox, oy, err := windowOrigin(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("click: %w", err)
	}

	absX, absY = ox+relX, oy+relY
	if !isOnDisplay(absX, absY) {
		return 0, 0, fmt.Errorf("target coordinates (%d,%d) are outside all displays — window may be off-screen", absX, absY)
	}
	ClickMouse(absX, absY, int(button), doubleClick)
	return absX, absY, nil
}

// DragInWindow performs a drag from one window-relative position to another.
// Returns the absolute screen coordinates of both endpoints.
func DragInWindow(appName string, fromRelX, fromRelY, toRelX, toRelY int) (fromAbsX, fromAbsY, toAbsX, toAbsY int, err error) {
	ox, oy, err := windowOrigin(appName)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("drag: %w", err)
	}

	fromAbsX, fromAbsY = ox+fromRelX, oy+fromRelY
	toAbsX, toAbsY = ox+toRelX, oy+toRelY
	if !isOnDisplay(fromAbsX, fromAbsY) {
		return 0, 0, 0, 0, fmt.Errorf("drag start coordinates (%d,%d) are outside all displays — window may be off-screen", fromAbsX, fromAbsY)
	}
	if !isOnDisplay(toAbsX, toAbsY) {
		return 0, 0, 0, 0, fmt.Errorf("drag end coordinates (%d,%d) are outside all displays — window may be off-screen", toAbsX, toAbsY)
	}
	DragMouse(fromAbsX, fromAbsY, toAbsX, toAbsY)
	return fromAbsX, fromAbsY, toAbsX, toAbsY, nil
}
