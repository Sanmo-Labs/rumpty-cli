package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
			want: "rumpty API error. HTTP 500.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}
