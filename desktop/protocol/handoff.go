package protocol

import (
	"sync"
	"time"
)

type HandoffState string

const (
	StateIdle       HandoffState = "idle"
	StateRequested  HandoffState = "requested"
	StatePeerReady  HandoffState = "peerReady"
	StateReleasing  HandoffState = "releasing"
	StateReleased   HandoffState = "released"
	StateConnecting  HandoffState = "connecting"
	StateActive     HandoffState = "active"
	StateRollback   HandoffState = "rollback"
)

type HandoffDirection string

const (
	DirPhoneToComputer HandoffDirection = "phoneToComputer"
	DirComputerToPhone HandoffDirection = "computerToPhone"
)

type HandoffFailureReason string

const (
	ReasonDeviceUnavailable HandoffFailureReason = "deviceUnavailable"
	ReasonBusy             HandoffFailureReason = "busy"
	ReasonTimeout          HandoffFailureReason = "timeout"
	ReasonDisconnectFailed HandoffFailureReason = "disconnectFailed"
	ReasonConnectFailed    HandoffFailureReason = "connectFailed"
	ReasonOutOfRange       HandoffFailureReason = "outOfRange"
	ReasonRejected         HandoffFailureReason = "rejected"
	ReasonUnknown          HandoffFailureReason = "unknown"
)

type HandoffResult struct {
	Success bool
	Reason  HandoffFailureReason
	Detail  string
}

type HandoffUpdate struct {
	BtAddress string
	State     HandoffState
	Reason    HandoffFailureReason
	Detail    string
}

type BtDevice struct {
	Address string
	Name    string
}

type BluetoothController interface {
	PairedDevices() ([]BtDevice, error)
	IsConnected(addr string) (bool, error)
	Connect(addr string) error
	Disconnect(addr string) error
	IsMultipointConnected(addr string) (bool, error)
}

type AudioController interface {
	SetDefaultOutput(addr string) error
	PausePlayback() error
}

type session struct {
	btAddress      string
	peerID         string
	direction      HandoffDirection
	state          HandoffState
	readyCh        chan map[string]any
	doneCh         chan map[string]any
	failureReason  HandoffFailureReason
	mu             sync.Mutex
}

type HandoffController struct {
	LocalDeviceID  string
	Bluetooth      BluetoothController
	Audio          AudioController
	Send           func(peerID string, msg map[string]any) error
	HandoffTimeout time.Duration
	Cooldown       time.Duration

	mu         sync.Mutex
	sessions   map[string]*session
	lastHandoff map[string]time.Time
	Updates    chan HandoffUpdate
}

func NewHandoffController(localID string, bt BluetoothController, audio AudioController, send func(string, map[string]any) error) *HandoffController {
	return &HandoffController{
		LocalDeviceID:   localID,
		Bluetooth:       bt,
		Audio:           audio,
		Send:            send,
		HandoffTimeout:  10 * time.Second,
		Cooldown:        30 * time.Second,
		sessions:        map[string]*session{},
		lastHandoff:     map[string]time.Time{},
		Updates:         make(chan HandoffUpdate, 64),
	}
}

func (h *HandoffController) emit(u HandoffUpdate) {
	select {
	case h.Updates <- u:
	default:
	}
}

func (h *HandoffController) isCoolingDown(addr string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	last, ok := h.lastHandoff[addr]
	if !ok {
		return false
	}
	return time.Since(last) < h.Cooldown
}

