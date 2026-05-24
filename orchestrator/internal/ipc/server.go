package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type HandlerFunc func(msg *Message)

type Server struct {
	path     string
	listener net.Listener
	handler  HandlerFunc
	conn     net.Conn
	wmu      sync.Mutex
	closed   bool
	mu       sync.Mutex
}

func NewServer(runtimeDir string, handler HandlerFunc) (*Server, error) {
	path := filepath.Join(runtimeDir, "aurora-rag.sock")

	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		return nil, fmt.Errorf("create runtime dir: %w", err)
	}

	os.Remove(path)

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	return &Server{
		path:     path,
		listener: listener,
		handler:  handler,
	}, nil
}

func (s *Server) Path() string { return s.path }

func (s *Server) Accept() error {
	conn, err := s.listener.Accept()
	if err != nil {
		return fmt.Errorf("accept: %w", err)
	}

	s.mu.Lock()
	if s.conn != nil {
		s.mu.Unlock()
		conn.Close()
		return fmt.Errorf("second client rejected")
	}
	s.conn = conn
	s.mu.Unlock()
	return nil
}

func (s *Server) Serve() error {
	for {
		s.mu.Lock()
		conn := s.conn
		closed := s.closed
		s.mu.Unlock()

		if closed || conn == nil {
			return nil
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		msg, err := ReadMessage(conn)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		s.handler(msg)
	}
}

func (s *Server) Send(msg *Message) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()

	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("no client connected")
	}
	_, err := conn.Write(msg.Encode())
	return err
}

func (s *Server) SendError(reqID uint32, payload []byte) error {
	return s.Send(NewMessage(TypeError, reqID, payload))
}

func (s *Server) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	if s.conn != nil {
		s.conn.Close()
	}
	s.listener.Close()
	s.mu.Unlock()
	os.Remove(s.path)
}