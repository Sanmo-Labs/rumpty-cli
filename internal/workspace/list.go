package workspace

import (
	"context"
	"fmt"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/term"
)

func List(ctx context.Context, rt *app.Runtime, format term.OutputFormat) error {
	spin := term.StartLoading(rt.Streams.ErrOut, "Loading workspaces")
	workspaces, err := rt.API().ListWorkspaces(ctx)
	spin.Stop()
	if err != nil {
		return err
	}
	w := rt.Streams.Out
	if format == term.OutputJSON {
		return term.WriteJSON(w, workspaces)
	}
	if len(workspaces) == 0 {
		fmt.Fprintln(w, "No workspaces.")
		return nil
	}

	table := term.NewTable(w, "SLUG", "NAME")
	for _, ws := range workspaces {
		table.Row(term.Bold(w, ws.Slug), ws.Name)
	}
	if err := table.Render(); err != nil {
		return err
	}
	if len(workspaces) == 1 {
		term.Summary(w, "1 workspace")
	} else {
		term.Summary(w, fmt.Sprintf("%d workspaces", len(workspaces)), "use one with --ws <slug>")
	}
	return nil
}
