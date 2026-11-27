package lb_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/lb"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func int64p(v int64) *int64       { return &v }
func float64p(v float64) *float64 { return &v }

func liveTraffic() api.LoadBalancerTraffic {
	return api.LoadBalancerTraffic{
		LoadBalancerUID: "lb-1", Available: true, Range: "1h", StepSeconds: 240,
		RequestsTotal: 1200, ErrorsTotal: 6, ClientErrorsTotal: int64p(31), ErrorRate: 0.005,
		AvgLatencyMs: float64p(18.4), RXBytes: 2048, TXBytes: 5 << 20,
		Points: []api.LoadBalancerTrafficPoint{{Requests: 10}, {Requests: 80}, {Requests: 40}},
	}
}

func TestTraffic_printsHealthAndTotals(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", "1h").Return(liveTraffic(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{
			Status: "healthy", HealthyTargets: 2, TotalTargets: 2,
		}, nil),
	)

	require.NoError(t, lb.Traffic(context.Background(), rt, "web", "1h", term.OutputTable))
	for _, want := range []string{
		"Health:          healthy (2/2 targets)\n",
		"Traffic (1h)",
		"▂█▅",
		"Requests:      1.2k (20/min)",
		"5xx errors:    6 (0.50%)",
		"4xx:           31",
		"Avg response:  18 ms",
		"Bandwidth:     2.0 KB in, 5.0 MB out",
	} {
		assert.Contains(t, out.String(), want)
	}
}

func TestTraffic_unknownFiguresShowAsDash(t *testing.T) {
	rt, mock, out := newRuntime(t)
	traffic := liveTraffic()
	traffic.ClientErrorsTotal, traffic.AvgLatencyMs = nil, nil
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", "1h").Return(traffic, nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{
			Status: "unknown", UnavailableReason: "no health data yet",
		}, nil),
	)

	require.NoError(t, lb.Traffic(context.Background(), rt, "web", "1h", term.OutputTable))
	assert.Contains(t, out.String(), "Health:          unknown (no health data yet)")
	assert.Contains(t, out.String(), "4xx:           —")
	assert.Contains(t, out.String(), "Avg response:  —")
}

func TestTraffic_downAndUnavailable(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", "7d").Return(api.LoadBalancerTraffic{
			Available: false, UnavailableReason: "traffic metrics are temporarily unavailable",
		}, nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{Status: "down", TotalTargets: 2}, nil),
	)

	require.NoError(t, lb.Traffic(context.Background(), rt, "web", "7d", term.OutputTable))
	assert.Contains(t, out.String(), "down (0/2 targets healthy; visitors get a 503 page)")
	assert.Contains(t, out.String(), "Traffic (7d): traffic metrics are temporarily unavailable")
}

func TestTraffic_json(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", "24h").Return(liveTraffic(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{Status: "healthy", HealthyTargets: 2, TotalTargets: 2}, nil),
	)

	require.NoError(t, lb.Traffic(context.Background(), rt, "web", "24h", term.OutputJSON))
	var got lb.TrafficReport
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "healthy", got.Health.Status)
	assert.Equal(t, int64(1200), got.Traffic.RequestsTotal)
	assert.Equal(t, int64(31), *got.Traffic.ClientErrorsTotal)
}

func TestTraffic_rejectsUnknownRange(t *testing.T) {
	rt, _, _ := newRuntime(t)
	err := lb.Traffic(context.Background(), rt, "web", "30d", term.OutputTable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--range must be one of 1h, 6h, 24h, 7d")
}

func TestGet_healthErrorDoesNotFailGet(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{}, errors.New("503")),
	)

	require.NoError(t, lb.Get(context.Background(), rt, "web", term.OutputTable))
	assert.Contains(t, out.String(), "Health:          unknown (could not load health)")
	assert.Contains(t, out.String(), "Web Two")
}

func TestTraffic_healthFailureStillShowsTraffic(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", "1h").Return(liveTraffic(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{}, errors.New("timeout")),
	)

	require.NoError(t, lb.Traffic(context.Background(), rt, "web", "1h", term.OutputTable))
	assert.Contains(t, out.String(), "Health:          unknown (could not load health)")
	assert.Contains(t, out.String(), "Requests:      1.2k")
}

func TestTraffic_unrecognisedStatusIsShownAsIs(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", "1h").Return(liveTraffic(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{Status: "draining"}, nil),
	)

	require.NoError(t, lb.Traffic(context.Background(), rt, "web", "1h", term.OutputTable))
	assert.Contains(t, out.String(), "Health:          draining")
}

