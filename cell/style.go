package cell

import "strings"

// Color is a 24-bit RGB color. The zero value means "default": the terminal's
// current foreground/background color is left untouched.
type Color struct {
	R, G, B uint8
}

// Style controls how a cell is presented. It is a plain value type; all
// mutation happens through the style package's chainable API.
type Style struct {
	Fg, Bg    Color
	Bold      bool
	Italic    bool
	Underline bool
	Faint     bool
	Reverse   bool
	Strike    bool
}

// SGR returns the ANSI Select Graphic Rendition sequence that switches the
// terminal to this style, or "" for the default style. Because SGR state is
// additive, callers switching styles must reset first; Diff does this.
func (s Style) SGR() string {
	if s == (Style{}) {
		return ""
	}
	p := make([]string, 0, 8)
	add := func(on bool, code string) {
		if on {
			p = append(p, code)
		}
	}
	add(s.Bold, "1")
	add(s.Faint, "2")
	add(s.Italic, "3")
	add(s.Underline, "4")
	add(s.Reverse, "7")
	add(s.Strike, "9")
	if s.Fg != (Color{}) {
		p = append(p, "38;2;"+fgRGB(s.Fg))
	}
	if s.Bg != (Color{}) {
		p = append(p, "48;2;"+fgRGB(s.Bg))
	}
	return "\x1b[" + strings.Join(p, ";") + "m"
}

func fgRGB(c Color) string {
	return strings.Join([]string{itou8(c.R), itou8(c.G), itou8(c.B)}, ";")
}

func itou8(v uint8) string {
	if v == 0 {
		return "0"
	}
	var b [3]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = '0' + v%10
		v /= 10
	}
	return string(b[i:])
}
