//go:build darwin

package notify

/*
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics
#include <stdlib.h>

extern void playTinkSound(void);
extern void showOverlay(void);
extern void startWarnFlash(void);
extern void stopWarnFlash(void);
extern void showBalloon(const char* text);
extern void hideBalloon(void);
*/
import "C"

import (
	"time"
	"unsafe"
)

// StartWarn activates a flashing red border overlay to alert the user.
func StartWarn() {
	C.startWarnFlash()
}

// StopWarn stops the red flash and shows a brief green border.
func StopWarn() {
	C.stopWarnFlash()
}

// ShowBalloon displays a native popover anchored to the systray icon.
// Stays visible for at least 5 seconds. Call HideBalloon to dismiss after that.
func ShowBalloon(text string) {
	cstr := C.CString(text)
	defer C.free(unsafe.Pointer(cstr))
	C.showBalloon(cstr)
}

// HideBalloon hides the balloon notification immediately.
func HideBalloon() {
	C.hideBalloon()
}

// BeforeAction plays a sound and shows a visual overlay if notifications are enabled.
// Call this before executing any screen-affecting action.
func BeforeAction() {
	if !enabled.Load() {
		return
	}
	C.playTinkSound()
	C.showOverlay()
	// Give the sound time to start before the action executes.
	time.Sleep(50 * time.Millisecond)
}
