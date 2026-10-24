package cell

import (
	"strings"
	"testing"
)

func fillScreen(w, h int, base rune, styleEvery int) *Screen {
	s := New(w, h)
	var row strings.Builder
	for y := 0; y < h; y++ {
		row.Reset()
		row.Grow(w)
		for x := 0; x < w; x++ {
			row.WriteRune(base + rune((x+y*7)%26))
		}
		st := Style{Fg: Color{R: uint8(y * 5 % 255), G: 128, B: 200}, Bold: styleEvery > 0 && y%styleEvery == 0}
		s.DrawString(0, y, row.String(), st)
	}
	return s
}

func BenchmarkDiff200x50AllChanged(b *testing.B) {
	prev := fillScreen(200, 50, 'A', 2)
	next := fillScreen(200, 50, 'M', 2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(prev, next)
	}
}

func BenchmarkDiff200x50OneCellChanged(b *testing.B) {
	prev := fillScreen(200, 50, 'A', 2)
	next := fillScreen(200, 50, 'A', 2)
	next.Set(100, 25, Cell{Grapheme: "✗", Width: 1, Style: Style{Reverse: true}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(prev, next)
	}
}
