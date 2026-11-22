package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
)

func TestCheckBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url string
		ok  bool
	}{
		{"https://api.rumptycloud.com", true},
		{"https://api.stg.rumptycloud.com/", true},
		{"http://localhost:8889", true},
		{"http://127.0.0.1:8889", true},
		{"http://[::1]:8889", true},
		{"http://api.rumptycloud.com", false},
		{"http://10.0.0.5:8889", false},
		{"ftp://api.rumptycloud.com", false},
		{"api.rumptycloud.com", false},
		{"", false},
	}
	for _, tt := range tests {
		err := api.CheckBaseURL(tt.url)
		if tt.ok {
			assert.NoError(t, err, tt.url)
		} else {
			assert.Error(t, err, tt.url)
		}
	}
}

func TestClient_refusesPlainHTTPBeforeSending(t *testing.T) {
	t.Parallel()

	client := api.NewClient("http://api.example.com", "secret")
	_, err := client.Me(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plain HTTP")
}
