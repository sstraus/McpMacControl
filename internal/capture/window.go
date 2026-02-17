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
	windows, err := ListWindows(appName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list windows: %w", err)
	}

	if len(windows) == 0 {
		return nil, nil, fmt.Errorf("no windows found for app %q", appName)
	}

	// Two-pass match: prefer owner name match, fall back to title match.
	// This lets callers use either the process name ("Safari") or the
	// window title ("TUI Commander") as appName.
	appNameLower := strings.ToLower(appName)
	windowTitleLower := strings.ToLower(windowTitle)

	var target *WindowInfo
	// Pass 1: match on owner name (original behavior)
	for i := range windows {
		w := &windows[i]
		if !strings.EqualFold(w.OwnerName, appName) {
			continue
		}
		if windowTitle != "" && !strings.Contains(strings.ToLower(w.Name), windowTitleLower) {
			continue
		}
		target = w
		break
	}
	// Pass 2: match on window title if no owner match found
	if target == nil {
		for i := range windows {
			w := &windows[i]
			if !strings.Contains(strings.ToLower(w.Name), appNameLower) {
				continue
			}
			if windowTitle != "" && !strings.Contains(strings.ToLower(w.Name), windowTitleLower) {
				continue
			}
			target = w
			break
		}
	}

	if target == nil {
		if windowTitle != "" {
			return nil, nil, fmt.Errorf("no window found for app %q with title containing %q", appName, windowTitle)
		}
		return nil, nil, fmt.Errorf("no window found for app %q", appName)
	}

	img, err := CaptureWindow(target.ID, hideShadow)
	if err != nil {
		return nil, nil, err
	}

	return img, target, nil
}
