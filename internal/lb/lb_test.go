package lb_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/lb"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

var (
	web1 = api.VM{UID: "vm-1", Slug: "web-1", Name: "web-1"}
	web2 = api.VM{UID: "vm-2", Slug: "web-2", Name: "Web Two"}
	web3 = api.VM{UID: "vm-3", Slug: "web-3", Name: "web-3"}
)

func readyLB() api.LoadBalancer {
	return api.LoadBalancer{
		UID: "lb-1", Name: "web", Slug: "web", Status: "ready",
		URL: "https://web-5d6e7f80.rumptycloud.app", TargetPort: 8080, TargetProtocol: "http",
		Targets: []api.LoadBalancerTarget{
			{VMUID: "vm-1", VMName: "web-1", VMStatus: "running"},
			{VMUID: "vm-2", VMName: "Web Two", VMStatus: "running"},
		},
		HealthCheck: api.LoadBalancerHealthCheck{
			Protocol: "http", Path: "/healthz", Port: 8080, IntervalSeconds: 10,
			TimeoutSeconds: 5, HealthyThreshold: 3, UnhealthyThreshold: 3,
		},
		StickySessions: api.LoadBalancerStickySessions{CookieName: "rumpty-lb", TTLSeconds: 300},
	}
}

func newRuntime(t *testing.T) (*app.Runtime, *mocks.MockClientAPI, *bytes.Buffer) {
	t.Helper()
	mock := mocks.NewMockClientAPI(gomock.NewController(t))
	out := &bytes.Buffer{}
	rt := &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "acme"},
		Streams:   term.Streams{In: strings.NewReader(""), Out: out, ErrOut: io.Discard},
		APIClient: mock,
	}
	return rt, mock, out
}

func intp(v int) *int       { return &v }
func strp(v string) *string { return &v }
func boolp(v bool) *bool    { return &v }

func TestCreate_resolvesVMsWaitsAndPrintsURL(t *testing.T) {
	rt, mock, out := newRuntime(t)
	pending := readyLB()
	pending.Status = "pending"

	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1, web2, web3}, nil),
		mock.EXPECT().CreateLoadBalancer(gomock.Any(), "acme", &api.CreateLoadBalancerRequest{
			Name:           "web",
			TargetPort:     8080,
			VMUIDs:         []string{"vm-1", "vm-2"},
			HealthCheck:    &api.LoadBalancerHealthCheckInput{Path: strp("/healthz")},
			StickySessions: boolp(true),
			AllowedCIDRs:   []string{"203.0.113.0/24"},
		}, gomock.Any()).Return(api.LoadBalancerOperationResult{OperationID: "op-1", LoadBalancer: &pending}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-1").Return(api.Operation{Status: "succeeded"}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
	)

	err := lb.Create(context.Background(), rt, "web", &lb.Settings{
		VMs:          []string{"web-1", "Web Two", "vm-1"},
		Port:         intp(8080),
		HealthPath:   strp("/healthz"),
		Sticky:       boolp(true),
		AllowedCIDRs: &[]string{"203.0.113.0/24"},
	})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Created load balancer web")
	assert.Contains(t, out.String(), "URL: https://web-5d6e7f80.rumptycloud.app")
	assert.Contains(t, out.String(), "Targets: web-1, Web Two")
	assert.Contains(t, out.String(), "$2/month")
}

func TestCreate_failureShowsLastFailure(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	pending := readyLB()
	failed := readyLB()
	failed.Status = "failed"
	failed.LastFailure = "route apply failed"

	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1}, nil),
		mock.EXPECT().CreateLoadBalancer(gomock.Any(), "acme", gomock.Any(), gomock.Any()).
			Return(api.LoadBalancerOperationResult{OperationID: "op-1", LoadBalancer: &pending}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-1").Return(api.Operation{Status: "failed"}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(failed, nil),
	)

	err := lb.Create(context.Background(), rt, "web", &lb.Settings{VMs: []string{"web-1"}, Port: intp(8080)})
	require.ErrorContains(t, err, "create failed for web: route apply failed")
	require.ErrorContains(t, err, "still exists and is billed")
	require.ErrorContains(t, err, "rumpty lb delete web")
}

func TestCreate_tooManyDistinctVMs(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	vms := make([]api.VM, 11)
	refs := make([]string, 11)
	for i := range vms {
		vms[i] = api.VM{UID: fmt.Sprintf("vm-%d", i), Slug: fmt.Sprintf("web-%d", i)}
		refs[i] = vms[i].Slug
	}
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return(vms, nil)

	err := lb.Create(context.Background(), rt, "web", &lb.Settings{VMs: refs, Port: intp(8080)})
	require.ErrorContains(t, err, "at most 10 VMs")
}

