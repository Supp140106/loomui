package term

import (
	"reflect"
	"testing"
)

func TestDetectProfile(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want ColorProfile
	}{
		{"empty env", map[string]string{}, ProfileNone},
		{"no color", map[string]string{"TERM": "xterm-256color", "NO_COLOR": "1"}, ProfileNone},
		{"no_color empty", map[string]string{"TERM": "xterm", "NO_COLOR": ""}, ProfileANSI},
		{"colorterm truecolor", map[string]string{"TERM": "xterm", "COLORTERM": "truecolor"}, ProfileTrueColor},
		{"colorterm 24bit", map[string]string{"TERM": "screen", "COLORTERM": "24bit"}, ProfileTrueColor},
		{"TERM 256color", map[string]string{"TERM": "xterm-256color"}, Profile256},
		{"TERM direct", map[string]string{"TERM": "xterm-direct"}, ProfileTrueColor},
		{"TERM plain", map[string]string{"TERM": "xterm"}, ProfileANSI},
		{"TERM dumb", map[string]string{"TERM": "dumb"}, ProfileNone},
		{"TERM truecolor wins over no COLORTERM", map[string]string{"TERM": "xterm-truecolor"}, ProfileTrueColor},
		{"no_color beats colorterm", map[string]string{"COLORTERM": "truecolor", "NO_COLOR": "1"}, ProfileNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectProfile(tt.env); got != tt.want {
				t.Errorf("DetectProfile(%v) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}

func TestKeyAndModStrings(t *testing.T) {
	if got := KeyUp.String(); got != "Up" {
		t.Errorf("KeyUp = %q", got)
	}
	if got := KeyF12.String(); got != "F12" {
		t.Errorf("KeyF12 = %q", got)
	}
	if got := (ModShift | ModCtrl).String(); got != "shift+ctrl" {
		t.Errorf("Mod = %q", got)
	}
	if got := ModNone.String(); got != "" {
		t.Errorf("ModNone = %q", got)
	}
	if got := BtnWheelDown.String(); got != "wheelDown" {
		t.Errorf("BtnWheelDown = %q", got)
	}
	if !reflect.TypeOf(FocusEvent{}).Implements(reflect.TypeOf((*Event)(nil)).Elem()) {
		t.Error("FocusEvent does not implement Event")
	}
}
