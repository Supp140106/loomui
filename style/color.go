// Package style provides string-level styling for terminal output: colors
// with profile downgrades, ANSI-aware truncation and wrapping, and a
// Lip Gloss-style immutable Style with borders, padding, margins and
// alignment. Unlike the cell/element packages it renders plain strings
// with escape codes, not cell buffers.
package style

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/rivo/uniseg"
)

// Color is a string-typed color in the Lip Gloss tradition. Two forms are
// accepted: "#rrggbb" (or "#rgb") hex for truecolor, and a decimal string
// "0".."255" for an ANSI 256 palette index. Any other value leaves the
// terminal color untouched.
type Color string

// TerminalColor is implemented by Color and AdaptiveColor: a color whose
// literal value may depend on whether the terminal background is dark.
type TerminalColor interface {
	value(isDark bool) Color
}

// value implements TerminalColor.
func (c Color) value(isDark bool) Color {
	_ = isDark
	return c
}

// AdaptiveColor picks between a light and a dark variant based on the
// terminal's background color, queried via OSC 11.
type AdaptiveColor struct {
	Light Color
	Dark  Color
}

// value implements TerminalColor.
func (a AdaptiveColor) value(isDark bool) Color {
	if isDark {
		return a.Dark
	}
	return a.Light
}

type colorKind uint8

const (
	colorNone colorKind = iota
	colorHex
	color256
)

// parsedColor is the internal form of a color: both its declared kind and
// the raw RGB values needed for downgrades.
type parsedColor struct {
	kind    colorKind
	r, g, b uint8
	idx     int
}

// parse converts a Color to its internal representation. It returns
// ok=false for anything that leaves the terminal color untouched.
func (c Color) parse() (parsedColor, bool) {
	s := strings.TrimSpace(string(c))
	if s == "" {
		return parsedColor{}, false
	}
	if allDigits(s) {
		n, err := strconv.Atoi(s)
		if err != nil || n > 255 {
			return parsedColor{}, false
		}
		r, g, b := xterm256ToRGB(n)
		return parsedColor{kind: color256, r: r, g: g, b: b, idx: n}, true
	}
	if r, g, b, ok := parseHex(s); ok {
		return parsedColor{kind: colorHex, r: r, g: g, b: b}, true
	}
	return parsedColor{}, false
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

// parseHex accepts "#rgb", "#rrggbb" and the same three-or-six lowercase or
// uppercase digit forms without the leading '#'.
func parseHex(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if !allHex(s) {
		return 0, 0, 0, false
	}
	switch len(s) {
	case 3:
		return uint8(hexVal(s[0]) * 17), uint8(hexVal(s[1]) * 17), uint8(hexVal(s[2]) * 17), true
	case 6:
		return uint8(hexVal(s[0])<<4 | hexVal(s[1])), uint8(hexVal(s[2])<<4 | hexVal(s[3])),
			uint8(hexVal(s[4])<<4 | hexVal(s[5])), true
	}
	return 0, 0, 0, false
}

func allHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return len(s) > 0
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	default:
		return int(c-'A') + 10
	}
}

// hexColor builds a Color from raw values, the inverse of parseHex.
func hexColor(r, g, b uint8) Color {
	return Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// Profile controls how colors are downgraded when rendered. ProfileAuto
// detects the terminal's capabilities from the environment (once, lazily).
type Profile uint8

const (
	ProfileAuto Profile = iota
	ProfileNone
	ProfileANSI
	Profile256
	ProfileTrueColor
)

func (p Profile) String() string {
	switch p {
	case ProfileAuto:
		return "auto"
	case ProfileNone:
		return "none"
	case ProfileANSI:
		return "ansi"
	case Profile256:
		return "256"
	case ProfileTrueColor:
		return "truecolor"
	}
	return "unknown"
}

var (
	profileOnce     sync.Once
	detectedProfile Profile
)

// effective resolves ProfileAuto against the environment.
func (p Profile) effective() Profile {
	for p == ProfileAuto {
		profileOnce.Do(func() { detectedProfile = detectProfile() })
		p = detectedProfile
	}
	return p
}

// detectProfile reads the usual environment variables. It exists here
// rather than in term/ so style stays independent of the runtime packages.
func detectProfile() Profile {
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return ProfileNone
	}
	if ct := strings.ToLower(os.Getenv("COLORTERM")); ct == "truecolor" || ct == "24bit" {
		return ProfileTrueColor
	}
	if os.Getenv("WT_SESSION") != "" {
		return ProfileTrueColor
	}
	t := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(t, "direct") {
		return ProfileTrueColor
	}
	if strings.Contains(t, "256color") {
		return Profile256
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "vscode":
		return ProfileTrueColor
	}
	return ProfileANSI
}

