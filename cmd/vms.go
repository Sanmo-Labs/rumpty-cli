package commands

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/vm"
)

func newVMsCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vms",
		Short: "Manage workspace VMs",
		Long:  "List and manage virtual machines in the configured Rumpty workspace.",
		Example: `  rumpty vms ls --ws production-team-019e2b95
  rumpty vms stop test-vm7 --ws production-team-019e2b95
  rumpty vms start test-vm7
  rumpty vms reboot test-vm7
  rumpty vms delete test-vm`,
	}

	cmd.AddCommand(
		newVMListCmd(rt),
		newVMLifecycleCmd(rt, "start", "Start a stopped VM", vm.Start),
		newVMLifecycleCmd(rt, "stop", "Stop a running VM", vm.Stop),
		newVMLifecycleCmd(rt, "reboot", "Reboot a VM", vm.Reboot),
		newVMLifecycleCmd(rt, "delete", "Delete a VM", vm.Delete),
	)

	return cmd
}

func newVMListCmd(rt *app.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List VMs in the workspace",
		Example: `  rumpty vms ls
  rumpty vms ls --ws production-team-019e2b95`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return vm.List(cmd.Context(), rt)
		},
	}
}

func newVMLifecycleCmd(rt *app.Runtime, use, short string, run func(context.Context, *app.Runtime, string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <vm>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return run(cmd.Context(), rt, args[0])
		},
	}
}
