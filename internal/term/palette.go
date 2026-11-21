package term

import (
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RumptyCloud brand colors. lipgloss converts them to the nearest color on
// terminals without true-color support.
var (
	skyBlue     = lipgloss.Color("#59C2FF")
	emerald     = lipgloss.Color("#4FD99A")
	brandOrange = lipgloss.Color("#FF6B2C")
)

var (
	accentStyle  = lipgloss.NewStyle().Foreground(skyBlue)
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(skyBlue)
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(emerald)
	greenStyle   = lipgloss.NewStyle().Foreground(emerald)
	yellowStyle  = lipgloss.NewStyle().Foreground(brandOrange)
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

// Accent returns s in the RumptyCloud sky blue when w is a color-capable terminal.
func Accent(w io.Writer, s string) string {
	return stylize(w, s, &accentStyle)
}

// Success returns a green "✓ s" line on terminals and plain s elsewhere.
func Success(w io.Writer, s string) string {
	if !colorEnabled(w) {
		return s
	}
	return successStyle.Render("✓ " + s)
}

// StatusBadge prefixes a resource status with a colored symbol on terminals,
// for example "● running". Piped output keeps the plain status.
func StatusBadge(w io.Writer, status string) string {
	status = strings.TrimSpace(status)
	if status == "" || !colorEnabled(w) {
		return status
	}
	switch strings.ToLower(status) {
	case "running", "ready", "active", "healthy", "live", "succeeded":
		return greenStyle.Render("● " + status)
	case "stopped", "offline", "inactive", "suspended":
		return mutedStyle.Render("○ " + status)
	case "failed", "error", "unhealthy", "unreachable":
		return redStyle.Render("✗ " + status)
	default:
		return yellowStyle.Render("◐ " + status)
	}
}
