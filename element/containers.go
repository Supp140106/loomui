package element

import (
	"github.com/Supp140106/loomui/cell"
	"github.com/Supp140106/loomui/layout"
)

// Separator draws a centered line: horizontal when its region is wider than
// tall, vertical otherwise.
type Separator struct {
	style  cell.Style
	gX, gY uint8
}

func NewSeparator() *Separator { return &Separator{} }

func (sp *Separator) Requirement() layout.Requirement {
	return layout.Requirement{
		MinW: 1, MinH: 1,
		FlexGrowX: sp.gX, FlexGrowY: sp.gY,
		FlexShrinkX: 1, FlexShrinkY: 1,
	}
}

func (sp *Separator) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	if r.W >= r.H {
		y := r.Y + r.H/2
		for x := r.X; x < r.X+r.W; x++ {
			s.Set(x, y, cell.Cell{Grapheme: "─", Width: 1, Style: sp.style})
		}
		return
	}
	x := r.X + r.W/2
	for y := r.Y; y < r.Y+r.H; y++ {
		s.Set(x, y, cell.Cell{Grapheme: "│", Width: 1, Style: sp.style})
	}
}

func (sp *Separator) Bold() *Separator              { sp.style.Bold = true; return sp }
func (sp *Separator) Dim() *Separator               { sp.style.Faint = true; return sp }
func (sp *Separator) Color(c cell.Color) *Separator { sp.style.Fg = c; return sp }
func (sp *Separator) Flex() *Separator              { sp.gX, sp.gY = 1, 1; return sp }

type BorderStyle uint8

const (
	BorderNormal BorderStyle = iota
	BorderRounded
	BorderThick
	BorderDouble
)

type borderSet struct {
	tl, tr, bl, br, h, v rune
}

var borderSets = [4]borderSet{
	{'┌', '┐', '└', '┘', '─', '│'},
	{'╭', '╮', '╰', '╯', '─', '│'},
	{'┏', '┓', '┗', '┛', '━', '┃'},
	{'╔', '╗', '╚', '╝', '═', '║'},
}

// Border draws a frame of the given style around an inner element, with an
// optional title overlaid on the top edge.
type Border struct {
	inner  Element
	kind   BorderStyle
	title  string
	style  cell.Style
	gX, gY uint8
}

func NewBorder(inner Element) *Border { return &Border{inner: inner} }

func (b *Border) Requirement() layout.Requirement {
	req := layout.Requirement{}
	if b.inner != nil {
		req = b.inner.Requirement()
	}
	req.MinW, req.MinH = max(req.MinW, 0)+2, max(req.MinH, 0)+2
	req.FlexGrowX = maxU8(req.FlexGrowX, b.gX)
	req.FlexGrowY = maxU8(req.FlexGrowY, b.gY)
	return req
}

func (b *Border) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	set := borderSets[b.kind]
	y0, y1 := r.Y, r.Y+r.H-1
	x0, x1 := r.X, r.X+r.W-1
	s.Set(x0, y0, cell.Cell{Grapheme: string(set.tl), Width: 1, Style: b.style})
	s.Set(x1, y0, cell.Cell{Grapheme: string(set.tr), Width: 1, Style: b.style})
	if r.H >= 2 {
		s.Set(x0, y1, cell.Cell{Grapheme: string(set.bl), Width: 1, Style: b.style})
		s.Set(x1, y1, cell.Cell{Grapheme: string(set.br), Width: 1, Style: b.style})
	}
	for x := r.X + 1; x < r.X+r.W-1; x++ {
		s.Set(x, y0, cell.Cell{Grapheme: string(set.h), Width: 1, Style: b.style})
		if r.H >= 2 {
			s.Set(x, y1, cell.Cell{Grapheme: string(set.h), Width: 1, Style: b.style})
		}
	}
	if r.H >= 3 {
		for y := r.Y + 1; y < r.Y+r.H-1; y++ {
			s.Set(x0, y, cell.Cell{Grapheme: string(set.v), Width: 1, Style: b.style})
			s.Set(x1, y, cell.Cell{Grapheme: string(set.v), Width: 1, Style: b.style})
		}
	}
	if title := b.title; title != "" && cell.StringWidth(title)+2 <= r.W-2 {
		s.DrawString(r.X+2, y0, " "+title+" ", b.style)
	}
	if b.inner != nil {
		if v := s.View(r.X+1, r.Y+1, r.W-2, r.H-2); v != nil {
			b.inner.Render(v, layout.Rect{X: 0, Y: 0, W: r.W - 2, H: r.H - 2})
		}
	}
}

func (b *Border) Title(title string) *Border  { b.title = title; return b }
func (b *Border) Bold() *Border               { b.style.Bold = true; return b }
func (b *Border) Dim() *Border                { b.style.Faint = true; return b }
func (b *Border) Color(c cell.Color) *Border  { b.style.Fg = c; return b }
func (b *Border) Style(k BorderStyle) *Border { b.kind = k; return b }
func (b *Border) Flex() *Border               { b.gX, b.gY = 1, 1; return b }

// Center sizes its inner element to its own minimum and places it in the
// middle of the region.
type Center struct {
	inner Element
}

func NewCenter(inner Element) *Center { return &Center{inner: inner} }

func (c *Center) Requirement() layout.Requirement {
	if c.inner == nil {
		return layout.Requirement{}
	}
	return c.inner.Requirement()
}

