package agent

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/sstraus/mcpmaccontrol/internal/capture"
	"github.com/sstraus/mcpmaccontrol/internal/input"
	"github.com/sstraus/mcpmaccontrol/internal/permissions"
	"github.com/sstraus/mcpmaccontrol/internal/protocol"
)

// Version is the agent protocol version.
const Version = "1.0.0"

// Server is the agent's RPC server that dispatches requests.
type Server struct {
	protoServer *protocol.Server
	lifecycle   *Lifecycle
}

// ServerConfig contains configuration for the agent server.
type ServerConfig struct {
	SocketPath  string
	IdleTimeout time.Duration
	OnStop      func(reason string)
}

// NewServer creates a new agent server.
func NewServer(config ServerConfig) *Server {
	protoServer := protocol.NewServer(config.SocketPath)

	// Create lifecycle with connection tracking callbacks
	lifecycle := NewLifecycle(LifecycleConfig{
		IdleTimeout: config.IdleTimeout,
		SocketPath:  config.SocketPath,
		PIDFile:     pidFilePath(),
		OnStop:      config.OnStop,
	})

	s := &Server{
		protoServer: protoServer,
		lifecycle:   lifecycle,
	}

	s.registerHandlers()
	return s
}

func pidFilePath() string {
	return fmt.Sprintf("/tmp/mcpmaccontrol-agent-%d.pid", os.Getuid())
}

// Start begins the agent server.
func (s *Server) Start() error {
	if err := s.protoServer.Start(); err != nil {
		return err
	}

	if err := s.lifecycle.WritePIDFile(); err != nil {
		s.protoServer.Stop()
		return err
	}

	s.lifecycle.SetReady()
	s.lifecycle.StartIdleTimer()

	return nil
}

// Stop stops the agent server.
func (s *Server) Stop() error {
	s.lifecycle.InitiateShutdown()
	s.lifecycle.RemovePIDFile()
	return s.protoServer.Stop()
}

// Lifecycle returns the lifecycle manager.
func (s *Server) Lifecycle() *Lifecycle {
	return s.lifecycle
}

// ProtoServer returns the protocol server for direct access.
func (s *Server) ProtoServer() *protocol.Server {
	return s.protoServer
}

// BroadcastStopping sends a stopping notification to all clients.
func (s *Server) BroadcastStopping(reason string) {
	notif := protocol.NewNotification(protocol.MethodStopping, map[string]any{
		"reason": reason,
	})
	s.protoServer.BroadcastNotification(notif)
}

func (s *Server) registerHandlers() {
	// Control methods
	s.protoServer.RegisterHandler(protocol.MethodHello, s.handleHello)
	s.protoServer.RegisterHandler(protocol.MethodPing, s.handlePing)
	s.protoServer.RegisterHandler(protocol.MethodShutdown, s.handleShutdown)
	s.protoServer.RegisterHandler(protocol.MethodOpenSettings, s.handleOpenSettings)

	// Capture methods
	s.protoServer.RegisterHandler(protocol.MethodListWindows, s.handleListWindows)
	s.protoServer.RegisterHandler(protocol.MethodCaptureWindow, s.handleCaptureWindow)
	s.protoServer.RegisterHandler(protocol.MethodCaptureScreen, s.handleCaptureScreen)

	// Input methods
	s.protoServer.RegisterHandler(protocol.MethodClick, s.handleClick)
	s.protoServer.RegisterHandler(protocol.MethodMove, s.handleMove)
	s.protoServer.RegisterHandler(protocol.MethodType, s.handleType)
	s.protoServer.RegisterHandler(protocol.MethodKey, s.handleKey)
	s.protoServer.RegisterHandler(protocol.MethodScroll, s.handleScroll)

	// Window control methods
	s.protoServer.RegisterHandler(protocol.MethodFocus, s.handleFocus)
	s.protoServer.RegisterHandler(protocol.MethodMinimize, s.handleMinimize)
	s.protoServer.RegisterHandler(protocol.MethodRestore, s.handleRestore)
	s.protoServer.RegisterHandler(protocol.MethodClose, s.handleClose)
	s.protoServer.RegisterHandler(protocol.MethodResize, s.handleResize)
}

// Control handlers

func (s *Server) handleHello(req *protocol.Request) *protocol.Response {
	perms := permissions.CheckAllPermissions()
	return protocol.NewResponse(req, map[string]any{
		"version": Version,
		"permissions": map[string]any{
			"accessibility":    perms.Accessibility,
			"screen_recording": perms.ScreenRecording,
		},
	})
}

func (s *Server) handlePing(req *protocol.Request) *protocol.Response {
	return protocol.NewResponse(req, map[string]any{
		"pong":        true,
		"connections": s.lifecycle.ConnectionCount(),
		"state":       s.lifecycle.State().String(),
	})
}

