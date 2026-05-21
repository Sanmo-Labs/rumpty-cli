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

func TestClient_IssueSSHCert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		workspace   string
		status      int
		envelope    map[string]any
		want        api.CertResponse
		wantErr     bool
		wantAPIErr  string
		errContains string
	}{
		{
			name:      "success",
			workspace: "acme",
			status:    http.StatusOK,
			envelope: apiEnvelope(true, "", "", map[string]any{
				"router_user": "rumpty+vm-1",
				"edge_host":   "edge.example",
				"edge_port":   2222,
				"certificate": "ssh-ed25519-cert-v01@openssh.com AAAA",
			}),
			want: api.CertResponse{
				RouterUser:  "rumpty+vm-1",
				EdgeHost:    "edge.example",
				EdgePort:    2222,
				Certificate: "ssh-ed25519-cert-v01@openssh.com AAAA",
			},
		},
		{
			name:      "default edge port when omitted",
			workspace: "acme",
			status:    http.StatusOK,
			envelope: apiEnvelope(true, "", "", map[string]any{
				"router_user": "rumpty+vm-1",
				"edge_host":   "edge.example",
				"certificate": "ssh-ed25519-cert-v01@openssh.com AAAA",
			}),
			want: api.CertResponse{
				RouterUser:  "rumpty+vm-1",
				EdgeHost:    "edge.example",
				EdgePort:    22,
				Certificate: "ssh-ed25519-cert-v01@openssh.com AAAA",
			},
		},
		{
			name:       "API forbidden",
			workspace:  "acme",
			status:     http.StatusForbidden,
			envelope:   apiEnvelope(false, "forbidden", "", nil),
			wantErr:    true,
			wantAPIErr: "forbidden",
		},
		{
			name:      "incomplete payload",
			workspace: "acme",
			status:    http.StatusOK,
			envelope: apiEnvelope(true, "", "", map[string]any{
				"edge_host": "edge.example",
			}),
			wantErr:     true,
			errContains: "incomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotWorkspace string
			srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/ssh-sessions/cert", r.URL.Path)
				gotWorkspace = r.Header.Get("X-Workspace-Slug")
				writeJSON(t, w, tt.status, tt.envelope)
			}))

			client := api.NewClient(srv.URL, "token")
			got, err := client.IssueSSHCert(context.Background(), tt.workspace, api.CertRequest{
				VM:        "vm-1",
				PublicKey: "ssh-ed25519 AAA",
			})

			assert.Equal(t, tt.workspace, gotWorkspace)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantAPIErr != "" {
					var apiErr *api.Error
					require.ErrorAs(t, err, &apiErr)
					assert.Equal(t, tt.wantAPIErr, apiErr.Message)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_baseURL_trailingSlash(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me", r.URL.Path)
		data, _ := json.Marshal(api.User{Username: "bob"})
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "", "", json.RawMessage(data)))
	}))

	client := api.NewClient(srv.URL+"/", "tok")
	user, err := client.Me(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "bob", user.Username)
}

func TestClient_decodeEnvelope_nonJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>error</html>"))
	}))

	_, err := api.NewClient(srv.URL, "").Me(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
	assert.NotErrorAs(t, err, new(*api.Error))
}
