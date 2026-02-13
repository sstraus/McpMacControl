package protocol

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
)

// Handler is a function that processes a request and returns a response.
type Handler func(req *Request) *Response

// Server is a JSON-RPC server over Unix socket.
type Server struct {
	socketPath string
	listener   net.Listener
	handlers   map[string]Handler
	handlersMu sync.RWMutex

	connCount atomic.Int32
	conns     sync.Map // map[net.Conn]struct{}

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewServer creates a new server listening on the given Unix socket path.
func NewServer(socketPath string) *Server {
	return &Server{
		socketPath: socketPath,
		handlers:   make(map[string]Handler),
		stopCh:     make(chan struct{}),
	}
}

// RegisterHandler registers a handler for the given method.
func (s *Server) RegisterHandler(method string, handler Handler) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.handlers[method] = handler
}

// Start begins listening for connections.
func (s *Server) Start() error {
	// Remove stale socket file if exists
	if _, err := os.Stat(s.socketPath); err == nil {
		os.Remove(s.socketPath)
	}

	// Set umask to ensure socket has 0600 permissions
	oldMask := syscall.Umask(0077)
	defer syscall.Umask(oldMask)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = listener

	// Ensure permissions (umask may not be sufficient on all systems)
	os.Chmod(s.socketPath, 0600)

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop closes the server and all connections.
func (s *Server) Stop() error {
	close(s.stopCh)

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all connections
	s.conns.Range(func(key, value any) bool {
		conn := key.(net.Conn)
		conn.Close()
		return true
	})

	s.wg.Wait()

	// Remove socket file
	os.Remove(s.socketPath)

	return nil
}

// ConnectionCount returns the number of active connections.
func (s *Server) ConnectionCount() int32 {
	return s.connCount.Load()
}

// SocketPath returns the socket path.
func (s *Server) SocketPath() string {
	return s.socketPath
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				continue
			}
		}

		s.conns.Store(conn, struct{}{})
		s.connCount.Add(1)

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.conns.Delete(conn)
		s.connCount.Add(-1)
		s.wg.Done()
	}()

	reader := bufio.NewReader(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		// Read line-delimited JSON
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := &Response{
				JSONRPC: JSONRPCVersion,
				Error: &Error{
					Code:    ErrParseError,
					Message: "Parse error: " + err.Error(),
				},
			}
			encoder.Encode(resp)
			continue
		}

		resp := s.dispatch(&req)
		if err := encoder.Encode(resp); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(req *Request) *Response {
	s.handlersMu.RLock()
	handler, ok := s.handlers[req.Method]
	s.handlersMu.RUnlock()

	if !ok {
		return NewErrorResponse(req, ErrMethodNotFound, "Method not found: "+req.Method)
	}

	return handler(req)
}

// BroadcastNotification sends a notification to all connected clients.
func (s *Server) BroadcastNotification(notif *Notification) {
	data, err := json.Marshal(notif)
	if err != nil {
		return
	}
	data = append(data, '\n')

	s.conns.Range(func(key, value any) bool {
		conn := key.(net.Conn)
		conn.Write(data)
		return true
	})
}
