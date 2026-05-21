package auth_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/auth"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func testRuntime(t *testing.T, apiClient api.ClientAPI, cfg config.Config) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	errOut := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &cfg,
		Streams:   term.Streams{In: strings.NewReader("\n"), Out: io.Discard, ErrOut: errOut},
		APIClient: apiClient,
		Browser:   term.NoopBrowser{},
	}, errOut
}

func TestLogin_apiKey(t *testing.T) {
	tests := []struct {
		name       string
		meUser     api.User
		meErr      error
		wantErr    string
		wantSaved  bool
		wantOutSub string
	}{
		{
			name:       "valid key saves session",
			meUser:     api.User{Username: "alice"},
			wantSaved:  true,
			wantOutSub: "Logged in as alice",
		},
		{
			name:    "API error surfaces message",
			meErr:   &api.Error{Message: "invalid API key"},
			wantErr: "invalid API key",
		},
		{
			name:    "generic failure",
			meErr:   errors.New("connection refused"),
			wantErr: "API key is invalid or expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			ctrl := gomock.NewController(t)

			mock := mocks.NewMockClientAPI(ctrl)
			mock.EXPECT().Me(gomock.Any()).Return(tt.meUser, tt.meErr)

			rt, errOut := testRuntime(t, mock, config.Config{APIURL: "https://api.example"})
			err := auth.Login(context.Background(), rt, "rumpty_prefix_secret", auth.LoginOptions{})

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, errOut.String(), tt.wantOutSub)

			creds, err := credentials.Load()
			require.NoError(t, err)
			assert.Equal(t, "https://api.example", creds.APIURL)
			assert.Equal(t, "rumpty_prefix_secret", creds.Token)
			assert.Equal(t, "alice", creds.Username)
		})
	}
}

func TestLogin_device_authorization(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctrl := gomock.NewController(t)

	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().StartDevice(gomock.Any()).Return(api.DeviceAuthStartResponse{
		DeviceCode:              "dc-1",
		UserCode:                "WXYZ-9999",
		VerificationURI:         "https://app.example/device",
		VerificationURIComplete: "https://app.example/device?user_code=WXYZ-9999",
		ExpiresIn:               60,
		Interval:                1,
	}, nil)
	mock.EXPECT().PollDeviceToken(gomock.Any(), "dc-1").Return(api.DeviceAuthPollResponse{
		Token: "jwt",
		User:  api.User{Username: "bob"},
	}, nil)

	rt, errOut := testRuntime(t, mock, config.Config{APIURL: "https://api.example"})
	err := auth.Login(context.Background(), rt, "", auth.LoginOptions{})
	require.NoError(t, err)

	out := errOut.String()
	assert.Contains(t, out, "WXYZ-9999")
	assert.Contains(t, out, "Logged in as bob")

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Equal(t, "jwt", creds.Token)
}

func TestLogin_device_unexpectedStatus(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().StartDevice(gomock.Any()).Return(api.DeviceAuthStartResponse{
		DeviceCode: "dc-1",
		UserCode:   "CODE",
		ExpiresIn:  60,
		Interval:   1,
	}, nil)
	mock.EXPECT().PollDeviceToken(gomock.Any(), "dc-1").Return(api.DeviceAuthPollResponse{
		Status: "denied",
	}, nil)

	rt, _ := testRuntime(t, mock, config.Config{APIURL: "https://api.example"})
	err := auth.Login(context.Background(), rt, "", auth.LoginOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected authorization status")
}

func TestSaveSession(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		user    string
		wantErr string
	}{
		{name: "requires token", token: "", wantErr: "token is required"},
		{name: "persists credentials", token: "tok", user: "alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			rt, errOut := testRuntime(t, nil, config.Config{APIURL: "https://api.example"})
			err := auth.SaveSession(rt, "https://api.example", tt.token, tt.user)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, errOut.String(), "Logged in as alice")
		})
	}
}

func TestAlreadyLoggedIn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Save(credentials.File{
		APIURL:   "https://api.example",
		Token:    "stored-token",
		Username: "legacy",
	}))

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().Me(gomock.Any()).Return(api.User{Username: "alice"}, nil)

	rt, errOut := testRuntime(t, mock, config.Config{APIURL: "https://api.example"})
	ok, err := auth.AlreadyLoggedIn(context.Background(), rt)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Contains(t, errOut.String(), "Already logged in as alice")
}

func TestAlreadyLoggedIn_staleToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Save(credentials.File{Token: "stale"}))

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().Me(gomock.Any()).Return(api.User{}, &api.Error{StatusCode: 401, Message: "unauthorized"})

	rt, _ := testRuntime(t, mock, config.Config{APIURL: "https://api.example"})
	ok, err := auth.AlreadyLoggedIn(context.Background(), rt)
	require.NoError(t, err)
	assert.False(t, ok)
}
