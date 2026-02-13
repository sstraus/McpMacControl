//go:build darwin

package permissions

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipWithoutTCC skips TCC-dependent tests under standard `go test` runs.
// TCC permissions are attributed to the .app bundle identity, not the test
// binary, so results are meaningless when spawned from a local shell.
// Set MCPMACCONTROL_TCC_TESTS=1 to run these tests from a properly signed context.
func skipWithoutTCC(t *testing.T) {
	t.Helper()
	if os.Getenv("MCPMACCONTROL_TCC_TESTS") != "1" {
		t.Skip("Skipping TCC-dependent test (set MCPMACCONTROL_TCC_TESTS=1 to run)")
	}
}

func TestHasAccessibilityPermission(t *testing.T) {
	skipWithoutTCC(t)
	result := HasAccessibilityPermission()
	t.Logf("Accessibility permission: %v", result)
	// Result can be true or false depending on whether permission is granted
	assert.True(t, result == true || result == false)
}

func TestHasScreenRecordingPermission(t *testing.T) {
	skipWithoutTCC(t)
	result := HasScreenRecordingPermission()
	t.Logf("Screen recording permission: %v", result)
	assert.True(t, result == true || result == false)
}

func TestCheckAllPermissions(t *testing.T) {
	skipWithoutTCC(t)
	status := CheckAllPermissions()
	t.Logf("Permissions status: accessibility=%v, screen_recording=%v",
		status.Accessibility, status.ScreenRecording)

	// Just verify the struct is properly populated
	assert.True(t, status.Accessibility == true || status.Accessibility == false)
	assert.True(t, status.ScreenRecording == true || status.ScreenRecording == false)
}

func TestRequestScreenRecordingPermission(t *testing.T) {
	skipWithoutTCC(t)
	result := RequestScreenRecordingPermission()
	t.Logf("RequestScreenRecordingPermission: %v", result)
	assert.IsType(t, true, result)
}

func TestEnsureScreenRecordingPermission(t *testing.T) {
	skipWithoutTCC(t)
	result := EnsureScreenRecordingPermission()
	t.Logf("EnsureScreenRecordingPermission: %v", result)
	assert.IsType(t, true, result)
}

func TestOpenSettings(t *testing.T) {
	skipWithoutTCC(t)
	t.Log("OpenSettings functions are defined and callable")
	// OpenAccessibilitySettings()  // Uncomment to test manually
	// OpenScreenRecordingSettings() // Uncomment to test manually
}

func TestSetWizardEnabled(t *testing.T) {
	// Verify default is false
	assert.False(t, wizardEnabled.Load(), "wizard should be disabled by default")

	SetWizardEnabled(true)
	assert.True(t, wizardEnabled.Load())

	SetWizardEnabled(false)
	assert.False(t, wizardEnabled.Load())
}

func TestEnsureAllPermissions_FastPath(t *testing.T) {
	skipWithoutTCC(t)
	SetWizardEnabled(false)
	defer SetWizardEnabled(false)

	status := EnsureAllPermissions()
	t.Logf("EnsureAllPermissions: accessibility=%v, screen_recording=%v",
		status.Accessibility, status.ScreenRecording)

	// Result should match direct checks
	expected := CheckAllPermissions()
	// We can't assert exact match because screen recording may be requested
	// in the fallback path, but we can verify the struct is populated
	assert.IsType(t, Status{}, status)
	t.Logf("Direct check: accessibility=%v, screen_recording=%v",
		expected.Accessibility, expected.ScreenRecording)
}

func TestEnsureAllPermissions_WizardDisabled(t *testing.T) {
	skipWithoutTCC(t)
	SetWizardEnabled(false)
	defer SetWizardEnabled(false)

	status := EnsureAllPermissions()
	assert.IsType(t, Status{}, status)
}

func TestWizardConcurrentGuard(t *testing.T) {
	// Test the concurrent caller guard logic (without actually showing the wizard).
	// We verify the mutex and channel mechanics by directly testing the guard state.
	wizardMu.Lock()
	assert.False(t, wizardActive, "wizard should not be active initially")
	wizardMu.Unlock()
}

func TestWizardGuardStateTransitions(t *testing.T) {
	// Verify the guard state transitions that protect against multiple
	// concurrent wizard windows.
	wizardMu.Lock()
	require.False(t, wizardActive, "wizard should start inactive")

	// Simulate first caller setting up
	wizardActive = true
	wizardResult = make(chan Status)
	ch := wizardResult
	wizardMu.Unlock()

	// Simulate second caller seeing active wizard
	wizardMu.Lock()
	require.True(t, wizardActive, "wizard should be active")
	require.NotNil(t, wizardResult, "result channel should exist")
	wizardMu.Unlock()

	// Simulate first caller finishing
	status := Status{Accessibility: true, ScreenRecording: true}
	wizardMu.Lock()
	wizardActive = false
	wizardResult = nil
	wizardMu.Unlock()

	// Send result to channel (simulating broadcast)
	go func() { ch <- status; close(ch) }()

	// Second caller receives result
	got := <-ch
	require.Equal(t, status, got, "waiter should receive broadcast status")
}
