package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLifecycle_States(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 100 * time.Millisecond,
	})

	assert.Equal(t, StateStarting, lc.State())

	lc.SetReady()
	assert.Equal(t, StateReady, lc.State())

	lc.ConnectionOpened()
	assert.Equal(t, StateActive, lc.State())

	lc.ConnectionClosed()
	assert.Equal(t, StateReady, lc.State())

	lc.InitiateShutdown()
	assert.Equal(t, StateShuttingDown, lc.State())
}

func TestLifecycle_ConnectionTracking(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 1 * time.Second,
	})
	lc.SetReady()

	assert.Equal(t, 0, lc.ConnectionCount())

	lc.ConnectionOpened()
	assert.Equal(t, 1, lc.ConnectionCount())
	assert.Equal(t, StateActive, lc.State())

	lc.ConnectionOpened()
	assert.Equal(t, 2, lc.ConnectionCount())

	lc.ConnectionClosed()
	assert.Equal(t, 1, lc.ConnectionCount())
	assert.Equal(t, StateActive, lc.State())

	lc.ConnectionClosed()
	assert.Equal(t, 0, lc.ConnectionCount())
	assert.Equal(t, StateReady, lc.State())
}

func TestLifecycle_IdleTimeout(t *testing.T) {
	shutdownCalled := make(chan struct{})

	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 50 * time.Millisecond,
		OnShutdown: func() {
			close(shutdownCalled)
		},
	})

	lc.SetReady()
	lc.StartIdleTimer()

	// Wait for idle timeout
	select {
	case <-shutdownCalled:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("idle timeout did not trigger shutdown")
	}

	assert.Equal(t, StateShuttingDown, lc.State())
}

func TestLifecycle_IdleTimerPausedWhileActive(t *testing.T) {
	shutdownCalled := make(chan struct{}, 1)

	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout: 50 * time.Millisecond,
		OnShutdown: func() {
			shutdownCalled <- struct{}{}
		},
	})

	lc.SetReady()
	lc.StartIdleTimer()

	// Open connection immediately - should pause timer
	lc.ConnectionOpened()

	// Wait longer than idle timeout
	time.Sleep(100 * time.Millisecond)

	// Should still be active, not shut down
	assert.Equal(t, StateActive, lc.State())

	select {
	case <-shutdownCalled:
		t.Fatal("shutdown should not be called while active")
	default:
		// Expected - no shutdown
	}

	// Close connection - timer should resume
	lc.ConnectionClosed()

	// Now wait for timeout
	select {
	case <-shutdownCalled:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("idle timeout did not trigger after connection closed")
	}
}

func TestLifecycle_PIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")
	socketPath := filepath.Join(tmpDir, "test.sock")

	lc := NewLifecycle(LifecycleConfig{
		PIDFile:    pidPath,
		SocketPath: socketPath,
	})

	err := lc.WritePIDFile()
	require.NoError(t, err)

	// Verify file exists and has correct content
	content, err := os.ReadFile(pidPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), socketPath)

	// Cleanup removes PID file
	lc.RemovePIDFile()
	_, err = os.Stat(pidPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLifecycle_GracefulShutdown(t *testing.T) {
	lc := NewLifecycle(LifecycleConfig{
		IdleTimeout:         1 * time.Second,
		GracefulShutdownMax: 100 * time.Millisecond,
	})

	lc.SetReady()
	lc.ConnectionOpened()
	lc.ConnectionOpened()

	// Initiate shutdown
	lc.InitiateShutdown()
	assert.Equal(t, StateShuttingDown, lc.State())

	// New connections should be rejected
	assert.False(t, lc.CanAcceptConnection())

	// Close existing connections
	lc.ConnectionClosed()
	lc.ConnectionClosed()

	// Wait for shutdown to complete
	done := lc.WaitForShutdown()
	select {
	case <-done:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("shutdown did not complete")
	}

	assert.Equal(t, StateStopped, lc.State())
}

func TestLifecycle_UserStop(t *testing.T) {
	var stopReason string

	lc := NewLifecycle(LifecycleConfig{
		OnShutdown: func() {
			// Will be called
		},
		OnStop: func(reason string) {
			stopReason = reason
		},
	})

	lc.SetReady()
	lc.UserStop()

	assert.Equal(t, StateShuttingDown, lc.State())
	assert.Equal(t, "user_initiated", stopReason)
}
