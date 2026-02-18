//go:build darwin

package notify

/*
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics

extern void playTinkSound(void);
extern void showOverlay(void);
*/
import "C"

import "time"

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
