package capture

import (
	"testing"
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

	// All returned windows should match the filter
	for _, w := range windows {
		if w.OwnerName != "Finder" {
			t.Errorf("Window owner %q does not match filter 'Finder'", w.OwnerName)
		}
	}

	t.Logf("Found %d Finder windows", len(windows))
}
