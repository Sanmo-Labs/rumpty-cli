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

func testPolicyWithRules() api.FirewallPolicy {
	p := testPolicy()
	p.Rules = []api.FirewallRule{
		{UID: "r-1", Direction: "inbound", Action: "allow", Protocol: "tcp", PortStart: 80, PortEnd: 80, CIDR: "0.0.0.0/0", IsDefault: true},
		{UID: "r-2", Direction: "inbound", Action: "allow", Protocol: "tcp", PortStart: 22, PortEnd: 22, CIDR: "0.0.0.0/0", IsDefault: true},
	}
	return p
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

func TestAddRule_addsAndApplies(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListFirewallPolicies(gomock.Any(), "prod-team").Return([]api.FirewallPolicy{testPolicy()}, nil)
	mock.EXPECT().AddFirewallRule(gomock.Any(), "prod-team", "fw-1", &api.AddFirewallRuleRequest{
		Direction: "inbound",
		Action:    "allow",
		Protocol:  "tcp",
		PortStart: 443,
		PortEnd:   443,
		CIDR:      "0.0.0.0/0",
	}).Return(api.FirewallRuleResult{
		FirewallRule: api.FirewallRule{UID: "r-9", Direction: "inbound", Action: "allow", Protocol: "tcp", PortStart: 443, PortEnd: 443, CIDR: "0.0.0.0/0"},
		OperationID:  "op-7",
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-7").Return(api.Operation{UID: "op-7", Status: "succeeded"}, nil)

	rt, out := newFirewallRuntime(t, mock)

	require.NoError(t, firewall.Add(context.Background(), rt, "web", &firewall.RuleOpts{Port: "443", Source: "0.0.0.0/0"}))
	assert.Contains(t, out.String(), "Added rule tcp/443 from 0.0.0.0/0 to web")
}

func TestAddRule_requiresSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	rt, _ := newFirewallRuntime(t, mock)

	err := firewall.Add(context.Background(), rt, "web", &firewall.RuleOpts{Port: "80"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--source is required")
}

func TestRmRule_byCompactRef(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListFirewallPolicies(gomock.Any(), "prod-team").Return([]api.FirewallPolicy{testPolicyWithRules()}, nil)
	mock.EXPECT().DeleteFirewallRule(gomock.Any(), "prod-team", "fw-1", "r-2").Return(nil)

	rt, _ := newFirewallRuntime(t, mock)

	require.NoError(t, firewall.RmRule(context.Background(), rt, "web", "tcp/22", true))
}

func TestRmRule_unknownRefListsRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListFirewallPolicies(gomock.Any(), "prod-team").Return([]api.FirewallPolicy{testPolicyWithRules()}, nil)

	rt, _ := newFirewallRuntime(t, mock)

	err := firewall.RmRule(context.Background(), rt, "web", "udp/99", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no rule")
	assert.Contains(t, err.Error(), "tcp/22")
}
