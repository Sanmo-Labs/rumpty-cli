package term

import (
	"io"
	"os"
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
