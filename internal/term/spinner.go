package term

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Spinner shows a simple progress indicator on interactive terminals.
type Spinner struct {
	w     io.Writer
	label string
	done  chan struct{}
	once  sync.Once
	noop  bool
}

// StartSpinner begins animating label on w. Call Stop before taking over the terminal.
func StartSpinner(w io.Writer, label string) *Spinner {
	if w == nil || !IsInteractive(w) {
		if w != nil {
			_, _ = fmt.Fprintf(w, "%s\n", label)
		}
		return &Spinner{noop: true}
	}

	s := &Spinner{w: w, label: label, done: make(chan struct{})}
	go s.run()
	return s
}

// Stop clears the spinner line. Safe to call more than once.
func (s *Spinner) Stop() {
	if s == nil || s.noop {
		return
	}
	s.once.Do(func() { close(s.done) })
	_, _ = fmt.Fprint(s.w, "\r\033[K")
}

// StopSpinnerOnWrite wraps w and stops spin on the first non-empty write.
func StopSpinnerOnWrite(w io.Writer, spin *Spinner) io.Writer {
	if spin == nil {
		return w
	}
	return &stopOnWrite{w: w, spin: spin}
}

type stopOnWrite struct {
	w    io.Writer
	spin *Spinner
	once sync.Once
}

func (s *stopOnWrite) Write(p []byte) (int, error) {
	if len(p) > 0 {
		s.once.Do(s.spin.Stop)
	}
	return s.w.Write(p)
}

func (s *Spinner) run() {
	frames := `-\|/`
	i := 0
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			_, _ = fmt.Fprintf(s.w, "\r%s %c ", s.label, frames[i%len(frames)])
			i++
		}
	}
}
