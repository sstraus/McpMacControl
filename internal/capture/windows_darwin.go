//go:build darwin

package capture

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

// Window info keys we need
extern const CFStringRef kCGWindowNumber;
extern const CFStringRef kCGWindowOwnerPID;
extern const CFStringRef kCGWindowOwnerName;
extern const CFStringRef kCGWindowName;
extern const CFStringRef kCGWindowBounds;
extern const CFStringRef kCGWindowIsOnscreen;
extern const CFStringRef kCGWindowLayer;

// Get window list as CFArray
CFArrayRef getWindowList() {
    return CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID
    );
}

// Get count of windows in array
CFIndex getWindowCount(CFArrayRef windows) {
    if (windows == NULL) return 0;
    return CFArrayGetCount(windows);
}

// Get window dict at index
CFDictionaryRef getWindowAt(CFArrayRef windows, CFIndex index) {
    return (CFDictionaryRef)CFArrayGetValueAtIndex(windows, index);
}

// Extract window number (ID)
uint32_t getWindowNumber(CFDictionaryRef window) {
    CFNumberRef num = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowNumber);
    if (num == NULL) return 0;
    uint32_t windowID = 0;
    CFNumberGetValue(num, kCFNumberSInt32Type, &windowID);
    return windowID;
}

// Extract window owner PID
int32_t getWindowOwnerPID(CFDictionaryRef window) {
    CFNumberRef num = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowOwnerPID);
    if (num == NULL) return 0;
    int32_t pid = 0;
    CFNumberGetValue(num, kCFNumberSInt32Type, &pid);
    return pid;
}

// Extract window layer
int32_t getWindowLayer(CFDictionaryRef window) {
    CFNumberRef num = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowLayer);
    if (num == NULL) return 0;
    int32_t layer = 0;
    CFNumberGetValue(num, kCFNumberSInt32Type, &layer);
    return layer;
}

// Extract on-screen status
int getWindowOnScreen(CFDictionaryRef window) {
    CFBooleanRef val = (CFBooleanRef)CFDictionaryGetValue(window, kCGWindowIsOnscreen);
    if (val == NULL) return 0;
    return CFBooleanGetValue(val) ? 1 : 0;
}

// Extract string value (owner name or window name)
// Returns NULL if not present, caller must free result
char* getWindowString(CFDictionaryRef window, CFStringRef key) {
    CFStringRef str = (CFStringRef)CFDictionaryGetValue(window, key);
    if (str == NULL) return NULL;

    CFIndex length = CFStringGetLength(str);
    CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
    char* buffer = (char*)malloc(maxSize);
    if (buffer == NULL) return NULL;

    if (!CFStringGetCString(str, buffer, maxSize, kCFStringEncodingUTF8)) {
        free(buffer);
        return NULL;
    }
    return buffer;
}

// Get window owner name
char* getWindowOwnerName(CFDictionaryRef window) {
    return getWindowString(window, kCGWindowOwnerName);
}

// Get window name (title)
char* getWindowName(CFDictionaryRef window) {
    return getWindowString(window, kCGWindowName);
}

// Get window bounds
void getWindowBounds(CFDictionaryRef window, int* x, int* y, int* width, int* height) {
    *x = 0; *y = 0; *width = 0; *height = 0;

    CFDictionaryRef bounds = (CFDictionaryRef)CFDictionaryGetValue(window, kCGWindowBounds);
    if (bounds == NULL) return;

    CGRect rect;
    if (!CGRectMakeWithDictionaryRepresentation(bounds, &rect)) return;

    *x = (int)rect.origin.x;
    *y = (int)rect.origin.y;
    *width = (int)rect.size.width;
    *height = (int)rect.size.height;
}

// Release window list
void releaseWindowList(CFArrayRef windows) {
    if (windows != NULL) {
        CFRelease(windows);
    }
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

func listWindowsDarwin(filter string) ([]WindowInfo, error) {
	windows := C.getWindowList()
	if windows == 0 {
		return nil, nil
	}
	defer C.releaseWindowList(windows)

	count := int(C.getWindowCount(windows))
	result := make([]WindowInfo, 0, count)
	filterLower := strings.ToLower(filter)

	for i := 0; i < count; i++ {
		window := C.getWindowAt(windows, C.CFIndex(i))
		if window == 0 {
			continue
		}

		// Get window layer - skip non-normal windows (layer != 0)
		layer := int(C.getWindowLayer(window))
		if layer != 0 {
			continue
		}

		// Get owner name
		ownerNameC := C.getWindowOwnerName(window)
		if ownerNameC == nil {
			continue
		}
		ownerName := C.GoString(ownerNameC)
		C.free(unsafe.Pointer(ownerNameC))

		// Apply filter if provided
		if filter != "" && !strings.Contains(strings.ToLower(ownerName), filterLower) {
			continue
		}

		// Get window name (may be empty)
		windowName := ""
		windowNameC := C.getWindowName(window)
		if windowNameC != nil {
			windowName = C.GoString(windowNameC)
			C.free(unsafe.Pointer(windowNameC))
		}

		// Get window ID
		windowID := uint32(C.getWindowNumber(window))
		if windowID == 0 {
			continue
		}

		// Get owner PID
		ownerPID := int(C.getWindowOwnerPID(window))

		// Get on-screen status
		onScreen := C.getWindowOnScreen(window) != 0

		// Get bounds
		var x, y, width, height C.int
		C.getWindowBounds(window, &x, &y, &width, &height)

		// Skip windows with no size
		if width <= 0 || height <= 0 {
			continue
		}

		result = append(result, WindowInfo{
			ID:        windowID,
			OwnerPID:  ownerPID,
			OwnerName: ownerName,
			Name:      windowName,
			X:         int(x),
			Y:         int(y),
			Width:     int(width),
			Height:    int(height),
			OnScreen:  onScreen,
			Layer:     layer,
		})
	}

	return result, nil
}
