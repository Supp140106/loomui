package element

import (
	"fmt"
	"testing"

	"github.com/Supp140106/loomui/cell"
	tuitest "github.com/Supp140106/loomui/testing"
)

func goldenElements() map[string]func() Element {
	blue := cell.Color{R: 80, G: 140, B: 255}
	red := cell.Color{R: 255, G: 100, B: 90}
	return map[string]func() Element{
		"text": func() Element {
			return NewHBox(
				NewText("Hello").Bold(),
				NewText("Loom").Color(blue).Width(8),
				NewText("!").Dim(),
			)
		},
		"paragraph": func() Element {
			return NewParagraph("The quick brown fox jumps over the lazy dog. 你好世界 loom!")
		},
		"hbox": func() Element {
			return NewHBox(
				NewText("left").Color(red),
				NewFiller(),
				NewText("right").Bold(),
			)
		},
		"vbox": func() Element {
			return NewVBox(
				NewText("top"),
				NewFiller(),
				NewText("bottom").Dim(),
			)
		},
		"filler": func() Element {
			return NewHBox(NewText("[A]"), NewFiller(), NewText("[B]"))
		},
		"separator": func() Element {
			return NewVBox(NewText("above"), NewSeparator(), NewText("below"))
		},
		"border-normal": func() Element {
			return NewBorder(NewText("inner content")).Title("loom")
		},
		"border-rounded": func() Element {
			return NewBorder(NewText("inner content")).Style(BorderRounded).Title("loom")
		},
		"border-thick": func() Element {
			return NewBorder(NewText("inner content")).Style(BorderThick).Title("loom")
		},
		"border-double": func() Element {
			return NewBorder(NewText("inner content")).Style(BorderDouble).Title("loom")
		},
		"center": func() Element {
			return NewCenter(NewText("middle"))
		},
		"padding": func() Element {
			return NewPadding(NewText("padded"), 2)
		},
		"size": func() Element {
			return NewVBox(
				NewText("full width line that is quite long"),
				NewSize(NewText("narrow")).FixedWidth(10),
				NewSize(NewText("capped")).MaxWidth(8),
			)
		},
		"zstack": func() Element {
			return NewZStack(
				NewParagraph("a longer sentence in the bottom layer of the z stack"),
				NewText("★").Bold().Color(red),
			)
		},
	}
}

var goldenSizes = [][2]int{{24, 8}, {44, 14}}

func TestGolden(t *testing.T) {
	for name, mk := range goldenElements() {
		for _, sz := range goldenSizes {
			name := fmt.Sprintf("%s-%dx%d", name, sz[0], sz[1])
			t.Run(name, func(t *testing.T) {
				scr := Render(mk(), sz[0], sz[1])
				tuitest.AssertGolden(t, name, tuitest.RenderANSI(scr))
			})
		}
	}
}
