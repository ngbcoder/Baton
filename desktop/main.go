package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ngbcoder/baton/desktop/protocol"
)

func main() {
	if len(os.Args) < 2 {
		RunGUI()
		return
	}
	switch os.Args[1] {
	case "gui":
		RunGUI()
	case "serve":
		addr := ":4040"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		ts, err := protocol.ListenAndServe(addr, func(c *protocol.TransportConnection) {
			go func() {
				for {
					m, err := c.Receive()
					if err != nil {
						return
					}
					fmt.Printf("received: %s\n", protocol.MsgType(m))
					c.Send(m)
				}
			}()
		})
		if err != nil {
			fmt.Println("serve error:", err)
			return
		}
		_ = ts
		fmt.Println("serving on", addr)
		select {}
	case "qr":
		if len(os.Args) < 3 {
			fmt.Println("usage: baton qr <text>")
			return
		}
		out, err := protocol.RenderQrAscii(os.Args[2])
		if err != nil {
			fmt.Println("qr error:", err)
			return
		}
		fmt.Println(out)
	case "connect":
		if len(os.Args) < 3 {
			fmt.Println("usage: baton connect <addr>")
			return
		}
		c, err := protocol.Dial(os.Args[2])
		if err != nil {
			fmt.Println("connect error:", err)
			return
		}
		defer c.Close()
		fmt.Println("connected to", os.Args[2])
		c.Send(protocol.NewHeartbeat())
		for {
			m, err := c.Receive()
			if err != nil {
				fmt.Println("recv error:", err)
				return
			}
			fmt.Printf("received: %s\n", protocol.MsgType(m))
		}
	case "bt-list":
		bt := newMockBluetooth()
		devices, err := bt.PairedDevices()
		if err != nil {
			fmt.Println("bt-list error:", err)
			return
		}
		fmt.Println("Paired Bluetooth devices:")
		for _, d := range devices {
			fmt.Printf("- %s (%s)\n", d.Name, d.Address)
		}
	case "pair":
		addr := "0.0.0.0:4040"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		secret := "baton-pairing-secret-777"
		cert, err := protocol.GenerateSelfSignedCert("Baton Desktop")
		if err != nil {
			fmt.Println("cert error:", err)
			return
		}
		qr := protocol.QrPayload{
			Address:         "127.0.0.1",
			Port:            4040,
			CertFingerprint: cert.Fingerprint,
			Secret:          secret,
			ExpiresAt:       time.Now().Add(2 * time.Minute),
		}
		encoded, _ := qr.Encode()
		ascii, err := protocol.RenderQrAscii(encoded)
		if err != nil {
			fmt.Println("qr render error:", err)
			return
		}
		fmt.Println("Scan this QR code with the Baton Android App to pair:")
		fmt.Println(ascii)
		fmt.Println("Pairing server listening on", addr)

		tlsCfg, err := protocol.TLSConfigFromPEM(cert.CertPEM, cert.KeyPEM, true)
		if err != nil {
			fmt.Println("tls config error:", err)
			return
		}
		ts, err := protocol.ListenAndServeTLS(addr, tlsCfg, func(c *protocol.TransportConnection) {
			go func() {
				defer c.Close()
				msg, err := c.Receive()
				if err != nil {
					fmt.Println("pairing receive error:", err)
					return
				}
				if protocol.MsgType(msg) == protocol.TypePairRequest {
					reqSecret := protocol.GetString(msg, "secret")
					if reqSecret != secret {
						fmt.Println("Pairing rejected: secret mismatch")
						c.Send(map[string]any{"type": protocol.TypePairReject, "reason": "invalid secret"})
						return
					}
					code := protocol.ComputeConfirmationCode(cert.Fingerprint)
					fmt.Printf("Confirmation Code: %s\n", code)
					err = c.Send(map[string]any{
						"type":             protocol.TypePairAccept,
						"id":               "pair-accept-1",
						"ts":               time.Now().UnixMilli(),
						"deviceName":       "Baton Desktop",
						"certFingerprint":  cert.Fingerprint,
						"confirmationCode": code,
					})
					if err != nil {
						fmt.Println("send accept error:", err)
						return
					}
					confirmMsg, err := c.Receive()
					if err != nil {
						fmt.Println("receive confirm error:", err)
						return
					}
					if protocol.MsgType(confirmMsg) == protocol.TypePairConfirm {
						fmt.Println("🎉 Successfully paired with Android device!")
					}
				}
			}()
		})
		if err != nil {
			fmt.Println("pairing server error:", err)
			return
		}
		_ = ts
		select {}
	default:
		fmt.Println("unknown command. Usage: baton [gui|serve|connect|qr|bt-list|pair]")
	}
}