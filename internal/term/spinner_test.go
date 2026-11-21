package term

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpinner_printsEachStepWhenNotATerminal(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	spin := StartSpinner(&out, "Finding web")
	spin.Update("Starting web")
	spin.Stop()
	spin.Stop()

	assert.Equal(t, "› Finding web\n› Starting web\n", out.String())
}

func TestStartLoading_isSilentWhenNotATerminal(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	spin := StartLoading(&out, "Loading VMs")
	spin.Update("still loading")
	spin.Stop()

	assert.Empty(t, out.String())
}

func TestSpinner_nilIsSafe(t *testing.T) {
	t.Parallel()

	var spin *Spinner
	spin.Update("x")
	spin.Stop()
}

func TestSpinner_noEscapeWhenNotTerminal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	spin := StartSpinner(&buf, "Connecting")
	spin.Stop()
	assert.Contains(t, buf.String(), "Connecting")
	assert.NotContains(t, buf.String(), "\r")
}

func TestSpinner_nilWriterIsSafe(t *testing.T) {
	t.Parallel()

	spin := StartSpinner(nil, "Connecting")
	spin.Update("Still connecting")
	spin.Stop()
	spin.Stop()
}
