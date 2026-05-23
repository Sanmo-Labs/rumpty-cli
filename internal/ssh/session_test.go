package ssh_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	xssh "golang.org/x/crypto/ssh"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestNewKeyPair_AuthorizedKeyLine(t *testing.T) {
	t.Parallel()

	key, err := rumptyssh.NewKeyPair()
	require.NoError(t, err)

	line, err := key.AuthorizedKeyLine()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(line, "ssh-ed25519 "))

	pub, _, _, _, err := xssh.ParseAuthorizedKey([]byte(line))
	require.NoError(t, err)
	assert.Equal(t, "ssh-ed25519", pub.Type())
}

func TestOpen_issueCertError(t *testing.T) {
	t.Parallel()
	t.Cleanup(rumptyssh.ResetCertCacheForTest)
	rumptyssh.ResetCertCacheForTest()

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().IssueSSHCert(gomock.Any(), "acme", gomock.Any()).Return(api.CertResponse{}, &api.Error{
		Message: "vm not running",
	})

	rt := &app.Runtime{
		Config:    &config.Config{APIURL: "https://api.example", Token: "tok", Workspace: "acme"},
		APIClient: mock,
	}

	err := rumptyssh.Open(context.Background(), rt, "my-vm", &rumptyssh.Options{})
	require.Error(t, err)
	var apiErr *api.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "vm not running", apiErr.Message)
}

func TestOpen_sendsPublicKey(t *testing.T) {
	t.Parallel()
	t.Cleanup(rumptyssh.ResetCertCacheForTest)
	rumptyssh.ResetCertCacheForTest()

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().IssueSSHCert(gomock.Any(), "acme", gomock.Any()).
		DoAndReturn(func(_ context.Context, ws string, req api.CertRequest) (api.CertResponse, error) {
			assert.Equal(t, "acme", ws)
			assert.Equal(t, "my-vm", req.VM)
			assert.Equal(t, "ubuntu", req.Username)
			assert.True(t, strings.HasPrefix(req.PublicKey, "ssh-ed25519 "))
			return api.CertResponse{}, &api.Error{Message: "stop"}
		})

	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		APIClient: mock,
	}
	_ = rumptyssh.Open(context.Background(), rt, "my-vm", &rumptyssh.Options{GuestUser: "ubuntu"})
}

func TestExitError(t *testing.T) {
	t.Parallel()
	err := &rumptyssh.ExitError{Code: 255}
	assert.Equal(t, "ssh exited with status 255", err.Error())
}
