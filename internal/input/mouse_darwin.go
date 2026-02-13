//go:build darwin

package input

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation

#include <ApplicationServices/ApplicationServices.h>

// Move mouse to position
void moveMouse(int x, int y) {
    CGPoint point = CGPointMake(x, y);
    CGEventRef event = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, point, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

// Click at position
void clickMouse(int x, int y, int button, int doubleClick) {
    CGPoint point = CGPointMake(x, y);

    // Determine event types based on button
    CGEventType downType, upType;
    CGMouseButton cgButton;

    switch (button) {
        case 1: // right
            downType = kCGEventRightMouseDown;
            upType = kCGEventRightMouseUp;
            cgButton = kCGMouseButtonRight;
            break;
        case 2: // middle
            downType = kCGEventOtherMouseDown;
            upType = kCGEventOtherMouseUp;
            cgButton = kCGMouseButtonCenter;
            break;
        default: // left (0)
            downType = kCGEventLeftMouseDown;
            upType = kCGEventLeftMouseUp;
            cgButton = kCGMouseButtonLeft;
            break;
    }

    // Move to position first
    CGEventRef moveEvent = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, point, cgButton);
    CGEventPost(kCGHIDEventTap, moveEvent);
    CFRelease(moveEvent);

    // Click count (1 for single, 2 for double)
    int clickCount = doubleClick ? 2 : 1;

    // Perform click(s)
    for (int i = 0; i < clickCount; i++) {
        CGEventRef downEvent = CGEventCreateMouseEvent(NULL, downType, point, cgButton);
        CGEventSetIntegerValueField(downEvent, kCGMouseEventClickState, i + 1);
        CGEventPost(kCGHIDEventTap, downEvent);
        CFRelease(downEvent);

        CGEventRef upEvent = CGEventCreateMouseEvent(NULL, upType, point, cgButton);
        CGEventSetIntegerValueField(upEvent, kCGMouseEventClickState, i + 1);
        CGEventPost(kCGHIDEventTap, upEvent);
        CFRelease(upEvent);
    }
}

// Get current mouse position
void getMousePosition(int *x, int *y) {
    CGEventRef event = CGEventCreate(NULL);
    CGPoint point = CGEventGetLocation(event);
    CFRelease(event);
    *x = (int)point.x;
    *y = (int)point.y;
}

// Drag from one position to another.
// Sends intermediate drag events along the path so macOS and apps
// recognize the gesture as a drag (not an instant teleport).
void dragMouse(int fromX, int fromY, int toX, int toY) {
    CGPoint from = CGPointMake(fromX, fromY);
    CGPoint to = CGPointMake(toX, toY);

    // Move to start position
    CGEventRef moveEvent = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, from, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, moveEvent);
    CFRelease(moveEvent);
    usleep(50000); // 50ms settle

    // Mouse down at start
    CGEventRef downEvent = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDown, from, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, downEvent);
    CFRelease(downEvent);
    usleep(100000); // 100ms hold before dragging

    // Interpolate drag events along the path
    int steps = 10;
    for (int i = 1; i <= steps; i++) {
        double t = (double)i / steps;
        CGPoint p = CGPointMake(fromX + t * (toX - fromX), fromY + t * (toY - fromY));
        CGEventRef dragEvent = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDragged, p, kCGMouseButtonLeft);
        CGEventPost(kCGHIDEventTap, dragEvent);
        CFRelease(dragEvent);
        usleep(10000); // 10ms between steps
    }

    usleep(50000); // 50ms settle before release

    // Mouse up at end
    CGEventRef upEvent = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, to, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, upEvent);
    CFRelease(upEvent);
}
*/
import "C"

// MoveMouse moves the mouse cursor to the specified screen coordinates.
func MoveMouse(x, y int) {
	C.moveMouse(C.int(x), C.int(y))
}

// ClickMouse clicks at the specified screen coordinates.
// button: 0=left, 1=right, 2=middle
// doubleClick: true for double-click
func ClickMouse(x, y int, button int, doubleClick bool) {
	dc := 0
	if doubleClick {
		dc = 1
	}
	C.clickMouse(C.int(x), C.int(y), C.int(button), C.int(dc))
}

// GetMousePosition returns the current mouse cursor position.
func GetMousePosition() (x, y int) {
	var cx, cy C.int
	C.getMousePosition(&cx, &cy)
	return int(cx), int(cy)
}

// DragMouse performs a drag operation from one screen position to another.
func DragMouse(fromX, fromY, toX, toY int) {
	C.dragMouse(C.int(fromX), C.int(fromY), C.int(toX), C.int(toY))
}
