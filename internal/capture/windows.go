package capture

// WindowInfo contains information about a macOS window.
type WindowInfo struct {
	ID        uint32 // CGWindowID
	OwnerPID  int    // Process ID of the owning application
	OwnerName string // Application name (e.g., "Safari")
	Name      string // Window title (e.g., "Welcome to Safari")
	X         int    // Window X position
	Y         int    // Window Y position
	Width     int    // Window width
	Height    int    // Window height
	OnScreen  bool   // Whether window is currently on screen
	Layer     int    // Window layer (0 = normal windows)
}

// ListWindows returns all visible windows on the system.
// The filter parameter is optional; if provided, only windows with matching
// owner name or window title (case-insensitive substring) are returned.
func ListWindows(filter string) ([]WindowInfo, error) {
	return listWindowsDarwin(filter)
}
