package firewall_test

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
	"github.com/cloverworks-hq/rumpty-cli/internal/firewall"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newFirewallRuntime(t *testing.T, mock *mocks.MockClientAPI) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}, out
}

func testPolicy() api.FirewallPolicy {
	return api.FirewallPolicy{UID: "fw-1", Name: "web", Slug: "web", IsDefault: false}
}

func TestCreate_reportsTheNewPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().CreateFirewallPolicy(gomock.Any(), "prod-team", api.CreateFirewallPolicyRequest{
		Name:        "web",
		Description: "public sites",
	}).Return(api.FirewallPolicy{UID: "fw-1", Name: "web", Slug: "web"}, nil)

	rt, out := newFirewallRuntime(t, mock)

	require.NoError(t, firewall.Create(context.Background(), rt, "web", "public sites"))
	assert.Contains(t, out.String(), "Created firewall policy web")
}

func TestRm_withYesDeletesAndReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListFirewallPolicies(gomock.Any(), "prod-team").Return([]api.FirewallPolicy{testPolicy()}, nil)
	mock.EXPECT().DeleteFirewallPolicy(gomock.Any(), "prod-team", "fw-1").Return(nil)

	rt, out := newFirewallRuntime(t, mock)

	require.NoError(t, firewall.Rm(context.Background(), rt, "web", true))
	assert.Contains(t, out.String(), "Deleted firewall policy web")
}

func TestRm_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListFirewallPolicies(gomock.Any(), "prod-team").Return([]api.FirewallPolicy{}, nil)

	rt, _ := newFirewallRuntime(t, mock)

	err := firewall.Rm(context.Background(), rt, "missing", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no firewall policy")
}

func TestRm_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListFirewallPolicies(gomock.Any(), "prod-team").Return([]api.FirewallPolicy{testPolicy()}, nil)

	rt, _ := newFirewallRuntime(t, mock)

	err := firewall.Rm(context.Background(), rt, "web", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}
