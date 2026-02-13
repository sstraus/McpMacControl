package capture

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSaveScreenToFile(t *testing.T) {
	numDisplays := NumDisplays()
	if numDisplays == 0 {
		t.Skip("No displays available")
	}

	path, err := SaveScreenToFile(0)
	if err != nil {
		if errors.Is(err, ErrPermissionDenied) {
			t.Skipf("Screen Recording permission not granted, skipping: %v", err)
		}
		t.Fatalf("SaveScreenToFile failed: %v", err)
	}
	defer os.Remove(path)

	t.Logf("Saved screenshot to: %s", path)

	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("File not found: %v", err)
	}

	t.Logf("File size: %.1f KB", float64(info.Size())/1024)

	// Verify it's in temp directory
	if !strings.HasPrefix(path, os.TempDir()) {
		t.Errorf("File not in temp directory: %s", path)
	}

	// Verify it's a supported image format (WebP preferred, PNG fallback)
	if !strings.HasSuffix(path, ".webp") && !strings.HasSuffix(path, ".png") {
		t.Errorf("File is not a supported image format: %s", path)
	}
}

func TestSaveWindowToFile(t *testing.T) {
	path, info, err := SaveWindowToFile("Finder", "", true)
	if err != nil {
		t.Skipf("Could not capture Finder window: %v", err)
	}
	defer os.Remove(path)

	t.Logf("Saved %s - %s to: %s", info.OwnerName, info.Name, path)

	// Verify file exists
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("File not found: %v", err)
	}

	t.Logf("File size: %.1f KB", float64(stat.Size())/1024)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Safari", "Safari"},
		{"Microsoft Teams", "MicrosoftTeams"},
		{"Finder/Desktop", "FinderDesktop"},
		{"App (v2.0)", "Appv20"},
		{"", "window"},
		{"///", "window"},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
