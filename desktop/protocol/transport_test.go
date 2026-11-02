package protocol

import (
	"testing"
	"time"
)

func TestTransportRoundTrip(t *testing.T) {
	ts, err := ListenAndServe("127.0.0.1:4060", func(c *TransportConnection) {
		go func() {
			m, _ := c.Receive()
			c.Send(m)
		}()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer ts.Close()

	time.Sleep(100 * time.Millisecond)
	c, err := Dial("127.0.0.1:4060")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	msg := NewHeartbeat()
	if err := c.Send(msg); err != nil {
		t.Fatal(err)
	}
	got, err := c.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if MsgType(got) != TypeHeartbeat {
		t.Fatalf("expected heartbeat, got %s", MsgType(got))
	}
}