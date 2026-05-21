package term_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func TestSpinner_noEscapeWhenNotTerminal(t *testing.T) {
	var buf bytes.Buffer
	spin := term.StartSpinner(&buf, "Connecting...")
	spin.Stop()
	assert.Contains(t, buf.String(), "Connecting...")
	assert.NotContains(t, buf.String(), "\r")
}

func TestSpinner_stopIsIdempotent(t *testing.T) {
	spin := term.StartSpinner(nil, "Connecting...")
	spin.Stop()
	spin.Stop()
}
