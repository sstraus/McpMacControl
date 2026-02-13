//go:build darwin

// Package permissions provides macOS permission detection and helpers.
package permissions

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework ScreenCaptureKit

#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <time.h>

#import <ScreenCaptureKit/ScreenCaptureKit.h>

// Check if process has accessibility permission.
// Check if process has accessibility permission.
// Uses AXIsProcessTrustedWithOptions (without prompting) which queries
// the TCC database directly.
int checkAccessibilityPermission() {
    @autoreleasepool {
        NSDictionary *options = @{
            (__bridge NSString *)kAXTrustedCheckOptionPrompt: @NO
        };
        return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
    }
}

// Prompt user for accessibility permission via system dialog
int requestAccessibilityPermission() {
    NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @YES};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
}

// Trigger screen recording TCC prompt via ScreenCaptureKit.
// Requesting shareable content triggers the macOS screen recording dialog.
int requestScreenRecordingPermission() {
    __block int result = 0;
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);

    [SCShareableContent getShareableContentExcludingDesktopWindows:YES
        onScreenWindowsOnly:YES
        completionHandler:^(SCShareableContent * _Nullable content, NSError * _Nullable error) {
            if (content != nil && error == nil) {
                result = 1;
            }
            dispatch_semaphore_signal(sem);
        }];

    dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, 10 * NSEC_PER_SEC));
    return result;
}

// Check if process has screen recording permission using the native preflight API.
// Note: CGPreflightScreenCaptureAccess may not reflect a newly granted permission
// until the app is restarted. This is a macOS limitation.
int checkScreenRecordingPermission() {
    return CGPreflightScreenCaptureAccess() ? 1 : 0;
}

// Live check for screen recording permission using SCShareableContent.
// Unlike CGPreflightScreenCaptureAccess, this reflects newly granted permissions
// without requiring an app restart. Uses async dispatch + cached result to avoid
// blocking the main thread. Safe to call from a polling timer.
//
// After a denial, waits _scrCooldownSec seconds before re-probing to avoid
// triggering repeated TCC prompts (SCShareableContent fires a system dialog
// each time it's called without permission).
static int _scrLiveResult = -1;  // -1 = not yet checked, 0 = denied, 1 = granted
static int _scrLiveInFlight = 0;
static time_t _scrLastDenied = 0;
static const int _scrCooldownSec = 5;

int checkScreenRecordingLive() {
    // Return preflight result if it says yes (fast path)
    if (CGPreflightScreenCaptureAccess()) {
        _scrLiveResult = 1;
        return 1;
    }

    // If already granted via SCShareableContent, trust it
    if (_scrLiveResult == 1) {
        return 1;
    }

    // After a denial, wait before re-probing to avoid prompt spam
    if (_scrLiveResult == 0) {
        time_t now = time(NULL);
        if (now - _scrLastDenied < _scrCooldownSec) {
            return 0;
        }
    }

    // Kick off async check if not already in flight
    if (!_scrLiveInFlight) {
        _scrLiveInFlight = 1;
        [SCShareableContent getShareableContentExcludingDesktopWindows:YES
            onScreenWindowsOnly:NO
            completionHandler:^(SCShareableContent * _Nullable content, NSError * _Nullable error) {
                if (content != nil && content.displays.count > 0) {
                    _scrLiveResult = 1;
                } else {
                    _scrLiveResult = 0;
                    _scrLastDenied = time(NULL);
                }
                _scrLiveInFlight = 0;
            }];
    }

    // Return cached result (may be stale by one poll cycle)
    return (_scrLiveResult > 0) ? 1 : 0;
}
*/
import "C"

import (
	"os/exec"
	"sync/atomic"
)

// Status represents the current permission status.
type Status struct {
	Accessibility   bool
	ScreenRecording bool
}

var wizardEnabled atomic.Bool

// SetWizardEnabled controls whether EnsureAllPermissions shows the native
// permission wizard. Should be called once at startup in server mode (when
// the app has a GUI run loop).
func SetWizardEnabled(enabled bool) {
	wizardEnabled.Store(enabled)
}

// EnsureAllPermissions checks both permissions and, if any are missing and
// the wizard is enabled, shows the native permission wizard. Returns the
// final permission status.
func EnsureAllPermissions() Status {
	status := CheckAllPermissions()
	if status.Accessibility && status.ScreenRecording {
		return status
	}

	if wizardEnabled.Load() {
		return ShowPermissionWizard()
	}

	// Wizard disabled (headless/bridge mode) - fall back to TCC prompts.
	if !status.ScreenRecording {
		RequestScreenRecordingPermission()
	}
	return CheckAllPermissions()
}

// HasAccessibilityPermission checks if the app has accessibility permission.
func HasAccessibilityPermission() bool {
	return C.checkAccessibilityPermission() != 0
}

// HasScreenRecordingPermission checks if the app has screen recording permission.
func HasScreenRecordingPermission() bool {
	return C.checkScreenRecordingPermission() != 0
}

// RequestScreenRecordingPermission triggers the macOS TCC dialog for screen
// recording permission via ScreenCaptureKit. Blocks up to 10 seconds waiting
// for the user to respond.
func RequestScreenRecordingPermission() bool {
	return C.requestScreenRecordingPermission() != 0
}

// EnsureScreenRecordingPermission checks for screen recording permission and,
// if not granted, triggers the TCC dialog and re-checks. Returns true if
// permission is granted after the check/request cycle.
func EnsureScreenRecordingPermission() bool {
	if HasScreenRecordingPermission() {
		return true
	}
	RequestScreenRecordingPermission()
	return HasScreenRecordingPermission()
}

// CheckAllPermissions returns the status of all required permissions.
func CheckAllPermissions() Status {
	return Status{
		Accessibility:   HasAccessibilityPermission(),
		ScreenRecording: HasScreenRecordingPermission(),
	}
}

// RequestPermissions triggers the macOS permission prompts for any missing permissions.
// This causes the app to appear in Privacy & Security settings.
func RequestPermissions() Status {
	accessibility := C.requestAccessibilityPermission() != 0
	screenRecording := C.requestScreenRecordingPermission() != 0
	return Status{
		Accessibility:   accessibility,
		ScreenRecording: screenRecording,
	}
}

// OpenAccessibilitySettings opens System Settings to the Accessibility pane.
func OpenAccessibilitySettings() error {
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
}

// OpenScreenRecordingSettings opens System Settings to the Screen Recording pane.
func OpenScreenRecordingSettings() error {
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture").Run()
}

// OpenSettings opens System Settings to the specified pane.
// Valid panes: "accessibility", "screen_recording"
func OpenSettings(pane string) error {
	switch pane {
	case "accessibility":
		return OpenAccessibilitySettings()
	case "screen_recording":
		return OpenScreenRecordingSettings()
	default:
		return OpenAccessibilitySettings()
	}
}
