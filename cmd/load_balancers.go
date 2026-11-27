package commands

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/lb"
)

const lbPricing = "Each load balancer costs $2/month, billed hourly until you delete it."

func newLoadBalancersCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "load-balancer",
		Aliases: []string{"lb", "load-balancers"},
		Short:   "Manage load balancers",
		Long: `Put 1 to 10 VMs behind one HTTPS address. Requests go to the VMs that pass
the health check, round robin, or pinned to one VM with sticky sessions.

` + lbPricing,
		Example: `  rumpty lb ls
  rumpty lb create web --vm web-1 --vm web-2 --port 8080 --health-path /healthz
  rumpty lb get web
  rumpty lb traffic web --range 24h
  rumpty lb add-target web web-3
  rumpty lb remove-target web web-1
  rumpty lb update web --sticky --allow 203.0.113.0/24
  rumpty lb delete web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return config.NewUsageError("unknown command %q for %q", args[0], "rumpty load-balancer")
			}
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newLBListCmd(rt),
		newLBGetCmd(rt),
		newLBTrafficCmd(rt),
		newLBCreateCmd(rt),
		newLBUpdateCmd(rt),
		newLBDeleteCmd(rt),
		newLBAddTargetCmd(rt),
		newLBRemoveTargetCmd(rt),
	)
	return cmd
}

func newLBListCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List load balancers in the workspace",
		Example: `  rumpty lb ls
  rumpty lb ls -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return lb.List(cmd.Context(), rt, format)
		},
	}
	addOutputFlag(cmd, &output)
	return cmd
}

func newLBGetCmd(rt *app.Runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "get <load-balancer>",
		Short: "Show a load balancer's settings and targets",
		Example: `  rumpty lb get web
  rumpty lb get web -o json`,
		Args:              exactArgs(1, "missing load balancer name"),
		ValidArgsFunction: completeLoadBalancerNames(rt),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return lb.Get(cmd.Context(), rt, args[0], format)
		},
	}
	addOutputFlag(cmd, &output)
	return cmd
}

func newLBTrafficCmd(rt *app.Runtime) *cobra.Command {
	var output, window string
	cmd := &cobra.Command{
		Use:   "traffic <load-balancer>",
		Short: "Show live health and request traffic for a load balancer",
		Long: `Show whether the load balancer's targets are healthy right now, and the
requests, errors, response time and bandwidth it served over a time range.`,
		Example: `  rumpty lb traffic web
  rumpty lb traffic web --range 7d
  rumpty lb traffic web -o json`,
		Args:              exactArgs(1, "missing load balancer name"),
		ValidArgsFunction: completeLoadBalancerNames(rt),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return lb.Traffic(cmd.Context(), rt, args[0], window, format)
		},
	}
	cmd.Flags().StringVar(&window, "range", "1h", "Time range: "+strings.Join(lb.TrafficRanges, ", "))
	_ = cmd.RegisterFlagCompletionFunc("range", cobra.FixedCompletions(lb.TrafficRanges, cobra.ShellCompDirectiveNoFileComp))
	addOutputFlag(cmd, &output)
	return cmd
}

type lbFlags struct {
	vms                []string
	port               int
	protocol           string
	healthProtocol     string
	healthPath         string
	healthInterval     int
	healthTimeout      int
	healthyThreshold   int
	unhealthyThreshold int
	sticky             bool
	allow              []string
	allowAll           bool
}

