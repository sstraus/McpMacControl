package capture

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"
)

// SaveToFile saves the image to a file and returns the file path.
// The file is saved in the system temp directory with a timestamp-based name.
// Uses WebP lossless by default, falls back to PNG if WebP fails.
func SaveToFile(img image.Image, prefix string) (string, error) {
	return SaveToFileWithFormat(img, prefix, FormatWebP, 0)
}

// SaveToFileWithFormat saves the image to a file in the specified format.
// quality controls compression (1-100, 0 = default). Ignored for PNG.
func SaveToFileWithFormat(img image.Image, prefix string, format ImageFormat, quality int) (string, error) {
	// Optimize the image
	result, err := OptimizeImageWithFormat(img, format, quality)
	if err != nil {
		return "", fmt.Errorf("failed to optimize image: %w", err)
	}

	// Determine file extension
	ext := ".webp"
	if result.Format == FormatPNG {
		ext = ".png"
	}

	// Create filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("mcpmaccontrol-%s-%s%s", prefix, timestamp, ext)
	fpath := filepath.Join(os.TempDir(), filename)

	// Write to file
	if err := os.WriteFile(fpath, result.Data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fpath, nil
}

// SaveScreenToFile captures the screen and saves it to a temp file.
// Returns the file path.
func SaveScreenToFile(displayIndex int) (string, error) {
	img, err := CaptureScreen(displayIndex)
	if err != nil {
		return "", err
	}
	return SaveToFile(img, fmt.Sprintf("screen%d", displayIndex))
}

// SaveWindowToFile captures a window and saves it to a temp file.
// Returns the file path and window info.
// The image is resized to window point dimensions for accurate click coordinates.
func SaveWindowToFile(appName, windowTitle string, hideShadow bool) (string, *WindowInfo, error) {
	img, info, err := CaptureWindowByName(appName, windowTitle, hideShadow)
	if err != nil {
		return "", nil, err
	}

	// Resize to window point dimensions so AI coordinates match click coordinates.
	// Screenshots are captured at Retina resolution (2x pixels), but window bounds
	// and mouse events use points.
	resized := ResizeToTarget(img, info.Width, info.Height)

	prefix := SanitizeFilename(appName)
	path, err := SaveToFileWithFormat(resized, prefix, FormatWebP, 0)
	if err != nil {
		return "", nil, err
	}

	return path, info, nil
}

// SanitizeFilename removes characters that aren't safe for filenames.
func SanitizeFilename(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, byte(c))
		}
	}
	if len(result) == 0 {
		return "window"
	}
	return string(result)
}
