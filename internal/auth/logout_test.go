package auth_test

import (
	"context"
	"os"
	"path/filepath"
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
