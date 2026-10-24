package element

import (
	"strings"
	"unicode"

	"github.com/Supp140106/loomui/cell"
	"github.com/Supp140106/loomui/layout"
	"github.com/rivo/uniseg"
)

// Text displays a single line of styled text.
type Text struct {
	content  string
	style    cell.Style
	minWidth int
	gX, gY   uint8
}

func NewText(content string) *Text { return &Text{content: content} }

func (t *Text) Requirement() layout.Requirement {
	w := max(t.minWidth, 0)
	if cw := cell.StringWidth(t.content); cw > w {
		w = cw
	}
	return layout.Requirement{
		MinW: w, MinH: 1,
		FlexGrowX: t.gX, FlexGrowY: t.gY,
		FlexShrinkX: 1, FlexShrinkY: 1,
	}
}

func (t *Text) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	s.DrawString(r.X, r.Y, t.content, t.style)
}

func (t *Text) Bold() *Text              { t.style.Bold = true; return t }
func (t *Text) Dim() *Text               { t.style.Faint = true; return t }
func (t *Text) Color(c cell.Color) *Text { t.style.Fg = c; return t }
func (t *Text) Width(n int) *Text        { t.minWidth = max(n, 0); return t }
func (t *Text) Flex() *Text              { t.gX, t.gY = 1, 1; return t }

// Paragraph displays a block of text word-wrapped to the given width.
type Paragraph struct {
	content  string
	style    cell.Style
	minWidth int
	gX, gY   uint8
}

func NewParagraph(content string) *Paragraph { return &Paragraph{content: content} }

func (p *Paragraph) Requirement() layout.Requirement {
	w := max(p.minWidth, 0)
	for _, word := range words(p.content) {
		if cw := cell.StringWidth(word); cw > w {
			w = cw
		}
	}
	return layout.Requirement{
		MinW: w, MinH: 1,
		FlexGrowX: p.gX, FlexGrowY: p.gY,
		FlexShrinkX: 1, FlexShrinkY: 1,
	}
}

func (p *Paragraph) Render(s *cell.Screen, r layout.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	for i, line := range Wrap(p.content, r.W) {
		if i >= r.H {
			return
		}
		s.DrawString(r.X, r.Y+i, line, p.style)
	}
}

func (p *Paragraph) Bold() *Paragraph              { p.style.Bold = true; return p }
func (p *Paragraph) Dim() *Paragraph               { p.style.Faint = true; return p }
func (p *Paragraph) Color(c cell.Color) *Paragraph { p.style.Fg = c; return p }
func (p *Paragraph) Width(n int) *Paragraph        { p.minWidth = max(n, 0); return p }
func (p *Paragraph) Flex() *Paragraph              { p.gX, p.gY = 1, 1; return p }

func words(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		out = append(out, strings.FieldsFunc(line, unicode.IsSpace)...)
	}
	return out
}

// Wrap word-wraps content into lines of at most maxWidth columns, breaking
// words wider than maxWidth at grapheme-cluster boundaries. Newlines force
// line breaks.
func Wrap(content string, maxWidth int) []string {
	if maxWidth <= 0 {
		return nil
	}
	var lines []string
	for _, para := range strings.Split(content, "\n") {
		emptyLine := false
		ws := strings.FieldsFunc(para, unicode.IsSpace)
		if len(ws) == 0 {
			emptyLine = true
		}
		cur, curW := "", 0
		for _, word := range ws {
			wd := cell.StringWidth(word)
			if curW > 0 && curW+1+wd <= maxWidth {
				cur += " " + word
				curW += 1 + wd
				continue
			}
			if curW > 0 {
				lines = append(lines, cur)
				cur, curW = "", 0
			}
			for {
				if wd <= maxWidth {
					cur, curW = word, wd
					break
				}
				head, rest := splitAtWidth(word, maxWidth)
				lines = append(lines, head)
				restW := cell.StringWidth(rest)
				if restW == 0 {
					cur, curW = "", 0
					break
				}
				word, wd = rest, restW
			}
		}
		if curW > 0 {
			lines = append(lines, cur)
		} else if emptyLine {
			lines = append(lines, "")
		}
	}
	return lines
}

// splitAtWidth splits s at the first grapheme cluster that does not fit into
// maxWidth columns. At least one cluster is always kept in the head.
func splitAtWidth(s string, maxWidth int) (head, rest string) {
	var b strings.Builder
	w := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		cw := cell.StringWidth(g.Str())
		if w+cw > maxWidth && b.Len() > 0 {
			break
		}
		b.WriteString(g.Str())
		w += cw
	}
	if b.Len() == 0 {
		g.Next()
		first := g.Str()
		return first, s[len(first):]
	}
	return b.String(), s[b.Len():]
}
