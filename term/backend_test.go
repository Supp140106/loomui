package term

import (
	"bytes"
	"strings"
	"testing"
)

func TestTerminalRestoreWritesExitSequences(t *testing.T) {
	out := &bytes.Buffer{}
	tt := &Terminal{
		out:     out,
		stopped: make(chan struct{}),
		done:    make(chan struct{}),
		events:  make(chan Event, 1),
	}
	close(tt.done)
	if err := tt.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := tt.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got := out.String()
	for _, want := range []string{"\x1b[?1049l", "\x1b[?25h", "\x1b[?2004l", "\x1b[?1006l", "\x1b[?1004l"} {
		if !strings.Contains(got, want) {
			t.Errorf("exit output missing %q", want)
		}
	}
	if n := strings.Count(got, "\x1b[?1049l"); n != 1 {
		t.Errorf("exit alt screen emitted %d times, want 1", n)
	}
	if _, ok := <-tt.events; ok {
		t.Error("events channel is still open after Close")
	}
}

func TestEnterSequencesEnableEverything(t *testing.T) {
	for _, want := range []string{
		"\x1b[?1049h", "\x1b[?25l", "\x1b[?2004h",
		"\x1b[?1000h", "\x1b[?1002h", "\x1b[?1006h", "\x1b[?1004h",
	} {
		if !strings.Contains(seqEnter, want) {
			t.Errorf("enter sequence missing %q", want)
		}
	}
}