func (c *Center) Render(s *cell.Screen, r layout.Rect) {
	if c.inner == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	want := c.inner.Requirement()
	w := clampI32(want.MinW, 0, r.W)
	h := clampI32(want.MinH, 0, r.H)
	if w == 0 || h == 0 {
		return
	}
	v := s.View(r.X+(r.W-w)/2, r.Y+(r.H-h)/2, w, h)
	if v != nil {
		c.inner.Render(v, layout.Rect{X: 0, Y: 0, W: w, H: h})
	}
}

// Padding surrounds its inner element with pad empty cells on every side.
type Padding struct {
	inner Element
	pad   int
}

func NewPadding(inner Element, pad int) *Padding { return &Padding{inner: inner, pad: max(pad, 0)} }

func (p *Padding) Requirement() layout.Requirement {
	req := layout.Requirement{}
	if p.inner != nil {
		req = p.inner.Requirement()
	}
	req.MinW += 2 * p.pad
	req.MinH += 2 * p.pad
	return req
}

func (p *Padding) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	w, h := r.W-2*p.pad, r.H-2*p.pad
	if w <= 0 || h <= 0 {
		return
	}
	v := s.View(r.X+p.pad, r.Y+p.pad, w, h)
	if v != nil && p.inner != nil {
		p.inner.Render(v, layout.Rect{X: 0, Y: 0, W: w, H: h})
	}
}

type SizeMode uint8

const (
	SizeNone SizeMode = iota
	SizeFixed
	SizeMin
	SizeMax
)

type sizeSpec struct {
	mode SizeMode
	size int
}

// Size constrains its inner element's width/height to be fixed, at least, or
// at most a value.
type Size struct {
	inner Element
	wSpec sizeSpec
	hSpec sizeSpec
}

func NewSize(inner Element) *Size { return &Size{inner: inner} }

func (sz *Size) Requirement() layout.Requirement {
	req := layout.Requirement{}
	if sz.inner != nil {
		req = sz.inner.Requirement()
	}
	req = applySpec(req, sz.wSpec, sz.hSpec)
	return req
}

func applySpec(req layout.Requirement, ws, hs sizeSpec) layout.Requirement {
	switch ws.mode {
	case SizeFixed:
		req.MinW = max(ws.size, 0)
		req.FlexGrowX, req.FlexShrinkX = 0, 0
	case SizeMin:
		req.MinW = max(req.MinW, ws.size)
	case SizeMax:
		req.MinW = max(min(req.MinW, ws.size), 0)
	}
	switch hs.mode {
	case SizeFixed:
		req.MinH = max(hs.size, 0)
		req.FlexGrowY, req.FlexShrinkY = 0, 0
	case SizeMin:
		req.MinH = max(req.MinH, hs.size)
	case SizeMax:
		req.MinH = max(min(req.MinH, hs.size), 0)
	}
	return req
}

func (sz *Size) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	w, h := r.W, r.H
	if sz.wSpec.mode == SizeFixed || sz.wSpec.mode == SizeMax {
		w = clampI32(sz.wSpec.size, 0, r.W)
	}
	if sz.hSpec.mode == SizeFixed || sz.hSpec.mode == SizeMax {
		h = clampI32(sz.hSpec.size, 0, r.H)
	}
	if w <= 0 || h <= 0 {
		return
	}
	v := s.View(r.X, r.Y, w, h)
	if v != nil && sz.inner != nil {
		sz.inner.Render(v, layout.Rect{X: 0, Y: 0, W: w, H: h})
	}
}

func (sz *Size) FixedWidth(v int) *Size  { sz.wSpec = sizeSpec{SizeFixed, v}; return sz }
func (sz *Size) FixedHeight(v int) *Size { sz.hSpec = sizeSpec{SizeFixed, v}; return sz }
func (sz *Size) MinWidth(v int) *Size    { sz.wSpec = sizeSpec{SizeMin, v}; return sz }
func (sz *Size) MinHeight(v int) *Size   { sz.hSpec = sizeSpec{SizeMin, v}; return sz }
func (sz *Size) MaxWidth(v int) *Size    { sz.wSpec = sizeSpec{SizeMax, v}; return sz }
func (sz *Size) MaxHeight(v int) *Size   { sz.hSpec = sizeSpec{SizeMax, v}; return sz }

// ZStack overlays layers on top of each other, all filling the region.
type ZStack struct {
	layers []Element
}

func NewZStack(layers ...Element) *ZStack { return &ZStack{layers: compact(layers)} }

func (z *ZStack) Requirement() layout.Requirement {
	req := layout.Requirement{}
	for _, l := range z.layers {
		r := l.Requirement()
		r.MinW, r.MinH = max(r.MinW, 0), max(r.MinH, 0)
		req.MinW = max(req.MinW, r.MinW)
		req.MinH = max(req.MinH, r.MinH)
		req.FlexGrowX = maxU8(req.FlexGrowX, r.FlexGrowX)
		req.FlexGrowY = maxU8(req.FlexGrowY, r.FlexGrowY)
		req.FlexShrinkX = maxU8(req.FlexShrinkX, r.FlexShrinkX)
		req.FlexShrinkY = maxU8(req.FlexShrinkY, r.FlexShrinkY)
	}
	return req
}

func (z *ZStack) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	for _, l := range z.layers {
		v := s.View(r.X, r.Y, r.W, r.H)
		if v != nil {
			l.Render(v, layout.Rect{X: 0, Y: 0, W: r.W, H: r.H})
		}
	}
}
