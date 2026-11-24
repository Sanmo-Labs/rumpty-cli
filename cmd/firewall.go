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
	cmd.AddCommand(newFirewallRuleCmd(rt))
	cmd.AddCommand(newFirewallAttachCmd(rt))
	cmd.AddCommand(newFirewallDetachCmd(rt))
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

func newFirewallRuleCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Add and remove firewall rules",
		Long:  "Add and remove allow rules on a firewall policy. The policy re-applies to every resource it is attached to.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty firewall rule")
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(newFirewallRuleAddCmd(rt))
	cmd.AddCommand(newFirewallRuleRmCmd(rt))
	return cmd
}

func newFirewallRuleAddCmd(rt *app.Runtime) *cobra.Command {
	var opts firewall.RuleOpts
	cmd := &cobra.Command{
		Use:   "add <policy>",
		Short: "Add an allow rule to a firewall policy",
		Long:  "Add an allow rule to a firewall policy. The source must be a CIDR block (a single address needs its prefix, such as 198.51.100.7/32); use 0.0.0.0/0 for any IPv4 source. Leave out --port to match all ports.",
		Example: `  rumpty firewall rule add web --port 80-443 --source 0.0.0.0/0
  rumpty firewall rule add web --port 22 --source 198.51.100.0/24 --description "office SSH"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return firewall.Add(cmd.Context(), rt, args[0], &opts)
		},
	}
	cmd.Flags().StringVar(&opts.Port, "port", "", "Port or range (80 or 80-443); omit for all ports")
	cmd.Flags().StringVar(&opts.PortEnd, "port-end", "", "End of the port range when --port holds only the start")
	cmd.Flags().StringVar(&opts.Source, "source", "", "Source address range in CIDR notation, required")
	cmd.Flags().StringVar(&opts.Direction, "direction", "", "inbound (default) or outbound")
	cmd.Flags().StringVar(&opts.Protocol, "protocol", "", "tcp (default), udp, or all")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Optional note about what the rule is for")
	return cmd
}

func newFirewallRuleRmCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm <policy> <rule>",
		Short: "Delete a rule from a firewall policy",
		Long:  "Delete one rule from a firewall policy by uid, or a compact reference such as tcp/22. Default rules can be deleted too; removing the SSH rule from a policy attached to a VM can block access unless another policy still allows port 22.",
		Example: `  rumpty firewall rule rm web tcp/22
  rumpty firewall rule rm web 019a9e10-7a8c-7d0e-9f1a-2b3c4d5e6f7a`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return firewall.RmRule(cmd.Context(), rt, args[0], args[1], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return cmd
}

func newFirewallAttachCmd(rt *app.Runtime) *cobra.Command {
	var resourceType string
	cmd := &cobra.Command{
		Use:   "attach <policy> <resource>",
		Short: "Attach a firewall policy to a resource",
		Long:  "Attach a firewall policy to a VM, database or deployment in the workspace. The resource is resolved by uid, name or slug; VMs are checked first, then databases. For a deployment pass --type deployment and its uid. From then on the resource only accepts traffic allowed by its attached policies.",
		Example: `  rumpty firewall attach web web-1
  rumpty firewall attach web my-postgres --type database`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return firewall.Attach(cmd.Context(), rt, args[0], args[1], resourceType)
		},
	}
	cmd.Flags().StringVar(&resourceType, "type", "", "Resource kind: vm (default), database, or deployment")
	return cmd
}

func newFirewallDetachCmd(rt *app.Runtime) *cobra.Command {
	var resourceType string
	cmd := &cobra.Command{
		Use:     "detach <policy> <resource>",
		Short:   "Detach a firewall policy from a resource",
		Long:    "Detach a firewall policy from the resource it is attached to, matched by the resource's uid, name or slug. If no other policy is attached, the resource is no longer filtered.",
		Example: `  rumpty firewall detach web web-1`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return firewall.Detach(cmd.Context(), rt, args[0], args[1], resourceType)
		},
	}
	cmd.Flags().StringVar(&resourceType, "type", "", "Resource kind when the reference is ambiguous: vm, database, or deployment")
	return cmd
}
