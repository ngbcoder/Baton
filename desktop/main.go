package main

import (
	"fmt"
	"github.com/ngbcoder/baton/desktop/protocol"
)

func main() {
	m := map[string]any{
		"type": protocol.TypeHello,
		"id":   "1",
		"ts":   1,
	}
	b, _ := protocol.EncodeFrame(m)
	fmt.Printf("encoded %d bytes\n", len(b))
	msg, n, err := protocol.DecodeFrame(b)
	fmt.Printf("decoded %+v consumed %d err %v\n", msg, n, err)
}