package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

// Message is the common envelope for all Baton messages.
type Message struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	TS   int64  `json:"ts"`
	// Raw holds the full decoded map for type-specific access.
	Raw map[string]any `json:"-"`
}

// Known message type constants.
const (
	TypeHello          = "HELLO"
	TypeHandoffRequest = "HANDOFF_REQUEST"
	TypeHandoffReady   = "HANDOFF_READY"
	TypeReleased       = "RELEASED"
	TypeHandoffDone    = "HANDOFF_DONE"
	TypeHandoffFailed  = "HANDOFF_FAILED"
	TypeBusy           = "BUSY"
	TypeDeviceList     = "DEVICE_LIST"
	TypeHeartbeat      = "HEARTBEAT"

	// Pairing
	TypePairRequest = "PAIR_REQUEST"
	TypePairAccept  = "PAIR_ACCEPT"
	TypePairConfirm = "PAIR_CONFIRM"
	TypePairReject  = "PAIR_REJECT"
)

// FrameCodec helpers for length-prefixed JSON.
func EncodeFrame(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(b)))
	out := append(header, b...)
	return out, nil
}

func DecodeFrame(data []byte) (map[string]any, int, error) {
	if len(data) < 4 {
		return nil, 0, fmt.Errorf("incomplete header")
	}
	n := int(binary.BigEndian.Uint32(data[:4]))
	if len(data) < 4+n {
		return nil, 0, fmt.Errorf("incomplete body")
	}
	var m map[string]any
	if err := json.Unmarshal(data[4:4+n], &m); err != nil {
		return nil, 0, err
	}
	return m, 4 + n, nil
}

// GetString safely extracts a string field.
func GetString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// MsgType returns the "type" field.
func MsgType(m map[string]any) string {
	return GetString(m, "type")
}

// MsgBtAddress extracts the Bluetooth address from a message.
func MsgBtAddress(m map[string]any) string {
	if a := GetString(m, "btAddress"); a != "" {
		return a
	}
	return GetString(m, "address")
}

// MsgDirection extracts the handoff direction.
func MsgDirection(m map[string]any) string {
	return GetString(m, "direction")
}

// MsgReason extracts a failure reason.
func MsgReason(m map[string]any) string {
	return GetString(m, "reason")
}

// NewMessage creates a base message map with type, id and timestamp.
func NewMessage(msgType string) map[string]any {
	return map[string]any{
		"type": msgType,
		"id":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"ts":   time.Now().UnixMilli(),
	}
}

// NewHello builds a HELLO capability exchange message.
func NewHello(deviceName, version string, caps []string) map[string]any {
	m := NewMessage(TypeHello)
	m["deviceName"] = deviceName
	m["version"] = version
	m["capabilities"] = caps
	return m
}

// NewHeartbeat builds a keep-alive message.
func NewHeartbeat() map[string]any {
	return NewMessage(TypeHeartbeat)
}

// NewHandoffRequest asks the peer to take ownership of a Bluetooth device.
// Signature matches handoff.go: (btAddr, deviceName, direction).
func NewHandoffRequest(btAddr, deviceName, direction string) map[string]any {
	m := NewMessage(TypeHandoffRequest)
	m["btAddress"] = btAddr
	m["btName"] = deviceName
	m["direction"] = direction
	return m
}

// NewHandoffReady signals the peer is prepared to accept the device.
// Signature matches handoff.go: no arguments.
func NewHandoffReady() map[string]any {
	return NewMessage(TypeHandoffReady)
}

// NewReleased signals that the sender has disconnected the device.
func NewReleased(btAddr string) map[string]any {
	m := NewMessage(TypeReleased)
	m["btAddress"] = btAddr
	return m
}

// NewHandoffDone signals that the receiver is now the active owner.
func NewHandoffDone(btAddr string) map[string]any {
	m := NewMessage(TypeHandoffDone)
	m["btAddress"] = btAddr
	return m
}

// NewHandoffFailed reports an error during handoff.
// Signature matches handoff.go: (btAddr, reason, detail).
func NewHandoffFailed(btAddr, reason, detail string) map[string]any {
	m := NewMessage(TypeHandoffFailed)
	m["btAddress"] = btAddr
	m["reason"] = reason
	m["detail"] = detail
	return m
}

// NewBusy rejects a concurrent handoff request.
func NewBusy(btAddr string) map[string]any {
	m := NewMessage(TypeBusy)
	m["btAddress"] = btAddr
	return m
}