package agent

import (
	"fmt"
	"os/exec"

	"github.com/getlantern/systray"
)

// MenuBar manages the system tray menu bar icon.
type MenuBar struct {
	lifecycle    *Lifecycle
	onStop       func()
	headless     bool

	statusItem   *systray.MenuItem
	connItem     *systray.MenuItem
	stopItem     *systray.MenuItem

	updateCh     chan struct{}
	quitCh       chan struct{}
}

// MenuBarConfig contains configuration for the menu bar.
type MenuBarConfig struct {
	Lifecycle *Lifecycle
	OnStop    func()
	Headless  bool
}

// NewMenuBar creates a new menu bar manager.
func NewMenuBar(config MenuBarConfig) *MenuBar {
	return &MenuBar{
		lifecycle: config.Lifecycle,
		onStop:    config.OnStop,
		headless:  config.Headless,
		updateCh:  make(chan struct{}, 1),
		quitCh:    make(chan struct{}),
	}
}

// Run starts the menu bar. This blocks until the menu bar is closed.
// In headless mode, this returns immediately.
func (m *MenuBar) Run() {
	if m.headless {
		// In headless mode, just wait for quit signal
		<-m.quitCh
		return
	}

	systray.Run(m.onReady, m.onExit)
}

// Quit signals the menu bar to exit.
func (m *MenuBar) Quit() {
	close(m.quitCh)
	if !m.headless {
		systray.Quit()
	}
}

// Update triggers a menu bar status update.
func (m *MenuBar) Update() {
	select {
	case m.updateCh <- struct{}{}:
	default:
	}
}

func (m *MenuBar) onReady() {
	// Set icon - using a simple template icon
	systray.SetTemplateIcon(iconData, iconData)
	systray.SetTitle("")
	systray.SetTooltip("MCPMacControl Agent")

	// Status section
	m.statusItem = systray.AddMenuItem("Status: Starting...", "Agent status")
	m.statusItem.Disable()

	m.connItem = systray.AddMenuItem("Connections: 0", "Active connections")
	m.connItem.Disable()

	systray.AddSeparator()

	// Actions
	permItem := systray.AddMenuItem("Check Permissions...", "Open System Settings")

	systray.AddSeparator()

	m.stopItem = systray.AddMenuItem("Stop Agent", "Stop the agent and notify clients")

	// Start update loop
	go m.updateLoop()

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-permItem.ClickedCh:
				openSystemSettings()
			case <-m.stopItem.ClickedCh:
				if m.onStop != nil {
					m.onStop()
				}
			case <-m.quitCh:
				return
			}
		}
	}()
}

func (m *MenuBar) onExit() {
	// Cleanup if needed
}

func (m *MenuBar) updateLoop() {
	// Initial update
	m.doUpdate()

	for {
		select {
		case <-m.updateCh:
			m.doUpdate()
		case <-m.quitCh:
			return
		}
	}
}

// doUpdate performs the actual UI update, handling nil items gracefully.
func (m *MenuBar) doUpdate() {
	if m.headless || m.statusItem == nil {
		return
	}
	m.updateStatus()
}

func (m *MenuBar) updateStatus() {
	if m.statusItem == nil {
		return
	}

	state := m.lifecycle.State()
	connCount := m.lifecycle.ConnectionCount()

	m.statusItem.SetTitle(FormatStatusText(state))
	m.connItem.SetTitle(FormatConnectionText(connCount))
	systray.SetTooltip(FormatTooltip(state, connCount))
}

// FormatStatusText returns the status text for the given state.
func FormatStatusText(state State) string {
	switch state {
	case StateStarting:
		return "Status: Starting..."
	case StateReady:
		return "Status: Ready (idle)"
	case StateActive:
		return "Status: Active"
	case StateShuttingDown:
		return "Status: Shutting down..."
	case StateStopped:
		return "Status: Stopped"
	default:
		return "Status: Unknown"
	}
}

// FormatConnectionText returns the connection count text.
func FormatConnectionText(count int) string {
	return fmt.Sprintf("Connections: %d", count)
}

// FormatTooltip returns the tooltip text.
func FormatTooltip(state State, connCount int) string {
	return fmt.Sprintf("MCPMacControl Agent - %s (%d conn)", state.String(), connCount)
}

func openSystemSettings() {
	// Open accessibility settings by default
	_ = OpenAccessibilitySettings()
}

// OpenAccessibilitySettings opens System Settings to Accessibility.
func OpenAccessibilitySettings() error {
	return openURL("x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility")
}

func openURL(url string) error {
	return runCommand("open", url)
}

func runCommand(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// 22x22 template icon for macOS menu bar
// Displays a screen outline with a camera/capture indicator
// Template icons should be black on transparent - macOS handles color inversion
var iconData = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x16,
	0x08, 0x06, 0x00, 0x00, 0x00, 0xc4, 0xb4, 0x6c, 0x3b, 0x00, 0x00, 0x00,
	0x6c, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x62, 0x62, 0xa0, 0x11, 0x18,
	0x7a, 0x06, 0xb3, 0xa0, 0xf1, 0xff, 0x53, 0x68, 0x1e, 0x23, 0x2e, 0x83,
	0x51, 0x24, 0x49, 0x04, 0x28, 0x8e, 0x1a, 0x7a, 0x61, 0x3c, 0x6a, 0xf0,
	0x08, 0x36, 0x18, 0x67, 0x86, 0xc2, 0x96, 0x41, 0x48, 0x35, 0x10, 0xab,
	0xe1, 0xe4, 0xb8, 0x18, 0xd9, 0x20, 0x9c, 0xb9, 0x14, 0x9b, 0x8b, 0x89,
	0x2d, 0x2f, 0x18, 0x91, 0x68, 0x4a, 0xcb, 0x18, 0xb0, 0x01, 0xb8, 0x0c,
	0x41, 0x91, 0x23, 0x35, 0x28, 0x18, 0x91, 0x0c, 0x41, 0x37, 0x14, 0x59,
	0x9e, 0xa2, 0x54, 0xf1, 0x1f, 0x87, 0x25, 0xa8, 0x36, 0x90, 0x69, 0x28,
	0x35, 0xcc, 0x22, 0xda, 0x02, 0xda, 0x02, 0x40, 0x00, 0x00, 0x00, 0xff,
	0xff, 0x3a, 0x44, 0x12, 0x24, 0x45, 0xcc, 0x37, 0xfb, 0x00, 0x00, 0x00,
	0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}
