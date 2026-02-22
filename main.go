package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sstraus/mcpmaccontrol/internal/agent"
	"github.com/sstraus/mcpmaccontrol/internal/bridge"
	"github.com/sstraus/mcpmaccontrol/internal/notify"
	"github.com/sstraus/mcpmaccontrol/internal/permissions"
	"github.com/sstraus/mcpmaccontrol/internal/shell"
	"github.com/sstraus/mcpmaccontrol/internal/tools"
)

var (
	mcpServer  *server.MCPServer
	statusItem *systray.MenuItem
	isRunning  atomic.Bool
	quitCh     = make(chan struct{})

	// Flags
	serverMode = flag.Bool("server", false, "Run in server mode (launched by bridge)")
	initMode   = flag.Bool("init", false, "Request macOS permissions and quit")
)

func main() {
	flag.Parse()

	headless := os.Getenv("MCPMACCONTROL_HEADLESS") == "1"

	switch {
	case *initMode:
		// Init mode: request macOS permissions (triggers TCC dialogs) and quit.
		// Run via `make init` which launches the .app bundle.
		log.Println("Requesting macOS permissions...")
		perms := permissions.RequestPermissions()
		log.Printf("Accessibility: %v, Screen Recording: %v", perms.Accessibility, perms.ScreenRecording)
		if !perms.Accessibility || !perms.ScreenRecording {
			log.Println("Some permissions missing. Grant them in System Settings > Privacy & Security, then re-run.")
		} else {
			log.Println("All permissions granted.")
		}
		return

	case *serverMode:
		// Server mode: launched by bridge via `open -a`, listens on Unix socket.
		// Shows systray since we're running as a proper .app.
		systray.Run(onReady, onExit)

	case headless:
		// Headless mode: direct stdio, no systray, no relaunch.
		runMCPServer(nil)

	default:
		// Bridge mode: relaunch ourselves inside .app bundle for TCC permissions,
		// then relay stdio <-> Unix socket.
		if err := bridge.Run(); err != nil {
			// Fallback: if no .app bundle found (e.g., `go run .`), serve directly.
			log.Printf("Bridge unavailable (%v), falling back to direct stdio", err)
			runMCPServer(nil)
		}
	}
}

func onReady() {
	// Set up menu bar
	systray.SetTemplateIcon(iconData, iconData)
	systray.SetTitle("")
	systray.SetTooltip("MCPMacControl - AI Mac Control")

	// About section
	aboutItem := systray.AddMenuItem("MCPMacControl", "")
	aboutItem.Disable()
	descItem := systray.AddMenuItem("  AI-powered Mac control via MCP", "")
	descItem.Disable()

	systray.AddSeparator()

	statusItem = systray.AddMenuItem("Status: Starting...", "MCP server status")
	statusItem.Disable()

	systray.AddSeparator()

	notifyItem := systray.AddMenuItemCheckbox("Notify on Actions", "Sound + visual flash before automation", notify.IsEnabled())

	systray.AddSeparator()

	permItem := systray.AddMenuItem("Check Permissions...", "Open System Settings")

	systray.AddSeparator()

	quitItem := systray.AddMenuItem("Quit", "Stop MCP server")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-notifyItem.ClickedCh:
				if notifyItem.Checked() {
					notifyItem.Uncheck()
					notify.SetEnabled(false)
				} else {
					notifyItem.Check()
					notify.SetEnabled(true)
				}
			case <-permItem.ClickedCh:
				perms := permissions.CheckAllPermissions()
				if !perms.Accessibility || !perms.ScreenRecording {
					go func() {
						result := permissions.ShowPermissionWizard()
						if result.Accessibility && result.ScreenRecording {
							statusItem.SetTitle("Status: Running")
						}
					}()
				} else {
					statusItem.SetTitle("Status: Running")
				}
			case <-quitItem.ClickedCh:
				close(quitCh)
				systray.Quit()
				return
			case <-quitCh:
				return
			}
		}
	}()

	// Start MCP server in background on the well-known socket.
	go func() {
		sockPath := bridge.SocketPath

		// Stale socket detection: if another instance is alive, shut down gracefully.
		if conn, err := net.DialTimeout("unix", sockPath, 500*time.Millisecond); err == nil {
			conn.Close()
			log.Printf("Another instance is already listening on %s, shutting down", sockPath)
			close(quitCh)
			systray.Quit()
			return
		}
		os.Remove(sockPath) // Remove stale socket from a crashed instance.

		listener, err := net.Listen("unix", sockPath)
		if err != nil {
			log.Printf("Failed to listen on socket %s: %v", sockPath, err)
			close(quitCh)
			systray.Quit()
			return
		}
		defer listener.Close()
		defer os.Remove(sockPath)

		runMCPServer(listener)

		// MCP server exited (idle timeout or signal)
		close(quitCh)
		systray.Quit()

		// Safety net: if systray.Quit() doesn't terminate the process
		// (e.g., NSApp event loop doesn't process the quit message),
		// force exit after a grace period.
		time.AfterFunc(2*time.Second, func() { os.Exit(0) })
	}()
}

