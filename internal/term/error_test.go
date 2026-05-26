package term_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func TestPrintError_noRumptyPrefix(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	var buf bytes.Buffer
	term.PrintError(&buf, &api.Error{StatusCode: 409, Message: "app exposure name already exists for this vm"})
	out := buf.String()
	assert.NotContains(t, out, "rumpty:")
	assert.Contains(t, out, "app exposure name already exists for this vm")
}

func TestError_prefix(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	assert.Equal(t, "✗ failed", term.Error(bytes.NewBuffer(nil), "failed"))
}

func TestPrintError_wrapped(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	var buf bytes.Buffer
	err := fmt.Errorf("list exposed apps for test-vm8: %w", api.TransportError(context.DeadlineExceeded))
	term.PrintError(&buf, err)
	out := buf.String()
	require.Contains(t, out, "list exposed apps for test-vm8")
	require.Contains(t, out, "request timed out")
}
