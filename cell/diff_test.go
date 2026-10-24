package cell

import (
	"testing"
)

func buildScreen(w, h int, fn func(s *Screen)) *Screen {
	s := New(w, h)
	fn(s)
	return s
}

func draw(s *Screen, x, y int, str string, st Style) *Screen {
	s.DrawString(x, y, str, st)
	return s
}

func TestDiff(t *testing.T) {
	const begin = "\x1b[?2026h"
	const end = "\x1b[?2026l"

	tests := []struct {
		name string
		prev func() *Screen
		next func() *Screen
		want string
	}{
		{
			name: "no changes yields no output",
			prev: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abcd", Style{}) },
			next: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abcd", Style{}) },
			want: "",
		},
		{
			name: "first frame is a full redraw",
			prev: func() *Screen { return nil },
			next: func() *Screen { return draw(buildScreen(2, 1, func(s *Screen) {}), 0, 0, "ab", Style{}) },
			want: begin + "\x1b[1;1Hab" + end,
		},
		{
			name: "resize triggers a full redraw",
			prev: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abcd", Style{}) },
			next: func() *Screen { return draw(buildScreen(2, 1, func(s *Screen) {}), 0, 0, "hi", Style{}) },
			want: begin + "\x1b[1;1Hhi" + end,
		},
		{
			name: "single changed cell",
			prev: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abcd", Style{}) },
			next: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abXd", Style{}) },
			want: begin + "\x1b[1;3HX" + end,
		},
		{
			name: "adjacent changed cells join into one run",
			prev: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abcd", Style{}) },
			next: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abXY", Style{}) },
			want: begin + "\x1b[1;3HXY" + end,
		},
		{
			name: "separated runs get separate cursor moves",
			prev: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abcd", Style{}) },
			next: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "aXcY", Style{}) },
			want: begin + "\x1b[1;2HX\x1b[1;4HY" + end,
		},
		{
			name: "changed cell on a later row moves the cursor down",
			prev: func() *Screen {
				return buildScreen(4, 2, func(s *Screen) {
					draw(s, 0, 0, "abcd", Style{})
					draw(s, 0, 1, "efgh", Style{})
				})
			},
			next: func() *Screen {
				return buildScreen(4, 2, func(s *Screen) {
					draw(s, 0, 0, "abcd", Style{})
					draw(s, 0, 1, "efXi", Style{})
				})
			},
			want: begin + "\x1b[2;3HXi" + end,
		},
		{
			name: "style-only change",
			prev: func() *Screen { return draw(buildScreen(2, 1, func(s *Screen) {}), 0, 0, "ab", Style{Bold: true}) },
			next: func() *Screen { return draw(buildScreen(2, 1, func(s *Screen) {}), 0, 0, "ab", Style{}) },
			want: begin + "\x1b[1;1Hab" + end,
		},
		{
			name: "sgr emitted once for same style across runs",
			prev: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "abcd", Style{Bold: true}) },
			next: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "XbYd", Style{Bold: true}) },
			want: begin + "\x1b[1;1H\x1b[0m\x1b[1mX\x1b[1;3HY\x1b[0m" + end,
		},
		{
			name: "sgr changes inside a run",
			prev: func() *Screen { return nil },
			next: func() *Screen {
				s := buildScreen(2, 1, func(s *Screen) {})
				draw(s, 0, 0, "a", Style{Fg: Color{R: 10, G: 0, B: 0}})
				draw(s, 1, 0, "b", Style{Bold: true})
				return s
			},
			want: begin + "\x1b[1;1H\x1b[0m\x1b[38;2;10;0;0ma\x1b[0m\x1b[1mb\x1b[0m" + end,
		},
		{
			name: "wide char continuation is skipped after its left half",
			prev: func() *Screen { return nil },
			next: func() *Screen { return draw(buildScreen(4, 1, func(s *Screen) {}), 0, 0, "你x", Style{}) },
			want: begin + "\x1b[1;1H你x " + end,
		},
		{
			name: "covered continuation cell",
			prev: func() *Screen { return draw(buildScreen(2, 1, func(s *Screen) {}), 0, 0, "你", Style{}) },
			next: func() *Screen {
				s := draw(buildScreen(2, 1, func(s *Screen) {}), 0, 0, "你", Style{})
				s.Cells[1].Style = Style{Bold: true}
				return s
			},
			want: begin + "\x1b[1;2H\x1b[0m\x1b[1m \x1b[0m" + end,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Diff(tt.prev(), tt.next())
			if string(got) != tt.want {
				t.Errorf("Diff output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiffDegenerateInputs(t *testing.T) {
	if got := Diff(nil, nil); got != nil {
		t.Errorf("Diff(nil, nil) = %q, want nil", got)
	}
	if got := Diff(nil, New(0, 3)); got != nil {
		t.Errorf("empty next = %q, want nil", got)
	}
	s := New(2, 1)
	if got := Diff(s, New(0, 0)); got != nil {
		t.Errorf("resize to empty = %q, want nil", got)
	}
}

func TestDiffNeverPanicsOnOddPrev(t *testing.T) {
	next := draw(buildScreen(3, 2, func(s *Screen) {}), 0, 0, "abc", Style{})
	for _, prev := range []*Screen{nil, New(0, 0), New(1, 1), New(100, 100)} {
		_ = Diff(prev, next)
	}
}
