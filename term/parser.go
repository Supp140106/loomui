package term

import (
	"bytes"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	esc            byte = 0x1b
	defaultEscTime      = 50 * time.Millisecond
	pasteStart          = "200"
	pasteEndSeq         = "\x1b[201~"
)

// Parser converts raw terminal input bytes into Events.
//
// It is not safe for concurrent use: feed it sequentially from one goroutine.
// Escape sequences that may continue in a following read are buffered until
// complete. If nothing completes them within Timeout of the first buffered
// byte, a lone ESC is reported as a key press. Call Parse with an empty
// slice when no input has arrived past the time returned by Deadline, so
// such a timeout can fire.
type Parser struct {
	Timeout  time.Duration
	pending  []byte
	deadline time.Time
	inPaste  bool
}

func NewParser() *Parser {
	return &Parser{Timeout: defaultEscTime}
}

// Deadline returns the time at which Parse should be called again with an
// empty slice if no input has arrived, and whether such a deadline exists.
func (p *Parser) Deadline() (time.Time, bool) {
	if len(p.pending) == 0 {
		return time.Time{}, false
	}
	return p.deadline, true
}

// Parse feeds a chunk of terminal input (may be nil) observed at time now,
// and returns the events decoded so far.
func (p *Parser) Parse(b []byte, now time.Time) []Event {
	var evs []Event

	if len(p.pending) != 0 && !now.Before(p.deadline) {
		evs = p.flushStale(evs, now)
	}
	if len(p.pending) == 0 && len(b) != 0 {
		p.deadline = now.Add(p.Timeout)
	}
	p.pending = append(p.pending, b...)

	for len(p.pending) > 0 {
		ev, n, ok := p.tryConsume()
		if ok {
			if ev != nil {
				evs = append(evs, ev)
			}
			p.shift(n)
			if ev == nil && len(p.pending) > 0 && p.pending[0] == esc {
				p.deadline = now.Add(p.Timeout)
			}
			continue
		}
		if !now.Before(p.deadline) {
			evs = p.flushStale(evs, now)
			continue
		}
		break
	}
	return evs
}

// shift drops the first n pending bytes.
func (p *Parser) shift(n int) {
	if n >= len(p.pending) {
		p.pending = p.pending[:0]
		return
	}
	copy(p.pending, p.pending[n:])
	p.pending = p.pending[:len(p.pending)-n]
}

// flushStale fires the escape-sequence timeout: buffered input that could
// not be completed is resolved to a lone ESC key (or a partial paste), or
// discarded when unresolvable.
func (p *Parser) flushStale(evs []Event, now time.Time) []Event {
	if p.inPaste {
		evs = append(evs, PasteEvent{Text: string(p.pending)})
		p.pending = p.pending[:0]
		p.inPaste = false
		return evs
	}
	switch {
	case len(p.pending) > 1 && p.pending[0] == esc && p.pending[1] == ']':
		// unterminated OSC: cannot recover meaningfully, discard silently
		p.pending = p.pending[:0]
	case len(p.pending) > 0 && p.pending[0] == esc:
		evs = append(evs, KeyEvent{Key: KeyEsc})
		p.shift(1)
		if len(p.pending) > 0 && p.pending[0] == esc {
			p.deadline = now.Add(p.Timeout)
		}
	default:
		// undecodable non-escape bytes: discard
		p.pending = p.pending[:0]
	}
	return evs
}

// tryConsume attempts to decode one event from the head of the pending
// buffer. ok implies a complete (possibly ignored, ev == nil) sequence of n
// bytes was consumed; !ok means more input is needed.
func (p *Parser) tryConsume() (ev Event, n int, ok bool) {
	if p.inPaste {
		return p.tryPaste()
	}
	b := p.pending

	if b[0] == esc {
		switch {
		case len(b) == 1:
			return nil, 0, false
		case b[1] == esc:
			return nil, 1, true
		case b[1] == '[':
			return p.tryCSI()
		case b[1] == 'O':
			if len(b) < 3 {
				return nil, 0, false
			}
			if k := decodeSS3Key(b[2]); k != KeyNil {
				return KeyEvent{Key: k}, 3, true
			}
			return nil, 3, true
		case b[1] == ']':
			return p.tryOSC()
		default:
			r, size := utf8.DecodeRune(b[1:])
			if r == utf8.RuneError && size <= 1 && size < utf8.UTFMax {
				return nil, 0, false // possibly incomplete rune after ESC
			}
			return decodeControl(r, ModAlt), 1 + size, true
		}
	}

	if b[0] < utf8.RuneSelf {
		return decodeControl(rune(b[0]), ModNone), 1, true
	}
	r, size := utf8.DecodeRune(b)
	if r == utf8.RuneError && size <= 1 && len(b) < utf8.UTFMax {
		return nil, 0, false // possibly incomplete rune
	}
	return KeyEvent{Rune: r}, size, true
}

