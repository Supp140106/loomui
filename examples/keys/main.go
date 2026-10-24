package main

import (
	"fmt"
	"os"

	"github.com/Supp140106/loomui/cell"
	"github.com/Supp140106/loomui/term"
)

func main() {
	t, err := term.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer t.Close()

	obold := cell.Style{Bold: true}.SGR()
	reset := "\x1b[0m"
	t.Write([]byte("\x1b[2J\x1b[1;1H" + obold + "Key monitor — every terminal event is printed. Ctrl+C quits." + reset + "\r\n\r\n"))

	for ev := range t.Events() {
		t.Write([]byte(describe(ev) + "\r\n"))
	}
	t.Write([]byte("\r\nbye\r\n"))
}

func describe(ev term.Event) string {
	switch e := ev.(type) {
	case term.KeyEvent:
		name := e.Key.String()
		if e.Key == term.KeyNil {
			name = fmt.Sprintf("%q", e.Rune)
		}
		return fmt.Sprintf("Key    %s%s", prefix(e.Mod), name)
	case term.MouseEvent:
		action := "release"
		if e.Pressed {
			action = "press"
		}
		if e.Motion {
			action = "motion"
		}
		return fmt.Sprintf("Mouse  %s%s %s at %d,%d", prefix(e.Mods), e.Button, action, e.X, e.Y)
	case term.PasteEvent:
		text := e.Text
		if len(text) > 40 {
			text = text[:37] + "..."
		}
		return fmt.Sprintf("Paste  %d bytes %q", len(e.Text), text)
	case term.FocusEvent:
		if e.Gained {
			return "Focus  gained"
		}
		return "Focus  lost"
	case term.ResizeEvent:
		return fmt.Sprintf("Resize %dx%d", e.W, e.H)
	default:
		return fmt.Sprintf("Unknown event %#v", ev)
	}
}

func prefix(m term.Mod) string {
	if m == term.ModNone {
		return ""
	}
	return m.String() + " "
}
