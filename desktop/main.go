package main

import (
	"fmt"
	"os"

	"github.com/ngbcoder/baton/desktop/protocol"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: baton serve|connect <addr>")
		return
	}
	switch os.Args[1] {
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
	default:
		fmt.Println("unknown command")
	}
}