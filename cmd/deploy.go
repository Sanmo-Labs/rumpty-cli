package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/deploy"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func newDeployCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Inspect your deployments",
		Long:  "List and inspect the deployments running on RumptyCloud.",
		Example: `  rumpty deploy ls
  rumpty deploy ls -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty deploy")
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(newDeployLsCmd(rt))
	return cmd
}

func newDeployLsCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List deployments",
		Example: `  rumpty deploy ls
  rumpty deploy ls -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			format, err := term.ParseOutputFormat(output)
			if err != nil {
				return config.NewUsageError("%v", err)
			}
			return deploy.Ls(cmd.Context(), rt, format)
		},
	}
	addOutputFlag(cmd, &output)
	return cmd
}