func (h *HandoffController) Initiate(peerID, btAddr, deviceName string, direction HandoffDirection, force bool) HandoffResult {
	if !force && h.isCoolingDown(btAddr) {
		return HandoffResult{Success: false, Reason: ReasonBusy, Detail: "cooldown"}
	}

	h.mu.Lock()
	existing, ok := h.sessions[btAddr]
	if ok && existing.state != StateIdle {
		h.mu.Unlock()
		return HandoffResult{Success: false, Reason: ReasonBusy, Detail: "in-flight"}
	}
	sess := &session{
		btAddress: btAddr,
		peerID:    peerID,
		direction: direction,
		state:     StateRequested,
		readyCh:   make(chan map[string]any, 1),
		doneCh:    make(chan map[string]any, 1),
	}
	h.sessions[btAddr] = sess
	h.mu.Unlock()

	h.emit(HandoffUpdate{BtAddress: btAddr, State: StateRequested})

	// send request
	req := NewHandoffRequest(btAddr, deviceName, string(direction))
	if err := h.Send(peerID, req); err != nil {
		h.rollback(sess, ReasonUnknown)
		return HandoffResult{Success: false, Reason: ReasonUnknown, Detail: err.Error()}
	}

	// wait ready
	select {
	case <-sess.readyCh:
	case <-time.After(h.HandoffTimeout):
		h.rollback(sess, ReasonTimeout)
		return HandoffResult{Success: false, Reason: ReasonTimeout}
	}
	sess.state = StatePeerReady
	h.emit(HandoffUpdate{BtAddress: btAddr, State: StatePeerReady})

	// multipoint shortcut
	mp, _ := h.Bluetooth.IsMultipointConnected(btAddr)
	if mp {
		_ = h.Audio.PausePlayback()
		sess.state = StateActive
		h.emit(HandoffUpdate{BtAddress: btAddr, State: StateActive})
		h.mu.Lock()
		h.lastHandoff[btAddr] = time.Now()
		h.mu.Unlock()
		h.cleanup(sess)
		return HandoffResult{Success: true}
	}

	// releasing
	sess.state = StateReleasing
	h.emit(HandoffUpdate{BtAddress: btAddr, State: StateReleasing})
	if err := h.Bluetooth.Disconnect(btAddr); err != nil {
		h.Send(peerID, NewHandoffFailed(btAddr, string(ReasonDisconnectFailed), err.Error()))
		h.rollback(sess, ReasonDisconnectFailed)
		return HandoffResult{Success: false, Reason: ReasonDisconnectFailed}
	}

	// released
	sess.state = StateReleased
	h.emit(HandoffUpdate{BtAddress: btAddr, State: StateReleased})
	// Assign done channel before sending RELEASED to handle race.
	doneCh := make(chan map[string]any, 1)
	sess.mu.Lock()
	sess.doneCh = doneCh
	sess.mu.Unlock()

	if err := h.Send(peerID, NewReleased(btAddr)); err != nil {
		h.rollback(sess, ReasonUnknown)
		return HandoffResult{Success: false, Reason: ReasonUnknown}
	}

	// wait done
	select {
	case res := <-doneCh:
		if res == nil {
			reason := sess.failureReason
			if reason == "" {
				reason = ReasonUnknown
			}
			h.rollback(sess, reason)
			return HandoffResult{Success: false, Reason: reason}
		}
	case <-time.After(h.HandoffTimeout):
		reason := sess.failureReason
		if reason == "" {
			reason = ReasonTimeout
		}
		h.rollback(sess, reason)
		return HandoffResult{Success: false, Reason: reason}
	}

	sess.state = StateActive
	h.emit(HandoffUpdate{BtAddress: btAddr, State: StateActive})
	h.mu.Lock()
	h.lastHandoff[btAddr] = time.Now()
	h.mu.Unlock()
	h.cleanup(sess)
	return HandoffResult{Success: true}
}

func (h *HandoffController) HandleRequest(peerID string, req map[string]any) {
	btAddr := MsgBtAddress(req)

	h.mu.Lock()
	existing, ok := h.sessions[btAddr]
	if ok && existing.state != StateIdle {
		h.mu.Unlock()
		h.Send(peerID, NewBusy(btAddr))
		return
	}
	// availability
	devs, _ := h.Bluetooth.PairedDevices()
	known := false
	for _, d := range devs {
		if d.Address == btAddr {
			known = true
			break
		}
	}
	if !known {
		h.mu.Unlock()
		h.Send(peerID, NewHandoffFailed(btAddr, string(ReasonDeviceUnavailable), "not paired"))
		return
	}
	sess := &session{
		btAddress: btAddr,
		peerID:    peerID,
		direction: HandoffDirection(MsgDirection(req)),
		state:     StatePeerReady,
		readyCh:   make(chan map[string]any, 1),
		doneCh:    make(chan map[string]any, 1),
	}
	h.sessions[btAddr] = sess
	h.mu.Unlock()

	h.emit(HandoffUpdate{BtAddress: btAddr, State: StatePeerReady})

	h.Send(peerID, NewHandoffReady())
}

