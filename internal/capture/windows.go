package capture

import (
	"fmt"
	"log"
	"strings"
)

// WindowInfo contains information about a macOS window.
type WindowInfo struct {
	ID        uint32 // CGWindowID
	OwnerPID  int    // Process ID of the owning application
	OwnerName string // Application name (e.g., "Safari")
	Name      string // Window title (e.g., "Welcome to Safari")
	X         int    // Window X position
	Y         int    // Window Y position
	Width     int    // Window width
	Height    int    // Window height
	OnScreen  bool   // Whether window is currently on screen
	Layer     int    // Window layer (0 = normal windows)
	Focused   bool   // Whether the owning app is the frontmost application
}

// ListWindows returns all visible windows on the system.
// The filter parameter is optional; if provided, only windows with matching
// owner name or window title (case-insensitive substring) are returned.
func ListWindows(filter string) ([]WindowInfo, error) {
	return listWindowsDarwin(filter)
}

// FindWindow finds the best matching window for appName.
// Two-pass: exact owner name match (case-insensitive) first,
// title substring match second. Prefers windows overlapping
// an active display.
func FindWindow(appName string) (*WindowInfo, error) {
	return FindWindowWithTitle(appName, "")
}

// FindWindowWithTitle is like FindWindow but also requires
// the window title to contain titleFilter (case-insensitive).
func FindWindowWithTitle(appName, titleFilter string) (*WindowInfo, error) {
	windows, err := ListWindows(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	appNameLower := strings.ToLower(appName)
	titleFilterLower := strings.ToLower(titleFilter)

	var matches []WindowInfo

	// Pass 1: exact owner name match (case-insensitive)
	for i := range windows {
		w := &windows[i]
		if !strings.EqualFold(w.OwnerName, appName) {
			continue
		}
		if titleFilter != "" && !strings.Contains(strings.ToLower(w.Name), titleFilterLower) {
			continue
		}
		matches = append(matches, *w)
	}

	// Pass 2: title substring match (only if no owner matches found)
	if len(matches) == 0 {
		for i := range windows {
			w := &windows[i]
			if !strings.Contains(strings.ToLower(w.Name), appNameLower) {
				continue
			}
			if titleFilter != "" && !strings.Contains(strings.ToLower(w.Name), titleFilterLower) {
				continue
			}
			matches = append(matches, *w)
		}
	}

	if len(matches) == 0 {
		if titleFilter != "" {
			return nil, fmt.Errorf("no window found for app %q with title containing %q", appName, titleFilter)
		}
		return nil, fmt.Errorf("no window found for app %q", appName)
	}

	// Prefer a window that overlaps an active display
	displays := ActiveDisplayBounds()
	for i := range matches {
		if windowOverlapsDisplay(&matches[i], displays) {
			return &matches[i], nil
		}
	}

	// Fall back to first match if none overlap a display
	log.Printf("FindWindow: no window for %q overlaps an active display; using off-screen window at (%d,%d)", appName, matches[0].X, matches[0].Y)
	return &matches[0], nil
}

// windowOverlapsDisplay returns true if any part of the window overlaps any
// active display.
func windowOverlapsDisplay(w *WindowInfo, displays []DisplayBounds) bool {
	for _, d := range displays {
		if w.X < d.X+d.Width && w.X+w.Width > d.X &&
			w.Y < d.Y+d.Height && w.Y+w.Height > d.Y {
			return true
		}
	}
	return false
}
