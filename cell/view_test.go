package cell

import (
	"testing"
)

func TestViewClip(t *testing.T) {
	s := New(6, 3)
	v := s.View(2, 1, 3, 1)
	if v == nil {
		t.Fatal("View nil")
	}
	v.DrawString(0, 0, "abcdef", Style{})
	if got, want := s.String(), "\n  abc\n"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got := s.Get(1, 1).Grapheme; got != "" {
		t.Errorf("view content leaked left: %q at (1,1)", got)
	}
	if got := s.Get(5, 1).Grapheme; got != "" {
		t.Errorf("view content leaked right: %q at (5,1)", got)
	}
}

func TestViewWideCharAtRightEdge(t *testing.T) {
	s := New(4, 1)
	v := s.View(0, 0, 3, 1)
	v.DrawString(0, 0, "你好", Style{})
	if got, want := s.String(), screenWant("你"); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if c := s.Get(2, 0); c.Grapheme != "" {
		t.Errorf("wide char drew past view at (2,0): %+v", c)
	}
}

func TestViewWideSpanningEdgeLeft(t *testing.T) {
	s := New(2, 1)
	s.DrawString(0, 0, "你", Style{})
	v := s.View(1, 0, 1, 1)
	v.Set(0, 0, Cell{Grapheme: "q", Width: 1})
	if c := s.Get(0, 0); c != (Cell{}) {
		t.Errorf("orphan left half outside the view wasn't cleared: %+v", c)
	}
	if c := s.Get(1, 0); c.Grapheme != "q" {
		t.Errorf("Get(1,0) = %+v", c)
	}
}

func TestViewWideStraddlingLeftEdgeNotDrawn(t *testing.T) {
	s := New(4, 1)
	v := s.View(1, 0, 3, 1)
	v.DrawString(-1, 0, "你a", Style{})
	if got := s.String(); got != "" {
		t.Errorf("String() = %q, want blank", got)
	}
}

func TestNestedViews(t *testing.T) {
	s := New(8, 4)
	outer := s.View(1, 1, 5, 2)
	inner := outer.View(2, 0, 2, 1)
	if inner == nil {
		t.Fatal("inner view nil")
	}
	inner.DrawString(0, 0, "ab", Style{Bold: true})
	if got, want := s.String(), "\n   ab\n\n"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if c := s.Get(3, 1); !c.Style.Bold {
		t.Errorf("style lost through nested views: %+v", c)
	}
}

func TestViewOutOfRangeClamped(t *testing.T) {
	s := New(3, 2)
	if v := s.View(2, 1, 5, 5); v == nil || v.W != 1 || v.H != 1 {
		t.Errorf("View past viewport = %+v", v)
	}
	if s.View(3, 0, 1, 1) != nil {
		t.Error("View entirely off screen must be nil")
	}
}

func TestViewClearClearsOnlyRegion(t *testing.T) {
	s := New(4, 2)
	s.DrawString(0, 0, "abcd", Style{})
	s.DrawString(0, 1, "efgh", Style{})
	s.View(1, 0, 2, 1).Clear()
	if got, want := s.String(), screenWant("a  d", "efgh"); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestViewResizeIsNoOp(t *testing.T) {
	s := New(3, 2)
	v := s.View(0, 0, 2, 1)
	v.Resize(10, 10)
	if s.W != 3 || s.H != 2 || v.W != 2 {
		t.Errorf("Resize leaked into view/base: base=%dx%d view=%dx%d", s.W, s.H, v.W, v.H)
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"ascii", "ab", 2},
		{"cjk", "你好", 4},
		{"emoji", "😀", 2},
		{"zwj family", "👨‍👩‍👧", 2},
		{"combining", "é", 1},
		{"empty", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringWidth(tt.in); got != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
