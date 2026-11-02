package protocol

import (
	"testing"
)

func TestTLSRoundTrip(t *testing.T) {
	cert, err := GenerateSelfSignedCert("baton")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := TLSConfigFromPEM(cert.CertPEM, cert.KeyPEM, true)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := ListenAndServeTLS("127.0.0.1:4070", cfg, func(c *TransportConnection) {
		go func() {
			m, _ := c.Receive()
			c.Send(m)
		}()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer ts.Close()

	clientCfg, _ := TLSConfigFromPEM(cert.CertPEM, cert.KeyPEM, false)
	clientCfg.InsecureSkipVerify = true
	conn, err := DialTLS("127.0.0.1:4070", clientCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	msg := NewHeartbeat()
	if err := conn.Send(msg); err != nil {
		t.Fatal(err)
	}
	got, err := conn.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if MsgType(got) != TypeHeartbeat {
		t.Fatalf("expected heartbeat, got %s", MsgType(got))
	}
}