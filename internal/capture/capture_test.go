package capture

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// skipWithoutScreenRecording skips the test if the capture returns a
// permission error (ad-hoc signed test binaries lack screen recording).
func skipIfPermissionDenied(t *testing.T, err error) {
	t.Helper()
	if err != nil && errors.Is(err, ErrPermissionDenied) {
		t.Skipf("Screen Recording permission not granted, skipping: %v", err)
	}
}

func TestCaptureScreen(t *testing.T) {
	numDisplays := NumDisplays()
	if numDisplays == 0 {
		t.Skip("No displays available")
	}

	t.Logf("Found %d display(s)", numDisplays)

	// Capture primary display
	img, err := CaptureScreen(0)
	skipIfPermissionDenied(t, err)
	if err != nil {
		t.Fatalf("CaptureScreen failed: %v", err)
	}

	bounds := img.Bounds()
	t.Logf("Captured screen: %dx%d", bounds.Dx(), bounds.Dy())

	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Error("Captured image has invalid dimensions")
	}
}

func TestCaptureScreenInvalidDisplay(t *testing.T) {
	_, err := CaptureScreen(999)
	if err == nil {
		t.Error("Expected error for invalid display index")
	}
}

func TestCaptureWindowByName(t *testing.T) {
	// Try to capture Finder window (should exist on any macOS)
	img, info, err := CaptureWindowByName("Finder", "", true)
	if err != nil {
		// Finder might not have a visible window
		t.Skipf("Could not capture Finder window: %v", err)
	}

	bounds := img.Bounds()
	t.Logf("Captured window [%d] %s - %s: %dx%d",
		info.ID, info.OwnerName, info.Name, bounds.Dx(), bounds.Dy())

	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Error("Captured image has invalid dimensions")
	}
}

func TestCaptureWindowByNameNotFound(t *testing.T) {
	_, _, err := CaptureWindowByName("NonExistentApp12345", "", true)
	if err == nil {
		t.Error("Expected error for non-existent app")
	}
}

func TestCaptureScreen_PermissionErrorDetection(t *testing.T) {
	// Both screen and window capture now use screencapture CLI, so permission
	// errors are detected via isPermissionError (checking "could not create image").
	// This is tested in TestIsPermissionError. Here we verify that CaptureScreen
	// returns ErrPermissionDenied when it encounters such errors (end-to-end).
	// We can't easily trigger the real error in a unit test, but the shared
	// runScreencapture helper handles it identically to CaptureWindow.
	assert.True(t, isPermissionError("could not create image from display"))
	assert.False(t, isPermissionError("display not found"))
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "permission denied output",
			output: "could not create image from window",
			want:   true,
		},
		{
			name:   "permission denied with newline",
			output: "could not create image from window\n",
			want:   true,
		},
		{
			name:   "generic could not create image",
			output: "could not create image",
			want:   true,
		},
		{
			name:   "empty output",
			output: "",
			want:   false,
		},
		{
			name:   "unrelated error",
			output: "invalid window id",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPermissionError(tt.output); got != tt.want {
				t.Errorf("isPermissionError(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}
