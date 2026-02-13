// Package agent provides the agent-side components for MCPMacControl.
package agent

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the agent lifecycle state.
type State int32

const (
	StateStarting State = iota
	StateReady
	StateActive
	StateShuttingDown
	StateStopped
)

func (s State) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateReady:
		return "ready"
	case StateActive:
		return "active"
	case StateShuttingDown:
		return "shutting_down"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// LifecycleConfig contains configuration for the lifecycle manager.
type LifecycleConfig struct {
	// IdleTimeout is how long to wait with no connections before shutting down.
	IdleTimeout time.Duration

	// GracefulShutdownMax is the maximum time to wait for connections to close.
	GracefulShutdownMax time.Duration

	// PIDFile is the path to write the PID file.
	PIDFile string

	// SocketPath is included in the PID file for client discovery.
	SocketPath string

	// OnShutdown is called when shutdown is triggered (idle or user-initiated).
	OnShutdown func()

	// OnStop is called when user initiates stop, with the reason.
	OnStop func(reason string)
}

// Lifecycle manages the agent's state machine and idle timeout.
type Lifecycle struct {
	config LifecycleConfig

	state    atomic.Int32
	connCount atomic.Int32

	idleTimer    *time.Timer
	idleTimerMu  sync.Mutex
	shutdownOnce sync.Once
	shutdownDone chan struct{}

	mu sync.Mutex
}

// NewLifecycle creates a new lifecycle manager.
func NewLifecycle(config LifecycleConfig) *Lifecycle {
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 5 * time.Minute
	}
	if config.GracefulShutdownMax == 0 {
		config.GracefulShutdownMax = 5 * time.Second
	}

	lc := &Lifecycle{
		config:       config,
		shutdownDone: make(chan struct{}),
	}
	lc.state.Store(int32(StateStarting))
	return lc
}

// State returns the current state.
func (lc *Lifecycle) State() State {
	return State(lc.state.Load())
}

// SetReady transitions from STARTING to READY.
func (lc *Lifecycle) SetReady() {
	lc.state.Store(int32(StateReady))
}

// ConnectionCount returns the number of active connections.
func (lc *Lifecycle) ConnectionCount() int {
	return int(lc.connCount.Load())
}

// ConnectionOpened is called when a new connection is established.
func (lc *Lifecycle) ConnectionOpened() {
	count := lc.connCount.Add(1)
	if count == 1 {
		lc.state.Store(int32(StateActive))
		lc.pauseIdleTimer()
	}
}

// ConnectionClosed is called when a connection is closed.
func (lc *Lifecycle) ConnectionClosed() {
	count := lc.connCount.Add(-1)
	if count <= 0 {
		lc.connCount.Store(0) // Ensure non-negative
		if lc.State() == StateActive {
			lc.state.Store(int32(StateReady))
			lc.resumeIdleTimer()
		}
	}
}

// CanAcceptConnection returns true if new connections should be accepted.
func (lc *Lifecycle) CanAcceptConnection() bool {
	state := lc.State()
	return state == StateReady || state == StateActive
}

// StartIdleTimer starts the idle timeout timer.
func (lc *Lifecycle) StartIdleTimer() {
	lc.idleTimerMu.Lock()
	defer lc.idleTimerMu.Unlock()

	if lc.idleTimer != nil {
		lc.idleTimer.Stop()
	}

	lc.idleTimer = time.AfterFunc(lc.config.IdleTimeout, func() {
		if lc.ConnectionCount() == 0 && lc.State() == StateReady {
			lc.InitiateShutdown()
		}
	})
}

func (lc *Lifecycle) pauseIdleTimer() {
	lc.idleTimerMu.Lock()
	defer lc.idleTimerMu.Unlock()

	if lc.idleTimer != nil {
		lc.idleTimer.Stop()
	}
}

func (lc *Lifecycle) resumeIdleTimer() {
	lc.idleTimerMu.Lock()
	defer lc.idleTimerMu.Unlock()

	if lc.idleTimer != nil {
		lc.idleTimer.Stop()
	}

	lc.idleTimer = time.AfterFunc(lc.config.IdleTimeout, func() {
		if lc.ConnectionCount() == 0 && lc.State() == StateReady {
			lc.InitiateShutdown()
		}
	})
}

// InitiateShutdown begins the graceful shutdown process.
func (lc *Lifecycle) InitiateShutdown() {
	lc.shutdownOnce.Do(func() {
		lc.state.Store(int32(StateShuttingDown))

		// Stop accepting new connections (handled by caller checking CanAcceptConnection)

		// Call shutdown callback
		if lc.config.OnShutdown != nil {
			lc.config.OnShutdown()
		}

		// Wait for connections to close or timeout
		go func() {
			deadline := time.After(lc.config.GracefulShutdownMax)
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-deadline:
					lc.state.Store(int32(StateStopped))
					close(lc.shutdownDone)
					return
				case <-ticker.C:
					if lc.ConnectionCount() == 0 {
						lc.state.Store(int32(StateStopped))
						close(lc.shutdownDone)
						return
					}
				}
			}
		}()
	})
}

// WaitForShutdown returns a channel that is closed when shutdown is complete.
func (lc *Lifecycle) WaitForShutdown() <-chan struct{} {
	return lc.shutdownDone
}

// UserStop initiates shutdown due to user action.
func (lc *Lifecycle) UserStop() {
	if lc.config.OnStop != nil {
		lc.config.OnStop("user_initiated")
	}
	lc.InitiateShutdown()
}

// WritePIDFile writes the PID file for client discovery.
func (lc *Lifecycle) WritePIDFile() error {
	if lc.config.PIDFile == "" {
		return nil
	}

	content := fmt.Sprintf("%d\n%s\n", os.Getpid(), lc.config.SocketPath)
	return os.WriteFile(lc.config.PIDFile, []byte(content), 0600)
}

// RemovePIDFile removes the PID file.
func (lc *Lifecycle) RemovePIDFile() {
	if lc.config.PIDFile != "" {
		os.Remove(lc.config.PIDFile)
	}
}