func TestCreate_duplicateRefsCountOnce(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	ready := readyLB()
	refs := make([]string, 11)
	for i := range refs {
		refs[i] = "web-1"
	}
	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1}, nil),
		mock.EXPECT().CreateLoadBalancer(gomock.Any(), "acme", gomock.Any(), gomock.Any()).
			Return(api.LoadBalancerOperationResult{IsCompleted: true, LoadBalancer: &ready}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(ready, nil),
	)

	require.NoError(t, lb.Create(context.Background(), rt, "web", &lb.Settings{VMs: refs, Port: intp(8080)}))
}

func TestCreate_unknownVM(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1}, nil)

	err := lb.Create(context.Background(), rt, "web", &lb.Settings{VMs: []string{"nope"}, Port: intp(8080)})
	require.ErrorContains(t, err, `vm "nope" not found`)
}

func TestCreate_replayCompletedSkipsWait(t *testing.T) {
	rt, mock, out := newRuntime(t)
	ready := readyLB()

	gomock.InOrder(
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1}, nil),
		mock.EXPECT().CreateLoadBalancer(gomock.Any(), "acme", gomock.Any(), gomock.Any()).
			Return(api.LoadBalancerOperationResult{OperationID: "op-1", IsReplay: true, IsCompleted: true, LoadBalancer: &ready}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(ready, nil),
	)

	require.NoError(t, lb.Create(context.Background(), rt, "web", &lb.Settings{VMs: []string{"web-1"}, Port: intp(8080)}))
	assert.Contains(t, out.String(), "Created load balancer web")
}

func TestSettingsValidate(t *testing.T) {
	cases := map[string]struct {
		s    lb.Settings
		want string
	}{
		"port":           {lb.Settings{Port: intp(70000)}, "port must be between"},
		"protocol":       {lb.Settings{Protocol: strp("tcp")}, "protocol must be http or grpc"},
		"grpc http":      {lb.Settings{Protocol: strp("grpc"), HealthProtocol: strp("http")}, "grpc load balancers use tcp"},
		"path":           {lb.Settings{HealthPath: strp("healthz")}, "must start with /"},
		"timeout":        {lb.Settings{HealthInterval: intp(5), HealthTimeout: intp(6)}, "must not exceed"},
		"threshold":      {lb.Settings{HealthyThreshold: intp(11)}, "between 1 and 10"},
		"cidr":           {lb.Settings{AllowedCIDRs: &[]string{"10.0.0.1"}}, `invalid CIDR "10.0.0.1"`},
		"path with tcp":  {lb.Settings{HealthProtocol: strp("tcp"), HealthPath: strp("/x")}, "only applies to http"},
		"interval range": {lb.Settings{HealthInterval: intp(301)}, "between 1 and 300"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, tc.s.Validate(), tc.want)
		})
	}
	ok := lb.Settings{Protocol: strp("grpc"), HealthProtocol: strp("tcp"), AllowedCIDRs: &[]string{"2001:db8::/32"}}
	require.NoError(t, ok.Validate())
}

func TestUpdate_sendsOnlyChangedFields(t *testing.T) {
	rt, mock, out := newRuntime(t)
	current := readyLB()

	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{current}, nil),
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1, web3}, nil),
		mock.EXPECT().UpdateLoadBalancer(gomock.Any(), "acme", "lb-1", &api.UpdateLoadBalancerRequest{
			VMUIDs:       &[]string{"vm-3"},
			HealthCheck:  &api.LoadBalancerHealthCheckInput{IntervalSeconds: intp(5)},
			AllowedCIDRs: &[]string{},
		}, gomock.Any()).Return(api.LoadBalancerOperationResult{OperationID: "op-2"}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-2").Return(api.Operation{Status: "succeeded"}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(current, nil),
	)

	err := lb.Update(context.Background(), rt, "lb-1", &lb.Settings{
		VMs:            []string{"web-3"},
		HealthInterval: intp(5),
		AllowedCIDRs:   &[]string{},
	})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Updated load balancer web")
}

func TestUpdate_nothingToUpdate(t *testing.T) {
	rt, _, _ := newRuntime(t)
	require.ErrorContains(t, lb.Update(context.Background(), rt, "web", &lb.Settings{}), "nothing to update")
}

func TestDelete_withYes(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().DeleteLoadBalancer(gomock.Any(), "acme", "lb-1", gomock.Any()).
			Return(api.LoadBalancerOperationResult{OperationID: "op-3"}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-3").Return(api.Operation{Status: "succeeded"}, nil),
	)

	require.NoError(t, lb.Delete(context.Background(), rt, "web", true))
	assert.Contains(t, out.String(), "Deleted load balancer web")
}

