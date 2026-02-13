package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client is a JSON-RPC client over Unix socket.
type Client struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Reader
	encoder    *json.Encoder

	mu        sync.Mutex
	connected atomic.Bool
	nextID    atomic.Int32
	timeout   time.Duration

	// NotificationHandler is called when a notification is received.
	NotificationHandler func(*Notification)
}

// NewClient creates a new client for the given Unix socket path.
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    30 * time.Second, // Default timeout
	}
}

// SetTimeout sets the timeout for requests.
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// Connect establishes a connection to the server.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected.Load() {
		return nil
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return err
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.encoder = json.NewEncoder(conn)
	c.connected.Store(true)

	return nil
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return nil
	}

	c.connected.Store(false)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsConnected returns true if connected.
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// Call sends a request and waits for the response.
func (c *Client) Call(method string, params map[string]any) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return nil, fmt.Errorf("not connected")
	}

	req := NewRequest(int(c.nextID.Add(1)), method, params)

	// Set deadline for read/write
	if c.timeout > 0 {
		c.conn.SetDeadline(time.Now().Add(c.timeout))
		defer c.conn.SetDeadline(time.Time{})
	}

	// Send request
	if err := c.encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("write error: %w", err)
	}

	// Read response
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("timeout waiting for response")
		}
		return nil, fmt.Errorf("read error: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &resp, nil
}

// Ping sends a ping request to check if the server is responsive.
func (c *Client) Ping() error {
	resp, err := c.Call(MethodPing, nil)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}