func (s *Server) handleShutdown(req *protocol.Request) *protocol.Response {
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.lifecycle.InitiateShutdown()
	}()
	return protocol.NewResponse(req, map[string]any{"ok": true})
}

func (s *Server) handleOpenSettings(req *protocol.Request) *protocol.Response {
	pane := "accessibility"
	if req.Params != nil {
		if p, ok := req.Params["pane"].(string); ok {
			pane = p
		}
	}

	if err := permissions.OpenSettings(pane); err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}
	return protocol.NewResponse(req, map[string]any{"ok": true})
}

// Capture handlers

func (s *Server) handleListWindows(req *protocol.Request) *protocol.Response {
	filter := ""
	if req.Params != nil {
		if f, ok := req.Params["filter"].(string); ok {
			filter = f
		}
	}

	// Check permission before listing windows
	if !permissions.EnsureScreenRecordingPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionScreenRecording,
			"Screen Recording permission required. Grant access in System Settings > Privacy & Security > Screen Recording")
	}

	windows, err := capture.ListWindows(filter)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	result := make([]map[string]any, len(windows))
	for i, w := range windows {
		result[i] = map[string]any{
			"id":         w.ID,
			"owner_pid":  w.OwnerPID,
			"owner_name": w.OwnerName,
			"name":       w.Name,
			"x":          w.X,
			"y":          w.Y,
			"width":      w.Width,
			"height":     w.Height,
			"layer":      w.Layer,
		}
	}

	return protocol.NewResponse(req, result)
}

func (s *Server) handleCaptureWindow(req *protocol.Request) *protocol.Response {
	// Validate parameters first
	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	windowTitle := getString(req.Params, "window_title", "")
	hideShadow := getBool(req.Params, "hide_shadow", true)
	format := getString(req.Params, "format", "webp")
	quality := getInt(req.Params, "quality", 0)

	// Region parameters
	regionX := getInt(req.Params, "region_x", 0)
	regionY := getInt(req.Params, "region_y", 0)
	regionW := getInt(req.Params, "region_width", 0)
	regionH := getInt(req.Params, "region_height", 0)

	// Check permission before capturing
	if !permissions.EnsureScreenRecordingPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionScreenRecording,
			"Screen Recording permission required. Grant access in System Settings > Privacy & Security > Screen Recording")
	}

	img, info, err := capture.CaptureWindowByName(app, windowTitle, hideShadow)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	// Resize to window point dimensions
	resized := capture.ResizeToTarget(img, info.Width, info.Height)

	// Crop to region if specified
	var finalImg = resized
	if regionW > 0 && regionH > 0 {
		if regionX < 0 || regionY < 0 || regionX+regionW > info.Width || regionY+regionH > info.Height {
			return protocol.NewErrorResponse(req, protocol.ErrInvalidParams,
				fmt.Sprintf("region out of bounds: region(%d,%d,%d,%d) exceeds window size(%d,%d)",
					regionX, regionY, regionW, regionH, info.Width, info.Height))
		}
		finalImg = cropImage(resized, regionX, regionY, regionW, regionH)
	}

	// Encode image
	imgFormat := capture.FormatWebP
	if format == "png" {
		imgFormat = capture.FormatPNG
	}

	result, err := capture.OptimizeImageWithFormat(finalImg, imgFormat, quality)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{
		"image":     base64.StdEncoding.EncodeToString(result.Data),
		"mime_type": result.MIMEType,
		"window": map[string]any{
			"owner_name": info.OwnerName,
			"name":       info.Name,
			"x":          info.X,
			"y":          info.Y,
			"width":      info.Width,
			"height":     info.Height,
		},
		"region": map[string]any{
			"x":      regionX,
			"y":      regionY,
			"width":  regionW,
			"height": regionH,
		},
	})
}

func (s *Server) handleCaptureScreen(req *protocol.Request) *protocol.Response {
	display := getInt(req.Params, "display", 0)
	format := getString(req.Params, "format", "webp")
	quality := getInt(req.Params, "quality", 0)

	// Check permission before capturing
	if !permissions.EnsureScreenRecordingPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionScreenRecording,
			"Screen Recording permission required. Grant access in System Settings > Privacy & Security > Screen Recording")
	}

	img, err := capture.CaptureScreen(display)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	imgFormat := capture.FormatWebP
	if format == "png" {
		imgFormat = capture.FormatPNG
	}

	result, err := capture.OptimizeImageWithFormat(img, imgFormat, quality)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	bounds := img.Bounds()
	return protocol.NewResponse(req, map[string]any{
		"image":     base64.StdEncoding.EncodeToString(result.Data),
		"mime_type": result.MIMEType,
		"width":     bounds.Dx(),
		"height":    bounds.Dy(),
	})
}

