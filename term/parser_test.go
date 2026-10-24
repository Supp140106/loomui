package term

import (
	"reflect"
	"testing"
	"time"
)

var base = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func feed(p *Parser, chunks ...string) []Event {
	var evs []Event
	for _, c := range chunks {
		evs = append(evs, p.Parse([]byte(c), base)...)
	}
	return evs
}

func feedAfter(p *Parser, d time.Duration, s string) []Event {
	return p.Parse([]byte(s), base.Add(d))
}

func feedTick(p *Parser, d time.Duration) []Event {
	return p.Parse(nil, base.Add(d))
}

func TestParserFixtures(t *testing.T) {
	tests := []struct {
		name  string
		chunk string
		want  []Event
	}{
		{"ascii", "ab", []Event{
			KeyEvent{Rune: 'a'}, KeyEvent{Rune: 'b'},
		}},
		{"utf8", "héllo 你", []Event{
			KeyEvent{Rune: 'h'}, KeyEvent{Rune: 'é'}, KeyEvent{Rune: 'l'},
			KeyEvent{Rune: 'l'}, KeyEvent{Rune: 'o'}, KeyEvent{Rune: ' '},
			KeyEvent{Rune: '你'},
		}},
		{"enter cr lf tab", "\r\n\t", []Event{
			KeyEvent{Key: KeyEnter}, KeyEvent{Key: KeyEnter}, KeyEvent{Key: KeyTab},
		}},
		{"backspace", "\x7f\x08", []Event{
			KeyEvent{Key: KeyBackspace}, KeyEvent{Key: KeyBackspace},
		}},
		{"ctrl combos", "\x03\x04\x05\x02", []Event{
			KeyEvent{Rune: 'c', Mod: ModCtrl},
			KeyEvent{Rune: 'd', Mod: ModCtrl},
			KeyEvent{Rune: 'e', Mod: ModCtrl},
			KeyEvent{Rune: 'b', Mod: ModCtrl},
		}},
		{"arrows", "\x1b[A\x1b[B\x1b[C\x1b[D", []Event{
			KeyEvent{Key: KeyUp}, KeyEvent{Key: KeyDown},
			KeyEvent{Key: KeyRight}, KeyEvent{Key: KeyLeft},
		}},
		{"ss3 arrows", "\x1bOA\x1bOB", []Event{
			KeyEvent{Key: KeyUp}, KeyEvent{Key: KeyDown},
		}},
		{"modified arrows", "\x1b[1;5A\x1b[1;2C\x1b[1;3D", []Event{
			KeyEvent{Key: KeyUp, Mod: ModCtrl},
			KeyEvent{Key: KeyRight, Mod: ModShift},
			KeyEvent{Key: KeyLeft, Mod: ModAlt},
		}},
		{"home end brackets", "\x1b[H\x1b[F", []Event{
			KeyEvent{Key: KeyHome}, KeyEvent{Key: KeyEnd},
		}},
		{"tilde nav", "\x1b[1~\x1b[2~\x1b[3~\x1b[4~\x1b[5~\x1b[6~", []Event{
			KeyEvent{Key: KeyHome}, KeyEvent{Key: KeyInsert},
			KeyEvent{Key: KeyDelete}, KeyEvent{Key: KeyEnd},
			KeyEvent{Key: KeyPgUp}, KeyEvent{Key: KeyPgDown},
		}},
		{"function keys ss3", "\x1bOP\x1bOQ\x1bOR\x1bOS", []Event{
			KeyEvent{Key: KeyF1}, KeyEvent{Key: KeyF2},
			KeyEvent{Key: KeyF3}, KeyEvent{Key: KeyF4},
		}},
		{"function keys tilde", "\x1b[11~\x1b[12~\x1b[15~\x1b[18~\x1b[21~\x1b[24~", []Event{
			KeyEvent{Key: KeyF1}, KeyEvent{Key: KeyF2}, KeyEvent{Key: KeyF5},
			KeyEvent{Key: KeyF7}, KeyEvent{Key: KeyF10}, KeyEvent{Key: KeyF12},
		}},
		{"function keys with mods", "\x1b[1;2P\x1b[1;5Q", []Event{
			KeyEvent{Key: KeyF1, Mod: ModShift},
			KeyEvent{Key: KeyF2, Mod: ModCtrl},
		}},
		{"alt letters", "\x1bn\x1bØ", []Event{
			KeyEvent{Rune: 'n', Mod: ModAlt},
			KeyEvent{Rune: 'Ø', Mod: ModAlt},
		}},
		{"alt special", "\x1b\r", []Event{
			KeyEvent{Key: KeyEnter, Mod: ModAlt},
		}},
		{"mouse press release", "\x1b[<0;5;3M\x1b[<0;5;3m", []Event{
			MouseEvent{X: 4, Y: 2, Button: BtnLeft, Pressed: true},
			MouseEvent{X: 4, Y: 2, Button: BtnLeft},
		}},
		{"mouse wheel and motion", "\x1b[<64;2;1M\x1b[<65;2;1M\x1b[<32;9;4M", []Event{
			MouseEvent{X: 1, Y: 0, Button: BtnWheelUp, Pressed: true},
			MouseEvent{X: 1, Y: 0, Button: BtnWheelDown, Pressed: true},
			MouseEvent{X: 8, Y: 3, Button: BtnLeft, Pressed: true, Motion: true},
		}},
		{"mouse with mods", "\x1b[<4;1;1M\x1b[<8;1;1M\x1b[<16;1;1M", []Event{
			MouseEvent{X: 0, Y: 0, Button: BtnLeft, Pressed: true, Mods: ModShift},
			MouseEvent{X: 0, Y: 0, Button: BtnLeft, Pressed: true, Mods: ModAlt},
			MouseEvent{X: 0, Y: 0, Button: BtnLeft, Pressed: true, Mods: ModCtrl},
		}},
		{"paste", "\x1b[200~hello 世界\x1b[201~", []Event{
			PasteEvent{Text: "hello 世界"},
		}},
		{"focus in out", "\x1b[I\x1b[O", []Event{
			FocusEvent{Gained: true}, FocusEvent{Gained: false},
		}},
		{"osc title ignored", "\x1b]0;title\x07zz", []Event{
			KeyEvent{Rune: 'z'}, KeyEvent{Rune: 'z'},
		}},
		{"unknown csi ignored", "\x1b[?2026;2$yz", []Event{
			KeyEvent{Rune: 'z'},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got := feed(p, tt.chunk)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) =\n%v\nwant\n%v", tt.chunk, got, tt.want)
			}
		})
	}
}

func TestParserChunkBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		want   []Event
	}{
		{"csi split", []string{"\x1b", "[", "1", ";", "5", "A"}, []Event{
			KeyEvent{Key: KeyUp, Mod: ModCtrl},
		}},
		{"rune split", []string{"\xe4", "\xbd", "\xa0"}, []Event{
			KeyEvent{Rune: '你'},
		}},
		{"paste split", []string{"\x1b[200~a", "bc\x1b", "[201~"}, []Event{
			PasteEvent{Text: "abc"},
		}},
		{"mixed", []string{"ab", "\x1b[C", "\x1b[<0;1;1M", "c"}, []Event{
			KeyEvent{Rune: 'a'}, KeyEvent{Rune: 'b'},
			KeyEvent{Key: KeyRight},
			MouseEvent{X: 0, Y: 0, Button: BtnLeft, Pressed: true},
			KeyEvent{Rune: 'c'},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got := feed(p, tt.chunks...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("chunks %q =\n%v\nwant\n%v", tt.chunks, got, tt.want)
			}
		})
	}
}

func TestParserEscapeTimeout(t *testing.T) {
	p := NewParser()

	if got := feed(p, "\x1b"); len(got) != 0 {
		t.Errorf("lone ESC before timeout = %v, want none", got)
	}
	if _, ok := p.Deadline(); !ok {
		t.Error("p.Deadline() missing for lone ESC")
	}
	got := feedTick(p, 30*time.Millisecond)
	if len(got) != 0 {
		t.Errorf("tick before timeout fired = %v, want none", got)
	}
	got = feedTick(p, 60*time.Millisecond)
	want := []Event{KeyEvent{Key: KeyEsc}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tick after timeout = %v, want %v", got, want)
	}
}

func TestParserEscapeThenLiteralInput(t *testing.T) {
	p := NewParser()
	got := feed(p, "\x1b")
	got = append(got, feedTick(p, 60*time.Millisecond)...)
	got = append(got, feedAfter(p, 70*time.Millisecond, "[A")...)
	want := []Event{KeyEvent{Key: KeyEsc}, KeyEvent{Rune: '['}, KeyEvent{Rune: 'A'}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ESC then literal = %v, want %v", got, want)
	}
}

func TestParserEscapeAbortedByNextBytes(t *testing.T) {
	p := NewParser()
	got := append(feed(p, "\x1b"), feedAfter(p, 10*time.Millisecond, "[A")...)
	want := []Event{KeyEvent{Key: KeyUp}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ESC aborted by CSI = %v, want %v", got, want)
	}
}

func TestParserMouseSplitChunks(t *testing.T) {
	p := NewParser()
	got := feed(p, "\x1b[<0;", "7;3M")
	want := []Event{MouseEvent{X: 6, Y: 2, Button: BtnLeft, Pressed: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mouse across chunks = %v, want %v", got, want)
	}
}
