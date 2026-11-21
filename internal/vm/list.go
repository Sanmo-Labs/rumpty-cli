package vm

import (
	"context"
	"fmt"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func List(ctx context.Context, rt *app.Runtime, format term.OutputFormat) error {
	workspace := strings.TrimSpace(rt.Config.Workspace)
	spin := term.StartLoading(out(rt), "Loading VMs in "+workspace)
	vms, err := rt.API().ListVMs(ctx, workspace)
	spin.Stop()
	if err != nil {
		return err
	}
	w := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(w, vms)
	}
	if len(vms) == 0 {
		fmt.Fprintf(w, "No VMs in %s.\n", workspace)
		fmt.Fprintln(w, term.Muted(w, "Create one at https://console.rumptycloud.com"))
		return nil
	}

	pretty := term.IsInteractive(w)
	table := term.NewTable(w, "NAME", "PLAN", "IMAGE", "STATUS", "PORT", "APP URL")
	counts := map[string]int{}
	var order []string
	for i := range vms {
		status := displayStatus(&vms[i])
		if counts[status] == 0 {
			order = append(order, status)
		}
		counts[status]++
		image, url := formatImage(&vms[i]), formatAppURL(&vms[i])
		if pretty {
			image, url = prettyImage(image), strings.TrimPrefix(url, "https://")
		}
		table.Row(
			term.Bold(w, vms[i].Slug),
			formatPlan(&vms[i]),
			image,
			term.StatusBadge(w, status),
			formatAppPort(&vms[i]),
			url,
		)
	}
	if err := table.Render(); err != nil {
		return err
	}

	summary := []string{plural(len(vms), "VM", "VMs")}
	for _, status := range order {
		summary = append(summary, fmt.Sprintf("%d %s", counts[status], status))
	}
	term.Summary(w, summary...)
	return nil
}

// prettyImage turns image slugs like ubuntu-2604-lts into "Ubuntu 26.04".
func prettyImage(slug string) string {
	parts := strings.Split(slug, "-")
	if len(parts) < 2 || parts[0] != "ubuntu" || len(parts[1]) != 4 {
		return slug
	}
	name := "Ubuntu " + parts[1][:2] + "." + parts[1][2:]
	if len(parts) > 2 && parts[2] == "desktop" {
		name += " Desktop"
	}
	return name
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func formatPlan(v *api.VM) string {
	if s := strings.TrimSpace(v.PlanSlug); s != "" {
		return s
	}
	return "—"
}

func formatImage(v *api.VM) string {
	if s := strings.TrimSpace(v.ImageSlug); s != "" {
		return s
	}
	if s := strings.TrimSpace(v.ImageName); s != "" {
		return s
	}
	return "—"
}

func displayStatus(v *api.VM) string {
	if s := strings.TrimSpace(v.DisplayStatus); s != "" {
		return s
	}
	return strings.TrimSpace(v.Status)
}

func formatAppPort(v *api.VM) string {
	ports := []string{"22", "8080"}
	if v.AppPort > 0 && v.AppPort != 8080 && v.AppPort != 22 {
		ports = append(ports, fmt.Sprintf("%d", v.AppPort))
	}
	return strings.Join(ports, ", ")
}

func formatAppURL(v *api.VM) string {
	if s := strings.TrimSpace(v.AppURL); s != "" {
		return s
	}
	return "—"
}
