package vm

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

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
	fmt.Fprintln(tw, "SLUG\tNAME\tSTATUS\tSPEC")
	for i := range vms {
		status := strings.TrimSpace(vms[i].DisplayStatus)
		if status == "" {
			status = vms[i].Status
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", vms[i].Slug, vms[i].Name, status, formatSpec(&vms[i]))
	}
	return tw.Flush()
}
