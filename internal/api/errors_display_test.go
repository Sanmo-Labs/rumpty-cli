package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func TestUserMessage_apiError(t *testing.T) {
	t.Parallel()

	title, hint := api.UserMessage(&api.Error{
		StatusCode: 409,
		Message:    "app exposure name already exists for this vm",
		Action:     "choose a different --name",
	})
	assert.Equal(t, "app exposure name already exists for this vm", title)
	assert.Equal(t, "choose a different --name", hint)
}

func TestUserMessage_wrapped(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("list exposed apps for test-vm8: %w",
		api.TransportError(context.DeadlineExceeded))
	title, hint := api.UserMessage(err)
	assert.Equal(t, "list exposed apps for test-vm8", title)
	assert.Contains(t, hint, "timed out")
}
