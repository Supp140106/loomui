package element

import (
	"github.com/Supp140106/loomui/cell"
	"github.com/Supp140106/loomui/layout"
)

// Element is a renderable, stateless screen component: an immutable tree
// node. Requirement describes how much space it wants; Render paints into a
// region of the screen and must never paint outside that region.
type Element interface {
	Requirement() layout.Requirement
	Render(*cell.Screen, layout.Rect)
}

// Render lays el out on a fresh w×h screen.
func Render(el Element, w, h int) *cell.Screen {
	scr := cell.New(w, h)
	if w <= 0 || h <= 0 || scr == nil || el == nil {
		return scr
	}
	el.Render(scr, layout.Rect{X: 0, Y: 0, W: w, H: h})
	return scr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampU8(v int) uint8 {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	}
	return uint8(v)
}

func clampI32(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
