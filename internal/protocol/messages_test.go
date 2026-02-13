package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		want    string
	}{
		{
			name: "simple request",
			request: Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "ping",
			},
			want: `{"jsonrpc":"2.0","id":1,"method":"ping"}`,
		},
		{
			name: "request with params",
			request: Request{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "click",
				Params: map[string]any{
					"app": "Safari",
					"x":   100,
					"y":   50,
				},
			},
			want: `{"jsonrpc":"2.0","id":2,"method":"click","params":{"app":"Safari","x":100,"y":50}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.request)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestRequest_Unmarshal(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":42,"method":"capture_window","params":{"app":"Safari","format":"webp"}}`

	var req Request
	err := json.Unmarshal([]byte(input), &req)
	require.NoError(t, err)

	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, 42, req.ID)
	assert.Equal(t, "capture_window", req.Method)
	assert.Equal(t, "Safari", req.Params["app"])
	assert.Equal(t, "webp", req.Params["format"])
}

func TestResponse_Success(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]any{
			"status": "ok",
		},
	}

	got, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`, string(got))
}

func TestResponse_Error(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Error: &Error{
			Code:    ErrPermissionAccessibility,
			Message: "Accessibility permission required",
		},
	}

	got, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(got), `"code":-32001`)
	assert.Contains(t, string(got), `"message":"Accessibility permission required"`)
}

func TestNotification_Marshal(t *testing.T) {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "stopping",
		Params: map[string]any{
			"reason": "user_initiated",
		},
	}

	got, err := json.Marshal(notif)
	require.NoError(t, err)
	assert.JSONEq(t, `{"jsonrpc":"2.0","method":"stopping","params":{"reason":"user_initiated"}}`, string(got))
}

func TestNewRequest(t *testing.T) {
	req := NewRequest(1, MethodPing, nil)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, 1, req.ID)
	assert.Equal(t, MethodPing, req.Method)
}

func TestNewResponse(t *testing.T) {
	req := NewRequest(42, MethodHello, nil)
	resp := NewResponse(req, map[string]any{"version": "1.0.0"})

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 42, resp.ID)
	assert.NotNil(t, resp.Result)
	assert.Nil(t, resp.Error)
}

func TestNewErrorResponse(t *testing.T) {
	req := NewRequest(42, MethodCapture, nil)
	resp := NewErrorResponse(req, ErrWindowNotFound, "Window not found: Safari")

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 42, resp.ID)
	assert.Nil(t, resp.Result)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrWindowNotFound, resp.Error.Code)
	assert.Equal(t, "Window not found: Safari", resp.Error.Message)
}

func TestError_Error(t *testing.T) {
	e := &Error{
		Code:    ErrPermissionAccessibility,
		Message: "Accessibility permission required",
	}
	assert.Equal(t, "Accessibility permission required", e.Error())
}

func TestMethodConstants(t *testing.T) {
	// Verify method constants are defined
	assert.Equal(t, "hello", MethodHello)
	assert.Equal(t, "ping", MethodPing)
	assert.Equal(t, "shutdown", MethodShutdown)
	assert.Equal(t, "list_windows", MethodListWindows)
	assert.Equal(t, "capture_window", MethodCaptureWindow)
	assert.Equal(t, "capture_screen", MethodCaptureScreen)
	assert.Equal(t, "click", MethodClick)
	assert.Equal(t, "move", MethodMove)
	assert.Equal(t, "type", MethodType)
	assert.Equal(t, "key", MethodKey)
	assert.Equal(t, "scroll", MethodScroll)
	assert.Equal(t, "focus", MethodFocus)
	assert.Equal(t, "minimize", MethodMinimize)
	assert.Equal(t, "restore", MethodRestore)
	assert.Equal(t, "close", MethodClose)
	assert.Equal(t, "resize", MethodResize)
	assert.Equal(t, "open_settings", MethodOpenSettings)
	assert.Equal(t, "stopping", MethodStopping)
}

func TestErrorCodeConstants(t *testing.T) {
	// Verify error codes match JSON-RPC spec and our custom codes
	assert.Equal(t, -32001, ErrPermissionAccessibility)
	assert.Equal(t, -32002, ErrPermissionScreenRecording)
	assert.Equal(t, -32003, ErrWindowNotFound)
	assert.Equal(t, -32004, ErrAgentShuttingDown)
	assert.Equal(t, -32600, ErrInvalidRequest)
	assert.Equal(t, -32601, ErrMethodNotFound)
	assert.Equal(t, -32602, ErrInvalidParams)
}

func TestNewNotification(t *testing.T) {
	notif := NewNotification(MethodStopping, map[string]any{"reason": "idle_timeout"})

	assert.Equal(t, "2.0", notif.JSONRPC)
	assert.Equal(t, MethodStopping, notif.Method)
	assert.NotNil(t, notif.Params)
	assert.Equal(t, "idle_timeout", notif.Params["reason"])
}

func TestNewNotification_NilParams(t *testing.T) {
	notif := NewNotification(MethodPing, nil)

	assert.Equal(t, "2.0", notif.JSONRPC)
	assert.Equal(t, MethodPing, notif.Method)
	assert.Nil(t, notif.Params)
}
