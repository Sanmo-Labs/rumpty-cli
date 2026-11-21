package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/workspace"
)

func newWorkspacesCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:     "workspaces",
		Aliases: []string{"ws", "workspace"},
		Short:   "List workspaces you can access",
		Long:    "List Rumpty workspaces for the authenticated user.",
		Example: `  rumpty login
  rumpty workspaces
  rumpty ws
  rumpty ws -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return workspace.List(cmd.Context(), rt, format)
		},
	}
	addOutputFlag(cmd, &output)
	return cmd
}