func TestTraffic_numberFormatting(t *testing.T) {
	cases := []struct {
		requests int64
		window   string
		latency  float64
		want     []string
	}{
		{requests: 400, window: "7d", latency: 999.6, want: []string{"Requests:      400 (<0.1/min)", "Avg response:  1.00 s"}},
		{requests: 999_950, window: "1h", latency: 12, want: []string{"Requests:      1.0M (17k/min)"}},
		{requests: 1_500_000_000, window: "7d", latency: 12, want: []string{"Requests:      1.5B (149k/min)"}},
		{requests: 0, window: "1h", latency: 0, want: []string{"Requests:      0 (0/min)", "Avg response:  —"}},
	}
	for _, tc := range cases {
		rt, mock, out := newRuntime(t)
		traffic := liveTraffic()
		traffic.RequestsTotal, traffic.AvgLatencyMs = tc.requests, float64p(tc.latency)
		gomock.InOrder(
			mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
			mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", tc.window).Return(traffic, nil),
			mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{Status: "healthy"}, nil),
		)
		require.NoError(t, lb.Traffic(context.Background(), rt, "web", tc.window, term.OutputTable))
		for _, want := range tc.want {
			assert.Contains(t, out.String(), want)
		}
	}
}

func perTargetHealth() api.LoadBalancerHealth {
	return api.LoadBalancerHealth{
		Status: "degraded", HealthyTargets: 1, TotalTargets: 2,
		Targets: []api.LoadBalancerTargetHealth{
			{VMUID: "vm-1", VMName: "web-1", State: "healthy"},
			{VMUID: "vm-2", VMName: "Web Two", State: "unhealthy", Reason: "failing health checks or returning errors"},
		},
	}
}

func TestGet_showsHealthPerTarget(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(perTargetHealth(), nil),
	)

	require.NoError(t, lb.Get(context.Background(), rt, "web", term.OutputTable))
	assert.Regexp(t, `TARGET\s+HEALTH\s+VM STATUS\s+VM UID`, out.String())
	assert.Regexp(t, `web-1\s+healthy\s+running\s+vm-1`, out.String())
	assert.Regexp(t, `Web Two\s+unhealthy\s+running\s+vm-2`, out.String())
}

func TestGet_olderServerWithoutTargetsShowsDash(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{Status: "healthy", HealthyTargets: 2, TotalTargets: 2}, nil),
	)

	require.NoError(t, lb.Get(context.Background(), rt, "web", term.OutputTable))
	assert.Regexp(t, `web-1\s+—\s+running`, out.String())
}

func TestTraffic_listsTargets(t *testing.T) {
	rt, mock, out := newRuntime(t)
	gomock.InOrder(
		mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
		mock.EXPECT().GetLoadBalancerTraffic(gomock.Any(), "acme", "lb-1", "1h").Return(liveTraffic(), nil),
		mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(perTargetHealth(), nil),
	)

	require.NoError(t, lb.Traffic(context.Background(), rt, "web", "1h", term.OutputTable))
	assert.Contains(t, out.String(), "  web-1    healthy\n")
	assert.Contains(t, out.String(), "  Web Two  unhealthy (failing health checks or returning errors)\n")
}

func TestGet_fallbackInfersOnlyWhatTheCountsProve(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   string
	}{
		{"healthy", `web-1\s+healthy\s+running`},
		{"down", `web-1\s+unhealthy\s+running`},
		{"degraded", `web-1\s+unknown\s+running`},
	} {
		rt, mock, out := newRuntime(t)
		gomock.InOrder(
			mock.EXPECT().ListLoadBalancers(gomock.Any(), "acme").Return([]api.LoadBalancer{readyLB()}, nil),
			mock.EXPECT().GetLoadBalancer(gomock.Any(), "acme", "lb-1").Return(readyLB(), nil),
			mock.EXPECT().GetLoadBalancerHealth(gomock.Any(), "acme", "lb-1").Return(api.LoadBalancerHealth{
				Status: tc.status, TotalTargets: 2,
				Targets: []api.LoadBalancerTargetHealth{{VMUID: "vm-1", VMName: "web-1", State: "unknown"}, {VMUID: "vm-2", VMName: "Web Two", State: "unknown"}},
			}, nil),
		)
		require.NoError(t, lb.Get(context.Background(), rt, "web", term.OutputTable))
		assert.Regexp(t, tc.want, out.String(), tc.status)
	}
}
