package cell

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// Cell is the smallest unit of screen state: one grapheme cluster occupying
// Width terminal columns, with its presentation Style.
//
// A wide character (CJK, emoji) always occupies two cells: the left cell holds
// the grapheme with Width >= 2, and the right cell is a blank continuation
// belonging to it. A Screen never leaves half of a wide character behind.
type Cell struct {
	Grapheme string
	Width    uint8
	Style    Style

	cont bool
}

// Continuation reports whether c is the blank right half of a wide character.
func (c Cell) Continuation() bool {
	return c.cont
}

// Screen is a rectangular grid of cells. A Screen obtained from View shares
// its cells with a parent screen and sees a clipped, offset region of it.
type Screen struct {
	W, H   int
	Cells  []Cell
	origin *Screen
	ox, oy int
}

// New returns a blank Screen of size w×h. Negative dimensions are clamped
// to zero.
func New(w, h int) *Screen {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Screen{W: w, H: h, Cells: make([]Cell, w*h)}
}

// View returns a sub-screen sharing s's cells limited to the region
// (x, y, w, h) in s's coordinates; coordinates outside s are clipped. It
// returns nil when the region is empty. Views may be nested.
func (s *Screen) View(x, y, w, h int) *Screen {
	if w <= 0 || h <= 0 {
		return nil
	}
	if s.origin != nil {
		x += s.ox
		y += s.oy
		s = s.origin
	}
	if x < 0 {
		w += x
		x = 0
	}
	if x+w > s.W {
		w = s.W - x
	}
	if y < 0 {
		h += y
		y = 0
	}
	if y+h > s.H {
		h = s.H - y
	}
	if w <= 0 || h <= 0 {
		return nil
	}
	return &Screen{W: w, H: h, origin: s, ox: x, oy: y}
}

// Contains reports whether (x, y) is inside the screen.
func (s *Screen) Contains(x, y int) bool {
	return x >= 0 && y >= 0 && x < s.W && y < s.H
}

// locate resolves view coordinates into the underlying root screen and an
// index into its Cells.
func (s *Screen) locate(x, y int) (*Screen, int) {
	for s.origin != nil {
		x += s.ox
		y += s.oy
		s = s.origin
	}
	return s, y*s.W + x
}

// Clear resets every cell to its blank state. On a View, only its region is
// cleared.
func (s *Screen) Clear() {
	if s.origin != nil {
		for y := 0; y < s.H; y++ {
			for x := 0; x < s.W; x++ {
				s.Set(x, y, Cell{})
			}
		}
		return
	}
	for i := range s.Cells {
		s.Cells[i] = Cell{}
	}
}

// Get returns the cell at (x, y), or the zero Cell if out of bounds.
func (s *Screen) Get(x, y int) Cell {
	if !s.Contains(x, y) {
		return Cell{}
	}
	root, i := s.locate(x, y)
	return root.Cells[i]
}

// unbind clears the neighbors of the cell being replaced at index i (column x)
// so that replacing a cell never leaves half of a wide character behind.
func (s *Screen) unbind(root *Screen, x, i int, old Cell) {
	if old.Width >= 2 {
		if x+1 < s.W {
			root.Cells[i+1] = Cell{}
		}
		return
	}
	if old.Continuation() && x > 0 && root.Cells[i-1].Width >= 2 {
		root.Cells[i-1] = Cell{}
	}
}

// Set places c at (x, y). Out-of-bounds writes are ignored.
//
// A cell with Width >= 2 also claims the cell to its right as its blank
// continuation, and replacing a cell that belongs to a wide character clears
// its partner, so half-displayed wide characters cannot be produced.
func (s *Screen) Set(x, y int, c Cell) {
	if !s.Contains(x, y) {
		return
	}
	if c.Width >= 2 && x+1 >= s.W {
		return
	}
	root, i := s.locate(x, y)
	rootCol := i % root.W
	s.unbind(root, rootCol, i, root.Cells[i])
	if c.Width >= 2 {
		if n := root.Cells[i+1]; n.Width >= 2 && x+2 < s.W {
			root.Cells[i+2] = Cell{}
		}
		root.Cells[i+1] = Cell{cont: true}
	}
	root.Cells[i] = c
}

// Resize changes the screen dimensions, preserving the overlapping cell
// contents. On a View it is a no-op.
func (s *Screen) Resize(w, h int) {
	if s.origin != nil {
		return
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	old, oldW, oldH := s.Cells, s.W, s.H
	s.W, s.H = w, h
	s.Cells = make([]Cell, w*h)
	for y := 0; y < h && y < oldH; y++ {
		copy(s.Cells[y*w:y*w+min(w, oldW)], old[y*oldW:y*oldW+min(w, oldW)])
	}
}

// DrawString writes str at (x, y) as a sequence of grapheme clusters, styled
// with st. Wide clusters (CJK, emoji, ZWJ emoji sequences) occupy two cells,
// with the second cell as an empty continuation. Combining marks join their
// base cluster and count as zero additional width.
//
// Out-of-bounds columns are clipped without panicking. A wide cluster is
// never split across the right edge: drawing stops rather than leaving half
// of it behind.
func (s *Screen) DrawString(x, y int, str string, st Style) {
	if y < 0 || y >= s.H {
		return
	}
	g := uniseg.NewGraphemes(str)
	cx := x
	for g.Next() {
		gr := g.Str()
		w := StringWidth(gr)
		if cx >= s.W {
			return
		}
		switch {
		case w <= 0:
			if s.Contains(cx, y) {
				s.Set(cx, y, Cell{Grapheme: gr, Width: 0, Style: st})
			}
		case w == 1:
			if cx < 0 {
				cx++
				continue
			}
			s.Set(cx, y, Cell{Grapheme: gr, Width: 1, Style: st})
			cx++
		case w == 2:
			switch {
			case cx+1 < 0:
				cx += 2
				continue
			case cx < 0 || cx+1 >= s.W:
				return
			default:
				s.Set(cx, y, Cell{Grapheme: gr, Width: 2, Style: st})
				cx += 2
			}
		}
	}
}

// String renders the screen as text: one line per row, empty cells as
// spaces, trailing spaces trimmed. It is intended for tests and debugging,
// not for final presentation (that is the term backend's job).
func (s *Screen) String() string {
	var b strings.Builder
	for y := 0; y < s.H; y++ {
		row := &strings.Builder{}
		row.Grow(s.W)
		for x := 0; x < s.W; x++ {
			c := s.Get(x, y)
			if c.Continuation() {
				continue
			}
			if c.Grapheme != "" {
				row.WriteString(c.Grapheme)
			} else {
				row.WriteByte(' ')
			}
		}
		b.WriteString(strings.TrimRight(row.String(), " "))
		if y < s.H-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// StringWidth reports the presentation width of s in terminal columns,
// counting grapheme clusters as units and clamping wide clusters to 2
// columns, matching DrawString's advancing semantics.
func StringWidth(s string) int {
	w := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		cw := runewidth.StringWidth(g.Str())
		if cw > 2 {
			cw = 2
		}
		w += cw
	}
	return w
}
