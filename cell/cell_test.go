package cell

import (
	"strings"
	"testing"
)

func joinRows(rows []string) string { return strings.Join(rows, "\n") }

func screenWant(rows ...string) string { return joinRows(rows) }

func TestNewClearAndDefaults(t *testing.T) {
	s := New(3, 2)
	if got, want := s.String(), "\n"; got != want {
		t.Errorf("blank screen = %q, want %q", got, want)
	}
	s.DrawString(0, 0, "abc", Style{})
	s.Clear()
	if got, want := s.String(), "\n"; got != want {
		t.Errorf("after Clear = %q, want %q", got, want)
	}
}

func TestNewNegativeClampsToZero(t *testing.T) {
	s := New(-3, -1)
	if s.String() != "" {
		t.Errorf("String() = %q, want empty", s.String())
	}
	s.DrawString(0, 0, "ab", Style{})
}

func TestGetOutOfBounds(t *testing.T) {
	s := New(2, 2)
	for _, p := range [][2]int{{5, 5}, {-1, 0}, {0, -1}, {2, 0}, {0, 2}} {
		if got := s.Get(p[0], p[1]); got != (Cell{}) {
			t.Errorf("Get(%d,%d) = %+v, want zero Cell", p[0], p[1], got)
		}
	}
}

func TestSetOutOfBoundsIsIgnored(t *testing.T) {
	s := New(2, 1)
	s.Set(-1, 0, Cell{Grapheme: "a", Width: 1})
	s.Set(2, 0, Cell{Grapheme: "b", Width: 1})
	s.Set(0, 1, Cell{Grapheme: "c", Width: 1})
	if got := s.String(); got != "" {
		t.Errorf("String() = %q, want empty", got)
	}
}

func TestSetWideClaimsContinuation(t *testing.T) {
	s := New(3, 1)
	s.Set(0, 0, Cell{Grapheme: "你", Width: 2})
	if c := s.Get(0, 0); c.Grapheme != "你" || c.Width != 2 {
		t.Errorf("Get(0,0) = %+v, want 你 width 2", c)
	}
	if c := s.Get(1, 0); !c.Continuation() {
		t.Errorf("Get(1,0) = %+v, want continuation", c)
	}
	s.Set(2, 0, Cell{Grapheme: "好", Width: 2})
	if c := s.Get(2, 0); c.Grapheme != "" {
		t.Errorf("wide Set with only one cell left was applied: %+v", c)
	}
}

func TestSetClobbersWideCharWithoutStrandingHalf(t *testing.T) {
	s := New(4, 1)
	s.DrawString(1, 0, "你", Style{})
	s.DrawString(0, 0, "你", Style{})
	if left, right := s.Get(0, 0), s.Get(1, 0); left.Width != 2 || !right.Continuation() {
		t.Errorf("cells 0/1 = %+v/%+v, want wide pair at 0", left, right)
	}
	if c := s.Get(2, 0); c != (Cell{}) {
		t.Errorf("Get(2,0) = %+v, want cleared right half of old wide char", c)
	}
}

func TestResizePreservesOverlap(t *testing.T) {
	s := New(4, 2)
	s.DrawString(0, 0, "abcd", Style{})
	s.DrawString(0, 1, "efgh", Style{})
	s.Resize(2, 3)
	if got, want := s.String(), screenWant("ab", "ef", ""); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	s.Resize(1, -1)
	if s.W != 1 || s.H != 0 || len(s.Cells) != 0 {
		t.Errorf("negative Resize ignored: W=%d H=%d len=%d", s.W, s.H, len(s.Cells))
	}
}

