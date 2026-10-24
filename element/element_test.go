package element

import (
	"testing"

	"github.com/Supp140106/loomui/cell"
	"github.com/Supp140106/loomui/layout"
)

func TestTextRequirement(t *testing.T) {
	tests := []struct {
		name string
		mk   func() *Text
		want layout.Requirement
	}{
		{"empty", func() *Text { return NewText("") }, layout.Requirement{MinH: 1, FlexShrinkX: 1, FlexShrinkY: 1}},
		{"ascii", func() *Text { return NewText("hello") }, layout.Requirement{MinW: 5, MinH: 1, FlexShrinkX: 1, FlexShrinkY: 1}},
		{"cjk", func() *Text { return NewText("你好") }, layout.Requirement{MinW: 4, MinH: 1, FlexShrinkX: 1, FlexShrinkY: 1}},
		{"width floor", func() *Text { return NewText("hi").Width(10) }, layout.Requirement{MinW: 10, MinH: 1, FlexShrinkX: 1, FlexShrinkY: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mk().Requirement(); got != tt.want {
				t.Errorf("Requirement = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxWidth int
		want     []string
	}{
		{"fit on one line", "hello world", 11, []string{"hello world"}},
		{"wraps on space", "hello world", 5, []string{"hello", "world"}},
		{"multiple breaks", "a b c d e", 3, []string{"a b", "c d", "e"}},
		{"hard split", "aaaaaa", 4, []string{"aaaa", "aa"}},
		{"no space long word", "abcdefgh", 3, []string{"abc", "def", "h"}},
		{"hard split cjk", "你好世界", 3, []string{"你", "好", "世", "界"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.content, tt.maxWidth)
			if len(got) != len(tt.want) {
				t.Fatalf("Wrap = %q, want %q", got, tt.want)
			}
		})
	}
	for _, tt := range tests {
		got := Wrap(tt.content, tt.maxWidth)
		for _, line := range got {
			if w := cell.StringWidth(line); w > tt.maxWidth {
				t.Fatalf("Wrap(%q,%d) produced overlong line %q", tt.content, tt.maxWidth, line)
			}
		}
	}
}

func TestWrapDetail(t *testing.T) {
	if got := Wrap("hello world", 5); !equalStr(got, []string{"hello", "world"}) {
		t.Errorf("Wrap = %q, want %q", got, []string{"hello", "world"})
	}
	if got := Wrap("line1\n\nline2", 20); !equalStr(got, []string{"line1", "", "line2"}) {
		t.Errorf("Wrap newline = %q, want [line1  line2]", got)
	}
	if got := Wrap("你好世界!", 5); !equalStr(got, []string{"你好", "世界!"}) {
		t.Errorf("Wrap cjk = %q", got)
	}
}

func equalStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBoxRequirements(t *testing.T) {
	h := NewHBox(NewText("ab"), NewText("cd"))
	if r := h.Requirement(); r.MinW != 4 || r.MinH != 1 {
		t.Errorf("HBox requirement = %+v", r)
	}
	v := NewVBox(NewText("ab"), NewText("cd"))
	if r := v.Requirement(); r.MinW != 2 || r.MinH != 2 {
		t.Errorf("VBox requirement = %+v", r)
	}
	stack := NewZStack(NewText("ab"), NewText("cdef"))
	if r := stack.Requirement(); r.MinW != 4 || r.MinH != 1 {
		t.Errorf("ZStack requirement = %+v", r)
	}
	flexed := NewHBox(NewFiller(), NewText("x"))
	if r := flexed.Requirement(); r.FlexGrowX != 1 || r.FlexGrowY != 1 {
		t.Errorf("filler flex missing: %+v", r)
	}
}

func TestRenderStaysInRegion(t *testing.T) {
	scr := Render(NewHBox(NewText("AAA"), NewText("bbbbbbbbbb")), 10, 1)
	for x := 0; x < 3; x++ {
		if got := scr.Get(x, 0).Grapheme; got != "A" {
			t.Errorf("cell %d = %q, want A", x, got)
		}
	}
	if got := scr.Get(3, 0).Grapheme; got != "b" {
		t.Errorf("cell 3 = %q, want b", got)
	}
	if got := scr.Get(9, 0).Grapheme; got != "b" {
		t.Errorf("cell 9 = %q, want b", got)
	}

	// filler distributes the leftover width; last text sits at the right edge
	scr2 := Render(NewHBox(NewText("AAA"), NewFiller(), NewText("Z")), 10, 1)
	if got := scr2.Get(9, 0).Grapheme; got != "Z" {
		t.Errorf("right edge cell = %q, want Z (screen %q)", got, scr2.String())
	}

	// zero-size renders must not panic
	Render(NewText("long text"), 0, 0)
	for _, el := range []Element{nil, NewText("x")} {
		Render(el, 1, 0)
		Render(el, 0, 1)
	}
}
