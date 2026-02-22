package capture

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrPermissionDenied indicates screen recording permission is not granted.
var ErrPermissionDenied = errors.New("screen recording permission denied")

// runScreencapture executes screencapture with the given args (excluding the
// output file path, which is handled internally). Returns the decoded image.
// Handles temp file creation, permission error detection, PNG decode, and cleanup.
func runScreencapture(args []string) (image.Image, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("mcpmaccontrol_%d.png", os.Getpid()))
	defer os.Remove(tmpFile)

	fullArgs := append(args, tmpFile)
	cmd := exec.Command("screencapture", fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if isPermissionError(string(output)) {
			return nil, fmt.Errorf("%w: screencapture cannot access contents", ErrPermissionDenied)
		}
		return nil, fmt.Errorf("screencapture failed: %w (output: %s)", err, string(output))
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open captured image: %w", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode captured image: %w", err)
	}

	return img, nil
}

// CaptureWindow captures a specific window by its window ID.
// hideShadow controls whether the window shadow is included.
func CaptureWindow(windowID uint32, hideShadow bool) (image.Image, error) {
	args := []string{"-l", fmt.Sprintf("%d", windowID), "-x"}
	if hideShadow {
		args = append(args, "-o")
	}
	return runScreencapture(args)
}

// isPermissionError checks if screencapture output indicates a screen recording
// permission issue. macOS returns "could not create image" when the app lacks
// Screen Recording permission.
func isPermissionError(output string) bool {
	return strings.Contains(output, "could not create image")
}

// CaptureWindowByName finds and captures a window by app name and optional title.
// Returns the captured image and the window info.
func CaptureWindowByName(appName, windowTitle string, hideShadow bool) (image.Image, *WindowInfo, error) {
	target, err := FindWindowWithTitle(appName, windowTitle)
	if err != nil {
		return nil, nil, err
	}

	img, err := CaptureWindow(target.ID, hideShadow)
	if err != nil {
		return nil, nil, err
	}

	return img, target, nil
}
