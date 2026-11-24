package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

// errOut returns the error stream so loading spinners stay off stdout.
func errOut(rt *app.Runtime) io.Writer {
	return rt.Streams.ErrOut
}

// Ls lists the workspace's deployments, newest first.
func Ls(ctx context.Context, rt *app.Runtime, format term.OutputFormat) error {
	spin := term.StartLoading(errOut(rt), "Listing deployments")
	deployments, err := rt.API().ListDeployments(ctx, strings.TrimSpace(rt.Config.Workspace))
	spin.Stop()
	if err != nil {
		return err
	}

	out := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(out, deployments)
	}
	if len(deployments) == 0 {
		fmt.Fprintln(out, term.Muted(out, "No deployments yet."))
		return nil
	}
	table := term.NewTable(out, "NAME", "STATUS", "PLAN", "BRANCH", "URL")
	for i := range deployments {
		d := &deployments[i]
		table.Row(d.Name, statusText(d), d.PlanSlug, d.Branch, d.PublicURL)
	}
	return table.Render()
}

// statusText prefers the display status, which shows updating while a new
// version builds for a live app.
func statusText(d *api.Deployment) string {
	if d.DisplayStatus != "" {
		return d.DisplayStatus
	}
	return d.Status
}
