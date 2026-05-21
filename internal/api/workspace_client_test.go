package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func TestClient_ListWorkspaces(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal([]api.Workspace{
		{UID: "ws-1", Name: "Production", Slug: "production-team", IsDefault: true},
		{UID: "ws-2", Name: "Dev", Slug: "acme-dev"},
	})
	require.NoError(t, err)

	var auth string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/workspaces", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		auth = r.Header.Get("Authorization")
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", json.RawMessage(payload)))
	}))

	client := api.NewClient(srv.URL, "tok")
	got, err := client.ListWorkspaces(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok", auth)
	require.Len(t, got, 2)
	assert.Equal(t, "production-team", got[0].Slug)
	assert.True(t, got[0].IsDefault)
	assert.Equal(t, "acme-dev", got[1].Slug)
}

func TestClient_ListWorkspaces_empty(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", json.RawMessage("[]")))
	}))

	client := api.NewClient(srv.URL, "tok")
	got, err := client.ListWorkspaces(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)
}
