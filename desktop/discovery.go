package main

import (
	"context"
	"sync"

	"github.com/grandcat/zeroconf"
)

// DiscoveredPeer mirrors the protocol's DiscoveredPeer concept.
type DiscoveredPeer struct {
	Name    string
	Address string
	Port    int
	TXT     []string
}

// MDNSDiscovery advertises and browses _baton._tcp services.
type MDNSDiscovery struct {
	mu      sync.Mutex
	server  *zeroconf.Server
	entries chan *DiscoveredPeer
	cancel  context.CancelFunc
}

func NewMDNSDiscovery() *MDNSDiscovery {
	return &MDNSDiscovery{entries: make(chan *DiscoveredPeer, 64)}
}

// Peers returns the channel of discovered peers.
func (d *MDNSDiscovery) Peers() <-chan *DiscoveredPeer { return d.entries }

// Advertise starts advertising this device.
func (d *MDNSDiscovery) Advertise(name string, port int) error {
	srv, err := zeroconf.Register(name, "_baton._tcp", "local.", port, []string{"version=1.0"}, nil)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.server = srv
	d.mu.Unlock()
	return nil
}

// Browse starts browsing for peers.
func (d *MDNSDiscovery) Browse() error {
	ctx, cancel := context.WithCancel(context.Background())
	d.mu.Lock()
	d.cancel = cancel
	d.mu.Unlock()

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		cancel()
		return err
	}
	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		for entry := range entries {
			addr := ""
			if len(entry.AddrIPv4) > 0 {
				addr = entry.AddrIPv4[0].String()
			} else if len(entry.AddrIPv6) > 0 {
				addr = entry.AddrIPv6[0].String()
			}
			if addr == "" {
				continue
			}
			peer := &DiscoveredPeer{
				Name:    entry.Instance,
				Address: addr,
				Port:    entry.Port,
				TXT:     entry.Text,
			}
			select {
			case d.entries <- peer:
			default:
			}
		}
	}()
	return resolver.Browse(ctx, "_baton._tcp", "local.", entries)
}

// Stop stops advertising and browsing.
func (d *MDNSDiscovery) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.server != nil {
		d.server.Shutdown()
		d.server = nil
	}
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
}
