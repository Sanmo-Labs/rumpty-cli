package term

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTable_alignsColoredCells(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	table := NewTable(&out, "NAME", "STATUS", "URL")
	table.Row("web", greenStyle.Render("● running"), "web.example")
	table.Row("database-01", redStyle.Render("✗ failed"), "—")
	require.NoError(t, table.Render())

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 3)
	col := func(line, cell string) int {
		i := strings.Index(line, cell)
		require.GreaterOrEqual(t, i, 0, cell)
		return lipgloss.Width(line[:i])
	}
	assert.Equal(t, col(lines[0], "STATUS"), col(lines[1], "●"))
	assert.Equal(t, col(lines[0], "STATUS"), col(lines[2], "✗"))
	assert.Equal(t, col(lines[0], "URL"), col(lines[1], "web.example"))
	for _, line := range lines {
		assert.Equal(t, strings.TrimRight(line, " "), line, "no trailing spaces")
	}
}

func TestStatusBadge_plainWhenNotATerminal(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "running", StatusBadge(&bytes.Buffer{}, "running"))
	assert.Equal(t, "Done", Success(&bytes.Buffer{}, "Done"))
}

func TestParseOutputFormat(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]OutputFormat{"": OutputTable, "table": OutputTable, "JSON": OutputJSON} {
		got, err := ParseOutputFormat(in)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := ParseOutputFormat("yaml")
	assert.Error(t, err)
}
