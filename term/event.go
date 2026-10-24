package term

// Event is anything that arrives from the terminal: key presses, mouse,
// paste, focus, and resize notifications. Consume via a type switch.
type Event interface {
	event()
}

// Key identifies non-printable keys. Rune-based keys (letters, digits,
// space, Ctrl+letter) use KeyNil.
type Key uint8

const (
	KeyNil Key = iota // rune-based key event, see KeyEvent.Rune
	KeyEnter
	KeyTab
	KeyEsc
	KeyBackspace
	KeyDelete
	KeyInsert
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDown
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

var keyNames = [...]string{
	"Ignored", "Enter", "Tab", "Esc", "Backspace", "Delete", "Insert",
	"Home", "End", "PgUp", "PgDown", "Up", "Down", "Left", "Right",
	"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12",
}

func (k Key) String() string {
	if int(k) < len(keyNames) {
		return keyNames[k]
	}
	return "key"
}

// Mod is a bitmask of keyboard modifiers.
type Mod uint8

const (
	ModNone  Mod = 0
	ModShift Mod = 1 << 0
	ModAlt   Mod = 1 << 1
	ModCtrl  Mod = 1 << 2
)

func (m Mod) String() string {
	if m == ModNone {
		return ""
	}
	var parts []string
	if m&ModShift != 0 {
		parts = append(parts, "shift")
	}
	if m&ModAlt != 0 {
		parts = append(parts, "alt")
	}
	if m&ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	s := parts[0]
	for _, p := range parts[1:] {
		s += "+" + p
	}
	return s
}

// Button identifies a mouse button for MouseEvent.
type Button uint8

const (
	BtnNone Button = iota
	BtnLeft
	BtnMiddle
	BtnRight
	BtnWheelUp
	BtnWheelDown
)

var buttonNames = [...]string{"none", "left", "middle", "right", "wheelUp", "wheelDown"}

func (b Button) String() string {
	if int(b) < len(buttonNames) {
		return buttonNames[b]
	}
	return "button"
}

// KeyEvent is a key press. Special keys carry Key != KeyNil and no Rune;
// rune-based keys (including Ctrl+letter and Alt+key) carry Rune and KeyNil.
type KeyEvent struct {
	Key  Key
	Rune rune
	Mod  Mod
}

// MouseEvent is a mouse action in 0-based screen coordinates. Release events
// set Pressed false; WheelUp/WheelDown buttons are always Pressed true.
type MouseEvent struct {
	X, Y    int
	Button  Button
	Mods    Mod
	Pressed bool
	Motion  bool
}

// PasteEvent carries text pasted into the terminal (bracketed paste enabled
// by the backend).
type PasteEvent struct {
	Text string
}

// FocusEvent notifies that the terminal gained or lost focus.
type FocusEvent struct {
	Gained bool
}

// ResizeEvent notifies that the terminal size changed.
type ResizeEvent struct {
	W, H int
}

func (KeyEvent) event()    {}
func (MouseEvent) event()  {}
func (PasteEvent) event()  {}
func (FocusEvent) event()  {}
func (ResizeEvent) event() {}