func TestDelete_promptAccepted(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	rt.Streams.In = strings.NewReader("y\n")
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().DeleteLoadBalancer(gomock.Any(), "acme", "lb-1", gomock.Any()).
			Return(api.LoadBalancerOperationResult{OperationID: "op-3", IsCompleted: true}, nil),
	)

	require.NoError(t, lb.Delete(context.Background(), rt, "web", false))
}

func TestDelete_promptDeclined(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	rt.Streams.In = strings.NewReader("n\n")
	mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil)

	require.ErrorIs(t, lb.Delete(context.Background(), rt, "web", false), lb.ErrNotConfirmed)
}

func TestDelete_nonTerminalStdinNeedsYes(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	rt.Streams.In = r
	mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil)

	err = lb.Delete(context.Background(), rt, "web", false)
	require.ErrorContains(t, err, "pass --yes")
	assert.True(t, config.IsUsageError(err))
}

func TestDelete_notFound(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{}, nil)

	require.ErrorContains(t, lb.Delete(context.Background(), rt, "web", true), `load balancer "web" not found in workspace acme`)
}

func TestAddTargets(t *testing.T) {
	rt, mock, out := newRuntime(t)
	after := readyLB()
	after.Targets = append(after.Targets, api.LoadBalancerTarget{VMUID: "vm-3", VMName: "web-3"})

	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1, web2, web3}, nil),
		mock.EXPECT().AddLoadBalancerTargets(gomock.Any(), "acme", "lb-1", []string{"vm-3"}, gomock.Any()).
			Return(api.LoadBalancerOperationResult{OperationID: "op-4"}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-4").Return(api.Operation{Status: "succeeded"}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(after, nil),
	)

	require.NoError(t, lb.AddTargets(context.Background(), rt, "web", []string{"web-3"}))
	assert.Contains(t, out.String(), "Added web-3 to web")
	assert.Contains(t, out.String(), "Targets: web-1, Web Two, web-3")
}

func TestRemoveTarget_bySlugWhenTargetShowsName(t *testing.T) {
	rt, mock, out := newRuntime(t)

	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1, web2}, nil),
		mock.EXPECT().RemoveLoadBalancerTarget(gomock.Any(), "acme", "lb-1", "vm-2", gomock.Any()).
			Return(api.LoadBalancerOperationResult{OperationID: "op-5"}, nil),
		mock.EXPECT().GetOperation(gomock.Any(), "acme", "op-5").Return(api.Operation{Status: "succeeded"}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
	)

	require.NoError(t, lb.RemoveTarget(context.Background(), rt, "web", "web-2"))
	assert.Contains(t, out.String(), "Removed web-2 from web")
}

func TestRemoveTarget_notATarget(t *testing.T) {
	rt, mock, _ := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().ListVMs(gomock.Any(), "acme").Return([]api.VM{web1, web2, web3}, nil),
	)

	require.ErrorContains(t, lb.RemoveTarget(context.Background(), rt, "web", "web-3"), "vm web-3 is not a target of web")
}

func TestList_table(t *testing.T) {
	rt, mock, out := newRuntime(t)
	mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil)

	require.NoError(t, lb.List(context.Background(), rt, term.OutputTable))
	for _, want := range []string{"NAME", "TARGETS", "web", "ready", "HTTP", "8080", "web-1, Web Two", "https://web-5d6e7f80.rumptycloud.app"} {
		assert.Contains(t, out.String(), want)
	}
}

func TestList_emptyAndJSON(t *testing.T) {
	rt, mock, out := newRuntime(t)
	mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{}, nil).Times(2)

	require.NoError(t, lb.List(context.Background(), rt, term.OutputTable))
	assert.Contains(t, out.String(), "No load balancers in acme.")

	out.Reset()
	require.NoError(t, lb.List(context.Background(), rt, term.OutputJSON))
	assert.Equal(t, "[]\n", out.String())
}

func TestGet_detailsAndJSON(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
	)

	require.NoError(t, lb.Get(context.Background(), rt, "web", term.OutputTable))
	for _, want := range []string{
		"HTTP on port 8080",
		"HTTP /healthz on port 8080 every 10s, timeout 5s, healthy after 3, unhealthy after 3",
		"Sticky sessions: off",
		"Allowed IPs:     everyone",
		"TARGET", "Web Two", "vm-2",
	} {
		assert.Contains(t, out.String(), want)
	}

	out.Reset()
	require.NoError(t, lb.Get(context.Background(), rt, "web", term.OutputJSON))
	assert.Contains(t, out.String(), `"uid": "lb-1"`)
	assert.Contains(t, out.String(), `"target_port": 8080`)
}
