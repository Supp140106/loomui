package style

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Supp140106/loomui/cell"
	"github.com/rivo/uniseg"
)

// This file contains the ANSI escape-aware text operations used by
// Style.Render: measuring, truncating and word-wrapping strings that may
// contain escape sequences.

// token is a piece of a styled string: either an escape sequence or a run
// of plain text with its display width precomputed.
type token struct {
	esc   bool
	text  string
	width int
}

// escapeLen returns the byte length of the escape sequence starting at
// s[i], or 0 when s[i] does not start one. It understands CSI (ESC [ ...),
// OSC (ESC ] ..., terminated by BEL or ST) and simple two-byte sequences.
func escapeLen(s string, i int) int {
	if i >= len(s) || s[i] != '\x1b' {
		return 0
	}
	if i+1 >= len(s) {
		return 1
	}
	switch s[i+1] {
	case '[':
		for j := i + 2; j < len(s); j++ {
			if s[j] >= 0x40 && s[j] <= 0x7e {
				return j - i + 1
			}
		}
		return len(s) - i
	case ']':
		for j := i + 2; j < len(s); j++ {
			if s[j] == 0x07 {
				return j - i + 1
			}
			if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2 - i
			}
		}
		return len(s) - i
	default:
		return 2
	}
}

// tokenize splits s into escape and plain-text tokens.
func tokenize(s string) []token {
	var out []token
	for i := 0; i < len(s); {
		if n := escapeLen(s, i); n > 0 {
			out = append(out, token{esc: true, text: s[i : i+n]})
			i += n
			continue
		}
		j := i
		for j < len(s) && escapeLen(s, j) == 0 {
			j++
		}
		text := s[i:j]
		out = append(out, token{esc: false, text: text, width: cell.StringWidth(text)})
		i = j
	}
	return out
}

// ansiWidth returns the display width of s, ignoring escape sequences.
func ansiWidth(s string) int {
	w := 0
	for _, t := range tokenize(s) {
		w += t.width
	}
	return w
}

// isSGR reports whether an escape token is a Select Graphic Rendition
// sequence, the only escape kind that carries style state.
func isSGR(esc string) bool {
	return len(esc) > 3 && strings.HasPrefix(esc, "\x1b[") && esc[len(esc)-1] == 'm'
}

// sgrState tracks the SGR parameters active at a position, so that wrapped
// lines can re-emit them and cut lines can drop them.
type sgrState struct {
	parts []string
}

func (s *sgrState) apply(esc string) {
	if !isSGR(esc) {
		return
	}
	params := strings.Split(esc[2:len(esc)-1], ";")
	for _, p := range params {
		if n, err := strconv.Atoi(p); err == nil && n == 0 {
			s.parts = nil
			continue
		}
		s.parts = append(s.parts, p)
	}
}

func (s *sgrState) empty() bool {
	return len(s.parts) == 0
}

func (s *sgrState) clone() sgrState {
	c := sgrState{parts: make([]string, len(s.parts))}
	copy(c.parts, s.parts)
	return c
}

// seq returns the escape sequence that re-establishes this state, or "".
func (s *sgrState) seq() string {
	if len(s.parts) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(s.parts, ";") + "m"
}

// applyAll applies the escapes of a token list to state, in order.
func (s *sgrState) applyAll(toks []token) {
	for _, t := range toks {
		if t.esc {
			s.apply(t.text)
		}
	}
}

// tokensWidth sums the display widths of a token list.
func tokensWidth(toks []token) int {
	w := 0
	for _, t := range toks {
		w += t.width
	}
	return w
}

// tokensString renders a token list.
func tokensString(toks []token) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(t.text)
	}
	return b.String()
}

// truncateANSI cuts s to maxWidth columns, dropping any further content.
// A reset is appended when the cut happens inside styled text, but never to
// a string that fits entirely.
func truncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	toks := tokenize(s)
	var out strings.Builder
	had := 0
	cut := false
	for _, t := range toks {
		if t.esc {
			out.WriteString(t.text)
			continue
		}
		g := uniseg.NewGraphemes(t.text)
		for g.Next() {
			cl := g.Str()
			cw := cell.StringWidth(cl)
			if had+cw > maxWidth {
				cut = true
				break
			}
			out.WriteString(cl)
			had += cw
		}
		if cut {
			break
		}
	}
	res := out.String()
	if cut && strings.Contains(res, "\x1b") {
		res += "\x1b[0m"
	}
	return res
}

