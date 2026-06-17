package vm

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func ListApps(ctx context.Context, rt *app.Runtime, ref string) error {
	term.Statusf(out(rt), "Resolving VM %s", ref)
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	apps, err := rt.API().ListVMApps(ctx, workspace, target.UID)
	if err != nil {
		return fmt.Errorf("list exposed apps for %s: %w", target.Slug, err)
	}
	if len(apps) == 0 {
		fmt.Fprintf(rt.Streams.Out, "No exposed apps for %s.\n", target.Slug)
		return nil
	}

	tw := tabwriter.NewWriter(rt.Streams.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tPORT\tPROTOCOL\tSTATUS\tURL")
	for i := range apps {
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n",
			appName(&apps[i]),
			apps[i].Port,
			appProtocol(&apps[i]),
			appStatus(&apps[i]),
			appURL(&apps[i]),
		)
	}
	return tw.Flush()
}

func appName(app *api.VMApp) string {
	if s := strings.TrimSpace(app.Slug); s != "" {
		return s
	}
	if s := strings.TrimSpace(app.Name); s != "" {
		return s
	}
	return "—"
}

func appStatus(app *api.VMApp) string {
	if s := strings.TrimSpace(app.Status); s != "" {
		return s
	}
	return "—"
}

func appProtocol(app *api.VMApp) string {
	if s := strings.TrimSpace(app.Protocol); s != "" {
		return strings.ToUpper(s)
	}
	return "HTTP"
}

func appURL(app *api.VMApp) string {
	if s := strings.TrimSpace(app.URL); s != "" {
		return s
	}
	return "—"
}