func (p *Parser) tryOSC() (Event, int, bool) {
	b := p.pending
	for i := 2; i < len(b); i++ {
		switch {
		case b[i] == 0x07:
			return nil, i + 1, true
		case b[i] == esc && i+1 < len(b) && b[i+1] == '\\':
			return nil, i + 2, true
		case b[i] == esc:
			return nil, 0, false // terminator spans next chunk
		}
	}
	return nil, 0, false
}

func (p *Parser) tryPaste() (Event, int, bool) {
	idx := bytes.Index(p.pending, []byte(pasteEndSeq))
	if idx < 0 {
		return nil, 0, false
	}
	p.inPaste = false
	return PasteEvent{Text: string(p.pending[:idx])}, idx + len(pasteEndSeq), true
}

// tryCSI decodes "\x1b[..." from the head of the pending buffer.
func (p *Parser) tryCSI() (Event, int, bool) {
	b := p.pending
	i := 2
	for ; i < len(b); i++ {
		c := b[i]
		if c >= 0x40 && c <= 0x7e {
			break // final byte
		}
		if !isCSIParamByte(c) && !(c >= 0x20 && c <= 0x2f) {
			// garbage inside: drop everything up to and including it
			return nil, i + 1, true
		}
	}
	if i == len(b) {
		return nil, 0, false // no final byte yet
	}
	final := b[i]
	body := string(b[2:i])

	if body == "" && final == 'M' {
		// legacy X10 mouse encoding follows as three raw bytes
		if len(b) < i+4 {
			return nil, 0, false
		}
		return nil, i + 4, true
	}

	switch body {
	case "200":
		if final == '~' {
			p.inPaste = true
			return nil, i + 1, true
		}
	case "201":
		if final == '~' {
			return nil, i + 1, true
		}
	}

	if strings.HasPrefix(body, "<") && (final == 'M' || final == 'm') {
		ev, ok := parseSGRMouse(body[1:], final)
		if !ok {
			return nil, i + 1, true
		}
		return ev, i + 1, true
	}

	switch final {
	case 'I':
		if body == "" {
			return FocusEvent{Gained: true}, i + 1, true
		}
	case 'O':
		if body == "" {
			return FocusEvent{Gained: false}, i + 1, true
		}
	case 'A', 'B', 'C', 'D':
		if k, ok := decodeArrowKey(final); ok {
			return KeyEvent{Key: k, Mod: modifierFromCSI(body)}, i + 1, true
		}
	case 'H', 'F':
		k := KeyHome
		if final == 'F' {
			k = KeyEnd
		}
		if body == "" || body == "1" {
			return KeyEvent{Key: k}, i + 1, true
		}
		return KeyEvent{Key: k, Mod: modifierFromCSI(body)}, i + 1, true
	case 'P', 'Q', 'R', 'S':
		return KeyEvent{Key: decodeCSIKey(final), Mod: modifierFromCSI(body)}, i + 1, true
	case '~':
		if k, ok := decodeTildeKey(body); ok {
			return KeyEvent{Key: k, Mod: tildeModifier(body)}, i + 1, true
		}
	}
	return nil, i + 1, true
}

func isCSIParamByte(c byte) bool {
	switch {
	case c >= '0' && c <= '9':
		return true
	case c == ';' || c == ':' || c == '<' || c == '>' || c == '=' || c == '?':
		return true
	}
	return false
}

// modifierFromCSI decodes a CSI "1;N" (or ";N") modifier parameter into Mod.
func modifierFromCSI(body string) Mod {
	parts := strings.Split(body, ";")
	if len(parts) < 2 {
		return ModNone
	}
	m, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || m < 2 {
		return ModNone
	}
	mask := m - 1
	var mod Mod
	if mask&1 != 0 {
		mod |= ModShift
	}
	if mask&2 != 0 {
		mod |= ModAlt
	}
	if mask&4 != 0 {
		mod |= ModCtrl
	}
	return mod
}

// tildeModifier decodes the modifier parameter of a "~" key where the first
// parameter is the key code and the second is the modifier.
func tildeModifier(body string) Mod {
	parts := strings.Split(body, ";")
	if len(parts) < 2 {
		return ModNone
	}
	return modifierFromCSI(strings.Join(parts[1:], ";"))
}

