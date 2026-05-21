package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

func TestClient_StartDevice(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/auth/device", r.URL.Path)
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "", "", map[string]any{
			"device_code":               "dc-1",
			"user_code":                 "ABCD-1234",
			"verification_uri":          "https://app.example/device",
			"verification_uri_complete": "https://app.example/device?user_code=ABCD-1234",
			"expires_in":                900,
			"interval":                  5,
		}))
	}))

	got, err := api.NewClient(srv.URL, "").StartDevice(context.Background())
	require.NoError(t, err)
	assert.Equal(t, api.DeviceAuthStartResponse{
		DeviceCode:              "dc-1",
		UserCode:                "ABCD-1234",
		VerificationURI:         "https://app.example/device",
		VerificationURIComplete: "https://app.example/device?user_code=ABCD-1234",
		ExpiresIn:               900,
		Interval:                5,
	}, got)
}

func TestClient_PollDeviceToken(t *testing.T) {
	t.Parallel()

	deviceCode := "device-secret"
	var polls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		n := polls.Add(1)
		switch n {
		case 1:
			writeJSON(t, w, http.StatusOK, apiEnvelope(true, "", "", map[string]any{
				"status": api.DeviceAuthStatusPending,
			}))
		default:
			writeJSON(t, w, http.StatusOK, apiEnvelope(true, "", "", map[string]any{
				"token": "jwt-token",
				"user":  map[string]any{"username": "alice"},
			}))
		}
	})

	srv := newTestServer(t, mux)
	client := api.NewClient(srv.URL, "")

	poll1, err := client.PollDeviceToken(context.Background(), deviceCode)
	require.NoError(t, err)
	assert.Equal(t, api.DeviceAuthStatusPending, poll1.Status)
	assert.Empty(t, poll1.Token)

	poll2, err := client.PollDeviceToken(context.Background(), deviceCode)
	require.NoError(t, err)
	assert.Equal(t, "jwt-token", poll2.Token)
	assert.Equal(t, "alice", poll2.User.Username)
}

func TestClient_PollDeviceToken_expired(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusBadRequest, apiEnvelope(false, "device code expired", "", nil))
	}))

	_, err := api.NewClient(srv.URL, "").PollDeviceToken(context.Background(), "gone")
	require.Error(t, err)
	var apiErr *api.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "device code expired", apiErr.Message)
}
