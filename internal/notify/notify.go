// Package notify provides sound and visual notifications before automation actions.
// When enabled, it plays a brief system sound and flashes a border overlay
// so the user knows the machine is being controlled programmatically.
package notify

import "sync/atomic"

var enabled atomic.Bool

func init() {
	enabled.Store(true)
}

// SetEnabled controls whether action notifications are active.
func SetEnabled(v bool) {
	enabled.Store(v)
}

// IsEnabled returns whether action notifications are active.
func IsEnabled() bool {
	return enabled.Load()
}
