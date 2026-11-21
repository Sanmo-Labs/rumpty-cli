package term

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// OutputFormat selects how list commands print their results.
type OutputFormat string

const (
	OutputTable OutputFormat = "table"
	OutputJSON  OutputFormat = "json"
)

// ParseOutputFormat validates the value of an --output flag.
func ParseOutputFormat(s string) (OutputFormat, error) {
	switch f := OutputFormat(strings.ToLower(strings.TrimSpace(s))); f {
	case "", OutputTable:
		return OutputTable, nil
	case OutputJSON:
		return OutputJSON, nil
	default:
		return "", fmt.Errorf("unknown output format %q; use table or json", s)
	}
}

// WriteJSON prints v as indented JSON.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Summary prints a faint footer line such as "7 VMs · 6 running" on terminals.
// Piped output gets no footer so it stays easy to parse.
func Summary(w io.Writer, parts ...string) {
	if !colorEnabled(w) || len(parts) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s\n", Muted(w, strings.Join(parts, " · ")))
}
