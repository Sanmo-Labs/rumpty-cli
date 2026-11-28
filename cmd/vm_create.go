package commands

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/vm"
)

func newVMCreateCmd(rt *app.Runtime) *cobra.Command {
	var req api.CreateVMRequest
	var wait bool
	cmd := &cobra.Command{
		Use:               "create <name> --plan <slug> --image <slug> --ssh-key <uid>",
		Short:             "Create a VM from an OS image",
		Example:           `  rumpty vm create web-1 --plan micro --image ubuntu-2604-lts --ssh-key "$SSH_KEY_UID" --ws acme --wait`,
		Args:              exactArgs(1, "missing VM name"),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			req.Name = strings.TrimSpace(args[0])
			req.PlanSlug = strings.TrimSpace(req.PlanSlug)
			req.ImageSlug = strings.TrimSpace(req.ImageSlug)
			req.ZoneSlug = strings.TrimSpace(req.ZoneSlug)
			switch {
			case req.Name == "":
				return config.NewUsageError("VM name is empty")
			case req.PlanSlug == "":
				return config.NewUsageError("--plan is required")
			case req.ImageSlug == "":
				return config.NewUsageError("--image is required")
			case len(req.SSHKeyUIDs) == 0:
				return config.NewUsageError("--ssh-key is required")
			case cmd.Flags().Changed("zone") && req.ZoneSlug == "":
				return config.NewUsageError("--zone must not be empty")
			}
			for i, key := range req.SSHKeyUIDs {
				req.SSHKeyUIDs[i] = strings.TrimSpace(key)
				if req.SSHKeyUIDs[i] == "" {
					return config.NewUsageError("--ssh-key must not be empty")
				}
			}
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			req.VPCSlug = "default"
			return vm.Create(cmd.Context(), rt, &req, wait)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&req.PlanSlug, "plan", "", "Compute plan slug (required)")
	flags.StringVar(&req.ImageSlug, "image", "", "OS image slug (required)")
	flags.StringVar(&req.ZoneSlug, "zone", "", "Zone slug (defaults to the API's default zone)")
	flags.StringArrayVar(&req.SSHKeyUIDs, "ssh-key", nil, "SSH key UID to install (required, repeat for more keys)")
	flags.BoolVar(&wait, "wait", false, "Wait until the VM is running")
	return cmd
}
