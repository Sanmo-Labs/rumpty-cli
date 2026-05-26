package term_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func TestBold_noEscapeWhenNotTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	var buf bytes.Buffer
	assert.Equal(t, "hello", term.Bold(&buf, "hello"))
}

func TestBold_noEscapeWithNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	got := term.Bold(os.Stderr, "hello")
	assert.Equal(t, "hello", got)
}

func TestMuted_noEscapeWhenNotTerminal(t *testing.T) {
	var buf bytes.Buffer
	assert.Equal(t, "hello", term.Muted(&buf, "hello"))
}

func TestLink_noEscapeWhenNotTerminal(t *testing.T) {
	var buf bytes.Buffer
	assert.Equal(t, "https://example.com", term.Link(&buf, "https://example.com"))
}
