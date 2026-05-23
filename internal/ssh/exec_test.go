package ssh_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/api/mocks"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestExec_issueCertError(t *testing.T) {
	t.Parallel()
	t.Cleanup(rumptyssh.ResetCertCacheForTest)
	rumptyssh.ResetCertCacheForTest()

	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().IssueSSHCert(gomock.Any(), "acme", gomock.Any()).Return(api.CertResponse{}, &api.Error{
		Message: "vm not running",
	})

	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		APIClient: mock,
	}

	err := rumptyssh.Exec(context.Background(), rt, "my-vm", []string{"uptime"}, &rumptyssh.Options{})
	require.Error(t, err)
	var apiErr *api.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "vm not running", apiErr.Message)
}
