package term

import (
	"fmt"
	"io"
	"strings"
)

// Statusf prints a quiet progress line for work that may take a moment.
// It writes to stderr-style streams so stdout stays useful for command output.
func Statusf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	msg := strings.TrimSpace(fmt.Sprintf(format, args...))
	if msg == "" {
		return
	}
	_, _ = fmt.Fprintf(w, "%s\n", Muted(w, "› "+msg))
}
