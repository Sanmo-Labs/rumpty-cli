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

func newWSRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testWorkspace() api.Workspace {
	return api.Workspace{UID: "ws-1", Name: "Prod Team", Slug: "prod-team"}
}

func TestInvite_reportsTheInvitation(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{testWorkspace()}, nil)
	mock.EXPECT().InviteWorkspaceMember(gomock.Any(), "ws-1", "prod-team", api.InviteWorkspaceMemberRequest{
		Email: "ada@example.com",
		Role:  "developer",
	}).Return(api.WorkspaceInvitation{UID: "inv-1", Email: "ada@example.com", Role: "developer", ExpiresAt: "2026-11-26T03:00:00Z"}, nil)

	rt, out := newWSRuntime(t, mock)

	require.NoError(t, workspace.Invite(context.Background(), rt, "prod-team", "ada@example.com", "developer"))
	assert.Contains(t, out.String(), "Invited ada@example.com as developer")
}

func TestInvite_rejectsUnknownRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	rt, _ := newWSRuntime(t, mock)

	err := workspace.Invite(context.Background(), rt, "prod-team", "ada@example.com", "root")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin, developer or viewer")
}
