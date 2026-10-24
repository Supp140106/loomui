package term

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// Backend is the terminal I/O abstraction the runtime builds on.
type Backend interface {
	Size() (int, int)
	Events() <-chan Event
	Write(p []byte) (int, error)
	Close() error
}

const (
	seqEnter = "\x1b[?1049h\x1b[?25l" + // alt screen, hide cursor
		"\x1b[?2004h" + // bracketed paste
		"\x1b[?1000h\x1b[?1002h\x1b[?1006h" + // mouse press/motion, SGR
		"\x1b[?1004h" // focus events
	seqExit = "\x1b[?1004l\x1b[?1006l\x1b[?1002l\x1b[?1000l" +
		"\x1b[?2004l\x1b[?25h\x1b[?1049l"
)

// Terminal is the real Backend: it puts the terminal in raw mode, switches to
// the alternate screen, enables bracketed paste, SGR mouse and focus events,
// and renders new size via SIGWINCH.
//
// Terminal state is restored by Close, during a panic unwind (defer Close),
// and on SIGINT/SIGTERM (state is restored and the default handler then
// terminates the process).
type Terminal struct {
	events chan Event
	out    io.Writer

	raw     *term.State
	parser  *Parser
	stopped chan struct{}
	done    chan struct{}

	stopOnce  sync.Once
	evtOnce   sync.Once
	awaitOnce sync.Once
}

// New opens the terminal backend on stdin/stdout or errors when the
// environment is not a terminal.
func New() (*Terminal, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("term: stdin is not a terminal")
	}
	raw, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	t := &Terminal{
		events:  make(chan Event, 64),
		out:     os.Stdout,
		raw:     raw,
		parser:  NewParser(),
		stopped: make(chan struct{}),
		done:    make(chan struct{}),
	}
	if _, err := t.Write([]byte(seqEnter)); err != nil {
		return nil, fmt.Errorf("term: enter sequences: %w", err)
	}
	winch := make(chan os.Signal, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go t.readInput()
	go t.watchResize(winch)
	go t.watchSignals(sigs)
	return t, nil
}

// Size returns the current terminal dimensions, or (0, 0) when unavailable.
func (t *Terminal) Size() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 0 || h < 0 {
		return 0, 0
	}
	return w, h
}

// Events returns the stream of terminal events; it is closed by Close only
// after input processing has ended.
func (t *Terminal) Events() <-chan Event {
	return t.events
}

// Write writes bytes to the terminal output verbatim.
func (t *Terminal) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

// Close restores the terminal and stops event delivery. It is idempotent.
func (t *Terminal) Close() error {
	t.stopOnce.Do(func() { close(t.stopped) })
	t.restore()
	select {
	case <-t.done:
	case <-time.After(time.Second):
	}
	t.evtOnce.Do(func() { close(t.events) })
	return nil
}

func (t *Terminal) restore() {
	t.awaitOnce.Do(func() {
		if t.out != nil {
			t.out.Write([]byte(seqExit))
		}
		if t.raw != nil {
			term.Restore(int(os.Stdin.Fd()), t.raw)
		}
	})
}

func (t *Terminal) readInput() {
	defer close(t.done)
	buf := make([]byte, 1024)
	for {
		select {
		case <-t.stopped:
			return
		default:
		}
		deadline := time.Now().Add(500 * time.Millisecond)
		if err := os.Stdin.SetReadDeadline(deadline); err == nil {
			// deadlines let Close unblock a silent terminal
		}
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			t.sendAll(t.parser.Parse(buf[:n], time.Now()))
		}
		if err != nil {
			if isRetryable(err) {
				continue
			}
			return
		}
	}
}

func isRetryable(err error) bool {
	var ne interface{ Timeout() bool }
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	if errors.Is(err, syscall.EINTR) {
		return true
	}
	return false
}

func (t *Terminal) sendAll(evs []Event) {
	for _, ev := range evs {
		t.send(ev)
	}
}

func (t *Terminal) send(ev Event) {
	select {
	case <-t.stopped:
		return
	default:
	}
	select {
	case <-t.stopped:
	case t.events <- ev:
	}
}

func (t *Terminal) watchResize(winch <-chan os.Signal) {
	for {
		select {
		case <-t.stopped:
			return
		case <-winch:
			w, h := t.Size()
			t.send(ResizeEvent{W: w, H: h})
		}
	}
}

func (t *Terminal) watchSignals(sigs chan os.Signal) {
	sig, ok := <-sigs
	if !ok {
		return
	}
	select {
	case <-t.stopped:
		return
	default:
	}
	t.restore()
	signal.Stop(sigs)
	syscall.Kill(os.Getpid(), sig.(syscall.Signal))
}