// cutTokens splits toks at the first grapheme cluster that does not fit
// into w columns, keeping at least one cluster in the head. rest holds the
// leftover tokens after the cut; caller re-establishes the style state
// carried over from the head.
func cutTokens(toks []token, w int) (head, rest []token, cut bool) {
	var out []token
	had := 0
	for i, t := range toks {
		if t.esc {
			out = append(out, t)
			continue
		}
		g := uniseg.NewGraphemes(t.text)
		for g.Next() {
			cw := cell.StringWidth(g.Str())
			if had+cw > w && had > 0 {
				_, end := g.Position()
				out = append(out, token{esc: false, text: t.text[:end], width: cell.StringWidth(t.text[:end])})
				if end < len(t.text) {
					suffix := t.text[end:]
					rest = append(rest, token{esc: false, text: suffix, width: cell.StringWidth(suffix)})
				}
				rest = append(rest, toks[i+1:]...)
				return out, rest, true
			}
			out = append(out, token{esc: false, text: g.Str()})
			had += cw
		}
	}
	return out, nil, false
}

// atom is a run of text plus any escapes embedded in it: a word when space
// is false, a whitespace run otherwise.
type atom struct {
	toks  []token
	w     int
	space bool
}

// buildAtoms groups tokens into word and space atoms. Escapes attach to the
// currently open atom, or open a new one when none is.
func buildAtoms(toks []token) []atom {
	var atoms []atom
	open := -1
	add := func() int {
		atoms = append(atoms, atom{})
		return len(atoms) - 1
	}
	for _, t := range toks {
		if t.esc {
			if open < 0 {
				open = add()
			}
			atoms[open].toks = append(atoms[open].toks, t)
			continue
		}
		for _, r := range splitRuns(t.text) {
			w := cell.StringWidth(r.text)
			if r.space {
				open = -1
				sp := add()
				atoms[sp].space = true
				atoms[sp].w = w
				atoms[sp].toks = []token{{esc: false, text: r.text, width: w}}
				continue
			}
			if open < 0 || atoms[open].space {
				open = add()
			}
			atoms[open].toks = append(atoms[open].toks, token{esc: false, text: r.text, width: w})
			atoms[open].w += w
			atoms[open].space = false
		}
	}
	return atoms
}

// run is a contiguous block of spaces or non-spaces inside a text chunk.
type run struct {
	space bool
	text  string
}

// splitRuns splits text into runs of whitespace and non-whitespace runes.
func splitRuns(text string) []run {
	var out []run
	start := 0
	sp := unicode.IsSpace(firstRune(text))
	for i := 0; i < len(text); {
		_, sz := utf8.DecodeRuneInString(text[i:])
		r, _ := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) != sp {
			out = append(out, run{space: sp, text: text[start:i]})
			start = i
			sp = !sp
		}
		i += sz
	}
	out = append(out, run{space: sp, text: text[start:]})
	return out
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// wrapANSI word-wraps s to lines of at most maxWidth columns. Word runs are
// separated by single spaces (multiple spaces collapse), newlines in s
// force breaks, words wider than the limit are hard-split at cluster
// boundaries, escape sequences are preserved, each output line re-emits the
// style state active where it begins, and lines broken mid-word get a
// trailing reset.
func wrapANSI(s string, maxWidth int) []string {
	maxWidth = max(maxWidth, 1)
	var lines []string
	var state sgrState
	for _, src := range strings.Split(s, "\n") {
		atoms := buildAtoms(tokenize(src))
		var cur strings.Builder
		curW := 0
		pendingSpace := false
		lineStart := sgrState{}
		snapNeeded := true
		emitted := false
		flush := func(cut bool) {
			line := lineStart.seq() + cur.String()
			if cut && strings.Contains(line, "\x1b") {
				line += "\x1b[0m"
			}
			lines = append(lines, line)
			cur.Reset()
			curW = 0
			pendingSpace = false
			snapNeeded = true
			emitted = true
		}
		for _, a := range atoms {
			if a.space {
				if curW > 0 {
					pendingSpace = true
				}
				continue
			}
			rest := a.toks
			for {
				wd := tokensWidth(rest)
				if wd == 0 {
					// Escape-only remainder.
					if snapNeeded {
						lineStart = state.clone()
						snapNeeded = false
					}
					state.applyAll(rest)
					cur.WriteString(tokensString(rest))
					break
				}
				sep := 0
				if curW > 0 && pendingSpace {
					sep = 1
				}
				if curW+sep+wd <= maxWidth {
					if curW == 0 && cur.Len() == 0 {
						lineStart = state.clone()
						snapNeeded = false
					}
					if sep == 1 {
						cur.WriteByte(' ')
						curW++
					}
					state.applyAll(rest)
					cur.WriteString(tokensString(rest))
					curW += wd
					pendingSpace = false
					break
				}
				if cur.Len() > 0 {
					flush(false)
				}
				if wd > maxWidth {
					h, r, cut := cutTokens(rest, maxWidth)
					lineStart = state.clone()
					snapNeeded = false
					state.applyAll(h)
					cur.WriteString(tokensString(h))
					flush(true)
					rest = r
					continue
				}
				lineStart = state.clone()
				snapNeeded = false
				state.applyAll(rest)
				cur.WriteString(tokensString(rest))
				curW += wd
				pendingSpace = false
				break
			}
		}
		if cur.Len() > 0 {
			flush(false)
		} else if !emitted {
			lines = append(lines, "")
		}
	}
	return lines
}
