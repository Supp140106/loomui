package tuitest

import (
	"strings"

	"github.com/Supp140106/loomui/cell"
)

// Render renders s to a plain string, one line per row: grapheme clusters as
// themselves, blank cells as spaces, continuation cells of wide characters
// skipped (their columns are covered by the left half). Nothing is trimmed,
// so golden files stay stable.
func Render(s *cell.Screen) string {
	if s == nil {
		return ""
	}
	rows := make([]string, s.H)
	for y := 0; y < s.H; y++ {
		var row strings.Builder
		row.Grow(s.W)
		for x := 0; x < s.W; x++ {
			c := s.Get(x, y)
			if c.Continuation() {
				continue
			}
			if c.Grapheme == "" {
				row.WriteByte(' ')
			} else {
				row.WriteString(c.Grapheme)
			}
		}
		rows[y] = row.String()
	}
	return strings.Join(rows, "\n")
}

// RenderANSI renders s to a string with ANSI SGR styling applied at style
// boundaries. Style state resets at the end of every row.
func RenderANSI(s *cell.Screen) string {
	if s == nil {
		return ""
	}
	rows := make([]string, s.H)
	for y := 0; y < s.H; y++ {
		var row strings.Builder
		row.Grow(s.W)
		cur := cell.Style{}
		styled := false
		for x := 0; x < s.W; x++ {
			c := s.Get(x, y)
			if c.Continuation() {
				continue
			}
			if c.Style != cur {
				if c.Style == (cell.Style{}) {
					row.WriteString("\x1b[0m")
				} else {
					row.WriteString(c.Style.SGR())
				}
				cur = c.Style
				styled = true
			}
			if c.Grapheme == "" {
				row.WriteByte(' ')
			} else {
				row.WriteString(c.Grapheme)
			}
		}
		if styled && cur != (cell.Style{}) {
			row.WriteString("\x1b[0m")
		}
		rows[y] = row.String()
	}
	return strings.Join(rows, "\n")
}