func onExit() {
	// Cleanup
	if tools.ShellManager != nil {
		tools.ShellManager.CloseAll()
	}
}

// runMCPServer starts the MCP server. If listener is non-nil, it runs an
// accept loop serving multiple concurrent connections with a 10-minute idle
// timeout. Otherwise, it serves a single session on stdio.
func runMCPServer(listener net.Listener) {
	// In server mode (with systray), enable the permission wizard so tool
	// handlers can show the native permission window on first use.
	if statusItem != nil {
		permissions.SetWizardEnabled(true)
		perms := permissions.CheckAllPermissions()
		if !perms.Accessibility || !perms.ScreenRecording {
			statusItem.SetTitle("Status: Missing Permissions")
		}
	}

	// Initialize shell session manager
	tools.ShellManager = shell.NewManager()
	defer tools.ShellManager.CloseAll()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-quitCh:
			cancel()
		}
	}()

	// Update status
	isRunning.Store(true)
	if statusItem != nil {
		statusItem.SetTitle("Status: Running")
	}

	debug := os.Getenv("MCPMACCONTROL_DEBUG") == "1"

	hooks := &server.Hooks{}
	if statusItem != nil {
		// Cache the project label from the client's roots on first use.
		var projectLabel atomic.Value

		getProjectLabel := func(ctx context.Context) string {
			if v := projectLabel.Load(); v != nil {
				return v.(string)
			}
			label := resolveProjectLabel(ctx)
			projectLabel.Store(label)
			return label
		}

		balloonText := func(ctx context.Context, msg *mcp.CallToolRequest) string {
			project := getProjectLabel(ctx)
			tool := describeTool(msg)
			if project != "" {
				return project + "\n" + tool
			}
			return tool
		}

		hooks.AddBeforeCallTool(func(ctx context.Context, _ any, msg *mcp.CallToolRequest) {
			if debug {
				log.Printf("[DEBUG] BeforeCallTool tool=%s start", msg.Params.Name)
			}
			// Skip popover before screenshot tools so it doesn't appear in the capture.
			// It will show after the shot instead (in AfterCallTool).
			if !isCaptureCall(msg.Params.Name) {
				notify.ShowBalloon(balloonText(ctx, msg))
			}
			if debug {
				log.Printf("[DEBUG] BeforeCallTool tool=%s done", msg.Params.Name)
			}
		})
		hooks.AddAfterCallTool(func(ctx context.Context, _ any, msg *mcp.CallToolRequest, _ *mcp.CallToolResult) {
			if debug {
				log.Printf("[DEBUG] AfterCallTool tool=%s start", msg.Params.Name)
			}
			if isCaptureCall(msg.Params.Name) {
				// Show after the shot so user sees what happened
				notify.ShowBalloon(balloonText(ctx, msg))
			} else {
				notify.HideBalloon()
			}
			if debug {
				log.Printf("[DEBUG] AfterCallTool tool=%s done", msg.Params.Name)
			}
		})
	}

	mcpServer = server.NewMCPServer(
		"mcpmaccontrol",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithHooks(hooks),
	)

	// Register Mac control tools
	tools.Register(mcpServer)

	if listener != nil {
		// Socket mode: accept loop with idle timeout.
		lc := agent.NewLifecycle(agent.LifecycleConfig{
			IdleTimeout: 10 * time.Minute,
			OnShutdown: func() {
				log.Println("Idle timeout reached, shutting down")
				cancel()
			},
		})
		lc.SetReady()
		lc.StartIdleTimer()

		// Close the listener when context is cancelled so Accept() unblocks.
		go func() {
			<-ctx.Done()
			listener.Close()
		}()

		var wg sync.WaitGroup
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Expected when listener is closed during shutdown.
				if ctx.Err() != nil {
					break
				}
				log.Printf("Accept error: %v", err)
				break
			}

			if !lc.CanAcceptConnection() {
				log.Printf("Rejecting connection: shutting down")
				conn.Close()
				continue
			}

			lc.ConnectionOpened()
			log.Printf("Connection accepted (active: %d)", lc.ConnectionCount())

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				defer func() {
					lc.ConnectionClosed()
					log.Printf("Connection closed (active: %d)", lc.ConnectionCount())
				}()

				stdioServer := server.NewStdioServer(mcpServer)
				if err := stdioServer.Listen(ctx, conn, conn); err != nil {
					log.Printf("MCP session error: %v", err)
				}
			}()
		}

		// Wait for active connections to finish.
		wg.Wait()
	} else {
		// Direct stdio mode: single session, no idle timeout.
		stdioServer := server.NewStdioServer(mcpServer)
		if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
			log.Printf("MCP server error: %v", err)
		}
	}

	isRunning.Store(false)
	if statusItem != nil {
		statusItem.SetTitle("Status: Stopped")
	}
}

