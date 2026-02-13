package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMenuBar(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 5 * time.Minute,
	})

	mb := NewMenuBar(MenuBarConfig{
		Lifecycle: lc,
		OnStop:    func() {},
		Headless:  true,
	})

	assert.NotNil(t, mb)
	assert.True(t, mb.headless)
	assert.NotNil(t, mb.updateCh)
	assert.NotNil(t, mb.quitCh)
}

func TestMenuBar_Headless_RunAndQuit(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 5 * time.Minute,
	})

	mb := NewMenuBar(MenuBarConfig{
		Lifecycle: lc,
		OnStop:    func() {},
		Headless:  true,
	})

	// Run in background
	done := make(chan struct{})
	go func() {
		mb.Run()
		close(done)
	}()

	// Should not block, quit immediately
	time.Sleep(10 * time.Millisecond)
	mb.Quit()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("MenuBar.Run did not exit after Quit")
	}
}

func TestMenuBar_Update(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 5 * time.Minute,
	})

	mb := NewMenuBar(MenuBarConfig{
		Lifecycle: lc,
		OnStop:    func() {},
		Headless:  true,
	})

	// Update should not block even when no one is listening
	mb.Update()
	mb.Update()
	mb.Update()

	// Channel should only have one pending update
	select {
	case <-mb.updateCh:
		// One update received
	default:
		t.Fatal("Expected at least one update in channel")
	}

	// Should be empty now or have at most one more
	select {
	case <-mb.updateCh:
		// OK, there was one more
	default:
		// OK, channel is empty
	}
}

func TestMenuBar_OnStop(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 5 * time.Minute,
	})

	stopCalled := false
	mb := NewMenuBar(MenuBarConfig{
		Lifecycle: lc,
		OnStop: func() {
			stopCalled = true
		},
		Headless: true,
	})

	assert.NotNil(t, mb.onStop)
	mb.onStop()
	assert.True(t, stopCalled)
}

func TestOpenAccessibilitySettings(t *testing.T) {
	// Always skip - this opens system settings which disrupts workflow
	// To test manually: comment out the skip and run just this test
	t.Skip("Skipping - opens system settings window")
}

func TestFormatStatusText(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateStarting, "Status: Starting..."},
		{StateReady, "Status: Ready (idle)"},
		{StateActive, "Status: Active"},
		{StateShuttingDown, "Status: Shutting down..."},
		{StateStopped, "Status: Stopped"},
		{State(99), "Status: Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatStatusText(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatConnectionText(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "Connections: 0"},
		{1, "Connections: 1"},
		{5, "Connections: 5"},
		{100, "Connections: 100"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatConnectionText(tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTooltip(t *testing.T) {
	tests := []struct {
		state     State
		connCount int
		expected  string
	}{
		{StateReady, 0, "MCPMacControl Agent - ready (0 conn)"},
		{StateActive, 3, "MCPMacControl Agent - active (3 conn)"},
		{StateShuttingDown, 1, "MCPMacControl Agent - shutting_down (1 conn)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatTooltip(tt.state, tt.connCount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOpenSystemSettings(t *testing.T) {
	// Always skip - this opens system settings which disrupts workflow
	// To test manually: comment out the skip and run just this test
	t.Skip("Skipping - opens system settings window")
}

func TestMenuBar_UpdateLoop_Headless(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 5 * time.Minute,
	})
	lc.SetReady()

	mb := NewMenuBar(MenuBarConfig{
		Lifecycle: lc,
		OnStop:    func() {},
		Headless:  true,
	})

	// Run updateLoop in background
	done := make(chan struct{})
	go func() {
		mb.updateLoop()
		close(done)
	}()

	// Send some updates
	mb.Update()
	mb.Update()

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	// Quit
	close(mb.quitCh)

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("updateLoop did not exit")
	}
}

func TestMenuBar_DoUpdate_Headless(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 5 * time.Minute,
	})

	mb := NewMenuBar(MenuBarConfig{
		Lifecycle: lc,
		OnStop:    func() {},
		Headless:  true,
	})

	// doUpdate should not panic in headless mode
	mb.doUpdate()
}

func TestMenuBar_DoUpdate_NilItems(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 5 * time.Minute,
	})

	mb := NewMenuBar(MenuBarConfig{
		Lifecycle: lc,
		OnStop:    func() {},
		Headless:  false, // Not headless but no items
	})

	// doUpdate should not panic with nil items
	mb.doUpdate()
}
