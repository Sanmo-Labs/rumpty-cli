package term

import (
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiCyan      = "\033[36m"
	ansiUnderline = "\033[4m"
)

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd())) //nolint:gosec // file descriptors are small on supported platforms
}

func stylize(w io.Writer, s, open string) string {
	if s == "" || !colorEnabled(w) {
		return s
	}
	return open + s + ansiReset
}

// Bold returns s in bold when w is a color-capable terminal.
func Bold(w io.Writer, s string) string {
	return stylize(w, s, ansiBold)
}

// Link styles URLs and other copy targets (bold, cyan, underlined).
func Link(w io.Writer, s string) string {
	if s == "" || !colorEnabled(w) {
		return s
	}
	return ansiBold + ansiUnderline + ansiCyan + s + ansiReset
}
