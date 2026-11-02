package protocol

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"sync"
)

// TransportConnection wraps a TCP connection with length-prefixed JSON framing.
type TransportConnection struct {
	conn net.Conn
	mu   sync.Mutex
	r    *bufio.Reader
}

func NewTransportConnection(conn net.Conn) *TransportConnection {
	return &TransportConnection{conn: conn, r: bufio.NewReader(conn)}
}

func (t *TransportConnection) Send(msg map[string]any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	b, err := EncodeFrame(msg)
	if err != nil {
		return err
	}
	_, err = t.conn.Write(b)
	return err
}

// Receive reads one framed message.
func (t *TransportConnection) Receive() (map[string]any, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(t.r, header); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(header)
	body := make([]byte, n)
	if _, err := io.ReadFull(t.r, body); err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (t *TransportConnection) Close() error { return t.conn.Close() }

// Server
type TransportServer struct {
	listener net.Listener
}

func ListenAndServe(addr string, onConn func(*TransportConnection)) (*TransportServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	ts := &TransportServer{listener: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			onConn(NewTransportConnection(c))
		}
	}()
	return ts, nil
}

func (ts *TransportServer) Close() error { return ts.listener.Close() }

// Client
func Dial(addr string) (*TransportConnection, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewTransportConnection(c), nil
}

