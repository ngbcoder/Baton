package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/ngbcoder/baton/desktop/protocol"
)

// ConnectionState represents a transport connection's lifecycle.
type ConnectionState string

const (
	StateDisconnected  ConnectionState = "disconnected"
	StateConnecting    ConnectionState = "connecting"
	StateHandshaking   ConnectionState = "handshaking"
	StateConnected     ConnectionState = "connected"
	StateReconnecting  ConnectionState = "reconnecting"
)

// queuedMessage holds an outbound message with an expiry.
type queuedMessage struct {
	msg       map[string]any
	createdAt time.Time
	ttl       time.Duration
}

func (q queuedMessage) expired() bool {
	return time.Since(q.createdAt) > q.ttl
}

// ManagedConnection wraps a single peer connection with state, heartbeat,
// reconnect, and an outbound queue with backpressure/TTL.
type ManagedConnection struct {
	PeerID string
	Addr   string

	mu    sync.RWMutex
	state ConnectionState
	conn  *protocol.TransportConnection

	subs []chan ConnectionState

	outbound chan queuedMessage
	done     chan struct{}

	heartbeatInterval time.Duration
	reconnectInitial  time.Duration
	reconnectMax      time.Duration
	messageTTL        time.Duration
}

// NewManagedConnection creates a managed connection to addr.
func NewManagedConnection(peerID, addr string) *ManagedConnection {
	mc := &ManagedConnection{
		PeerID:            peerID,
		Addr:              addr,
		state:             StateDisconnected,
		outbound:          make(chan queuedMessage, 256),
		done:              make(chan struct{}),
		heartbeatInterval: 10 * time.Second,
		reconnectInitial:  1 * time.Second,
		reconnectMax:      30 * time.Second,
		messageTTL:       60 * time.Second,
	}
	go mc.run()
	go mc.flushLoop()
	return mc
}

func (mc *ManagedConnection) setState(s ConnectionState) {
	mc.mu.Lock()
	mc.state = s
	subs := append([]chan ConnectionState(nil), mc.subs...)
	mc.mu.Unlock()
	for _, c := range subs {
		select {
		case c <- s:
		default:
		}
	}
}

func (mc *ManagedConnection) State() ConnectionState {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.state
}

// Subscribe returns a channel receiving state changes.
func (mc *ManagedConnection) Subscribe() <-chan ConnectionState {
	ch := make(chan ConnectionState, 16)
	mc.mu.Lock()
	mc.subs = append(mc.subs, ch)
	mc.mu.Unlock()
	return ch
}

func (mc *ManagedConnection) run() {
	delay := mc.reconnectInitial
	for {
		select {
		case <-mc.done:
			return
		default:
		}

		mc.setState(StateConnecting)
		conn, err := protocol.Dial(mc.Addr)
		if err != nil {
			mc.setState(StateReconnecting)
			time.Sleep(delay)
			delay = minDuration(delay*2, mc.reconnectMax)
			continue
		}

		mc.mu.Lock()
		mc.conn = conn
		mc.mu.Unlock()
		mc.setState(StateHandshaking)

		// Hello exchange (best-effort)
		_ = conn.Send(protocol.NewHello("baton", "1.0", []string{"multipoint"}))
		mc.setState(StateConnected)
		delay = mc.reconnectInitial

		// Heartbeat loop
		hb := time.NewTicker(mc.heartbeatInterval)
		readErr := make(chan error, 1)
		go func() {
			for {
				_, err := conn.Receive()
				if err != nil {
					readErr <- err
					return
				}
			}
		}()

	heartbeatLoop:
		for {
			select {
			case <-mc.done:
				hb.Stop()
				conn.Close()
				return
			case <-hb.C:
				if err := conn.Send(protocol.NewHeartbeat()); err != nil {
					break heartbeatLoop
				}
			case err := <-readErr:
				_ = err
				break heartbeatLoop
			}
		}
		hb.Stop()
		conn.Close()
		mc.mu.Lock()
		mc.conn = nil
		mc.mu.Unlock()
		mc.setState(StateReconnecting)
		time.Sleep(delay)
		delay = minDuration(delay*2, mc.reconnectMax)
	}
}

func (mc *ManagedConnection) flushLoop() {
	for {
		select {
		case <-mc.done:
			return
		case qm := <-mc.outbound:
			if qm.expired() {
				continue
			}
			mc.mu.RLock()
			conn := mc.conn
			mc.mu.RUnlock()
			if conn == nil {
				// re-queue if not expired
				if !qm.expired() {
					select {
					case mc.outbound <- qm:
					default:
					}
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			_ = conn.Send(qm.msg)
		}
	}
}

// Send queues a message for delivery (backpressure via bounded channel).
func (mc *ManagedConnection) Send(msg map[string]any) error {
	qm := queuedMessage{msg: msg, createdAt: time.Now(), ttl: mc.messageTTL}
	select {
	case mc.outbound <- qm:
		return nil
	default:
		return fmt.Errorf("outbound queue full")
	}
}

// Receive is not supported on managed connection; use underlying conn.
func (mc *ManagedConnection) Close() {
	close(mc.done)
	mc.mu.Lock()
	if mc.conn != nil {
		mc.conn.Close()
		mc.conn = nil
	}
	mc.mu.Unlock()
}

// ConnectionPool manages multiple peer connections.
type ConnectionPool struct {
	mu    sync.RWMutex
	conns map[string]*ManagedConnection
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{conns: map[string]*ManagedConnection{}}
}

func (p *ConnectionPool) Get(peerID, addr string) *ManagedConnection {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.conns[peerID]; ok {
		return c
	}
	c := NewManagedConnection(peerID, addr)
	p.conns[peerID] = c
	return c
}

func (p *ConnectionPool) Remove(peerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.conns[peerID]; ok {
		c.Close()
		delete(p.conns, peerID)
	}
}

func (p *ConnectionPool) All() []*ManagedConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*ManagedConnection, 0, len(p.conns))
	for _, c := range p.conns {
		out = append(out, c)
	}
	return out
}

// BroadcastDeviceList sends a DEVICE_LIST message to all peers.
func (p *ConnectionPool) BroadcastDeviceList(msg map[string]any) {
	for _, c := range p.All() {
		_ = c.Send(msg)
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