// resolveProjectLabel queries the client's roots to build a short label
// like "[MyProject] " for the popover. Returns empty string on failure.
func resolveProjectLabel(ctx context.Context) string {
	session := server.ClientSessionFromContext(ctx)
	if session == nil {
		return ""
	}

	home, _ := os.UserHomeDir()

	// Try to get the project path from roots (queries the CLIENT, e.g. Claude Code)
	if rs, ok := session.(server.SessionWithRoots); ok {
		result, err := rs.ListRoots(ctx, mcp.ListRootsRequest{})
		if err != nil {
			log.Printf("[popover] ListRoots failed: %v", err)
		} else if result != nil && len(result.Roots) > 0 {
			root := result.Roots[0]
			log.Printf("[popover] root: name=%q uri=%q", root.Name, root.URI)
			// Extract path from file:// URI and abbreviate home dir
			uri := root.URI
			if strings.HasPrefix(uri, "file://") {
				dir := strings.TrimPrefix(uri, "file://")
				if home != "" && strings.HasPrefix(dir, home) {
					dir = "~" + dir[len(home):]
				}
				if dir != "" && dir != "." {
					return dir
				}
			}
		}
	}

	// Fall back to client name + version (e.g. "claude-code 1.0.29")
	if ci, ok := session.(server.SessionWithClientInfo); ok {
		info := ci.GetClientInfo()
		log.Printf("[popover] clientInfo: name=%q version=%q", info.Name, info.Version)
		if info.Name != "" {
			return info.Name
		}
	}

	return ""
}

// isCaptureCall returns true for standalone screenshot tools.
func isCaptureCall(name string) bool {
	return name == "capture_window" || name == "capture_screen"
}

