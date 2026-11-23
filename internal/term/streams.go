package term

import (
	"io"
	"os"

	"golang.org/x/term"
)

type Streams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

func System() Streams {
	return Streams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// InputIsTerminal reports whether input is an interactive terminal.
func (s Streams) InputIsTerminal() bool {
	f, ok := s.In.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd())) //nolint:gosec // fd is small on supported platforms
}
