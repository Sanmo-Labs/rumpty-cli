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
		Use:     "vm",
		Aliases: []string{"vms"},
		Short:   "Manage workspace VMs",
		Long:    "List and manage virtual machines in the configured Rumpty workspace.",
		Example: `  rumpty vm ls --ws production-team-019e2b95
  rumpty vm stop test-vm7 --ws production-team-019e2b95
  rumpty vm start test-vm7
  rumpty vm reboot test-vm7
  rumpty vm delete test-vm
  rumpty vm expose ls test-vm8 --ws production-team-019e2b95`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty vm")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newVMListCmd(rt),
		newVMLifecycleCmd(rt, "start", "Start a stopped VM", vm.Start),
		newVMLifecycleCmd(rt, "stop", "Stop a running VM", vm.Stop),
		newVMLifecycleCmd(rt, "reboot", "Reboot a VM", vm.Reboot),
		newVMLifecycleCmd(rt, "delete", "Delete a VM", vm.Delete),
		newVMExposeCmd(rt),
	)

	return cmd
}

func newVMExposeCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "expose",
		Short:   "Manage exposed VM services",
		Example: `  rumpty vm expose ls test-vm8 --ws production-team-019e2b95`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty vm expose")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newVMExposeListCmd(rt))
	return cmd
}

func newVMExposeListCmd(rt *app.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:     "ls <vm>",
		Short:   "List exposed URLs for a VM",
		Example: `  rumpty vm expose ls test-vm8 --ws production-team-019e2b95`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return vm.ListApps(cmd.Context(), rt, args[0])
		},
	}
}

func newVMListCmd(rt *app.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List VMs in the workspace",
		Example: `  rumpty vm ls
  rumpty vm ls --ws production-team-019e2b95`,
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
