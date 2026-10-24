package cell

import "strconv"

const (
	syncBegin = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
	sgrReset  = "\x1b[0m"
)

// Diff computes the ANSI escape sequence that transforms prev into next on a
// terminal. Only runs of changed cells are written; cursor moves are skipped
// when the current position already matches, and SGR state is tracked so an
// unchanged style is never re-emitted (SGR state is additive, so switching
// styles emits a reset first).
//
// prev == nil means the first frame and triggers a full redraw, as does a
// change in screen dimensions. The output is wrapped in synchronized update
// (DECSET 2026) markers. An unchanged frame produces no output.
func Diff(prev, next *Screen) []byte {
	if next == nil || next.W <= 0 || next.H <= 0 {
		return nil
	}
	full := prev == nil || prev.W != next.W || prev.H != next.H
	var out []byte

	same := func(x, y int) bool {
		if full {
			return false
		}
		p, i := prev.locate(x, y)
		n, j := next.locate(x, y)
		return p.Cells[i] == n.Cells[j]
	}

	var curStyle Style
	styleOK := false
	setStyle := func(st Style) {
		if styleOK && st == curStyle {
			return
		}
		if !styleOK && st == (Style{}) {
			curStyle, styleOK = st, true
			return
		}
		out = append(out, sgrReset...)
		if sg := st.SGR(); sg != "" {
			out = append(out, sg...)
		}
		curStyle, styleOK = st, true
	}

	curX, curY := -1, -1

	for y := 0; y < next.H; y++ {
		x := 0
		for x < next.W {
			if same(x, y) {
				x++
				continue
			}
			start := x
			for x < next.W && !same(x, y) {
				x++
			}
			if curX != start || curY != y {
				out = append(out, cup(y+1, start+1)...)
			}
			widePending := false
			w := 0
			for k := start; k < x; k++ {
				n2, i2 := next.locate(k, y)
				c := n2.Cells[i2]
				if widePending && c.Continuation() {
					widePending = false
					continue
				}
				setStyle(c.Style)
				adv := 1
				if c.Grapheme != "" {
					out = append(out, c.Grapheme...)
					adv = int(c.Width)
				} else {
					out = append(out, ' ')
				}
				w += adv
				widePending = c.Width >= 2
			}
			curX, curY = start+w, y
			if curX >= next.W {
				curX, curY = -1, -1
			}
		}
	}

	if len(out) == 0 {
		return nil
	}
	// SGR state persists across frames: end the frame on a known default so
	// the next frame may safely assume it.
	if styleOK && curStyle != (Style{}) {
		out = append(out, sgrReset...)
	}
	final := make([]byte, 0, len(out)+len(syncBegin)+len(syncEnd))
	final = append(final, syncBegin...)
	final = append(final, out...)
	final = append(final, syncEnd...)
	return final
}

func cup(row, col int) []byte {
	b := make([]byte, 0, 12)
	b = append(b, "\x1b["...)
	b = strconv.AppendInt(b, int64(row), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64(col), 10)
	b = append(b, 'H')
	return b
}
