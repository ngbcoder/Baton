package protocol

import (
	"sync"
	"testing"
	"time"
)

type fakeBT struct {
	mu        sync.Mutex
	paired    map[string]bool
	connected map[string]bool
	multipoint bool
	failConn  bool
	failDisc  bool
}

func newFakeBT() *fakeBT { return &fakeBT{paired: map[string]bool{}, connected: map[string]bool{}} }

func (f *fakeBT) PairedDevices() ([]BtDevice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []BtDevice
	for a := range f.paired {
		out = append(out, BtDevice{Address: a, Name: a})
	}
	return out, nil
}
func (f *fakeBT) IsConnected(addr string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connected[addr], nil
}
func (f *fakeBT) Connect(addr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failConn {
		return errFail
	}
	f.connected[addr] = true
	return nil
}
func (f *fakeBT) Disconnect(addr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failDisc {
		return errFail
	}
	delete(f.connected, addr)
	return nil
}
func (f *fakeBT) IsMultipointConnected(addr string) (bool, error) {
	return f.multipoint, nil
}

type fakeAudio struct {
	mu     sync.Mutex
	output string
	pauses int
}
func (f *fakeAudio) SetDefaultOutput(addr string) error { f.mu.Lock(); defer f.mu.Unlock(); f.output = addr; return nil }
func (f *fakeAudio) PausePlayback() error { f.mu.Lock(); defer f.mu.Unlock(); f.pauses++; return nil }

var errFail = &simpleErr{"fail"}

type simpleErr struct{ msg string }
func (e *simpleErr) Error() string { return e.msg }

func TestHandoffSuccess(t *testing.T) {
	const hp = "AA:BB"
	phoneBT := newFakeBT()
	pcBT := newFakeBT()
	phoneBT.paired[hp] = true
	pcBT.paired[hp] = true
	phoneBT.connected[hp] = true

	phoneAudio := &fakeAudio{}
	pcAudio := &fakeAudio{}

	var phone, pc *HandoffController
	phone = NewHandoffController("phone", phoneBT, phoneAudio, func(peer string, msg map[string]any) error {
		go pc.HandleMessage("phone", msg)
		return nil
	})
	pc = NewHandoffController("pc", pcBT, pcAudio, func(peer string, msg map[string]any) error {
		go phone.HandleMessage("pc", msg)
		return nil
	})
	phone.HandoffTimeout = 2 * time.Second
	pc.HandoffTimeout = 2 * time.Second

	res := phone.Initiate("pc", hp, "hp", DirPhoneToComputer, true)
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if !pcBT.connected[hp] {
		t.Fatal("pc not connected")
	}
	if phoneBT.connected[hp] {
		t.Fatal("phone still connected")
	}
	if pcAudio.output != hp {
		t.Fatal("pc default output not set")
	}
}

func TestHandoffRollbackOnConnectFail(t *testing.T) {
	const hp = "AA:BB"
	phoneBT := newFakeBT()
	pcBT := newFakeBT()
	phoneBT.paired[hp] = true
	pcBT.paired[hp] = true
	phoneBT.connected[hp] = true
	pcBT.failConn = true

	phoneAudio := &fakeAudio{}
	pcAudio := &fakeAudio{}

	var phone, pc *HandoffController
	phone = NewHandoffController("phone", phoneBT, phoneAudio, func(peer string, msg map[string]any) error {
		go pc.HandleMessage("phone", msg)
		return nil
	})
	pc = NewHandoffController("pc", pcBT, pcAudio, func(peer string, msg map[string]any) error {
		go phone.HandleMessage("pc", msg)
		return nil
	})
	phone.HandoffTimeout = 2 * time.Second
	pc.HandoffTimeout = 2 * time.Second

	res := phone.Initiate("pc", hp, "hp", DirPhoneToComputer, true)
	if res.Success {
		t.Fatal("expected failure")
	}
	if !phoneBT.connected[hp] {
		t.Fatal("phone did not reconnect")
	}
}

func TestHandoffMultipoint(t *testing.T) {
	const hp = "AA:BB"
	phoneBT := newFakeBT()
	pcBT := newFakeBT()
	phoneBT.paired[hp] = true
	pcBT.paired[hp] = true
	phoneBT.connected[hp] = true
	phoneBT.multipoint = true

	phoneAudio := &fakeAudio{}
	pcAudio := &fakeAudio{}

	var phone, pc *HandoffController
	phone = NewHandoffController("phone", phoneBT, phoneAudio, func(peer string, msg map[string]any) error {
		go pc.HandleMessage("phone", msg)
		return nil
	})
	pc = NewHandoffController("pc", pcBT, pcAudio, func(peer string, msg map[string]any) error {
		go phone.HandleMessage("pc", msg)
		return nil
	})
	phone.HandoffTimeout = 2 * time.Second
	pc.HandoffTimeout = 2 * time.Second

	res := phone.Initiate("pc", hp, "hp", DirPhoneToComputer, true)
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if phoneAudio.pauses != 1 {
		t.Fatalf("expected 1 pause, got %d", phoneAudio.pauses)
	}
	if !phoneBT.connected[hp] {
		t.Fatal("phone disconnected in multipoint")
	}
}