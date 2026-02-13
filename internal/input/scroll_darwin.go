//go:build darwin

package input

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation

#include <ApplicationServices/ApplicationServices.h>

// Scroll wheel event
// deltaY: vertical scroll (positive = down, negative = up)
// deltaX: horizontal scroll (positive = right, negative = left)
void scrollWheel(int deltaY, int deltaX) {
    // Use pixel units for smooth scrolling
    CGEventRef event = CGEventCreateScrollWheelEvent(
        NULL,
        kCGScrollEventUnitPixel,
        2,        // wheelCount (2 = vertical + horizontal)
        deltaY,
        deltaX
    );
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}
*/
import "C"

// Scroll performs a scroll wheel event at the current mouse position.
// deltaY: vertical scroll (positive = scroll down, negative = scroll up)
// deltaX: horizontal scroll (positive = scroll right, negative = scroll left)
func Scroll(deltaY, deltaX int) {
	C.scrollWheel(C.int(deltaY), C.int(deltaX))
}
