package commands

import (
	"github.com/spf13/cobra"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/config"
	"github.com/Sanmo-Labs/rumpty-cli/internal/vm"
)

func newExposeCmd(rt *app.Runtime) *cobra.Command {
	var name string
	var port int
	var protocol string

	cmd := &cobra.Command{
		Use:   "expose <vm>",
		Short: "Expose a VM service with a public URL",
		Long: `Expose a service running inside a VM through Rumpty HTTP access.

The service inside the VM must listen on 0.0.0.0:<port>, not only 127.0.0.1.

Use --protocol grpc for gRPC services.`,
		Example: `  rumpty expose test-vm8 --ws production-team-019e2b95 --port 18789 --name openclaw
  rumpty expose api-box --ws production-team-019e2b95 -p 3000
  rumpty expose api-box --ws production-team-019e2b95 -p 50051 --protocol grpc`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return nil
			}
			if len(args) > 1 {
				return config.NewUsageError("unexpected argument %q\nTo list exposed URLs, run: rumpty vm expose ls %s --ws <workspace>", args[1], args[0])
			}
			return config.NewUsageError("missing VM name")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			if port == 0 {
				return config.NewUsageError("--port is required")
			}
			if port < 1 || port > 65535 {
				return config.NewUsageError("--port must be between 1 and 65535")
			}
			switch protocol {
			case "", "http", "grpc":
				// valid
			default:
				return config.NewUsageError("--protocol must be http or grpc")
			}
			return vm.Expose(cmd.Context(), rt, args[0], port, name, protocol)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Public app name; defaults to port-<port>")
	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port inside the VM to expose")
	cmd.Flags().StringVar(&protocol, "protocol", "", `Application protocol: http (default) or grpc`)
	return cmd
}

func newUnexposeCmd(rt *app.Runtime) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "unexpose <vm>",
		Short:   "Remove a VM service public URL",
		Example: `  rumpty unexpose test-vm8 --ws production-team-019e2b95 --name openclaw`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return nil
			}
			if len(args) > 1 {
				return config.NewUsageError("unexpected argument %q", args[1])
			}
			return config.NewUsageError("missing VM name")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			if name == "" {
				return config.NewUsageError("--name is required")
			}
			return vm.Unexpose(cmd.Context(), rt, args[0], name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Public app name to remove")
	return cmd
}
