package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/vm"
)

func newVMsCmd(rt *app.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "vms",
		Short: "List VMs in a workspace",
		Long:  "List virtual machines in the configured Rumpty workspace.",
		Example: `  rumpty login
  rumpty workspaces
  rumpty vms --ws production-team-019e2b95`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return vm.List(cmd.Context(), rt)
		},
	}
}
