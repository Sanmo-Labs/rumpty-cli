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

func newInvitationsRuntime(t *testing.T, mock *mocks.MockClientAPI, cfgWorkspace string) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: cfgWorkspace},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testInvitation() api.WorkspaceInvitation {
	return api.WorkspaceInvitation{
		UID:       "inv-1",
		Email:     "ada@example.com",
		Role:      "developer",
		Status:    "pending",
		ExpiresAt: "2026-11-21T10:24:00Z",
	}
}

func TestListInvitations_printsPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{UID: "ws-1", Name: "Production", Slug: "prod-team"},
	}, nil)
	mock.EXPECT().ListInvitations(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceInvitation{testInvitation()}, nil)

	rt, out := newInvitationsRuntime(t, mock, "")

	require.NoError(t, workspace.ListInvitations(context.Background(), rt, "prod-team", term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "ada@example.com")
	assert.Contains(t, s, "developer")
}

func TestListInvitations_empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{UID: "ws-1", Name: "Production", Slug: "prod-team"},
	}, nil)
	mock.EXPECT().ListInvitations(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceInvitation{}, nil)

	rt, out := newInvitationsRuntime(t, mock, "")

	require.NoError(t, workspace.ListInvitations(context.Background(), rt, "prod-team", term.OutputTable))
	assert.Contains(t, out.String(), "No pending invitations.")
}

func TestRevokeInvitation_byEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{UID: "ws-1", Name: "Production", Slug: "prod-team"},
	}, nil)
	mock.EXPECT().ListInvitations(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceInvitation{testInvitation()}, nil)
	mock.EXPECT().RevokeInvitation(gomock.Any(), "ws-1", "prod-team", "inv-1").Return(api.WorkspaceInvitation{
		UID: "inv-1", Email: "ada@example.com", Status: "revoked",
	}, nil)

	rt, out := newInvitationsRuntime(t, mock, "")

	require.NoError(t, workspace.RevokeInvitation(context.Background(), rt, "prod-team", "ada@example.com"))
	assert.Contains(t, out.String(), "Revoked invitation for ada@example.com")
}

func TestRevokeInvitation_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListWorkspaces(gomock.Any()).Return([]api.Workspace{
		{UID: "ws-1", Name: "Production", Slug: "prod-team"},
	}, nil)
	mock.EXPECT().ListInvitations(gomock.Any(), "ws-1", "prod-team").Return([]api.WorkspaceInvitation{}, nil)

	rt, _ := newInvitationsRuntime(t, mock, "")

	err := workspace.RevokeInvitation(context.Background(), rt, "prod-team", "missing@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending invitation")
}
