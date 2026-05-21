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

func TestClient_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		envelope   map[string]any
		want       api.AuthResponse
		wantErr    bool
		wantAPIErr string
	}{
		{
			name:   "requires OTP",
			status: http.StatusOK,
			envelope: apiEnvelope(true, "", "", map[string]any{
				"requires_otp": true,
				"otp_session":  "sess-1",
			}),
			want: api.AuthResponse{RequiresOTP: true, OTPSession: "sess-1"},
		},
		{
			name:   "direct token",
			status: http.StatusOK,
			envelope: apiEnvelope(true, "", "", map[string]any{
				"token": "jwt",
				"user":  map[string]any{"username": "alice"},
			}),
			want: api.AuthResponse{Token: "jwt", User: api.User{Username: "alice"}},
		},
		{
			name:       "API error message",
			status:     http.StatusUnauthorized,
			envelope:   apiEnvelope(false, "invalid credentials", "", nil),
			wantErr:    true,
			wantAPIErr: "invalid credentials",
		},
		{
			name:     "malformed JSON",
			status:   http.StatusOK,
			envelope: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v1/auth/login", r.URL.Path)
				if tt.envelope == nil {
					_, _ = w.Write([]byte("not-json"))
					return
				}
				writeJSON(t, w, tt.status, tt.envelope)
			}))

			client := api.NewClient(srv.URL, "")
			got, err := client.Login(context.Background(), api.LoginRequest{
				Username: "alice",
				Password: "secret",
			})

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantAPIErr != "" {
					var apiErr *api.Error
					require.ErrorAs(t, err, &apiErr)
					assert.Equal(t, tt.wantAPIErr, apiErr.Message)
				}
				return
			}
			if tt.envelope == nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_VerifyLoginOTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		envelope    map[string]any
		status      int
		wantToken   string
		wantErr     bool
		wantAPIErr  string
		errContains string
	}{
		{
			name:   "returns token",
			status: http.StatusOK,
			envelope: apiEnvelope(true, "", "", map[string]any{
				"token": "jwt-token",
				"user":  map[string]any{"username": "alice"},
			}),
			wantToken: "jwt-token",
		},
		{
			name:   "missing token in success payload",
			status: http.StatusOK,
			envelope: apiEnvelope(true, "", "", map[string]any{
				"user": map[string]any{"username": "alice"},
			}),
			wantErr:     true,
			errContains: "did not include a token",
		},
		{
			name:       "forbidden",
			status:     http.StatusForbidden,
			envelope:   apiEnvelope(false, "invalid code", "", nil),
			wantErr:    true,
			wantAPIErr: "invalid code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/auth/login/verify", r.URL.Path)
				writeJSON(t, w, tt.status, tt.envelope)
			}))

			client := api.NewClient(srv.URL, "")
			got, err := client.VerifyLoginOTP(context.Background(), api.VerifyLoginOTPRequest{
				OTPSession: "sess-1",
				Code:       "123456",
			})

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
			assert.Equal(t, tt.wantToken, got.Token)
			assert.Equal(t, "alice", got.User.Username)
		})
	}
}

func TestClient_Me(t *testing.T) {
	t.Parallel()

	userPayload, err := json.Marshal(api.User{UID: "u1", Username: "alice", Email: "a@b.c", Verified: true})
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		status      int
		envelope    map[string]any
		want        api.User
		wantErr     bool
		wantAPIErr  string
		wantAuthHdr bool
	}{
		{
			name:        "success",
			token:       "rumpty_key",
			status:      http.StatusOK,
			envelope:    apiEnvelope(true, "", "", json.RawMessage(userPayload)),
			want:        api.User{UID: "u1", Username: "alice", Email: "a@b.c", Verified: true},
			wantAuthHdr: true,
		},
		{
			name:        "unauthorized",
			token:       "bad",
			status:      http.StatusUnauthorized,
			envelope:    apiEnvelope(false, "unauthorized", "", nil),
			wantErr:     true,
			wantAPIErr:  "unauthorized",
			wantAuthHdr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var auth string
			srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/me", r.URL.Path)
				auth = r.Header.Get("Authorization")
				writeJSON(t, w, tt.status, tt.envelope)
			}))

			client := api.NewClient(srv.URL, tt.token)
			got, err := client.Me(context.Background())

			if tt.wantAuthHdr {
				assert.Equal(t, "Bearer "+tt.token, auth)
			}
			if tt.wantErr {
				require.Error(t, err)
				var apiErr *api.Error
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.wantAPIErr, apiErr.Message)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Logout(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/auth/logout", r.URL.Path)
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "", "", nil))
	}))

	client := api.NewClient(srv.URL, "token")
	require.NoError(t, client.Logout(context.Background()))
}

func TestClient_ResendLoginOTP(t *testing.T) {
	t.Parallel()

	var body map[string]string
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/auth/login/resend", r.URL.Path)
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(t, w, http.StatusOK, apiEnvelope(true, "", "", nil))
	}))

	client := api.NewClient(srv.URL, "")
	require.NoError(t, client.ResendLoginOTP(context.Background(), "sess-99"))
	assert.Equal(t, "sess-99", body["otp_session"])
}
