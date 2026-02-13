//go:build darwin

package input

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework Carbon

#include <ApplicationServices/ApplicationServices.h>
#include <Carbon/Carbon.h>

// Type a single key with modifiers.
// Always sets flags explicitly to avoid inheriting stale modifier state
// (e.g. Shift leaking from a previous event).
void typeKey(int keyCode, int modifierFlags) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, true);
    CGEventSetFlags(keyDown, (CGEventFlags)modifierFlags);
    CGEventPost(kCGHIDEventTap, keyDown);
    CFRelease(keyDown);

    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, false);
    CGEventSetFlags(keyUp, (CGEventFlags)modifierFlags);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyUp);
}

// Type a unicode character
void typeUnichar(UniChar character) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, 0, true);
    CGEventKeyboardSetUnicodeString(keyDown, 1, &character);
    CGEventPost(kCGHIDEventTap, keyDown);
    CFRelease(keyDown);

    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, 0, false);
    CGEventKeyboardSetUnicodeString(keyUp, 1, &character);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyUp);
}
*/
import "C"

import (
	"strings"
	"time"
)

// Modifier flags for key events
const (
	ModShift   = 0x00020000 // kCGEventFlagMaskShift
	ModControl = 0x00040000 // kCGEventFlagMaskControl
	ModOption  = 0x00080000 // kCGEventFlagMaskAlternate (Alt/Option)
	ModCommand = 0x00100000 // kCGEventFlagMaskCommand
)

// Virtual key codes for macOS
var keyCodes = map[string]int{
	// Letters
	"a": 0x00, "b": 0x0B, "c": 0x08, "d": 0x02, "e": 0x0E,
	"f": 0x03, "g": 0x05, "h": 0x04, "i": 0x22, "j": 0x26,
	"k": 0x28, "l": 0x25, "m": 0x2E, "n": 0x2D, "o": 0x1F,
	"p": 0x23, "q": 0x0C, "r": 0x0F, "s": 0x01, "t": 0x11,
	"u": 0x20, "v": 0x09, "w": 0x0D, "x": 0x07, "y": 0x10,
	"z": 0x06,

	// Numbers
	"0": 0x1D, "1": 0x12, "2": 0x13, "3": 0x14, "4": 0x15,
	"5": 0x17, "6": 0x16, "7": 0x1A, "8": 0x1C, "9": 0x19,

	// Special keys
	"tab":       0x30,
	"enter":     0x24,
	"return":    0x24,
	"escape":    0x35,
	"esc":       0x35,
	"delete":    0x33,
	"backspace": 0x33,
	"space":     0x31,

	// Arrow keys
	"left":  0x7B,
	"right": 0x7C,
	"down":  0x7D,
	"up":    0x7E,

	// Navigation
	"home":     0x73,
	"end":      0x77,
	"pageup":   0x74,
	"pagedown": 0x79,

	// Function keys
	"f1": 0x7A, "f2": 0x78, "f3": 0x63, "f4": 0x76,
	"f5": 0x60, "f6": 0x61, "f7": 0x62, "f8": 0x64,
	"f9": 0x65, "f10": 0x6D, "f11": 0x67, "f12": 0x6F,

	// Punctuation (US keyboard layout)
	"-":  0x1B,
	"=":  0x18,
	"[":  0x21,
	"]":  0x1E,
	"\\": 0x2A,
	";":  0x29,
	"'":  0x27,
	",":  0x2B,
	".":  0x2F,
	"/":  0x2C,
	"`":  0x32,
}

// modifierMap maps modifier names to flags
var modifierMap = map[string]int{
	"shift":   ModShift,
	"ctrl":    ModControl,
	"control": ModControl,
	"alt":     ModOption,
	"opt":     ModOption,
	"option":  ModOption,
	"cmd":     ModCommand,
	"command": ModCommand,
}

// KeyTap presses a key with optional modifiers.
// key: the key name (e.g., "a", "tab", "enter", "1")
// modifiers: list of modifier names (e.g., ["shift"], ["cmd"])
// Returns true if the key was recognized and pressed.
func KeyTap(key string, modifiers []string) bool {
	key = strings.ToLower(key)

	// Calculate modifier flags
	flags := 0
	for _, mod := range modifiers {
		if f, ok := modifierMap[strings.ToLower(mod)]; ok {
			flags |= f
		}
	}

	// Get key code
	if code, ok := keyCodes[key]; ok {
		C.typeKey(C.int(code), C.int(flags))
		return true
	}
	return false
}

// TypeText types a string of text character by character using Unicode input.
// This is keyboard-layout-independent — characters are inserted by their
// Unicode value, not by virtual key codes.
func TypeText(text string) {
	for _, r := range text {
		time.Sleep(10 * time.Millisecond)
		C.typeUnichar(C.UniChar(r))
	}
}
