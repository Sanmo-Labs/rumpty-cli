package api_test

import (
	"context"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
)

func TestClient_SendsCLIUserAgent(t *testing.T) {
	t.Parallel()

	var got string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "", "", map[string]any{"device_code": "dc-1", "user_code": "ABCD-1234"}))
	}))

	_, err := api.NewClient(srv.URL, "").StartDevice(context.Background())
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(got, "rumpty-cli/"), got)
	assert.Contains(t, got, runtime.GOOS)
	assert.Equal(t, api.UserAgent(), got)
}
