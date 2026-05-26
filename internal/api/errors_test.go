package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func TestError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *api.Error
		want string
	}{
		{
			name: "message preferred",
			err:  &api.Error{StatusCode: 403, Message: "forbidden"},
			want: "forbidden",
		},
		{
			name: "fallback to status code",
			err:  &api.Error{StatusCode: 500},
			want: "request failed HTTP 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestTransportError_timeout(t *testing.T) {
	t.Parallel()

	err := api.TransportError(context.DeadlineExceeded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.NotContains(t, err.Error(), "/v1/")
	assert.NotContains(t, err.Error(), "http://")
}

func TestTransportError_preservesAPIError(t *testing.T) {
	t.Parallel()

	apiErr := &api.Error{StatusCode: 409, Message: "vm is not running"}
	err := api.TransportError(apiErr)
	require.ErrorIs(t, err, apiErr)
}

func TestTransportError_hidesWrappedPath(t *testing.T) {
	t.Parallel()

	raw := fmt.Errorf(`get /v1/vms/uid/apps: Get "http://localhost:8889/v1/vms/uid/apps": %w`, context.DeadlineExceeded)
	err := api.TransportError(raw)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "localhost")
	assert.NotContains(t, err.Error(), "/v1/vms")
}
