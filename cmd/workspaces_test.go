package commands

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// The workspace is an argument on these commands, so they must work with a
// token but no configured default workspace.
func TestWSCommands_workWithoutDefaultWorkspace(t *testing.T) {
	t.Run("invite", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mock := mocks.NewMockClientAPI(ctrl)
		mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
			{UID: "ws-1", Name: "Production", Slug: "prod-team"},
		}, nil)
		mock.EXPECT().InviteWorkspaceMember(gomock.Any(), "ws-1", "prod-team", api.InviteWorkspaceMemberRequest{
			Email: "ada@example.com",
			Role:  "developer",
		}).Return(api.WorkspaceInvitation{Email: "ada@example.com", Role: "developer", ExpiresAt: "2026-11-25T10:00:00Z"}, nil)

		cmd := newWorkspacesCmd(newWSTestRuntime(t, mock))
		cmd.SetArgs([]string{"invite", "prod-team", "ada@example.com"})
		require.NoError(t, cmd.Execute())
	})

	t.Run("remove", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mock := mocks.NewMockClientAPI(ctrl)
		mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
			{UID: "ws-1", Name: "Production", Slug: "prod-team"},
		}, nil)
		mock.EXPECT().ListWorkspaceMembers(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceMember{
			{UID: "m-1", Username: "ada", Email: "ada@example.com"},
		}, nil)
		mock.EXPECT().RemoveWorkspaceMember(gomock.Any(), "ws-1", "prod-team", "m-1").Return(api.WorkspaceMember{}, nil)

		cmd := newWorkspacesCmd(newWSTestRuntime(t, mock))
		cmd.SetArgs([]string{"remove", "prod-team", "ada@example.com", "--yes"})
		require.NoError(t, cmd.Execute())
	})

	t.Run("role", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mock := mocks.NewMockClientAPI(ctrl)
		mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
			{UID: "ws-1", Name: "Production", Slug: "prod-team"},
		}, nil)
		mock.EXPECT().ListWorkspaceMembers(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceMember{
			{UID: "m-1", Username: "ada", Email: "ada@example.com"},
		}, nil)
		mock.EXPECT().UpdateWorkspaceMemberRole(gomock.Any(), "ws-1", "prod-team", "m-1", "viewer").Return(api.WorkspaceMember{}, nil)

		cmd := newWorkspacesCmd(newWSTestRuntime(t, mock))
		cmd.SetArgs([]string{"role", "prod-team", "ada@example.com", "viewer"})
		require.NoError(t, cmd.Execute())
	})
}

func newWSTestRuntime(t *testing.T, mock *mocks.MockClientAPI) *app.Runtime {
	t.Helper()
	return &app.Runtime{
		Config:    &config.Config{Token: "tok"},
		Streams:   term.Streams{Out: &bytes.Buffer{}, ErrOut: io.Discard},
		APIClient: mock,
	}
}