// Input handlers

func (s *Server) handleClick(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	x := getInt(req.Params, "x", 0)
	y := getInt(req.Params, "y", 0)
	button := getString(req.Params, "button", "left")
	double := getBool(req.Params, "double", false)

	absX, absY, err := input.ClickInWindow(app, x, y, input.ParseMouseButton(button), double)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{
		"ok":      true,
		"abs_x":   absX,
		"abs_y":   absY,
		"button":  button,
		"double":  double,
	})
}

func (s *Server) handleMove(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	x := getInt(req.Params, "x", 0)
	y := getInt(req.Params, "y", 0)

	absX, absY, err := input.MoveMouseToWindow(app, x, y)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{
		"ok":    true,
		"abs_x": absX,
		"abs_y": absY,
	})
}

func (s *Server) handleType(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	text := getString(req.Params, "text", "")
	if text == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "text parameter required")
	}

	input.TypeText(text)
	return protocol.NewResponse(req, map[string]any{
		"ok":    true,
		"chars": len(text),
	})
}

func (s *Server) handleKey(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	key := getString(req.Params, "key", "")
	if key == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "key parameter required")
	}

	modifiers := getStringSlice(req.Params, "modifiers")

	if !input.KeyTap(key, modifiers) {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "unknown key: "+key)
	}

	return protocol.NewResponse(req, map[string]any{
		"ok":        true,
		"key":       key,
		"modifiers": modifiers,
	})
}

func (s *Server) handleScroll(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	x := getInt(req.Params, "x", 0)
	y := getInt(req.Params, "y", 0)
	deltaY := getInt(req.Params, "delta_y", 0)
	deltaX := getInt(req.Params, "delta_x", 0)

	// Move mouse to position first
	_, _, err := input.MoveMouseToWindow(app, x, y)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	input.Scroll(deltaY, deltaX)

	return protocol.NewResponse(req, map[string]any{
		"ok":      true,
		"delta_x": deltaX,
		"delta_y": deltaY,
	})
}

// Window control handlers

func (s *Server) handleFocus(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	pid, err := findWindowPID(app)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	if err := input.FocusWindow(pid, 0); err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{"ok": true})
}

func (s *Server) handleMinimize(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	pid, err := findWindowPID(app)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	if err := input.MinimizeWindow(pid, 0); err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{"ok": true})
}

func (s *Server) handleRestore(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	pid, err := findWindowPID(app)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	if err := input.RestoreWindow(pid, 0); err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{"ok": true})
}

func (s *Server) handleClose(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	pid, err := findWindowPID(app)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	if err := input.CloseWindow(pid, 0); err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{"ok": true})
}

func (s *Server) handleResize(req *protocol.Request) *protocol.Response {
	if !permissions.HasAccessibilityPermission() {
		return protocol.NewErrorResponse(req, protocol.ErrPermissionAccessibility,
			"Accessibility permission required. Grant access in System Settings > Privacy & Security > Accessibility")
	}

	app := getString(req.Params, "app", "")
	if app == "" {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "app parameter required")
	}

	width := getInt(req.Params, "width", 0)
	height := getInt(req.Params, "height", 0)
	if width <= 0 || height <= 0 {
		return protocol.NewErrorResponse(req, protocol.ErrInvalidParams, "width and height must be positive")
	}

	pid, err := findWindowPID(app)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrWindowNotFound, err.Error())
	}

	if err := input.ResizeWindow(pid, 0, width, height); err != nil {
		return protocol.NewErrorResponse(req, protocol.ErrInternalError, err.Error())
	}

	return protocol.NewResponse(req, map[string]any{
		"ok":     true,
		"width":  width,
		"height": height,
	})
}

// Helper functions

func findWindowPID(appName string) (int, error) {
	windows, err := capture.ListWindows(appName)
	if err != nil {
		return 0, err
	}
	if len(windows) == 0 {
		return 0, fmt.Errorf("no window found for app: %s", appName)
	}
	return windows[0].OwnerPID, nil
}

func getString(params map[string]any, key, defaultVal string) string {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key].(string); ok {
		return v
	}
	return defaultVal
}

func getInt(params map[string]any, key string, defaultVal int) int {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	if v, ok := params[key].(int); ok {
		return v
	}
	return defaultVal
}

func getBool(params map[string]any, key string, defaultVal bool) bool {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key].(bool); ok {
		return v
	}
	return defaultVal
}

func getStringSlice(params map[string]any, key string) []string {
	if params == nil {
		return nil
	}
	if v, ok := params[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	// Handle single string as array
	if v, ok := params[key].(string); ok {
		return []string{v}
	}
	return nil
}
