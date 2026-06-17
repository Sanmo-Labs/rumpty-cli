package commands

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
)

const completionTimeout = 5 * time.Second

func completeVMNames(rt *app.Runtime) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		slugs, ok := vmSlugs(cmd, rt)
		if !ok {
			return nil, cobra.ShellCompDirectiveError
		}

		return slugs, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeWorkspaceSlugs(rt *app.Runtime) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		rt.Config.Resolve()
		if err := rt.Config.ValidateForAuth(); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), completionTimeout)
		defer cancel()

		workspaces, err := rt.API().ListWorkspaces(ctx)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		out := make([]string, 0, len(workspaces))
		for i := range workspaces {
			if s := strings.TrimSpace(workspaces[i].Slug); s != "" {
				out = append(out, s)
			}
		}

		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func vmSlugs(cmd *cobra.Command, rt *app.Runtime) ([]string, bool) {
	rt.Config.Resolve()
	if err := rt.Config.ValidateForAuth(); err != nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), completionTimeout)
	defer cancel()

	workspace := strings.TrimSpace(rt.Config.Workspace)
	if workspace == "" {
		workspace = defaultWorkspace(ctx, rt)
	}
	if workspace == "" {
		return nil, false
	}

	vms, err := rt.API().ListVMs(ctx, workspace)
	if err != nil {
		return nil, false
	}

	out := make([]string, 0, len(vms))
	for i := range vms {
		if s := strings.TrimSpace(vms[i].Slug); s != "" {
			out = append(out, s)
		}
	}

	return out, true
}

// defaultWorkspace picks a workspace to complete against when the user hasn't
// specified one: the workspace flagged IsDefault, or the only one if there's a
// single workspace. Returns "" when the choice is ambiguous.
func defaultWorkspace(ctx context.Context, rt *app.Runtime) string {
	workspaces, err := rt.API().ListWorkspaces(ctx)
	if err != nil {
		return ""
	}
	for i := range workspaces {
		if workspaces[i].IsDefault {
			return strings.TrimSpace(workspaces[i].Slug)
		}
	}
	if len(workspaces) == 1 {
		return strings.TrimSpace(workspaces[0].Slug)
	}
	return ""
}
