package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
)

func runLB(t *testing.T, setup func(m *mocks.MockClientAPIMockRecorder), args ...string) (string, error) {
	t.Helper()
	isolateCreds(t)
	rt, mock := newRuntime(t, &config.Config{})
	if setup != nil {
		setup(mock.EXPECT())
	}
	out := &bytes.Buffer{}
	rt.Streams.Out = out
	root := NewRoot(rt)
	root.SetArgs(append([]string{"--token", "tok", "--ws", "acme"}, args...))
	root.SetOut(out)
	root.SetErr(out)
	err := root.Execute()
	return out.String(), err
}

func lbFixture() api.LoadBalancer {
	return api.LoadBalancer{UID: "lb-1", Slug: "web", Status: "ready"}
}

func TestLBUpdate_sendsOnlyChangedFlags(t *testing.T) {
	_, err := runLB(t, func(m *mocks.MockClientAPIMockRecorder) {
		sticky := false
		gomock.InOrder(
			m.ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{lbFixture()}, nil),
			m.UpdateLoadBalancer(gomock.Any(), "acme", "lb-1", &api.UpdateLoadBalancerRequest{
				StickySessions: &sticky,
				AllowedCIDRs:   &[]string{},
			}, gomock.Any()).Return(api.LoadBalancerOperationResult{IsCompleted: true}, nil),
			m.GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(lbFixture(), nil),
		)
	}, "lb", "update", "web", "--sticky=false", "--allow-all")
	require.NoError(t, err)
}

func TestLBUpdate_usageErrors(t *testing.T) {
	cases := map[string][]string{
		"nothing to update":          {"lb", "update", "web"},
		"--vm needs at least one VM": {"lb", "update", "web", "--vm", ""},
		"--allow needs a CIDR":       {"lb", "update", "web", "--allow", " "},
		"grpc load balancers use tcp": {
			"lb", "update", "web", "--protocol", "grpc", "--health-protocol", "http",
		},
	}
	for want, args := range cases {
		t.Run(want, func(t *testing.T) {
			_, err := runLB(t, nil, args...)
			require.Error(t, err)
			assert.True(t, config.IsUsageError(err), err)
			assert.Contains(t, err.Error(), want)
		})
	}
}

func TestLBCreate_usageErrors(t *testing.T) {
	cases := map[string][]string{
		"--vm is required":             {"load-balancer", "create", "web", "--port", "8080"},
		"--port is required":           {"lb", "create", "web", "--vm", "web-1"},
		"missing load balancer name":   {"lb", "create"},
		`invalid CIDR "1.2.3.4"`:       {"lb", "create", "web", "--vm", "web-1", "-p", "80", "--allow", "1.2.3.4"},
		"health check path must start": {"lb", "create", "web", "--vm", "web-1", "-p", "80", "--health-path", "x"},
		"--allow needs a CIDR":         {"lb", "create", "web", "--vm", "web-1", "-p", "80", "--allow", ""},
	}
	for want, args := range cases {
		t.Run(want, func(t *testing.T) {
			_, err := runLB(t, nil, args...)
			require.Error(t, err)
			assert.True(t, config.IsUsageError(err), err)
			assert.Contains(t, err.Error(), want)
		})
	}
}

func TestLBCreate_helpMentionsPriceAndGRPCHealthChecks(t *testing.T) {
	out, err := runLB(t, nil, "lb", "create", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "$2/month")
	assert.Contains(t, out, "gRPC load balancers use TCP")
}

func TestLBRemoveTarget_needsTwoArgs(t *testing.T) {
	_, err := runLB(t, nil, "lb", "remove-target", "web")
	require.Error(t, err)
	assert.True(t, config.IsUsageError(err))
}
