package sshkeys_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/sshkeys"
)

func TestRm_withYesRemovesAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListSSHKeys(gomock.Any(), "prod-team").Return([]api.SSHKey{
		{UID: "key-1", Name: "laptop", Fingerprint: "SHA256:abc"},
		{UID: "key-2", Name: "server", Fingerprint: "SHA256:def"},
	}, nil)
	mock.EXPECT().DeleteSSHKey(gomock.Any(), "prod-team", "key-1").Return(nil)

	rt, out := newAddRuntime(t, mock)

	require.NoError(t, sshkeys.Rm(context.Background(), rt, "laptop", true))
	assert.Contains(t, out.String(), "Removed laptop (SHA256:abc)")
}

func TestRm_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListSSHKeys(gomock.Any(), "prod-team").Return([]api.SSHKey{}, nil)

	rt, _ := newAddRuntime(t, mock)

	err := sshkeys.Rm(context.Background(), rt, "missing", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no ssh key")
}

func TestRm_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListSSHKeys(gomock.Any(), "prod-team").Return([]api.SSHKey{
		{UID: "key-1", Name: "laptop", Fingerprint: "SHA256:abc"},
	}, nil)

	rt, _ := newAddRuntime(t, mock)

	err := sshkeys.Rm(context.Background(), rt, "laptop", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}

func TestRm_matchesFingerprint(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListSSHKeys(gomock.Any(), "prod-team").Return([]api.SSHKey{
		{UID: "key-1", Name: "laptop", Fingerprint: "SHA256:abc"},
	}, nil)
	mock.EXPECT().DeleteSSHKey(gomock.Any(), "prod-team", "key-1").Return(nil)

	rt, _ := newAddRuntime(t, mock)

	require.NoError(t, sshkeys.Rm(context.Background(), rt, "SHA256:abc", true))
}
