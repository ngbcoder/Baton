package protocol

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// QrPayload is the data encoded into the pairing QR code.
type QrPayload struct {
	Address         string    `json:"address"`
	Port            int       `json:"port"`
	CertFingerprint string    `json:"certFingerprint"`
	Secret          string    `json:"secret"`
	ExpiresAt       time.Time `json:"expiresAt"`
}

// Encode returns a compact base64url JSON representation suitable for a QR code.
func (q QrPayload) Encode() (string, error) {
	b, err := json.Marshal(q)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeQrPayload parses a base64url-encoded QrPayload.
func DecodeQrPayload(s string) (*QrPayload, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var q QrPayload
	if err := json.Unmarshal(b, &q); err != nil {
		return nil, err
	}
	return &q, nil
}

// ComputeConfirmationCode produces a short human-readable code from a fingerprint.
func ComputeConfirmationCode(fingerprint string) string {
	sum := sha256.Sum256([]byte(fingerprint))
	return fmt.Sprintf("%02X%02X%02X", sum[0], sum[1], sum[2])
}