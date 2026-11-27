package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
)

func lbQueued(t *testing.T) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(api.LoadBalancerOperationResult{
		OperationID:  "op-9",
		Status:       "queued",
		LoadBalancer: &api.LoadBalancer{UID: "lb-1", Slug: "web", Status: "pending"},
	})
	require.NoError(t, err)
	return payload
}

func TestClient_ListLoadBalancers(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal([]api.LoadBalancer{{
		UID: "lb-1", Slug: "web", Status: "ready", URL: "https://web-5d6e7f80.rumptycloud.app",
		TargetPort: 8080, TargetProtocol: "http",
		Targets:     []api.LoadBalancerTarget{{VMUID: "vm-1", VMName: "web-1", VMStatus: "running"}},
		HealthCheck: api.LoadBalancerHealthCheck{Protocol: "http", Path: "/healthz", IntervalSeconds: 10},
	}})
	require.NoError(t, err)

	var workspace, idem string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/load-balancers", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		workspace = r.Header.Get("X-Workspace-Slug")
		idem = r.Header.Get("X-Idempotency")
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", json.RawMessage(payload)))
	}))

	got, err := api.NewClient(srv.URL, "tok").ListLoadBalancers(context.Background(), "acme")
	require.NoError(t, err)
	assert.Equal(t, "acme", workspace)
	assert.Empty(t, idem)
	require.Len(t, got, 1)
	assert.Equal(t, "web", got[0].Slug)
	assert.Equal(t, "web-1", got[0].Targets[0].VMName)
	assert.Equal(t, "/healthz", got[0].HealthCheck.Path)
}

func TestClient_ListLoadBalancers_nullIsEmpty(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", nil))
	}))

	got, err := api.NewClient(srv.URL, "tok").ListLoadBalancers(context.Background(), "acme")
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestClient_CreateLoadBalancer(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var idem string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/load-balancers", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		idem = r.Header.Get("X-Idempotency")
		raw, _ := io.ReadAll(r.Body)
		assert.NoError(t, json.Unmarshal(raw, &body))
		writeJSON(t, w, http.StatusAccepted, apiEnvelope(true, "load balancer creation queued", "", lbQueued(t)))
	}))

	path := "/healthz"
	got, err := api.NewClient(srv.URL, "tok").CreateLoadBalancer(context.Background(), "acme", &api.CreateLoadBalancerRequest{
		Name:        "web",
		TargetPort:  8080,
		VMUIDs:      []string{"vm-1", "vm-2"},
		HealthCheck: &api.LoadBalancerHealthCheckInput{Path: &path},
	}, "key-1")
	require.NoError(t, err)
	assert.Equal(t, "key-1", idem)
	assert.Equal(t, "op-9", got.OperationID)
	require.NotNil(t, got.LoadBalancer)
	assert.Equal(t, "lb-1", got.LoadBalancer.UID)

	assert.Equal(t, map[string]any{
		"name":         "web",
		"target_port":  float64(8080),
		"vm_uids":      []any{"vm-1", "vm-2"},
		"health_check": map[string]any{"path": "/healthz"},
	}, body)
}

func TestClient_UpdateLoadBalancer_sendsOnlySetFields(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/load-balancers/lb-1", r.URL.Path)
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "key-2", r.Header.Get("X-Idempotency"))
		raw, _ := io.ReadAll(r.Body)
		assert.NoError(t, json.Unmarshal(raw, &body))
		writeJSON(t, w, http.StatusAccepted, apiEnvelope(true, "queued", "", lbQueued(t)))
	}))

	sticky := false
	_, err := api.NewClient(srv.URL, "tok").UpdateLoadBalancer(context.Background(), "acme", "lb-1", &api.UpdateLoadBalancerRequest{
		StickySessions: &sticky,
		AllowedCIDRs:   &[]string{},
	}, "key-2")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"sticky_sessions": false,
		"allowed_cidrs":   []any{},
	}, body)
}

