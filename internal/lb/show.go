package lb

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func List(ctx context.Context, rt *app.Runtime, format term.OutputFormat) error {
	workspace := strings.TrimSpace(rt.Config.Workspace)
	spin := term.StartLoading(out(rt), "Loading load balancers in "+workspace)
	lbs, err := rt.API().ListLoadBalancers(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}

	w := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(w, lbs)
	}
	if len(lbs) == 0 {
		fmt.Fprintf(w, "No load balancers in %s.\n", workspace)
		fmt.Fprintln(w, term.Muted(w, "Create one with: rumpty lb create <name> --vm <vm> --port <port>"))
		return nil
	}

	table := term.NewTable(w, "NAME", "STATUS", "PROTOCOL", "PORT", "TARGETS", "URL")
	for i := range lbs {
		table.Row(
			term.Bold(w, lbName(&lbs[i])),
			term.StatusBadge(w, orDash(lbs[i].Status)),
			strings.ToUpper(orDefault(lbs[i].TargetProtocol, "http")),
			fmt.Sprintf("%d", lbs[i].TargetPort),
			targetNames(&lbs[i]),
			term.Link(w, orDash(lbs[i].URL)),
		)
	}
	if err := table.Render(); err != nil {
		return err
	}
	term.Summary(w, plural(len(lbs), "load balancer", "load balancers"))
	return nil
}

func Get(ctx context.Context, rt *app.Runtime, ref string, format term.OutputFormat) error {
	spin := term.StartLoading(out(rt), "Loading "+ref)
	defer spin.Stop()
	found, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}
	workspace := strings.TrimSpace(rt.Config.Workspace)
	lb, err := rt.API().GetLoadBalancer(ctx, workspace, found.UID)
	if err != nil {
		return fmt.Errorf("get load balancer %s: %w", lbName(&found), err)
	}
	w := rt.Streams.Out
	if format == term.OutputJSON {
		spin.Stop()
		return term.WriteJSON(w, lb)
	}
	health := loadHealth(ctx, rt, workspace, lb.UID)
	spin.Stop()

	field := func(label, value string) {
		fmt.Fprintf(w, "%-17s%s\n", label+":", value)
	}
	field("Name", term.Bold(w, lbName(&lb)))
	field("Status", term.StatusBadge(w, orDash(lb.Status)))
	field("Health", healthSummary(w, &health))
	if s := strings.TrimSpace(lb.LastFailure); s != "" {
		field("Last failure", term.Error(w, s))
	}
	field("URL", term.Link(w, orDash(lb.URL)))
	field("Backend", fmt.Sprintf("%s on port %d", strings.ToUpper(orDefault(lb.TargetProtocol, "http")), lb.TargetPort))
	field("Health check", healthCheckSummary(&lb.HealthCheck))
	field("Sticky sessions", stickySummary(&lb.StickySessions))
	field("Allowed IPs", allowedSummary(lb.AllowedCIDRs))
	field("UID", lb.UID)

	fmt.Fprintln(w)
	if len(lb.Targets) == 0 {
		fmt.Fprintln(w, "No targets.")
		return nil
	}
	targets := term.NewTable(w, "TARGET", "VM STATUS", "VM UID")
	for i := range lb.Targets {
		targets.Row(
			orDash(lb.Targets[i].VMName),
			term.StatusBadge(w, orDash(lb.Targets[i].VMStatus)),
			lb.Targets[i].VMUID,
		)
	}
	return targets.Render()
}

func healthCheckSummary(hc *api.LoadBalancerHealthCheck) string {
	check := strings.ToUpper(orDefault(hc.Protocol, "http"))
	if strings.EqualFold(hc.Protocol, "http") && hc.Path != "" {
		check += " " + hc.Path
	}
	if hc.Port > 0 {
		check += fmt.Sprintf(" on port %d", hc.Port)
	}
	return fmt.Sprintf("%s every %ds, timeout %ds, healthy after %d, unhealthy after %d",
		check, hc.IntervalSeconds, hc.TimeoutSeconds, hc.HealthyThreshold, hc.UnhealthyThreshold)
}

func stickySummary(s *api.LoadBalancerStickySessions) string {
	if !s.Enabled {
		return "off"
	}
	return fmt.Sprintf("on (cookie %s, %ds)", orDash(s.CookieName), s.TTLSeconds)
}

func allowedSummary(cidrs []string) string {
	if len(cidrs) == 0 {
		return "everyone"
	}
	return strings.Join(cidrs, ", ")
}

func targetNames(lb *api.LoadBalancer) string {
	if len(lb.Targets) == 0 {
		return "—"
	}
	names := make([]string, len(lb.Targets))
	for i := range lb.Targets {
		names[i] = orDefault(lb.Targets[i].VMName, lb.Targets[i].VMUID)
	}
	return strings.Join(names, ", ")
}

func lbName(lb *api.LoadBalancer) string {
	if s := strings.TrimSpace(lb.Slug); s != "" {
		return s
	}
	if s := strings.TrimSpace(lb.Name); s != "" {
		return s
	}
	return "load balancer"
}

func orDash(s string) string {
	return orDefault(s, "—")
}

func orDefault(s, def string) string {
	if s = strings.TrimSpace(s); s != "" {
		return s
	}
	return def
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