// fgSeq returns the foreground SGR parameters (without the surrounding
// escape sequence) for a color at the given profile, or "" when the profile
// drops the color.
func fgSeq(p parsedColor, prof Profile) string {
	return colSeq(p, prof, false)
}

// bgSeq is fgSeq for the background.
func bgSeq(p parsedColor, prof Profile) string {
	return colSeq(p, prof, true)
}

func colSeq(p parsedColor, prof Profile, bg bool) string {
	switch prof {
	case ProfileNone:
		return ""
	case ProfileTrueColor:
		if p.kind == colorHex {
			return rgbSeq(p.r, p.g, p.b, bg)
		}
		return idxSeq(p.idx, bg)
	case Profile256:
		if p.kind == color256 {
			return idxSeq(p.idx, bg)
		}
		return idxSeq(rgbToXterm256(p.r, p.g, p.b), bg)
	case ProfileANSI:
		n := p.idx
		if p.kind == colorHex {
			n = nearestANSI16(p.r, p.g, p.b)
		}
		if bg {
			return basicSeq(n, "4", "10", bg)
		}
		return basicSeq(n, "3", "9", bg)
	}
	return ""
}

// basicSeq emits a basic ANSI color: 30-37/90-97 for foreground or
// 40-47/100-107 for background.
func basicSeq(n int, dark, bright string, bg bool) string {
	code := dark
	if n >= 8 {
		code = bright
		n -= 8
	}
	return code + strconv.Itoa(n)
}

func rgbSeq(r, g, b uint8, bg bool) string {
	pfx := "38;2;"
	if bg {
		pfx = "48;2;"
	}
	return pfx + strconv.Itoa(int(r)) + ";" + strconv.Itoa(int(g)) + ";" + strconv.Itoa(int(b))
}

func idxSeq(i int, bg bool) string {
	pfx := "38;5;"
	if bg {
		pfx = "48;5;"
	}
	return pfx + strconv.Itoa(i)
}

// Luminance returns the relative luminance of an sRGB color in [0,1], with
// 0 the darkest black. Pure black is 0, pure white is 1; anything below 0.5
// generally reads as dark.
func Luminance(r, g, b uint8) float64 {
	lin := func(v uint8) float64 {
		c := float64(v) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// Blend linearly interpolates between two colors; t=0 gives a and t=1 gives
// b. Out-of-range t is clamped. If either color cannot be parsed, a is
// returned unchanged.
func Blend(a, b Color, t float64) Color {
	pa, ok1 := a.parse()
	pb, ok2 := b.parse()
	if !ok1 || !ok2 {
		return a
	}
	r, g, bl := lerpRGB(pa, pb, t)
	return hexColor(r, g, bl)
}

// Gradient returns n colors interpolated from a to b, inclusive. It returns
// nil for n <= 0.
func Gradient(a, b Color, n int) []Color {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []Color{a}
	}
	out := make([]Color, n)
	for i := 0; i < n; i++ {
		out[i] = Blend(a, b, float64(i)/float64(n-1))
	}
	return out
}

// GradientText colors every visible grapheme cluster of str along a
// gradient from a to b, spread over the whole string. Unstyled whitespace is
// passed through untouched, newlines are preserved, clusters keep their
// composed glyph streams, and a reset is appended at the end when any color
// was emitted.
func GradientText(str string, a, b Color) string {
	if str == "" {
		return ""
	}
	pa, ok1 := a.parse()
	pb, ok2 := b.parse()
	if !ok1 || !ok2 {
		return str
	}
	visible := func(line string) int {
		n := 0
		g := uniseg.NewGraphemes(line)
		for g.Next() {
			if strings.TrimSpace(g.Str()) != "" {
				n++
			}
		}
		return n
	}
	lines := strings.Split(str, "\n")
	total := 0
	for _, line := range lines {
		total += visible(line)
	}
	if total == 0 {
		return str
	}
	var sb strings.Builder
	sb.Grow(len(str) + 8*total)
	k := 0
	for li, line := range lines {
		g := uniseg.NewGraphemes(line)
		for g.Next() {
			cluster := g.Str()
			if strings.TrimSpace(cluster) == "" {
				sb.WriteString(cluster)
				continue
			}
			t := 0.0
			if total > 1 {
				t = float64(k) / float64(total-1)
			}
			r, gg, bb := lerpRGB(pa, pb, t)
			sb.WriteString("\x1b[38;2;" + strconv.Itoa(int(r)) + ";" + strconv.Itoa(int(gg)) + ";" +
				strconv.Itoa(int(bb)) + "m" + cluster)
			k++
		}
		if strings.HasSuffix(str, "\n") || li < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String() + "\x1b[0m"
}

func lerpRGB(p1, p2 parsedColor, t float64) (r, g, b uint8) {
	if t <= 0 {
		return p1.r, p1.g, p1.b
	}
	if t >= 1 {
		return p2.r, p2.g, p2.b
	}
	mix := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t))
	}
	return mix(p1.r, p2.r), mix(p1.g, p2.g), mix(p1.b, p2.b)
}