func TestClient_LoadBalancerTargets(t *testing.T) {
	t.Parallel()

	var calls []string
	var addBody map[string]any
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost {
			raw, _ := io.ReadAll(r.Body)
			assert.NoError(t, json.Unmarshal(raw, &addBody))
		}
		writeJSON(t, w, http.StatusAccepted, apiEnvelope(true, "queued", "", lbQueued(t)))
	}))

	client := api.NewClient(srv.URL, "tok")
	_, err := client.AddLoadBalancerTargets(context.Background(), "acme", "lb-1", []string{"vm-3"}, "k")
	require.NoError(t, err)
	_, err = client.RemoveLoadBalancerTarget(context.Background(), "acme", "lb-1", "vm-3", "k")
	require.NoError(t, err)
	_, err = client.DeleteLoadBalancer(context.Background(), "acme", "lb-1", "k")
	require.NoError(t, err)
	_, err = client.GetLoadBalancer(context.Background(), "acme", "lb-1")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"POST /v1/load-balancers/lb-1/targets",
		"DELETE /v1/load-balancers/lb-1/targets/vm-3",
		"DELETE /v1/load-balancers/lb-1",
		"GET /v1/load-balancers/lb-1",
	}, calls)
	assert.Equal(t, map[string]any{"vm_uids": []any{"vm-3"}}, addBody)
}

func TestClient_CreateLoadBalancer_conflict(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusConflict, apiEnvelope(false, "a load balancer with this name already exists", "", nil))
	}))

	_, err := api.NewClient(srv.URL, "tok").CreateLoadBalancer(context.Background(), "acme", &api.CreateLoadBalancerRequest{Name: "web"}, "k")
	require.ErrorContains(t, err, "already exists")
}

func TestClient_GetLoadBalancerTraffic(t *testing.T) {
	t.Parallel()

	var path, window, workspace string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, window, workspace = r.URL.Path, r.URL.Query().Get("range"), r.Header.Get("X-Workspace-Slug")
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", json.RawMessage(`{
			"load_balancer_uid":"lb-1","available":true,"range":"24h","step_seconds":1440,
			"requests_total":10,"errors_total":1,"client_errors_total":null,"error_rate":0.1,
			"avg_latency_ms":12.5,"rx_bytes":1,"tx_bytes":2,"points":[{"timestamp":"2026-11-27T10:00:00Z","requests":10,"errors":1}]}`)))
	}))

	got, err := api.NewClient(srv.URL, "tok").GetLoadBalancerTraffic(context.Background(), "acme", "lb-1", "24h")
	require.NoError(t, err)
	assert.Equal(t, "/v1/load-balancers/lb-1/traffic", path)
	assert.Equal(t, "24h", window)
	assert.Equal(t, "acme", workspace)
	assert.Nil(t, got.ClientErrorsTotal, "null stays nil, not a false zero")
	require.NotNil(t, got.AvgLatencyMs)
	assert.InDelta(t, 12.5, *got.AvgLatencyMs, 1e-9)
	require.Len(t, got.Points, 1)
}

func TestClient_GetLoadBalancerHealth(t *testing.T) {
	t.Parallel()

	var path, workspace string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, workspace = r.URL.Path, r.Header.Get("X-Workspace-Slug")
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "ok", "", json.RawMessage(`{
			"load_balancer_uid":"lb-1","status":"degraded","healthy_targets":1,"total_targets":2,"checked_at":"2026-11-27T10:00:00Z"}`)))
	}))

	got, err := api.NewClient(srv.URL, "tok").GetLoadBalancerHealth(context.Background(), "acme", "lb-1")
	require.NoError(t, err)
	assert.Equal(t, "/v1/load-balancers/lb-1/health", path)
	assert.Equal(t, "acme", workspace)
	assert.Equal(t, api.LoadBalancerHealth{LoadBalancerUID: "lb-1", Status: "degraded", HealthyTargets: 1, TotalTargets: 2, CheckedAt: "2026-11-27T10:00:00Z"}, got)
}

func TestClient_GetLoadBalancerHealth_olderServerWithoutRoute(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found"))
	}))

	_, err := api.NewClient(srv.URL, "tok").GetLoadBalancerHealth(context.Background(), "acme", "lb-1")
	var apiErr *api.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.Contains(t, err.Error(), "may not support this command yet")
}
