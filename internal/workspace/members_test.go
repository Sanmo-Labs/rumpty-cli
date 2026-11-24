package workspace_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
	"github.com/cloverworks-hq/rumpty-cli/internal/workspace"
)

func newMembersRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func membersTestWorkspace() api.Workspace {
	return api.Workspace{UID: "ws-1", Name: "Prod Team", Slug: "prod-team"}
}

func TestSetRole_reportsTheChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{membersTestWorkspace()}, nil)
	mock.EXPECT().ListWorkspaceMembers(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceMember{
		{UID: "mem-1", UserUID: "u-1", Username: "ada", Email: "ada@example.com", Role: "viewer"},
	}, nil)
	mock.EXPECT().UpdateWorkspaceMemberRole(gomock.Any(), "ws-1", "prod-team", "mem-1", "developer").Return(api.WorkspaceMember{}, nil)

	rt, out := newMembersRuntime(t, mock)

	require.NoError(t, workspace.SetRole(context.Background(), rt, "prod-team", "ada", "developer"))
	assert.Contains(t, out.String(), "ada is now developer")
}

func TestSetRole_rejectsOwnerChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{membersTestWorkspace()}, nil)
	mock.EXPECT().ListWorkspaceMembers(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceMember{
		{UID: "mem-1", Username: "boss", Role: "owner"},
	}, nil)

	rt, _ := newMembersRuntime(t, mock)

	err := workspace.SetRole(context.Background(), rt, "prod-team", "boss", "viewer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner's role cannot be changed")
}

func TestRmMember_withYesRemovesAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{membersTestWorkspace()}, nil)
	mock.EXPECT().ListWorkspaceMembers(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceMember{
		{UID: "mem-1", Username: "ada", Role: "developer"},
	}, nil)
	mock.EXPECT().RemoveWorkspaceMember(gomock.Any(), "ws-1", "prod-team", "mem-1").Return(api.WorkspaceMember{}, nil)

	rt, out := newMembersRuntime(t, mock)

	require.NoError(t, workspace.RmMember(context.Background(), rt, "prod-team", "ada", true))
	assert.Contains(t, out.String(), "Removed ada")
}

func TestRmMember_refusesOwnerAndSelf(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	owner := api.WorkspaceMember{UID: "mem-1", Username: "boss", Role: "owner"}
	self := api.WorkspaceMember{UID: "mem-2", Username: "me", Role: "admin", IsCurrentUser: true}
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{membersTestWorkspace()}, nil).Times(2)
	mock.EXPECT().ListWorkspaceMembers(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceMember{owner, self}, nil).Times(2)
	rt, _ := newMembersRuntime(t, mock)

	err := workspace.RmMember(context.Background(), rt, "prod-team", "boss", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner cannot be removed")

	err = workspace.RmMember(context.Background(), rt, "prod-team", "me", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove yourself")
}

func TestRmMember_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{membersTestWorkspace()}, nil)
	mock.EXPECT().ListWorkspaceMembers(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceMember{
		{UID: "mem-1", Username: "ada", Role: "developer"},
	}, nil)
	rt, _ := newMembersRuntime(t, mock)

	err := workspace.RmMember(context.Background(), rt, "prod-team", "ada", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}
