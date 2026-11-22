package vm

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func ListApps(ctx context.Context, rt *app.Runtime, ref string, format term.OutputFormat) error {
	spin := term.StartLoading(out(rt), "Loading exposed apps for "+ref)
	defer spin.Stop()
	target, err := Find(ctx, rt, ref)
	if err != nil {
		return err
	}

	workspace := strings.TrimSpace(rt.Config.Workspace)
	apps, err := rt.API().ListVMApps(ctx, workspace, target.UID)
	if err != nil {
		return fmt.Errorf("list exposed apps for %s: %w", target.Slug, err)
	}
	spin.Stop()
	w := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(w, apps)
	}
	if len(apps) == 0 {
		fmt.Fprintf(w, "No exposed apps for %s.\n", target.Slug)
		fmt.Fprintln(w, term.Muted(w, fmt.Sprintf("Expose one with: rumpty expose %s --ws %s --port <port>", target.Slug, workspace)))
		return nil
	}

	table := term.NewTable(w, "NAME", "PORT", "PROTOCOL", "STATUS", "URL")
	for i := range apps {
		table.Row(
			term.Bold(w, appName(&apps[i])),
			fmt.Sprintf("%d", apps[i].Port),
			appProtocol(&apps[i]),
			term.StatusBadge(w, appStatus(&apps[i])),
			term.Link(w, appURL(&apps[i])),
		)
	}
	if err := table.Render(); err != nil {
		return err
	}
	term.Summary(w, plural(len(apps), "exposed app", "exposed apps"))
	return nil
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
