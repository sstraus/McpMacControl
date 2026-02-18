package notify

import "testing"

func TestEnabledByDefault(t *testing.T) {
	// Reset to default state
	enabled.Store(true)

	if !IsEnabled() {
		t.Error("expected notifications to be enabled by default")
	}
}

func TestSetEnabledFalse(t *testing.T) {
	enabled.Store(true)

	SetEnabled(false)
	if IsEnabled() {
		t.Error("expected notifications to be disabled after SetEnabled(false)")
	}
}

func TestSetEnabledTrue(t *testing.T) {
	enabled.Store(false)

	SetEnabled(true)
	if !IsEnabled() {
		t.Error("expected notifications to be enabled after SetEnabled(true)")
	}
}

func TestToggle(t *testing.T) {
	enabled.Store(true)

	SetEnabled(false)
	SetEnabled(true)
	SetEnabled(false)

	if IsEnabled() {
		t.Error("expected notifications to be disabled after toggle sequence ending with false")
	}
}
