package auth_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/auth"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
)

func TestLogout_clearsCredentials(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Save(credentials.File{
		APIURL: "https://api.example",
		Token:  "tok",
	}))

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().Logout(gomock.Any()).Return(nil)

	rt, errOut := testRuntime(t, mock, &config.Config{APIURL: "https://api.example"})
	require.NoError(t, auth.Logout(context.Background(), rt))
	assert.Contains(t, errOut.String(), "Logged out")

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Empty(t, creds.Token)
}

func TestLogout_noTokenStillClears(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	rt, errOut := testRuntime(t, nil, &config.Config{})
	require.NoError(t, auth.Logout(context.Background(), rt))
	assert.Contains(t, errOut.String(), "Logged out")
}

func TestLogout_ignores401FromServer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Save(credentials.File{Token: "revoked"}))

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().Logout(gomock.Any()).Return(&api.Error{StatusCode: 401, Message: "unauthorized"})

	rt, _ := testRuntime(t, mock, &config.Config{APIURL: "https://api.example"})
	require.NoError(t, auth.Logout(context.Background(), rt))

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Empty(t, creds.Token)
}

func TestLogout_revokesSyncKeysAndRemovesLocalSecrets(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, credentials.Save(credentials.File{APIURL: "https://api.example", Token: "tok"}))
	rumptyDir := filepath.Join(dir, "rumpty")
	require.NoError(t, os.WriteFile(filepath.Join(rumptyDir, "s3-keys.json"),
		[]byte(`{"b1":{"uid":"key-1","workspace":"acme","access_key_id":"AK1","secret_access_key":"s","endpoint":"e","bucket":"backups"}}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rumptyDir, "ssh-certs.json"), []byte(`{"version":1,"entries":[]}`), 0o600))

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	gomock.InOrder(
		mock.EXPECT().RevokeAssetAccessKey(gomock.Any(), "acme", "key-1").Return(nil),
		mock.EXPECT().Logout(gomock.Any()).Return(nil),
	)

	rt, errOut := testRuntime(t, mock, &config.Config{APIURL: "https://api.example"})
	require.NoError(t, auth.Logout(context.Background(), rt))
	assert.NotContains(t, errOut.String(), "Could not revoke")

	for _, name := range []string{"credentials.json", "s3-keys.json", "ssh-certs.json"} {
		assert.NoFileExists(t, filepath.Join(rumptyDir, name))
	}
}

func TestLogoutAll(t *testing.T) {
	tests := []struct {
		name      string
		serverErr error
		wantErr   string
		wantOut   string
		wantKept  bool
	}{
		{name: "success", wantOut: "Signed out of all sessions."},
		{name: "api key", serverErr: &api.Error{StatusCode: 403, Message: "forbidden"}, wantErr: "needs a browser login", wantKept: true},
		{name: "old server", serverErr: &api.Error{StatusCode: 404, Message: "not found"}, wantErr: "does not support", wantKept: true},
		{name: "session already ended", serverErr: &api.Error{StatusCode: 401, Message: "unauthorized"}, wantErr: "already ended", wantKept: true},
		{name: "server error", serverErr: &api.Error{StatusCode: 500, Message: "boom"}, wantErr: "Sync daemons were stopped", wantKept: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			require.NoError(t, credentials.Save(credentials.File{APIURL: "https://api.example", Token: "tok"}))

			ctrl := gomock.NewController(t)
			mock := mocks.NewMockClientAPI(ctrl)
			mock.EXPECT().LogoutAll(gomock.Any()).Return(tt.serverErr)

			rt, errOut := testRuntime(t, mock, &config.Config{APIURL: "https://api.example"})
			err := auth.LogoutAll(context.Background(), rt)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Contains(t, errOut.String(), tt.wantOut)
			}

			creds, err := credentials.Load()
			require.NoError(t, err)
			if tt.wantKept {
				assert.Equal(t, "tok", creds.Token)
				assert.NotContains(t, errOut.String(), "Signed out")
			} else {
				assert.Empty(t, creds.Token)
			}
		})
	}
}

func TestLogoutAll_apiKeyIsRefusedBeforeTouchingSync(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	apiKeyLogin := strings.Join([]string{"rumpty", strings.Repeat("0", 8), strings.Repeat("a", 64)}, "_")
	require.NoError(t, credentials.Save(credentials.File{APIURL: "https://api.example", Token: apiKeyLogin}))
	keysFile := filepath.Join(dir, "rumpty", "s3-keys.json")
	require.NoError(t, os.WriteFile(keysFile,
		[]byte(`{"b1":{"uid":"key-1","workspace":"acme","access_key_id":"AK1","secret_access_key":"s","endpoint":"e","bucket":"backups"}}`), 0o600))

	mock := mocks.NewMockClientAPI(gomock.NewController(t))
	rt, _ := testRuntime(t, mock, &config.Config{APIURL: "https://api.example"})
	require.ErrorContains(t, auth.LogoutAll(context.Background(), rt), "not an API key")
	assert.FileExists(t, keysFile)
}

func TestLogoutAll_failureStillListsUnrevokedSyncKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, credentials.Save(credentials.File{APIURL: "https://api.example", Token: "tok"}))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rumpty", "s3-keys.json"),
		[]byte(`{"b1":{"uid":"key-1","workspace":"acme","access_key_id":"AK1","secret_access_key":"s","endpoint":"e","bucket":"backups"}}`), 0o600))

	mock := mocks.NewMockClientAPI(gomock.NewController(t))
	gomock.InOrder(
		mock.EXPECT().RevokeAssetAccessKey(gomock.Any(), "acme", "key-1").Return(&api.Error{StatusCode: 500, Message: "boom"}),
		mock.EXPECT().LogoutAll(gomock.Any()).Return(&api.Error{StatusCode: 404, Message: "not found"}),
	)
	rt, errOut := testRuntime(t, mock, &config.Config{APIURL: "https://api.example"})
	require.ErrorContains(t, auth.LogoutAll(context.Background(), rt), "does not support")
	assert.Contains(t, errOut.String(), "Could not revoke these sync keys")
}

func TestLogoutAll_notLoggedIn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	rt, _ := testRuntime(t, nil, &config.Config{})
	require.ErrorContains(t, auth.LogoutAll(context.Background(), rt), "not logged in")
}
