//go:build darwin

package input

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation

#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>

// Error codes
#define WC_OK 0
#define WC_ERR_CREATE_APP 1
#define WC_ERR_GET_WINDOWS 2
#define WC_ERR_NO_WINDOWS 3
#define WC_ERR_WINDOW_INDEX 4
#define WC_ERR_ACTION_FAILED 5
#define WC_ERR_SET_ATTR 6

// Focus window (bring to front)
int focusWindow(pid_t pid, CFIndex windowIndex) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (app == NULL) return WC_ERR_CREATE_APP;

    CFArrayRef windows = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (err != kAXErrorSuccess || windows == NULL) {
        CFRelease(app);
        return WC_ERR_GET_WINDOWS;
    }

    CFIndex count = CFArrayGetCount(windows);
    if (count == 0) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_NO_WINDOWS;
    }

    if (windowIndex >= count) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_WINDOW_INDEX;
    }

    AXUIElementRef window = (AXUIElementRef)CFArrayGetValueAtIndex(windows, windowIndex);

    // Raise the window
    err = AXUIElementPerformAction(window, kAXRaiseAction);
    int result = (err == kAXErrorSuccess) ? WC_OK : WC_ERR_ACTION_FAILED;

    // Also activate the application
    AXUIElementSetAttributeValue(app, kAXFrontmostAttribute, kCFBooleanTrue);

    CFRelease(windows);
    CFRelease(app);
    return result;
}

// Minimize window
int minimizeWindow(pid_t pid, CFIndex windowIndex) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (app == NULL) return WC_ERR_CREATE_APP;

    CFArrayRef windows = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (err != kAXErrorSuccess || windows == NULL) {
        CFRelease(app);
        return WC_ERR_GET_WINDOWS;
    }

    CFIndex count = CFArrayGetCount(windows);
    if (count == 0) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_NO_WINDOWS;
    }

    if (windowIndex >= count) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_WINDOW_INDEX;
    }

    AXUIElementRef window = (AXUIElementRef)CFArrayGetValueAtIndex(windows, windowIndex);

    err = AXUIElementSetAttributeValue(window, kAXMinimizedAttribute, kCFBooleanTrue);
    int result = (err == kAXErrorSuccess) ? WC_OK : WC_ERR_SET_ATTR;

    CFRelease(windows);
    CFRelease(app);
    return result;
}

// Restore (unminimize) window
int restoreWindow(pid_t pid, CFIndex windowIndex) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (app == NULL) return WC_ERR_CREATE_APP;

    CFArrayRef windows = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (err != kAXErrorSuccess || windows == NULL) {
        CFRelease(app);
        return WC_ERR_GET_WINDOWS;
    }

    CFIndex count = CFArrayGetCount(windows);
    if (count == 0) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_NO_WINDOWS;
    }

    if (windowIndex >= count) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_WINDOW_INDEX;
    }

    AXUIElementRef window = (AXUIElementRef)CFArrayGetValueAtIndex(windows, windowIndex);

    err = AXUIElementSetAttributeValue(window, kAXMinimizedAttribute, kCFBooleanFalse);
    int result = (err == kAXErrorSuccess) ? WC_OK : WC_ERR_SET_ATTR;

    CFRelease(windows);
    CFRelease(app);
    return result;
}

// Close window
int closeWindow(pid_t pid, CFIndex windowIndex) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (app == NULL) return WC_ERR_CREATE_APP;

    CFArrayRef windows = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (err != kAXErrorSuccess || windows == NULL) {
        CFRelease(app);
        return WC_ERR_GET_WINDOWS;
    }

    CFIndex count = CFArrayGetCount(windows);
    if (count == 0) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_NO_WINDOWS;
    }

    if (windowIndex >= count) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_WINDOW_INDEX;
    }

    AXUIElementRef window = (AXUIElementRef)CFArrayGetValueAtIndex(windows, windowIndex);

    // Get the close button and press it
    AXUIElementRef closeButton = NULL;
    err = AXUIElementCopyAttributeValue(window, kAXCloseButtonAttribute, (CFTypeRef*)&closeButton);
    if (err != kAXErrorSuccess || closeButton == NULL) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_ACTION_FAILED;
    }

    err = AXUIElementPerformAction(closeButton, kAXPressAction);
    int result = (err == kAXErrorSuccess) ? WC_OK : WC_ERR_ACTION_FAILED;

    CFRelease(closeButton);
    CFRelease(windows);
    CFRelease(app);
    return result;
}