func TestDrawString(t *testing.T) {
	st := Style{Bold: true}

	tests := []struct {
		name     string
		w, h     int
		x, y     int
		str      string
		wantRows []string
	}{
		{"ascii", 4, 1, 0, 0, "ab", []string{"ab"}},
		{"fill", 2, 1, 0, 0, "abcdef", []string{"ab"}},
		{"cjk two cells each", 5, 1, 0, 0, "你好", []string{"你好"}},
		{"emoji", 3, 1, 0, 0, "😀x", []string{"😀x"}},
		{"zwj family emoji is one cluster", 3, 1, 0, 0, "👨‍👩‍👧x", []string{"👨‍👩‍👧x"}},
		{"keycap sequence", 3, 1, 0, 0, "5️⃣x", []string{"5️⃣x"}},
		{"combining mark joins base", 2, 1, 0, 0, "é", []string{"é"}},
		{"wide fits exactly at right edge", 2, 1, 0, 0, "你", []string{"你"}},
		{"wide with one cell left draws nothing", 3, 1, 2, 0, "你", nil},
		{"wide split at right edge stops drawing", 3, 1, 2, 0, "你好", nil},
		{"stop drawing after clipped wide char", 4, 1, 1, 0, "你a", []string{" 你a"}},
		{"negative x skips leading cells", 3, 1, -1, 0, "ab", []string{"b"}},
		{"wide spanning screen edge stops drawing", 3, 1, -1, 0, "你a", nil},
		{"wide fully off-screen left skipped", 3, 1, -2, 0, "你a", []string{"a"}},
		{"out of bounds entirely", 2, 1, 5, 0, "ab", nil},
		{"y beyond height ignored", 2, 2, 0, 2, "ab", []string{"", ""}},
		{"draw overwrites", 4, 1, 0, 0, "xyz", []string{"xyz"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.w, tt.h)
			s.DrawString(tt.x, tt.y, tt.str, st)
			want := ""
			if tt.wantRows != nil {
				want = screenWant(tt.wantRows...)
			}
			if got := s.String(); got != want {
				t.Errorf("DrawString(%d,%d,%q):\ngot %q\nwant %q", tt.x, tt.y, tt.str, got, want)
			}
		})
	}
}

func TestDrawStringCellDetails(t *testing.T) {
	flags := Style{Italic: true, Underline: true}

	t.Run("wide continuation has zero width", func(t *testing.T) {
		s := New(4, 1)
		s.DrawString(0, 0, "你a", flags)
		if c := s.Get(0, 0); c.Grapheme != "你" || c.Width != 2 || c.Style != flags {
			t.Errorf("Get(0,0) = %+v, want 你 width 2 styled %+v", c, flags)
		}
		if c := s.Get(1, 0); !c.Continuation() {
			t.Errorf("Get(1,0) = %+v, want blank continuation", c)
		}
		if c := s.Get(2, 0); c.Grapheme != "a" || c.Width != 1 || c.Style != flags {
			t.Errorf("Get(2,0) = %+v, want 'a' width 1 styled %+v", c, flags)
		}
	})

	t.Run("combining mark is one cell", func(t *testing.T) {
		s := New(2, 1)
		s.DrawString(0, 0, "é", Style{})
		if c := s.Get(0, 0); c.Grapheme != "é" || c.Width != 1 {
			t.Errorf("Get(0,0) = %+v, want decomposed é cluster width 1", c)
		}
		if c := s.Get(1, 0); c.Grapheme != "" {
			t.Errorf("Get(1,0) = %+v, want untouched", c)
		}
	})

	t.Run("overwriting the left half clears its continuation", func(t *testing.T) {
		s := New(3, 1)
		s.DrawString(0, 0, "你", Style{})
		s.DrawString(0, 0, "a", Style{})
		if c := s.Get(1, 0); c != (Cell{}) {
			t.Errorf("Get(1,0) = %+v, want cleared", c)
		}
	})

	t.Run("overwriting a continuation clears the left half", func(t *testing.T) {
		s := New(3, 1)
		s.DrawString(0, 0, "你x", Style{})
		s.DrawString(1, 0, "b", Style{})
		if c := s.Get(0, 0); c.Grapheme != "" {
			t.Errorf("Get(0,0) = %+v, want cleared left half", c)
		}
	})
}
