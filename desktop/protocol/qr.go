package protocol

import (
	"fmt"
	"strings"
)

// RenderQrAscii produces a simple block-style ASCII representation of the payload.
func RenderQrAscii(data string) (string, error) {
	const size = 21

	var b strings.Builder
	b.WriteString("┌" + strings.Repeat("─", size*2) + "┐\n")

	for y := 0; y < size; y++ {
		b.WriteString("│")
		for x := 0; x < size; x++ {
			on := false
			if (x < 7 && y < 7) || (x >= size-7 && y < 7) || (x < 7 && y >= size-7) {
				if x == 0 || x == 6 || y == 0 || y == 6 ||
					(x >= 2 && x <= 4 && y >= 2 && y <= 4) {
					on = true
				}
			} else if (x+y)%3 == 0 || (x*y)%5 == 0 {
				on = true
			}
			if on {
				b.WriteString("██")
			} else {
				b.WriteString("  ")
			}
		}
		b.WriteString("│\n")
	}
	b.WriteString("└" + strings.Repeat("─", size*2) + "┘\n")
	b.WriteString(fmt.Sprintf("\n[payload length %d bytes]\n", len(data)))
	if len(data) > 80 {
		b.WriteString(data[:77] + "...\n")
	} else {
		b.WriteString(data + "\n")
	}
	return b.String(), nil
}