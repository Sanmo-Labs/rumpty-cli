package commands

import (
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/firewall"
)

func newFirewallCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firewall",
		Short: "Manage firewall policies",
		Long:  "Create and delete firewall policies. Rules and attachments are managed with `rumpty firewall rule` and `rumpty firewall attach`.",
		Example: `  rumpty firewall create web
  rumpty firewall rm web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty firewall")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(newFirewallCreateCmd(rt))
	cmd.AddCommand(newFirewallRmCmd(rt))
	return cmd
}

func newFirewallCreateCmd(rt *app.Runtime) *cobra.Command {
	var description string
	cmd := &cobra.Command{
		Use:     "create <name>",
		Short:   "Create a firewall policy",
		Example: `  rumpty firewall create web --description "public sites"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return firewall.Create(cmd.Context(), rt, args[0], description)
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "Optional description for the policy")
	return cmd
}

func newFirewallRmCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "rm <name|uid>",
		Short:   "Delete a firewall policy",
		Long:    "Delete a firewall policy. Everything attached to it loses the policy and its firewall is rebuilt from the remaining policies.",
		Example: `  rumpty firewall rm web`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return firewall.Rm(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}