func (h *HandoffController) HandleMessage(peerID string, msg map[string]any) {
	switch MsgType(msg) {
	case TypeHandoffRequest:
		h.HandleRequest(peerID, msg)
	case TypeHandoffReady:
		h.mu.Lock()
		var sess *session
		for _, s := range h.sessions {
			if s.peerID == peerID && s.state == StateRequested {
				sess = s
				break
			}
		}
		h.mu.Unlock()
		if sess != nil {
			select {
			case sess.readyCh <- msg:
			default:
			}
		}
	case TypeReleased:
		btAddr := MsgBtAddress(msg)
		h.mu.Lock()
		sess, ok := h.sessions[btAddr]
		h.mu.Unlock()
		if !ok {
			return
		}
		sess.state = StateConnecting
		h.emit(HandoffUpdate{BtAddress: btAddr, State: StateConnecting})
		if err := h.Bluetooth.Connect(btAddr); err != nil {
			sess.state = StateIdle
			h.Send(peerID, NewHandoffFailed(btAddr, string(ReasonConnectFailed), err.Error()))
			h.mu.Lock()
			delete(h.sessions, btAddr)
			h.mu.Unlock()
			return
		}
		if err := h.Audio.SetDefaultOutput(btAddr); err != nil {
			// still done
		}
		sess.state = StateActive
		h.emit(HandoffUpdate{BtAddress: btAddr, State: StateActive})
		h.Send(peerID, NewHandoffDone(btAddr))
		h.mu.Lock()
		h.lastHandoff[btAddr] = time.Now()
		delete(h.sessions, btAddr)
		h.mu.Unlock()
	case TypeHandoffDone:
		btAddr := MsgBtAddress(msg)
		h.mu.Lock()
		sess, ok := h.sessions[btAddr]
		h.mu.Unlock()
		if !ok {
			return
		}
		sess.mu.Lock()
		ch := sess.doneCh
		sess.mu.Unlock()
		select {
		case ch <- msg:
		default:
		}
	case TypeHandoffFailed:
		btAddr := MsgBtAddress(msg)
		h.mu.Lock()
		sess, ok := h.sessions[btAddr]
		h.mu.Unlock()
		if !ok {
			return
		}
		reason := HandoffFailureReason(MsgReason(msg))
		if reason == "" {
			reason = ReasonUnknown
		}
		sess.failureReason = reason
		sess.mu.Lock()
		ch := sess.doneCh
		sess.mu.Unlock()
		h.emit(HandoffUpdate{BtAddress: btAddr, State: StateRollback, Reason: reason})
		h.rollback(sess, reason)
		select {
		case ch <- nil:
		default:
		}
		h.mu.Lock()
		delete(h.sessions, btAddr)
		h.mu.Unlock()
	case TypeBusy:
		btAddr := MsgBtAddress(msg)
		if btAddr == "" {
			// find first session
			h.mu.Lock()
			for k := range h.sessions {
				btAddr = k
				break
			}
			h.mu.Unlock()
		}
		h.mu.Lock()
		sess, ok := h.sessions[btAddr]
		h.mu.Unlock()
		if !ok {
			return
		}
		sess.failureReason = ReasonBusy
		h.rollback(sess, ReasonBusy)
		sess.mu.Lock()
		ch := sess.doneCh
		sess.mu.Unlock()
		select {
		case ch <- nil:
		default:
		}
		h.mu.Lock()
		delete(h.sessions, btAddr)
		h.mu.Unlock()
	}
}

func (h *HandoffController) rollback(sess *session, reason HandoffFailureReason) {
	sess.state = StateRollback
	h.emit(HandoffUpdate{BtAddress: sess.btAddress, State: StateRollback, Reason: reason})
	// reconnect locally
	connected, _ := h.Bluetooth.IsConnected(sess.btAddress)
	if !connected {
		_ = h.Bluetooth.Connect(sess.btAddress)
	}
	_ = h.Audio.SetDefaultOutput(sess.btAddress)
	sess.state = StateIdle
	h.emit(HandoffUpdate{BtAddress: sess.btAddress, State: StateIdle})
}

func (h *HandoffController) cleanup(sess *session) {
	h.mu.Lock()
	delete(h.sessions, sess.btAddress)
	h.mu.Unlock()
}