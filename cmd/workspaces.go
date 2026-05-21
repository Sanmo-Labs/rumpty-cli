package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/workspace"
)

func newWorkspacesCmd(rt *app.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "workspaces",
		Short: "List workspaces you can access",
		Long:  "List Rumpty workspaces for the authenticated user.",
		Example: `  rumpty login
  rumpty workspaces`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return workspace.List(cmd.Context(), rt)
		},
	}
}
