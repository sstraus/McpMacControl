package input

import (
	"fmt"

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

// windowOrigin returns the top-left screen coordinates of the first window
// matching appName.
func windowOrigin(appName string) (x, y int, err error) {
	windows, err := capture.ListWindows(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list windows: %w", err)
	}

	for i := range windows {
		if windows[i].OwnerName == appName {
			return windows[i].X, windows[i].Y, nil
		}
	}

	return 0, 0, fmt.Errorf("window not found for app: %s", appName)
}

// MoveMouseToWindow moves the mouse to a position relative to a window.
// Returns the absolute screen coordinates used.
func MoveMouseToWindow(appName string, relX, relY int) (absX, absY int, err error) {
	ox, oy, err := windowOrigin(appName)
	if err != nil {
		return 0, 0, fmt.Errorf("move mouse: %w", err)
	}

	absX, absY = ox+relX, oy+relY
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
	DragMouse(fromAbsX, fromAbsY, toAbsX, toAbsY)
	return fromAbsX, fromAbsY, toAbsX, toAbsY, nil
}
