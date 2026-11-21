package term

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table prints aligned columns. Unlike text/tabwriter it measures the visible
// width of each cell, so colored cells stay aligned.
type Table struct {
	w       io.Writer
	headers []string
	rows    [][]string
}

func NewTable(w io.Writer, headers ...string) *Table {
	return &Table{w: w, headers: headers}
}

func (t *Table) Row(cells ...string) {
	t.rows = append(t.rows, cells)
}

func (t *Table) Render() error {
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, row := range t.rows {
		for i := 0; i < len(row) && i < len(widths); i++ {
			widths[i] = max(widths[i], lipgloss.Width(row[i]))
		}
	}

	headers := make([]string, len(t.headers))
	for i, h := range t.headers {
		headers[i] = stylize(t.w, h, &headerStyle)
	}
	if err := t.line(headers, widths); err != nil {
		return err
	}
	for _, row := range t.rows {
		if err := t.line(row, widths); err != nil {
			return err
		}
	}
	return nil
}

func (t *Table) line(cells []string, widths []int) error {
	var b strings.Builder
	for i, cell := range cells {
		if i >= len(widths) {
			break
		}
		b.WriteString(cell)
		if i < len(cells)-1 {
			b.WriteString(strings.Repeat(" ", widths[i]-lipgloss.Width(cell)+3))
		}
	}
	_, err := fmt.Fprintln(t.w, strings.TrimRight(b.String(), " "))
	return err
}
