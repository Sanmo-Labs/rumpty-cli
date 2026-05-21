package workspace

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
)

func List(ctx context.Context, rt *app.Runtime) error {
	workspaces, err := rt.API().ListWorkspaces(ctx)
	if err != nil {
		return err
	}
	if len(workspaces) == 0 {
		fmt.Fprintln(rt.Streams.Out, "No workspaces.")
		return nil
	}

	tw := tabwriter.NewWriter(rt.Streams.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SLUG\tNAME")
	for _, ws := range workspaces {
		fmt.Fprintf(tw, "%s\t%s\n", ws.Slug, ws.Name)
	}
	return tw.Flush()
}
