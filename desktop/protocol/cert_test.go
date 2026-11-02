package protocol

import "testing"

func TestGenerateSelfSignedCert(t *testing.T) {
	c, err := GenerateSelfSignedCert("baton-device")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.CertPEM) == 0 || len(c.KeyPEM) == 0 {
		t.Fatal("empty pem")
	}
	if len(c.Fingerprint) != 64 {
		t.Fatalf("bad fingerprint length: %d", len(c.Fingerprint))
	}
}