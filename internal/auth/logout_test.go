package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
	"github.com/Sanmo-Labs/rumpty-cli/internal/auth"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
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

	rt, errOut := testRuntime(t, mock, config.Config{APIURL: "https://api.example"})
	require.NoError(t, auth.Logout(context.Background(), rt))
	assert.Contains(t, errOut.String(), "Logged out")

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Empty(t, creds.Token)
}

func TestLogout_noTokenStillClears(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	rt, errOut := testRuntime(t, nil, config.Config{})
	require.NoError(t, auth.Logout(context.Background(), rt))
	assert.Contains(t, errOut.String(), "Logged out")
}

func TestLogout_ignores401FromServer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, credentials.Save(credentials.File{Token: "revoked"}))

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().Logout(gomock.Any()).Return(&api.Error{StatusCode: 401, Message: "unauthorized"})

	rt, _ := testRuntime(t, mock, config.Config{APIURL: "https://api.example"})
	require.NoError(t, auth.Logout(context.Background(), rt))

	creds, err := credentials.Load()
	require.NoError(t, err)
	assert.Empty(t, creds.Token)
}
