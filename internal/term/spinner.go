package term

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const (
	spinnerInterval = 100 * time.Millisecond
	// The elapsed time appears once a step has taken this long.
	spinnerShowElapsed = 3 * time.Second
)

// Spinner shows progress for an action on interactive terminals. Its label
// can change as the action moves through steps. On other outputs each label
// is printed once as a status line.
type Spinner struct {
	w     io.Writer
	quiet bool
	noop  bool

	mu      sync.Mutex
	label   string
	started time.Time
	frame   int
	stopped bool
	done    chan struct{}
}

// StartSpinner begins animating label on w. Call Stop before taking over the
// terminal or printing a result.
func StartSpinner(w io.Writer, label string) *Spinner {
	return startSpinner(w, label, false)
}

// StartLoading is StartSpinner for read-only commands: it prints nothing when
// w is not a terminal, so piped output stays clean.
func StartLoading(w io.Writer, label string) *Spinner {
	return startSpinner(w, label, true)
}

func startSpinner(w io.Writer, label string, quiet bool) *Spinner {
	if w == nil {
		return &Spinner{noop: true}
	}
	if !IsInteractive(w) {
		s := &Spinner{w: w, quiet: quiet, noop: true}
		s.printStatus(label)
		return s
	}
	s := &Spinner{w: w, label: label, started: time.Now(), done: make(chan struct{})}
	s.draw()
	go s.run()
	return s
}

// Update changes the label, for example when an action moves to its next step.
func (s *Spinner) Update(label string) {
	if s == nil {
		return
	}
	if s.noop {
		s.printStatus(label)
		return
	}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.label = label
	s.started = time.Now()
	s.mu.Unlock()
	s.draw()
}

// Stop clears the spinner line. Safe to call more than once.
func (s *Spinner) Stop() {
	if s == nil || s.noop {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.stopped = true
	close(s.done)
	_, _ = fmt.Fprint(s.w, "\r\033[K")
}

func (s *Spinner) printStatus(label string) {
	if s.quiet || label == "" {
		return
	}
	Statusf(s.w, "%s", label)
}

func (s *Spinner) run() {
	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.draw()
		}
	}
}

func (s *Spinner) draw() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	line := Accent(s.w, spinnerFrames[s.frame%len(spinnerFrames)]) + " " + s.label + "…"
	if elapsed := time.Since(s.started); elapsed >= spinnerShowElapsed {
		line += " " + Muted(s.w, fmt.Sprintf("%ds", int(elapsed.Seconds())))
	}
	s.frame++
	_, _ = fmt.Fprint(s.w, "\r\033[K"+line)
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
