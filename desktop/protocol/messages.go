package protocol

import (
	"time"

	"github.com/google/uuid"
)

func nowTS() int64 { return time.Now().UnixMilli() }
func newID() string { return uuid.New().String() }

func NewHello(version string, caps []string) map[string]any {
	return map[string]any{
		"type": TypeHello, "id": newID(), "ts": nowTS(),
		"protocolVersion": version, "capabilities": caps,
	}
}

func NewHandoffRequest(btAddr, name string, dir string) map[string]any {
	return map[string]any{
		"type": TypeHandoffRequest, "id": newID(), "ts": nowTS(),
		"btAddress": btAddr, "deviceName": name, "direction": dir,
	}
}

func NewHandoffReady() map[string]any {
	return map[string]any{"type": TypeHandoffReady, "id": newID(), "ts": nowTS()}
}

func NewReleased(btAddr string) map[string]any {
	return map[string]any{"type": TypeReleased, "id": newID(), "ts": nowTS(), "btAddress": btAddr}
}

func NewHandoffDone(btAddr string) map[string]any {
	return map[string]any{"type": TypeHandoffDone, "id": newID(), "ts": nowTS(), "btAddress": btAddr}
}

func NewHandoffFailed(btAddr string, reason string, detail string) map[string]any {
	m := map[string]any{
		"type": TypeHandoffFailed, "id": newID(), "ts": nowTS(),
		"btAddress": btAddr, "reason": reason,
	}
	if detail != "" {
		m["detail"] = detail
	}
	return m
}

func NewBusy(btAddr string) map[string]any {
	m := map[string]any{"type": TypeBusy, "id": newID(), "ts": nowTS()}
	if btAddr != "" {
		m["btAddress"] = btAddr
	}
	return m
}

func NewHeartbeat() map[string]any {
	return map[string]any{"type": TypeHeartbeat, "id": newID(), "ts": nowTS()}
}

// Type accessors
func MsgType(m map[string]any) string { return GetString(m, "type") }
func MsgBtAddress(m map[string]any) string { return GetString(m, "btAddress") }
func MsgReason(m map[string]any) string { return GetString(m, "reason") }
func MsgDirection(m map[string]any) string { return GetString(m, "direction") }
func MsgDeviceName(m map[string]any) string { return GetString(m, "deviceName") }