package deploy_test

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
	"github.com/cloverworks-hq/rumpty-cli/internal/deploy"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newDeployRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testDeployment() api.Deployment {
	return api.Deployment{
		UID: "dep-1", Name: "cryptotax", Slug: "cryptotax",
		Status: "active", DisplayStatus: "updating",
		PlanSlug: "launch", Branch: "main",
		PublicURL: "https://cryptotax.rumptycloud.app",
	}
}

func TestLs_lists(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListDeployments(gomock.Any(), "prod-team").Return([]api.Deployment{testDeployment()}, nil)

	rt, out := newDeployRuntime(t, mock)

	require.NoError(t, deploy.Ls(context.Background(), rt, term.OutputTable))
	s := out.String()
	assert.Contains(t, s, "cryptotax")
	assert.Contains(t, s, "updating")
	assert.Contains(t, s, "launch")
	assert.Contains(t, s, "main")
	assert.Contains(t, s, "https://cryptotax.rumptycloud.app")
}

func TestLs_fallsBackToStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	d := testDeployment()
	d.DisplayStatus = ""
	mock.EXPECT().ListDeployments(gomock.Any(), "prod-team").Return([]api.Deployment{d}, nil)

	rt, out := newDeployRuntime(t, mock)

	require.NoError(t, deploy.Ls(context.Background(), rt, term.OutputTable))
	assert.Contains(t, out.String(), "active")
}

func TestLs_empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListDeployments(gomock.Any(), "prod-team").Return(nil, nil)

	rt, out := newDeployRuntime(t, mock)

	require.NoError(t, deploy.Ls(context.Background(), rt, term.OutputTable))
	assert.Contains(t, out.String(), "No deployments yet")
}

func TestLs_json(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListDeployments(gomock.Any(), "prod-team").Return([]api.Deployment{testDeployment()}, nil)

	rt, out := newDeployRuntime(t, mock)

	require.NoError(t, deploy.Ls(context.Background(), rt, term.OutputJSON))
	assert.Contains(t, out.String(), "\"uid\": \"dep-1\"")
	assert.Contains(t, out.String(), "\"display_status\": \"updating\"")
}

func TestLs_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListDeployments(gomock.Any(), "prod-team").Return(nil, assert.AnError)

	rt, _ := newDeployRuntime(t, mock)

	require.ErrorIs(t, deploy.Ls(context.Background(), rt, term.OutputTable), assert.AnError)
}
