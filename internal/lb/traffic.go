package lb

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// TrafficRanges are the windows the traffic endpoint accepts. The server
// keeps 8 days of history, so longer windows are not offered.
var TrafficRanges = []string{"1h", "6h", "24h", "7d"}

// TrafficReport is what `rumpty lb traffic -o json` prints.
type TrafficReport struct {
	Health  api.LoadBalancerHealth  `json:"health"`
	Traffic api.LoadBalancerTraffic `json:"traffic"`
}

func Traffic(ctx context.Context, rt *app.Runtime, ref, window string, format term.OutputFormat) error {
	if !validRange(window) {
		return config.NewUsageError("--range must be one of %s", strings.Join(TrafficRanges, ", "))
	}
	spin := term.StartLoading(out(rt), "Loading traffic for "+ref)
	defer spin.Stop()
	found, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}
	workspace := strings.TrimSpace(rt.Config.Workspace)
	traffic, err := rt.API().GetLoadBalancerTraffic(ctx, workspace, found.UID, window)
	if err != nil {
		return fmt.Errorf("get traffic for %s: %w", lbName(&found), err)
	}
	health := loadHealth(ctx, rt, workspace, found.UID)
	spin.Stop()

	w := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(w, TrafficReport{Health: health, Traffic: traffic})
	}

	field := func(label, value string) {
		fmt.Fprintf(w, "%-17s%s\n", label+":", value)
	}
	field("Load balancer", term.Bold(w, lbName(&found)))
	field("Health", healthSummary(w, &health))
	fmt.Fprintln(w)

	if !traffic.Available {
		reason := orDefault(traffic.UnavailableReason, "not available")
		fmt.Fprintf(w, "Traffic (%s): %s\n", window, term.Muted(w, reason))
		return nil
	}
	fmt.Fprintf(w, "Traffic (%s)\n", window)
	if line := sparkline(traffic.Points); line != "" {
		fmt.Fprintf(w, "  %s\n", term.Accent(w, line))
	}
	field("  Requests", fmt.Sprintf("%s (%s/min)", count(float64(traffic.RequestsTotal)), perMinuteText(traffic.RequestsTotal, window)))
	field("  5xx errors", fmt.Sprintf("%s (%s)", count(float64(traffic.ErrorsTotal)), rate(traffic.ErrorRate)))
	field("  4xx", optionalCount(traffic.ClientErrorsTotal))
	field("  Avg response", latency(traffic.AvgLatencyMs, traffic.RequestsTotal))
	field("  Bandwidth", fmt.Sprintf("%s in, %s out", bytesHuman(traffic.RXBytes), bytesHuman(traffic.TXBytes)))
	return nil
}

// healthTimeout bounds the health lookup so a slow metrics backend never
// holds up `lb get` or `lb traffic`; the line then reads "unknown".
const healthTimeout = 3 * time.Second

// loadHealth never fails: health is a best-effort extra on both commands.
func loadHealth(ctx context.Context, rt *app.Runtime, workspace, uid string) api.LoadBalancerHealth {
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()
	health, err := rt.API().GetLoadBalancerHealth(ctx, workspace, uid)
	if err != nil {
		return api.LoadBalancerHealth{Status: "unknown", UnavailableReason: "could not load health"}
	}
	return health
}

// healthSummary renders the health line shared by `lb get` and `lb traffic`.
func healthSummary(w io.Writer, h *api.LoadBalancerHealth) string {
	switch h.Status {
	case "healthy":
		return term.StatusBadge(w, "healthy") + fmt.Sprintf(" (%d/%d targets)", h.HealthyTargets, h.TotalTargets)
	case "degraded":
		return term.StatusBadge(w, "degraded") + fmt.Sprintf(" (%d/%d targets healthy)", h.HealthyTargets, h.TotalTargets)
	case "down":
		return term.Error(w, "down") + fmt.Sprintf(" (0/%d targets healthy; visitors get a 503 page)", h.TotalTargets)
	case "no_targets":
		return term.Muted(w, "no targets")
	case "unknown", "":
		line := term.Muted(w, "unknown")
		if r := strings.TrimSpace(h.UnavailableReason); r != "" {
			line += term.Muted(w, " ("+r+")")
		}
		return line
	default:
		return term.Muted(w, h.Status)
	}
}

func validRange(window string) bool {
	for _, r := range TrafficRanges {
		if r == window {
			return true
		}
	}
	return false
}

var sparkBlocks = []rune("▁▂▃▄▅▆▇█")

func sparkline(points []api.LoadBalancerTrafficPoint) string {
	maxReq := 0.0
	for _, p := range points {
		maxReq = math.Max(maxReq, p.Requests)
	}
	if maxReq <= 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range points {
		i := int(math.Round(p.Requests / maxReq * float64(len(sparkBlocks)-1)))
		b.WriteRune(sparkBlocks[max(0, min(i, len(sparkBlocks)-1))])
	}
	return b.String()
}

func perMinuteText(total int64, window string) string {
	minutes := map[string]float64{"1h": 60, "6h": 360, "24h": 1440, "7d": 10080}[window]
	if minutes == 0 || total == 0 {
		return "0"
	}
	if r := float64(total) / minutes; r < 0.05 {
		return "<0.1"
	} else {
		return count(r)
	}
}

// count picks the unit after rounding, so 999,950 reads 1.0M, not 1000k.
func count(n float64) string {
	units := []struct {
		div    float64
		suffix string
	}{{1e9, "B"}, {1e6, "M"}, {1e3, "k"}}
	for _, u := range units {
		if v := n / u.div; math.Round(v*10)/10 >= 1 {
			if v >= 10 && u.suffix == "k" {
				return fmt.Sprintf("%.0f%s", v, u.suffix)
			}
			return fmt.Sprintf("%.1f%s", v, u.suffix)
		}
	}
	if n > 0 && n < 10 && n != math.Trunc(n) {
		return fmt.Sprintf("%.1f", n)
	}
	return fmt.Sprintf("%.0f", n)
}

func optionalCount(n *int64) string {
	if n == nil {
		return "—"
	}
	return count(float64(*n))
}

func rate(r float64) string {
	switch {
	case r == 0:
		return "0%"
	case r < 0.0001:
		return "<0.01%"
	default:
		return fmt.Sprintf("%.2f%%", r*100)
	}
}

func latency(ms *float64, requests int64) string {
	if ms == nil || requests == 0 {
		return "—"
	}
	if math.Round(*ms) >= 1000 {
		return fmt.Sprintf("%.2f s", *ms/1000)
	}
	return fmt.Sprintf("%.0f ms", *ms)
}

func bytesHuman(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
