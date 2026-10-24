package main

import (
	"fmt"
	"strings"

	"github.com/Supp140106/loomui/cell"
)

func main() {
	s := cell.New(24, 3)
	blue := cell.Style{Fg: cell.Color{R: 100, G: 150, B: 255}, Bold: true}
	s.DrawString(0, 0, "Hello, Loom!", blue)
	s.DrawString(0, 1, "你好世界 🌍", cell.Style{Faint: true})
	s.DrawString(0, 2, "👨‍👩‍👧 ✌️", cell.Style{Reverse: true})

	for line := range strings.SplitSeq(s.String(), "\n") {
		fmt.Println(line)
	}
}
