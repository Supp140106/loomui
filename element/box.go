package element

import (
	"github.com/Supp140106/loomui/cell"
	"github.com/Supp140106/loomui/layout"
)

// HBox arranges children left to right; leftover width is shared by their
// FlexGrowX factors.
type HBox struct {
	children []Element
	gX, gY   uint8
}

func NewHBox(children ...Element) *HBox { return &HBox{children: compact(children)} }

func (b *HBox) Requirement() layout.Requirement {
	return boxRequirement(b.children, b.gX, b.gY, true)
}

func (b *HBox) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	reqs := childRequirements(b.children)
	sizes := layout.HorizontalSizes(r.W, reqs)
	x := r.X
	for i, ch := range b.children {
		if v := s.View(x, r.Y, sizes[i], r.H); v != nil {
			ch.Render(v, layout.Rect{X: 0, Y: 0, W: sizes[i], H: r.H})
		}
		x += sizes[i]
	}
}

func (b *HBox) Flex() *HBox { b.gX, b.gY = 1, 1; return b }

// VBox arranges children top to bottom; leftover height is shared by their
// FlexGrowY factors.
type VBox struct {
	children []Element
	gX, gY   uint8
}

func NewVBox(children ...Element) *VBox { return &VBox{children: compact(children)} }

func (b *VBox) Requirement() layout.Requirement {
	return boxRequirement(b.children, b.gX, b.gY, false)
}

func (b *VBox) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	reqs := childRequirements(b.children)
	sizes := layout.VerticalSizes(r.H, reqs)
	y := r.Y
	for i, ch := range b.children {
		if v := s.View(r.X, y, r.W, sizes[i]); v != nil {
			ch.Render(v, layout.Rect{X: 0, Y: 0, W: r.W, H: sizes[i]})
		}
		y += sizes[i]
	}
}

func (b *VBox) Flex() *VBox { b.gX, b.gY = 1, 1; return b }

// Filler grabs any leftover space and paints nothing.
type Filler struct{}

func NewFiller() *Filler { return &Filler{} }

func (f *Filler) Requirement() layout.Requirement {
	return layout.Requirement{FlexGrowX: 1, FlexGrowY: 1}
}

func (f *Filler) Render(*cell.Screen, layout.Rect) {}

func compact(children []Element) []Element {
	out := make([]Element, 0, len(children))
	for _, c := range children {
		if c != nil {
			out = append(out, c)
		}
	}
	return out
}

// boxRequirement combines children requirements: along the axis minima
// (respectively cross-axis maxima), grow factors sum along the box axis and
// max out across; shrink factors do the same. The box's own Flex adds at
// least grow gX/gY.
func boxRequirement(children []Element, gX, gY uint8, horizontal bool) layout.Requirement {
	req := layout.Requirement{}
	sumAxisGrow, sumAxisShrink := 0, 0
	for _, ch := range children {
		r := ch.Requirement()
		r.MinW, r.MinH = max(r.MinW, 0), max(r.MinH, 0)
		if horizontal {
			req.MinW += r.MinW
			req.MinH = max(req.MinH, r.MinH)
			sumAxisGrow += int(r.FlexGrowX)
			sumAxisShrink += int(r.FlexShrinkX)
			req.FlexGrowY = maxU8(req.FlexGrowY, r.FlexGrowY)
			req.FlexShrinkY = maxU8(req.FlexShrinkY, r.FlexShrinkY)
		} else {
			req.MinH += r.MinH
			req.MinW = max(req.MinW, r.MinW)
			sumAxisGrow += int(r.FlexGrowY)
			sumAxisShrink += int(r.FlexShrinkY)
			req.FlexGrowX = maxU8(req.FlexGrowX, r.FlexGrowX)
			req.FlexShrinkX = maxU8(req.FlexShrinkX, r.FlexShrinkX)
		}
	}
	if horizontal {
		req.FlexGrowX = maxU8(clampU8(sumAxisGrow), gX)
		req.FlexGrowY = maxU8(req.FlexGrowY, gY)
	} else {
		req.FlexGrowY = maxU8(clampU8(sumAxisGrow), gY)
		req.FlexGrowX = maxU8(req.FlexGrowX, gX)
	}
	req.FlexShrinkX = maxU8(req.FlexShrinkX, clampU8(sumAxisShrink))
	req.FlexShrinkY = maxU8(req.FlexShrinkY, clampU8(sumAxisShrink))
	return req
}

func childRequirements(children []Element) []layout.Requirement {
	out := make([]layout.Requirement, len(children))
	for i, ch := range children {
		r := ch.Requirement()
		r.MinW, r.MinH = max(r.MinW, 0), max(r.MinH, 0)
		out[i] = r
	}
	return out
}

func maxU8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}
