package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// Message is the common envelope for all Baton messages.
type Message struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	TS   int64  `json:"ts"`
	// Raw holds the full decoded map for type-specific access.
	Raw map[string]any `json:"-"`
}

// Envelope for JSON (de)serialization. Concrete types embed Message? We'll use maps.

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

// Helper to get string field.
func GetString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}