// describeTool returns a short human-readable description of an MCP tool call
// for the balloon notification.
func describeTool(msg *mcp.CallToolRequest) string {
	args := msg.GetArguments()
	switch msg.Params.Name {
	case "do":
		if actions, ok := args["actions"].([]any); ok && len(actions) > 0 {
			var app string
			var types []string
			for _, a := range actions {
				m, ok := a.(map[string]any)
				if !ok {
					continue
				}
				if t, ok := m["type"].(string); ok {
					types = append(types, t)
				}
				// Capture target app from first action that has one
				if app == "" {
					if v, ok := m["app"].(string); ok && v != "" {
						app = v
					}
				}
			}
			desc := strings.Join(types, " → ")
			if app != "" {
				return fmt.Sprintf("do [%s]: %s", app, desc)
			}
			if len(types) > 0 {
				return "do: " + desc
			}
		}
		return "do"
	case "shell":
		if action, ok := args["action"].(string); ok {
			if sid, ok := args["session_id"].(string); ok && sid != "" {
				return fmt.Sprintf("shell: %s (%s)", action, sid)
			}
			return fmt.Sprintf("shell: %s", action)
		}
		return "shell"
	case "list_windows":
		if filter, ok := args["app_filter"].(string); ok && filter != "" {
			return fmt.Sprintf("list_windows: %s", filter)
		}
		return "list_windows"
	case "processes":
		if name, ok := args["name"].(string); ok && name != "" {
			return fmt.Sprintf("processes: %s", name)
		}
		return "processes"
	case "help":
		if topic, ok := args["topic"].(string); ok && topic != "" {
			return fmt.Sprintf("help: %s", topic)
		}
		return "help"
	case "capture_window":
		if app, ok := args["app_name"].(string); ok && app != "" {
			return fmt.Sprintf("capture: %s", app)
		}
		return "capture_window"
	case "capture_screen":
		return "capture_screen"
	case "alert":
		if active, ok := args["active"].(bool); ok {
			if active {
				return "alert: ON"
			}
			return "alert: OFF"
		}
		return "alert"
	default:
		return msg.Params.Name
	}
}

// 22x22 template icon for macOS menu bar - monitor with capture dot
var iconData = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x16,
	0x08, 0x06, 0x00, 0x00, 0x00, 0xc4, 0xb4, 0x6c, 0x3b, 0x00, 0x00, 0x00,
	0x70, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0xd4, 0xd4, 0x41, 0x0e, 0x80,
	0x20, 0x10, 0x43, 0x51, 0x30, 0xde, 0xff, 0xca, 0xb8, 0x15, 0x82, 0xf2,
	0x3b, 0x99, 0x6a, 0xe8, 0xca, 0x8d, 0xcf, 0x32, 0x20, 0x47, 0x31, 0xc5,
	0x06, 0x9f, 0xb7, 0xe7, 0x96, 0x64, 0xd6, 0xf2, 0x55, 0xe3, 0xee, 0x8b,
	0x81, 0x74, 0x2b, 0xde, 0x6f, 0xf3, 0x54, 0x18, 0x6f, 0xb0, 0x02, 0x37,
	0x05, 0xa7, 0xf0, 0x88, 0x2d, 0x71, 0x0a, 0x8f, 0x27, 0x65, 0x79, 0x72,
	0x94, 0x51, 0x54, 0x8a, 0xaa, 0x30, 0x46, 0x23, 0x30, 0xce, 0xec, 0xcf,
	0x4b, 0xb9, 0x33, 0x6c, 0x8d, 0xc9, 0xcc, 0x9e, 0x56, 0xf0, 0xfa, 0xee,
	0x2f, 0x8d, 0xe9, 0xac, 0xa7, 0x86, 0xad, 0xb1, 0x2d, 0xb6, 0xc6, 0x57,
	0x00, 0x00, 0x00, 0xff, 0xff, 0xa6, 0x9b, 0x0a, 0x2e, 0x77, 0x7d, 0x49,
	0xd8, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60,
	0x82,
}