func decodeArrowKey(final byte) (Key, bool) {
	switch final {
	case 'A':
		return KeyUp, true
	case 'B':
		return KeyDown, true
	case 'C':
		return KeyRight, true
	case 'D':
		return KeyLeft, true
	}
	return KeyNil, false
}

func decodeSS3Key(c byte) Key {
	switch c {
	case 'A':
		return KeyUp
	case 'B':
		return KeyDown
	case 'C':
		return KeyRight
	case 'D':
		return KeyLeft
	case 'H':
		return KeyHome
	case 'F':
		return KeyEnd
	case 'P':
		return KeyF1
	case 'Q':
		return KeyF2
	case 'R':
		return KeyF3
	case 'S':
		return KeyF4
	}
	return KeyNil
}

func decodeCSIKey(final byte) Key {
	switch final {
	case 'P':
		return KeyF1
	case 'Q':
		return KeyF2
	case 'R':
		return KeyF3
	case 'S':
		return KeyF4
	}
	return KeyNil
}

// decodeTildeKey decodes CSI "N~" key codes from VT/xterm terminfo.
func decodeTildeKey(body string) (Key, bool) {
	parts := strings.Split(body, ";")
	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return KeyNil, false
	}
	switch code {
	case 1, 7:
		return KeyHome, true
	case 2:
		return KeyInsert, true
	case 3:
		return KeyDelete, true
	case 4, 8:
		return KeyEnd, true
	case 5:
		return KeyPgUp, true
	case 6:
		return KeyPgDown, true
	case 11:
		return KeyF1, true
	case 12:
		return KeyF2, true
	case 13:
		return KeyF3, true
	case 14:
		return KeyF4, true
	case 15:
		return KeyF5, true
	case 17:
		return KeyF6, true
	case 18:
		return KeyF7, true
	case 19:
		return KeyF8, true
	case 20:
		return KeyF9, true
	case 21:
		return KeyF10, true
	case 23:
		return KeyF11, true
	case 24:
		return KeyF12, true
	}
	return KeyNil, false
}

// parseSGRMouse decodes SGR enhanced mouse parameters "b;x;y" with final M
// (press or motion) or m (release). Coordinates arrive 1-based and are
// returned 0-based.
func parseSGRMouse(params string, final byte) (Event, bool) {
	parts := strings.Split(params, ";")
	if len(parts) != 3 {
		return nil, false
	}
	b, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, false
	}
	x, err := strconv.Atoi(parts[1])
	if err != nil || x < 1 {
		return nil, false
	}
	y, err := strconv.Atoi(parts[2])
	if err != nil || y < 1 {
		return nil, false
	}
	ev := MouseEvent{X: x - 1, Y: y - 1}
	if b&4 != 0 {
		ev.Mods |= ModShift
	}
	if b&8 != 0 {
		ev.Mods |= ModAlt
	}
	if b&16 != 0 {
		ev.Mods |= ModCtrl
	}
	switch {
	case b&64 != 0: // wheel
		ev.Button = BtnWheelUp
		if b&1 != 0 {
			ev.Button = BtnWheelDown
		}
		ev.Pressed = true
	case final == 'm':
		ev.Button = physicalButton(b)
	default:
		ev.Button = physicalButton(b)
		ev.Pressed = true
		ev.Motion = b&32 != 0
	}
	return ev, true
}

func physicalButton(b int) Button {
	switch b & 3 {
	case 0:
		return BtnLeft
	case 1:
		return BtnMiddle
	case 2:
		return BtnRight
	}
	return BtnNone
}

// decodeControl maps one non-escape input rune to an event. runeByte is the
// decoded byte value; mod carries Alt for ESC-prefixed input.
func decodeControl(r rune, mod Mod) Event {
	switch r {
	case '\r', '\n':
		return KeyEvent{Key: KeyEnter, Mod: mod}
	case '\t':
		return KeyEvent{Key: KeyTab, Mod: mod}
	case 0x08, 0x7f:
		return KeyEvent{Key: KeyBackspace, Mod: mod}
	case 0x00:
		return KeyEvent{Rune: ' ', Mod: mod | ModCtrl}
	}
	if r > 0 && r < 0x20 {
		return KeyEvent{Rune: 'a' + r - 1, Mod: mod | ModCtrl}
	}
	return KeyEvent{Rune: r, Mod: mod}
}
