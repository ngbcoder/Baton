package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	defaultReadTimeout  = 30 * time.Second
	defaultWriteTimeout = 10 * time.Second
	maxMessageSize      = 1 << 20 // 1 MiB
	headerSize          = 4
)

// TransportConnection is a single connection (plain TCP or TLS).
type TransportConnection struct {
	conn   net.Conn
	mu     sync.Mutex
	closed bool
}

// NewTransportConnection wraps a net.Conn.
func NewTransportConnection(c net.Conn) *TransportConnection {
	return &TransportConnection{conn: c}
}

// Close closes the underlying connection.
func (c *TransportConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.conn.Close()
}

// Send writes a length-prefixed JSON message.
func (c *TransportConnection) Send(m map[string]any) error {
	data, err := EncodeFrame(m)
	if err != nil {
		return err
	}
	if len(data) > maxMessageSize+headerSize {
		return fmt.Errorf("message too large: %d bytes", len(data))
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("connection closed")
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	_, err = c.conn.Write(data)
	return err
}

// Receive reads the next length-prefixed JSON message.
func (c *TransportConnection) Receive() (map[string]any, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("connection closed")
	}
	c.mu.Unlock()

	_ = c.conn.SetReadDeadline(time.Now().Add(defaultReadTimeout))

	var hdr [headerSize]byte
	if _, err := io.ReadFull(c.conn, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > maxMessageSize {
		return nil, fmt.Errorf("invalid message length: %d", n)
	}

	buf := make([]byte, n)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// TransportServer accepts incoming connections.
type TransportServer struct {
	listener net.Listener
}

// Close shuts down the listener.
func (s *TransportServer) Close() error {
	return s.listener.Close()
}

// ListenAndServe starts a plain TCP server.
func ListenAndServe(addr string, handler func(*TransportConnection)) (*TransportServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &TransportServer{listener: ln}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			handler(NewTransportConnection(conn))
		}
	}()
	return s, nil
}

// Dial connects to a remote Baton peer over plain TCP.
func Dial(addr string) (*TransportConnection, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return NewTransportConnection(conn), nil
}