// Resize window
int resizeWindow(pid_t pid, CFIndex windowIndex, int width, int height) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (app == NULL) return WC_ERR_CREATE_APP;

    CFArrayRef windows = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (err != kAXErrorSuccess || windows == NULL) {
        CFRelease(app);
        return WC_ERR_GET_WINDOWS;
    }

    CFIndex count = CFArrayGetCount(windows);
    if (count == 0) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_NO_WINDOWS;
    }

    if (windowIndex >= count) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_WINDOW_INDEX;
    }

    AXUIElementRef window = (AXUIElementRef)CFArrayGetValueAtIndex(windows, windowIndex);

    // Create size value
    CGSize size = CGSizeMake(width, height);
    AXValueRef sizeValue = AXValueCreate(kAXValueCGSizeType, &size);
    if (sizeValue == NULL) {
        CFRelease(windows);
        CFRelease(app);
        return WC_ERR_SET_ATTR;
    }

    err = AXUIElementSetAttributeValue(window, kAXSizeAttribute, sizeValue);
    int result = (err == kAXErrorSuccess) ? WC_OK : WC_ERR_SET_ATTR;

    CFRelease(sizeValue);
    CFRelease(windows);
    CFRelease(app);
    return result;
}

// Get window count for an app
CFIndex getAppWindowCount(pid_t pid) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (app == NULL) return 0;

    CFArrayRef windows = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (err != kAXErrorSuccess || windows == NULL) {
        CFRelease(app);
        return 0;
    }

    CFIndex count = CFArrayGetCount(windows);
    CFRelease(windows);
    CFRelease(app);
    return count;
}
*/
import "C"

import (
	"fmt"
)

// WindowControlError codes
const (
	wcOK            = 0
	wcErrCreateApp  = 1
	wcErrGetWindows = 2
	wcErrNoWindows  = 3
	wcErrIndex      = 4
	wcErrAction     = 5
	wcErrSetAttr    = 6
)

func windowControlError(code C.int, operation string) error {
	switch code {
	case wcOK:
		return nil
	case wcErrCreateApp:
		return fmt.Errorf("%s: failed to access application (check Accessibility permission)", operation)
	case wcErrGetWindows:
		return fmt.Errorf("%s: failed to get windows (check Accessibility permission)", operation)
	case wcErrNoWindows:
		return fmt.Errorf("%s: application has no windows", operation)
	case wcErrIndex:
		return fmt.Errorf("%s: window index out of range", operation)
	case wcErrAction:
		return fmt.Errorf("%s: action failed", operation)
	case wcErrSetAttr:
		return fmt.Errorf("%s: failed to set attribute", operation)
	default:
		return fmt.Errorf("%s: unknown error (%d)", operation, code)
	}
}

// FocusWindow brings a window to the front.
// pid is the process ID of the application.
// windowIndex is 0 for the first window.
func FocusWindow(pid int, windowIndex int) error {
	result := C.focusWindow(C.pid_t(pid), C.CFIndex(windowIndex))
	return windowControlError(result, "focus")
}

// MinimizeWindow minimizes a window to the dock.
func MinimizeWindow(pid int, windowIndex int) error {
	result := C.minimizeWindow(C.pid_t(pid), C.CFIndex(windowIndex))
	return windowControlError(result, "minimize")
}

// RestoreWindow restores a minimized window.
func RestoreWindow(pid int, windowIndex int) error {
	result := C.restoreWindow(C.pid_t(pid), C.CFIndex(windowIndex))
	return windowControlError(result, "restore")
}

// CloseWindow closes a window.
func CloseWindow(pid int, windowIndex int) error {
	result := C.closeWindow(C.pid_t(pid), C.CFIndex(windowIndex))
	return windowControlError(result, "close")
}

// ResizeWindow changes the size of a window.
func ResizeWindow(pid int, windowIndex int, width, height int) error {
	result := C.resizeWindow(C.pid_t(pid), C.CFIndex(windowIndex), C.int(width), C.int(height))
	return windowControlError(result, "resize")
}

// GetAppWindowCount returns the number of windows for an application.
func GetAppWindowCount(pid int) int {
	return int(C.getAppWindowCount(C.pid_t(pid)))
}
