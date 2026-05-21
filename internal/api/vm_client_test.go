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

func TestClient_ListVMs(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal([]api.VM{
		{
			UID: "vm-1", Name: "Test VM 7", Slug: "test-vm7", Status: "running", DisplayStatus: "running",
			PlanSlug: "micro", VCPU: 1, MemoryMiB: 1024, DiskGiB: 20, ZoneSlug: "olas-closet",
			ImageSlug: "ubuntu-24-04", ImageName: "Ubuntu 24.04 LTS",
		},
		{UID: "vm-2", Name: "Dev box", Slug: "dev-box", Status: "stopped"},
	})
	require.NoError(t, err)

	var auth, workspace string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/vms", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		auth = r.Header.Get("Authorization")
		workspace = r.Header.Get("X-Workspace-Slug")
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", json.RawMessage(payload)))
	}))

	client := api.NewClient(srv.URL, "tok")
	got, err := client.ListVMs(context.Background(), "production-team")
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok", auth)
	assert.Equal(t, "production-team", workspace)
	require.Len(t, got, 2)
	assert.Equal(t, "test-vm7", got[0].Slug)
	assert.Equal(t, "running", got[0].DisplayStatus)
	assert.Equal(t, "micro", got[0].PlanSlug)
	assert.Equal(t, 1, got[0].VCPU)
	assert.Equal(t, "Ubuntu 24.04 LTS", got[0].ImageName)
}

func TestClient_StopVM(t *testing.T) {
	t.Parallel()

	var workspace, idempotency string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/vms/vm-uid-7/stop", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		workspace = r.Header.Get("X-Workspace-Slug")
		idempotency = r.Header.Get("X-Idempotency")
		writeJSON(t, w, http.StatusAccepted, apiEnvelope(true, "vm operation queued", "", map[string]any{
			"operation_id": "op-1",
			"vm_uid":       "vm-uid-7",
			"action":       "stop",
			"status":       "queued",
		}))
	}))

	client := api.NewClient(srv.URL, "tok")
	got, err := client.StopVM(context.Background(), "acme", "vm-uid-7", "idem-1")
	require.NoError(t, err)
	assert.Equal(t, "acme", workspace)
	assert.Equal(t, "idem-1", idempotency)
	assert.Equal(t, "op-1", got.OperationID)
}

func TestClient_DeleteVM(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/vms/vm-uid-7", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		writeJSON(t, w, http.StatusAccepted, apiEnvelope(true, "vm operation queued", "", map[string]any{
			"operation_id": "op-2",
			"action":       "delete",
		}))
	}))

	client := api.NewClient(srv.URL, "tok")
	got, err := client.DeleteVM(context.Background(), "acme", "vm-uid-7", "idem-2")
	require.NoError(t, err)
	assert.Equal(t, "op-2", got.OperationID)
}

func TestClient_ListVMs_empty(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", json.RawMessage("[]")))
	}))

	client := api.NewClient(srv.URL, "tok")
	got, err := client.ListVMs(context.Background(), "acme")
	require.NoError(t, err)
	assert.Empty(t, got)
}