func (f *lbFlags) register(cmd *cobra.Command, rt *app.Runtime, update bool) {
	fs := cmd.Flags()
	if update {
		fs.StringSliceVar(&f.vms, "vm", nil, "VM name or uid to send traffic to; replaces all targets (repeatable)")
		fs.IntVarP(&f.port, "port", "p", 0, "Port your app listens on inside each VM")
	} else {
		fs.StringSliceVar(&f.vms, "vm", nil, "VM name or uid to send traffic to (repeatable, 1 to 10)")
		fs.IntVarP(&f.port, "port", "p", 0, "Port your app listens on inside each VM (required)")
	}
	fs.StringVar(&f.protocol, "protocol", "", "Backend protocol: http (default) or grpc")
	fs.StringVar(&f.healthProtocol, "health-protocol", "", "Health check: http or tcp (default http, tcp for grpc; grpc must use tcp)")
	fs.StringVar(&f.healthPath, "health-path", "", "Path for http health checks; passes on a 2xx or 3xx response (default /)")
	fs.IntVar(&f.healthInterval, "health-interval", 0, "Seconds between health checks, 1 to 300 (default 10)")
	fs.IntVar(&f.healthTimeout, "health-timeout", 0, "Seconds before a health check fails, at most the interval (default 5)")
	fs.IntVar(&f.healthyThreshold, "healthy-threshold", 0, "Passing checks before a VM gets traffic, 1 to 10 (default 3)")
	fs.IntVar(&f.unhealthyThreshold, "unhealthy-threshold", 0, "Failing checks before a VM is taken out, 1 to 10 (default 3)")
	if update {
		fs.BoolVar(&f.sticky, "sticky", false, "Turn sticky sessions on, or off with --sticky=false")
		fs.StringSliceVar(&f.allow, "allow", nil, "Client CIDR allowed to connect; replaces the allow list (repeatable)")
		fs.BoolVar(&f.allowAll, "allow-all", false, "Clear the allow list so every client can connect")
		cmd.MarkFlagsMutuallyExclusive("allow", "allow-all")
	} else {
		fs.BoolVar(&f.sticky, "sticky", false, "Pin each client to one VM with a cookie (rumpty-lb, 300 seconds)")
		fs.StringSliceVar(&f.allow, "allow", nil, "Client CIDR allowed to connect (repeatable); default allows everyone")
	}

	_ = cmd.RegisterFlagCompletionFunc("vm", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		slugs, ok := vmSlugs(cmd, rt)
		if !ok {
			return nil, cobra.ShellCompDirectiveError
		}
		return slugs, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("protocol", cobra.FixedCompletions([]string{"http", "grpc"}, cobra.ShellCompDirectiveNoFileComp))
	_ = cmd.RegisterFlagCompletionFunc("health-protocol", cobra.FixedCompletions([]string{"http", "tcp"}, cobra.ShellCompDirectiveNoFileComp))
}

// settings returns only the fields whose flags were set, so update leaves
// everything else unchanged and create gets the server defaults.
func (f *lbFlags) settings(cmd *cobra.Command) (*lb.Settings, error) {
	changed := cmd.Flags().Changed
	s := &lb.Settings{}
	if changed("vm") {
		s.VMs = append([]string{}, f.vms...)
	}
	if changed("port") {
		s.Port = &f.port
	}
	if changed("protocol") {
		v := strings.ToLower(strings.TrimSpace(f.protocol))
		s.Protocol = &v
	}
	if changed("health-protocol") {
		v := strings.ToLower(strings.TrimSpace(f.healthProtocol))
		s.HealthProtocol = &v
	}
	if changed("health-path") {
		s.HealthPath = &f.healthPath
	}
	if changed("health-interval") {
		s.HealthInterval = &f.healthInterval
	}
	if changed("health-timeout") {
		s.HealthTimeout = &f.healthTimeout
	}
	if changed("healthy-threshold") {
		s.HealthyThreshold = &f.healthyThreshold
	}
	if changed("unhealthy-threshold") {
		s.UnhealthyThreshold = &f.unhealthyThreshold
	}
	if changed("sticky") {
		s.Sticky = &f.sticky
	}
	if changed("allow") {
		cidrs := make([]string, 0, len(f.allow))
		for _, c := range f.allow {
			if c = strings.TrimSpace(c); c != "" {
				cidrs = append(cidrs, c)
			}
		}
		if len(cidrs) == 0 {
			return nil, config.NewUsageError("--allow needs a CIDR; to allow every client, leave out --allow on create or pass --allow-all on update")
		}
		s.AllowedCIDRs = &cidrs
	}
	if f.allowAll {
		s.AllowedCIDRs = &[]string{}
	}
	if err := s.Validate(); err != nil {
		return nil, config.NewUsageError("%v", err)
	}
	return s, nil
}

func newLBCreateCmd(rt *app.Runtime) *cobra.Command {
	var flags lbFlags
	cmd := &cobra.Command{
		Use:   "create <name> --vm <vm>... --port <port>",
		Short: "Create a load balancer in front of your VMs",
		Long: `Create a load balancer that spreads HTTPS traffic across 1 to 10 VMs at
https://<name>-<id>.<domain>. The command waits until it is ready.

Apps in each VM must listen on 0.0.0.0:<port>. By default the load balancer
checks HTTP GET / every 10 seconds; VMs that fail are taken out of rotation.
Use --protocol grpc for gRPC (HTTP/2) services; gRPC load balancers use TCP
health checks.

` + lbPricing,
		Example: `  rumpty lb create web --vm web-1 --vm web-2 --port 8080
  rumpty lb create web --vm web-1,web-2 -p 8080 --health-path /healthz --sticky
  rumpty lb create api --vm api-1 --port 50051 --protocol grpc
  rumpty lb create admin --vm admin-1 --port 3000 --allow 203.0.113.0/24`,
		Args: exactArgs(1, "missing load balancer name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			if len(flags.vms) == 0 {
				return config.NewUsageError("--vm is required")
			}
			if !cmd.Flags().Changed("port") {
				return config.NewUsageError("--port is required")
			}
			s, err := flags.settings(cmd)
			if err != nil {
				return err
			}
			return lb.Create(cmd.Context(), rt, args[0], s)
		},
	}
	flags.register(cmd, rt, false)
	return cmd
}

