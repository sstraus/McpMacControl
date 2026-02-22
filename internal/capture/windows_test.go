package capture

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWindows(t *testing.T) {
	windows, err := ListWindows("")
	if err != nil {
		t.Fatalf("ListWindows failed: %v", err)
	}

	// Should have at least one window (this test runner)
	if len(windows) == 0 {
		t.Log("Warning: no windows found. This might be expected in headless CI environment.")
		return
	}

	t.Logf("Found %d windows:", len(windows))
	for _, w := range windows {
		t.Logf("  [%d] %s - %s (%dx%d at %d,%d)",
			w.ID, w.OwnerName, w.Name, w.Width, w.Height, w.X, w.Y)
	}

	// Verify window properties are populated
	for _, w := range windows {
		if w.ID == 0 {
			t.Error("Window has zero ID")
		}
		if w.OwnerName == "" {
			t.Error("Window has empty owner name")
		}
		if w.Width <= 0 || w.Height <= 0 {
			t.Errorf("Window has invalid size: %dx%d", w.Width, w.Height)
		}
	}
}

func TestListWindowsWithFilter(t *testing.T) {
	// Filter for Finder (should exist on any macOS)
	windows, err := ListWindows("Finder")
	if err != nil {
		t.Fatalf("ListWindows with filter failed: %v", err)
	}

	// All returned windows should match the filter on owner name or title
	for _, w := range windows {
		ownerMatch := strings.Contains(strings.ToLower(w.OwnerName), "finder")
		titleMatch := strings.Contains(strings.ToLower(w.Name), "finder")
		if !ownerMatch && !titleMatch {
			t.Errorf("Window %q (title %q) does not match filter 'Finder' on owner or title", w.OwnerName, w.Name)
		}
	}

	t.Logf("Found %d Finder windows", len(windows))
}

func TestListWindowsFilterMatchesTitle(t *testing.T) {
	// Get all windows to find one with a non-empty title different from owner
	allWindows, err := ListWindows("")
	if err != nil {
		t.Fatalf("ListWindows failed: %v", err)
	}

	// Find a window whose title is non-empty and not a substring of owner name
	var titleToSearch string
	var expectedOwner string
	for _, w := range allWindows {
		if w.Name == "" {
			continue
		}
		if strings.Contains(strings.ToLower(w.OwnerName), strings.ToLower(w.Name)) {
			continue
		}
		titleToSearch = w.Name
		expectedOwner = w.OwnerName
		break
	}

	if titleToSearch == "" {
		t.Skip("No window found with a title distinct from its owner name")
	}

	t.Logf("Filtering by title %q (owner: %q)", titleToSearch, expectedOwner)

	// Filter by title — should return the window
	filtered, err := ListWindows(titleToSearch)
	if err != nil {
		t.Fatalf("ListWindows with title filter failed: %v", err)
	}

	found := false
	for _, w := range filtered {
		if strings.Contains(strings.ToLower(w.Name), strings.ToLower(titleToSearch)) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListWindows(%q) did not return a window with matching title", titleToSearch)
	}
}

func TestFindWindow_NotFound(t *testing.T) {
	_, err := FindWindow("NonExistentApp_XYZZY_12345")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no window found")
}

func TestFindWindow_MatchesOwner(t *testing.T) {
	// Finder should always exist on macOS
	w, err := FindWindow("Finder")
	if err != nil {
		t.Skip("Finder window not found; might be headless CI")
	}
	assert.Equal(t, "Finder", w.OwnerName)
	assert.Greater(t, w.OwnerPID, 0)
}

func TestFindWindow_CaseInsensitive(t *testing.T) {
	w, err := FindWindow("finder")
	if err != nil {
		t.Skip("Finder window not found; might be headless CI")
	}
	assert.Equal(t, "Finder", w.OwnerName)
}

func TestFindWindowWithTitle_NotFound(t *testing.T) {
	_, err := FindWindowWithTitle("Finder", "NonExistentTitle_XYZZY_12345")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no window found")
}

func TestFindWindowWithTitle_EmptyTitleMatchesAny(t *testing.T) {
	// Empty title filter should behave like FindWindow
	w, err := FindWindowWithTitle("Finder", "")
	if err != nil {
		t.Skip("Finder window not found; might be headless CI")
	}
	assert.Equal(t, "Finder", w.OwnerName)
}

func TestWindowOverlapsDisplay(t *testing.T) {
	displays := []DisplayBounds{
		{X: 0, Y: 0, Width: 1920, Height: 1080},
	}

	tests := []struct {
		name    string
		window  WindowInfo
		overlap bool
	}{
		{"fully on-screen", WindowInfo{X: 100, Y: 100, Width: 800, Height: 600}, true},
		{"partially on-screen", WindowInfo{X: -400, Y: 100, Width: 800, Height: 600}, true},
		{"fully off-screen left", WindowInfo{X: -900, Y: 100, Width: 800, Height: 600}, false},
		{"fully off-screen right", WindowInfo{X: 2000, Y: 100, Width: 800, Height: 600}, false},
		{"fully off-screen above", WindowInfo{X: 100, Y: -700, Width: 800, Height: 600}, false},
		{"fully off-screen below", WindowInfo{X: 100, Y: 1200, Width: 800, Height: 600}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.overlap, windowOverlapsDisplay(&tt.window, displays))
		})
	}
}
