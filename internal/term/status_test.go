package term_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func TestStatus_printsQuietLine(t *testing.T) {
	var buf bytes.Buffer
	term.Statusf(&buf, "exposing %s", "openclaw")
	assert.Equal(t, "› exposing openclaw\n", buf.String())
}

func TestStatus_ignoresBlankMessage(t *testing.T) {
	var buf bytes.Buffer
	term.Statusf(&buf, "   ")
	assert.Empty(t, buf.String())
}
