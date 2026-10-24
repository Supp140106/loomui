package tuitest

import (
	"testing"

	"github.com/Supp140106/loomui/cell"
)

func exampleScreen() *cell.Screen {
	s := cell.New(20, 4)
	s.DrawString(0, 0, "Hello, Loom!", cell.Style{Bold: true, Fg: cell.Color{R: 120, G: 170, B: 255}})
	s.DrawString(0, 1, "你好世界", cell.Style{Faint: true})
	s.DrawString(0, 2, "👨‍👩‍👧 ✌️", cell.Style{})
	s.DrawString(0, 3, "plain tail", cell.Style{})
	return s
}

func TestRender(t *testing.T) {
	s := cell.New(5, 1)
	s.DrawString(0, 0, "你a", cell.Style{})
	if got, want := Render(s), "你a  "; got != want {
		t.Errorf("Render = %q, want %q", got, want)
	}
}

func TestRenderANSI(t *testing.T) {
	s := cell.New(2, 1)
	s.DrawString(0, 0, "a", cell.Style{Bold: true})
	if got, want := RenderANSI(s), "\x1b[1ma\x1b[0m "; got != want {
		t.Errorf("RenderANSI = %q, want %q", got, want)
	}
}

func TestAssertGolden(t *testing.T) {
	AssertGolden(t, "render", Render(exampleScreen()))
	AssertGolden(t, "render-ansi", RenderANSI(exampleScreen()))
}
