//go:build darwin

package capture

/*
#cgo LDFLAGS: -framework CoreGraphics

#include <CoreGraphics/CoreGraphics.h>

// numActiveDisplays returns the count of active displays.
uint32_t numActiveDisplays() {
    uint32_t count = 0;
    CGGetActiveDisplayList(0, NULL, &count);
    return count;
}
*/
import "C"

import (
	"fmt"
	"image"
)

// NumDisplays returns the number of active displays.
func NumDisplays() int {
	return int(C.numActiveDisplays())
}

// CaptureScreen captures the entire screen for the given display.
// Display 0 is the primary display.
func CaptureScreen(displayIndex int) (image.Image, error) {
	numDisplays := NumDisplays()
	if numDisplays == 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	if displayIndex < 0 || displayIndex >= numDisplays {
		return nil, fmt.Errorf("display index %d out of range (0-%d)", displayIndex, numDisplays-1)
	}

	// screencapture -D uses 1-indexed display numbers
	args := []string{"-D", fmt.Sprintf("%d", displayIndex+1), "-x"}
	return runScreencapture(args)
}
