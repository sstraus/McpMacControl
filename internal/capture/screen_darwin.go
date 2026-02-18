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

// Flat array to hold display bounds: [x0, y0, w0, h0, x1, y1, w1, h1, ...]
// Caller provides buffer and max count.
uint32_t activeDisplayBounds(int* buf, uint32_t maxDisplays) {
    uint32_t count = 0;
    CGGetActiveDisplayList(0, NULL, &count);
    if (count == 0) return 0;
    if (count > maxDisplays) count = maxDisplays;

    CGDirectDisplayID displays[count];
    CGGetActiveDisplayList(count, displays, &count);

    for (uint32_t i = 0; i < count; i++) {
        CGRect r = CGDisplayBounds(displays[i]);
        buf[i*4+0] = (int)r.origin.x;
        buf[i*4+1] = (int)r.origin.y;
        buf[i*4+2] = (int)r.size.width;
        buf[i*4+3] = (int)r.size.height;
    }
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

// DisplayBounds describes the position and size of a display in global screen
// coordinates. The primary display has origin (0,0); secondary displays can
// have negative coordinates.
type DisplayBounds struct {
	X, Y, Width, Height int
}

// ActiveDisplayBounds returns the bounds of all active displays.
func ActiveDisplayBounds() []DisplayBounds {
	const maxDisplays = 16
	var buf [maxDisplays * 4]C.int
	count := int(C.activeDisplayBounds(&buf[0], maxDisplays))

	result := make([]DisplayBounds, count)
	for i := 0; i < count; i++ {
		result[i] = DisplayBounds{
			X:      int(buf[i*4+0]),
			Y:      int(buf[i*4+1]),
			Width:  int(buf[i*4+2]),
			Height: int(buf[i*4+3]),
		}
	}
	return result
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