// ansi16RGB holds the xterm defaults for the 16 basic ANSI colors, flattened
// as R,G,B triples for indexes 0..15.
var ansi16RGB = [48]uint8{
	0x00, 0x00, 0x00, // 0 black
	0x80, 0x00, 0x00, // 1 maroon
	0x00, 0x80, 0x00, // 2 green
	0x80, 0x80, 0x00, // 3 olive
	0x00, 0x00, 0x80, // 4 navy
	0x80, 0x00, 0x80, // 5 purple
	0x00, 0x80, 0x80, // 6 teal
	0xc0, 0xc0, 0xc0, // 7 silver
	0x80, 0x80, 0x80, // 8 gray
	0xff, 0x00, 0x00, // 9 red
	0x00, 0xff, 0x00, // 10 lime
	0xff, 0xff, 0x00, // 11 yellow
	0x00, 0x00, 0xff, // 12 blue
	0xff, 0x00, 0xff, // 13 fuchsia
	0x00, 0xff, 0xff, // 14 aqua
	0xff, 0xff, 0xff, // 15 white
}

// cubeRGB maps a 0..5 cube level to its 256-palette RGB component.
var cubeRGB = [6]uint8{0x00, 0x5f, 0x87, 0xaf, 0xd7, 0xff}

// xterm256ToRGB converts a 256-palette index to its reference RGB values.
// Out-of-range indexes are clamped.
func xterm256ToRGB(i int) (r, g, b uint8) {
	if i < 0 {
		i = 0
	} else if i > 255 {
		i = 255
	}
	switch {
	case i < 16:
		return ansi16RGB[3*i], ansi16RGB[3*i+1], ansi16RGB[3*i+2]
	case i < 232:
		n := i - 16
		return cubeRGB[n/36], cubeRGB[(n/6)%6], cubeRGB[n%6]
	default:
		v := uint8(8 + 10*(i-232))
		return v, v, v
	}
}

// rgbToXterm256 finds the nearest 256-palette index: either a cube color or
// a grayscale ramp value, whichever is closer in plain RGB distance.
func rgbToXterm256(r, g, b uint8) int {
	cr, cg, cb := v2ci(r), v2ci(g), v2ci(b)
	ci := 36*cr + 6*cg + cb
	avg := (int(r) + int(g) + int(b)) / 3
	if avg > 238 {
		avg = 238
	}
	gi := (avg - 3) / 10
	if gi < 0 {
		gi = 0
	}
	gv := 8 + 10*gi
	if distSq(int(r), int(g), int(b), int(cubeRGB[cr]), int(cubeRGB[cg]), int(cubeRGB[cb])) <=
		distSq(int(r), int(g), int(b), gv, gv, gv) {
		return 16 + ci
	}
	return 232 + gi
}

// v2ci quantizes one channel to the 0..5 cube level.
func v2ci(v uint8) int {
	if v < 48 {
		return 0
	}
	if v < 115 {
		return 1
	}
	return (int(v) - 35) / 40
}

func distSq(r1, g1, b1, r2, g2, b2 int) int {
	dr, dg, db := r1-r2, g1-g2, b1-b2
	return dr*dr + dg*dg + db*db
}

// nearestANSI16 maps an RGB color to the closest of the 16 basic ANSI
// colors by squared RGB distance.
func nearestANSI16(r, g, b uint8) int {
	best, bestDist := 0, 1<<31
	for i := 0; i < 16; i++ {
		d := distSq(int(r), int(g), int(b),
			int(ansi16RGB[3*i]), int(ansi16RGB[3*i+1]), int(ansi16RGB[3*i+2]))
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}
