package term

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	boldStyle  = lipgloss.NewStyle().Bold(true)
	mutedStyle = lipgloss.NewStyle().Faint(true)
	linkStyle  = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("14"))
	errorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
)

// IsInteractive reports whether w is a terminal suitable for spinners and styling.
func IsInteractive(w io.Writer) bool {
	return colorEnabled(w)
}

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

func stylize(w io.Writer, s string, style *lipgloss.Style) string {
	if s == "" || !colorEnabled(w) {
		return s
	}
	return style.Render(s)
}

// Bold returns s in bold when w is a color-capable terminal.
func Bold(w io.Writer, s string) string {
	return stylize(w, s, &boldStyle)
}

// Muted returns s in faint text when w is a color-capable terminal.
func Muted(w io.Writer, s string) string {
	return stylize(w, s, &mutedStyle)
}

// Link styles URLs and other copy targets (bold, cyan, underlined).
func Link(w io.Writer, s string) string {
	return stylize(w, s, &linkStyle)
}

// Error styles a failure line (bold red, with a leading cross).
func Error(w io.Writer, s string) string {
	msg := strings.TrimSpace(s)
	if msg == "" {
		return ""
	}
	return stylize(w, "✗ "+msg, &errorStyle)
}
