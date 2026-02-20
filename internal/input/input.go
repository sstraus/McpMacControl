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

// windowRect holds position and size of a window.
type windowRect struct {
	x, y, width, height int
}

// checkWindowBounds validates that relative coordinates fall within the window.
func checkWindowBounds(w windowRect, relX, relY int) error {
	if relX < 0 || relX >= w.width || relY < 0 || relY >= w.height {
		return fmt.Errorf("coordinates (%d,%d) are outside window bounds (0,0)–(%d,%d)", relX, relY, w.width-1, w.height-1)
	}
	return nil
}

// findWindow returns the position and size of the best matching window for
// appName. It uses a two-pass match (owner name first, then window title) and
// prefers windows whose bounds overlap an active display.
func findWindow(appName string) (windowRect, error) {
	windows, err := capture.ListWindows(appName)
	if err != nil {
		return windowRect{}, fmt.Errorf("failed to list windows: %w", err)
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
		return windowRect{}, fmt.Errorf("window not found for app: %s", appName)
	}

	toRect := func(w *capture.WindowInfo) windowRect {
		return windowRect{x: w.X, y: w.Y, width: w.Width, height: w.Height}
	}

	// Prefer a window that overlaps an active display.
	for i := range matches {
		if windowOverlapsDisplay(&matches[i], displays) {
			return toRect(&matches[i]), nil
		}
	}

	// Fall back to first match if none overlap a display.
	return toRect(&matches[0]), nil
}

// MoveMouseToWindow moves the mouse to a position relative to a window.
// Returns the absolute screen coordinates used.
func MoveMouseToWindow(appName string, relX, relY int) (absX, absY int, err error) {
	w, err := findWindow(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("move mouse: %w", err)
	}
	if err := checkWindowBounds(w, relX, relY); err != nil {
		return 0, 0, fmt.Errorf("move mouse: %w", err)
	}

	absX, absY = w.x+relX, w.y+relY
	MoveMouse(absX, absY)
	return absX, absY, nil
}

// ClickInWindow clicks at a position relative to a window.
// Returns the absolute screen coordinates used.
func ClickInWindow(appName string, relX, relY int, button MouseButton, doubleClick bool) (absX, absY int, err error) {
	w, err := findWindow(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("click: %w", err)
	}
	if err := checkWindowBounds(w, relX, relY); err != nil {
		return 0, 0, fmt.Errorf("click: %w", err)
	}

	absX, absY = w.x+relX, w.y+relY
	ClickMouse(absX, absY, int(button), doubleClick)
	return absX, absY, nil
}

// DragInWindow performs a drag from one window-relative position to another.
// Returns the absolute screen coordinates of both endpoints.
func DragInWindow(appName string, fromRelX, fromRelY, toRelX, toRelY int) (fromAbsX, fromAbsY, toAbsX, toAbsY int, err error) {
	w, err := findWindow(appName)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("drag: %w", err)
	}
	if err := checkWindowBounds(w, fromRelX, fromRelY); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("drag start: %w", err)
	}
	if err := checkWindowBounds(w, toRelX, toRelY); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("drag end: %w", err)
	}

	fromAbsX, fromAbsY = w.x+fromRelX, w.y+fromRelY
	toAbsX, toAbsY = w.x+toRelX, w.y+toRelY
	DragMouse(fromAbsX, fromAbsY, toAbsX, toAbsY)
	return fromAbsX, fromAbsY, toAbsX, toAbsY, nil
}
