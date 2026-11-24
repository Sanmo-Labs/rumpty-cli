package apikeys_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/apikeys"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newRevokeRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: out, ErrOut: out},
		APIClient: mock,
	}, out
}

func testAPIKey() api.APIKey {
	return api.APIKey{UID: "key-1", Name: "CI deploy", Prefix: "3f9a1c2e"}
}

func TestRevoke_withYesRevokesAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAPIKeys(gomock.Any()).Return([]api.APIKey{testAPIKey()}, nil)
	mock.EXPECT().RevokeAPIKey(gomock.Any(), "key-1").Return(nil)

	rt, out := newRevokeRuntime(t, mock)

	require.NoError(t, apikeys.Revoke(context.Background(), rt, "CI deploy", true))
	assert.Contains(t, out.String(), "Revoked API key CI deploy")
}

func TestRevoke_matchesPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAPIKeys(gomock.Any()).Return([]api.APIKey{testAPIKey()}, nil)
	mock.EXPECT().RevokeAPIKey(gomock.Any(), "key-1").Return(nil)

	rt, _ := newRevokeRuntime(t, mock)

	require.NoError(t, apikeys.Revoke(context.Background(), rt, "3f9a1c2e", true))
}

func TestRevoke_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAPIKeys(gomock.Any()).Return([]api.APIKey{}, nil)

	rt, _ := newRevokeRuntime(t, mock)

	err := apikeys.Revoke(context.Background(), rt, "nope", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no API key")
}

func TestRevoke_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListAPIKeys(gomock.Any()).Return([]api.APIKey{testAPIKey()}, nil)

	rt, _ := newRevokeRuntime(t, mock)

	err := apikeys.Revoke(context.Background(), rt, "CI deploy", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}

func TestCreate_printsTokenOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().CreateAPIKey(gomock.Any(), api.CreateAPIKeyRequest{Name: "CI deploy"}).Return(api.CreateAPIKeyResult{
		APIKey: api.APIKey{UID: "key-1", Name: "CI deploy", Prefix: "abc12345", Scopes: []string{"cli"}},
		Token:  "rumpty_secret_token",
	}, nil)

	rt, out := newRevokeRuntime(t, mock)

	require.NoError(t, apikeys.Create(context.Background(), rt, "CI deploy", apikeys.CreateOpts{}))
	assert.Contains(t, out.String(), "Created API key CI deploy (prefix abc12345, scope cli)")
	assert.Contains(t, out.String(), "shown only once")
	assert.Contains(t, out.String(), "rumpty_secret_token")
}

func TestCreate_rejectsConflictingExpiry(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	rt, _ := newRevokeRuntime(t, mock)

	err := apikeys.Create(context.Background(), rt, "x", apikeys.CreateOpts{ExpiresDays: 30, NeverExpiry: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined")
}

func TestCreate_rejectsBadScope(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	rt, _ := newRevokeRuntime(t, mock)

	err := apikeys.Create(context.Background(), rt, "x", apikeys.CreateOpts{Scope: "sdk"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cli or mcp")
}
