package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/apikeys"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
)

func newAPIKeyCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "api-key",
		Short:   "Manage your API keys",
		Long:    "Create and revoke the personal API keys that authenticate the CLI, the Rumpty MCP and your automation.",
		Example: `  rumpty api-key revoke "CI deploy"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty api-key")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newAPIKeyCreateCmd(rt))
	cmd.AddCommand(newAPIKeyRevokeCmd(rt))
	return cmd
}

func newAPIKeyRevokeCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "revoke <name|uid|prefix>",
		Short: "Revoke an API key",
		Example: `  rumpty api-key revoke "CI deploy"
  rumpty api-key revoke 7c1e2f9a-4b3d-4e8f-9a1b-2c3d4e5f6a7b`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return apikeys.Revoke(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}

func newAPIKeyCreateCmd(rt *app.Runtime) *cobra.Command {
	var opts apikeys.CreateOpts
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an API key",
		Long:  "Create a personal API key. The full token is shown once in the output; store it securely, it cannot be retrieved again. Keys expire after 365 days by default; set --expires-days (up to 730) or --never-expires, not both.",
		Example: `  rumpty api-key create "CI deploy"
  rumpty api-key create "mcp" --scope mcp --never-expires
  rumpty api-key create "read-only" --expires-days 90 --permissions workspace.read,audit.read`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForAuth(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return apikeys.Create(cmd.Context(), rt, args[0], opts)
		},
	}
	cmd.Flags().IntVar(&opts.ExpiresDays, "expires-days", 0, "Days until the key expires (default 365, capped at 730)")
	cmd.Flags().BoolVar(&opts.NeverExpiry, "never-expires", false, "Create a key that never expires")
	cmd.Flags().StringVar(&opts.Scope, "scope", "", "What the key is for: cli (default) or mcp")
	cmd.Flags().StringVar(&opts.Permissions, "permissions", "", "Comma-separated permissions; omit for the default set")
	return cmd
}