func newLBUpdateCmd(rt *app.Runtime) *cobra.Command {
	var flags lbFlags
	cmd := &cobra.Command{
		Use:   "update <load-balancer>",
		Short: "Change a load balancer's settings or targets",
		Long: `Change any setting except the name. Flags you leave out keep their current
values. --vm replaces the whole target list; use add-target and remove-target
to change one VM at a time. The command waits until the change is live.

gRPC load balancers use TCP health checks.`,
		Example: `  rumpty lb update web --health-path /ready --health-interval 5
  rumpty lb update web --vm web-1 --vm web-2 --vm web-3
  rumpty lb update web --sticky=false
  rumpty lb update web --allow 203.0.113.0/24 --allow 198.51.100.7/32
  rumpty lb update web --allow-all`,
		Args:              exactArgs(1, "missing load balancer name"),
		ValidArgsFunction: completeLoadBalancerNames(rt),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			s, err := flags.settings(cmd)
			if err != nil {
				return err
			}
			if s.VMs != nil && len(s.VMs) == 0 {
				return config.NewUsageError("--vm needs at least one VM")
			}
			if s.IsEmpty() {
				return config.NewUsageError("nothing to update; pass at least one flag (see --help)")
			}
			return lb.Update(cmd.Context(), rt, args[0], s)
		},
	}
	flags.register(cmd, rt, true)
	return cmd
}

func newLBDeleteCmd(rt *app.Runtime) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <load-balancer>",
		Aliases: []string{"rm"},
		Short:   "Delete a load balancer",
		Long: `Delete a load balancer and its routing. Billing stops at once. The VMs are not
changed. Asks for confirmation unless you pass --yes.`,
		Example: `  rumpty lb delete web
  rumpty lb delete web --yes`,
		Args:              exactArgs(1, "missing load balancer name"),
		ValidArgsFunction: completeLoadBalancerNames(rt),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return lb.Delete(cmd.Context(), rt, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Delete without asking for confirmation")
	return cmd
}

func newLBAddTargetCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-target <load-balancer> <vm>...",
		Short: "Send traffic to more VMs",
		Long:  "Add one or more VMs to a load balancer. VMs that are already targets are skipped. A load balancer can have at most 10 targets.",
		Example: `  rumpty lb add-target web web-3
  rumpty lb add-target web web-3 web-4`,
		Args: func(cmd *cobra.Command, args []string) error {
			switch len(args) {
			case 0:
				return config.NewUsageError("missing load balancer name")
			case 1:
				return config.NewUsageError("missing VM name")
			}
			return nil
		},
		ValidArgsFunction: completeLBThenVMs(rt, -1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return lb.AddTargets(cmd.Context(), rt, args[0], args[1:])
		},
	}
	return cmd
}

func newLBRemoveTargetCmd(rt *app.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove-target <load-balancer> <vm>",
		Short:             "Stop sending traffic to a VM",
		Long:              "Remove one VM from a load balancer. The VM itself is not changed.",
		Example:           `  rumpty lb remove-target web web-1`,
		Args:              exactArgs(2, "usage: rumpty lb remove-target <load-balancer> <vm>"),
		ValidArgsFunction: completeLBThenVMs(rt, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Config.ValidateForSSH(); err != nil {
				return config.NewUsageError("%v", err)
			}
			return lb.RemoveTarget(cmd.Context(), rt, args[0], args[1])
		},
	}
	return cmd
}

func exactArgs(n int, missing string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			return config.NewUsageError("%s", missing)
		}
		if len(args) > n {
			return config.NewUsageError("unexpected argument %q", args[n])
		}
		return nil
	}
}

func completeLoadBalancerNames(rt *app.Runtime) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return completeLBThenVMs(rt, 1)
}

// completeLBThenVMs completes a load balancer name first, then VM names up to
// maxArgs positional arguments in total (-1 for no limit).
func completeLBThenVMs(rt *app.Runtime, maxArgs int) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if maxArgs >= 0 && len(args) >= maxArgs {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var (
			names []string
			ok    bool
		)
		if len(args) == 0 {
			names, ok = loadBalancerSlugs(cmd, rt)
		} else {
			names, ok = vmSlugs(cmd, rt)
		}
		if !ok {
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func loadBalancerSlugs(cmd *cobra.Command, rt *app.Runtime) ([]string, bool) {
	rt.Config.Resolve()
	if err := rt.Config.ValidateForAuth(); err != nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), completionTimeout)
	defer cancel()

	workspace := strings.TrimSpace(rt.Config.Workspace)
	if workspace == "" {
		workspace = defaultWorkspace(ctx, rt)
	}
	if workspace == "" {
		return nil, false
	}

	lbs, err := rt.API().ListLoadBalancers(ctx, workspace)
	if err != nil {
		return nil, false
	}
	out := make([]string, 0, len(lbs))
	for i := range lbs {
		if s := strings.TrimSpace(lbs[i].Slug); s != "" {
			out = append(out, s)
		}
	}
	return out, true
}
