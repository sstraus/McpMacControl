// Package protocol provides JSON-RPC 2.0 types for agent-client IPC.
package protocol

// JSON-RPC version constant.
const JSONRPCVersion = "2.0"

// Method constants for all supported operations.
const (
	// Control methods
	MethodHello        = "hello"
	MethodPing         = "ping"
	MethodShutdown     = "shutdown"
	MethodOpenSettings = "open_settings"

	// Capture methods
	MethodListWindows   = "list_windows"
	MethodCaptureWindow = "capture_window"
	MethodCaptureScreen = "capture_screen"
	MethodCapture       = "capture" // Alias

	// Input methods
	MethodClick  = "click"
	MethodMove   = "move"
	MethodType   = "type"
	MethodKey    = "key"
	MethodScroll = "scroll"

	// Window control methods
	MethodFocus    = "focus"
	MethodMinimize = "minimize"
	MethodRestore  = "restore"
	MethodClose    = "close"
	MethodResize   = "resize"

	// Notification methods (agent → client)
	MethodStopping = "stopping"
)

// Error codes following JSON-RPC 2.0 spec plus custom codes.
const (
	// Custom application errors (-32000 to -32099)
	ErrPermissionAccessibility   = -32001
	ErrPermissionScreenRecording = -32002
	ErrWindowNotFound            = -32003
	ErrAgentShuttingDown         = -32004

	// JSON-RPC standard errors
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternalError  = -32603
	ErrParseError     = -32700
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification (no ID, no response expected).
type Notification struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Message
}

// NewRequest creates a new JSON-RPC request.
func NewRequest(id int, method string, params map[string]any) *Request {
	return &Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}
}

// NewResponse creates a successful JSON-RPC response.
func NewResponse(req *Request, result any) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

// NewErrorResponse creates an error JSON-RPC response.
func NewErrorResponse(req *Request, code int, message string) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// NewNotification creates a JSON-RPC notification (no response expected).
func NewNotification(method string, params map[string]any) *Notification {
	return &Notification{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  params,
	}
}

// StopReason constants for the "stopping" notification.
const (
	StopReasonUserInitiated = "user_initiated"
	StopReasonIdleTimeout   = "idle_timeout"
	StopReasonShutdown      = "shutdown"
)
