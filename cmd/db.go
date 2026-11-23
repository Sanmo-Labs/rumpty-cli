package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/db"
)

func newDBCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage workspace databases",
		Long:  "List and manage managed databases in the configured Rumpty workspace.",
		Example: `  rumpty db rm prod-db
  rumpty db rm prod-db --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty db")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newDBDeleteCmd(rt))
	return cmd
}

func newDBDeleteCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm <db>",
		Short: "Delete a database",
		Example: `  rumpty db rm prod-db
  rumpty db rm prod-db --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return db.Delete(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}
