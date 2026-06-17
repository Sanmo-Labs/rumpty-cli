package vm

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
)

func List(ctx context.Context, rt *app.Runtime) error {
	vms, err := rt.API().ListVMs(ctx, strings.TrimSpace(rt.Config.Workspace))
	if err != nil {
		return err
	}
	if len(vms) == 0 {
		fmt.Fprintln(rt.Streams.Out, "No VMs.")
		return nil
	}

	tw := tabwriter.NewWriter(rt.Streams.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tPLAN\tIMAGE\tSTATUS\tPORT\tAPP URL")
	for i := range vms {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			vms[i].Slug,
			formatPlan(&vms[i]),
			formatImage(&vms[i]),
			displayStatus(&vms[i]),
			formatAppPort(&vms[i]),
			formatAppURL(&vms[i]),
		)
	}
	return tw.Flush()
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
